/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authv1alpha1 "github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/pkg/common"
	"github.com/alex123012/database-users-operator/pkg/database/postgresql"
	"github.com/go-logr/logr"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const userFinalizer = "auth.alex123012.com/finalizer"

//+kubebuilder:rbac:groups=auth.alex123012.com,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=auth.alex123012.com,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=auth.alex123012.com,resources=users/finalizers,verbs=update
//+kubebuilder:rbac:groups=v1,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	userNameNamespace := []interface{}{"NAME", req.NamespacedName.Name, "NAMESPACE", req.NamespacedName.Namespace}
	logger := log.FromContext(ctx).WithValues(userNameNamespace...)
	errorGroup, errorGroupCtx := errgroup.WithContext(ctx)

	userResource := &authv1alpha1.User{}
	if err := r.Get(ctx, req.NamespacedName, userResource); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			logger.Info("User resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "unable to fetch User resource")
		return ctrl.Result{}, err
	}

	// Check if the User resource is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isUserMarkedToBeDeleted := userResource.GetDeletionTimestamp() != nil
	if isUserMarkedToBeDeleted {
		if err := r.processDeleteRequest(ctx, errorGroupCtx, userResource, errorGroup, logger); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	} else {
		if err := r.setUserDeleteDBUserFinalizer(ctx, userResource, logger); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.processCreateUpdateRequest(errorGroupCtx, userResource, errorGroup, logger); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil // Requeue: true, RequeueAfter: time.Minute
}

func (r *UserReconciler) GetV1Secret(ctx context.Context, name, namespace string, logger logr.Logger) (*v1.Secret, error) {
	secretV1Resource := &v1.Secret{}

	logger.Info("Trying to get v1.Secret with name '" + name + "' in namespace '" + namespace + "'")
	err := r.Client.Get(
		ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		secretV1Resource)
	if err != nil {
		logger.Info("Failed to get new v1.Secret '"+name+"' in namespace '"+namespace+"'", "ERROR", err)
		return nil, err
	}

	logger.Info("Getted v1.Secret with name '" + name + "' in namespace '" + namespace + "'")
	return secretV1Resource, nil
}

func (r *UserReconciler) CreateV1Secret(ctx context.Context, userResource *authv1alpha1.User, data map[string][]byte, logger logr.Logger) error {
	// userResource = userResource.(*authv1alpha1.User)
	logger.Info("Creating a new v1.Secret")
	secretV1Resource := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userResource.GetName(),
			Namespace: userResource.GetNamespace(),
		},
		Data: data,
		Type: v1.SecretTypeOpaque,
	}
	if err := ctrl.SetControllerReference(userResource, secretV1Resource, r.Scheme); err != nil {
		logger.Error(err, "Error setting reference for v1.Secret from User resource")
		return err
	}

	if err := r.Client.Create(ctx, secretV1Resource); err != nil {
		logger.Error(err, "Failed to create new v1.Secret")
		return err
	}
	return nil
}

func (r *UserReconciler) DeleteV1Secret(ctx context.Context, secretResource *v1.Secret, logger logr.Logger) error {

	logger.Info("Trying to delete v1.Secret")
	if err := r.Client.Delete(ctx, secretResource); err != nil {
		logger.Error(err, "Failed to create new v1.Secret '"+secretResource.GetName()+"' in namespace '"+secretResource.GetNamespace()+"'")
		return err
	}
	return nil
}

func (r *UserReconciler) processCreateUpdateRequest(errorGroupCtx context.Context, userResource *authv1alpha1.User, errorGroup *errgroup.Group, logger logr.Logger) error {

	r.runErrGroup(errorGroupCtx, userResource, errorGroup, logger, r.processUserWithConfig)
	if err := errorGroup.Wait(); err != nil {
		logger.Error(err, "Failed to create db user for User resource")
		return err
	}
	logger.Info("Created/Updated DB user for User resource")
	return nil
}

func (r *UserReconciler) processDeleteRequest(ctx, errorGroupCtx context.Context, userResource *authv1alpha1.User, errorGroup *errgroup.Group, logger logr.Logger) error {
	logger.Info("Received deletion event")
	if controllerutil.ContainsFinalizer(userResource, userFinalizer) {
		// Run finalization logic for userFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		r.runErrGroup(errorGroupCtx, userResource, errorGroup, logger, r.deleteUserWithConfig)
		if err := errorGroup.Wait(); err != nil {
			logger.Error(err, "Failed to delete db user for User resource")
			return err
		}
		// Remove userFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		controllerutil.RemoveFinalizer(userResource, userFinalizer)

		if err := r.Update(ctx, userResource); err != nil {
			return err
		}
	}
	return nil
}

func (r *UserReconciler) runErrGroup(ctx context.Context, userResource *authv1alpha1.User,
	errorGroup *errgroup.Group,
	logger logr.Logger,
	fn func(context.Context, *authv1alpha1.User, authv1alpha1.DatabaseConfig, logr.Logger) error,
) error {
	for _, dbConfig := range userResource.Spec.DatabaseConfig {
		tmpDbconfig := dbConfig
		errorGroup.Go(func() error {
			return fn(ctx, userResource, tmpDbconfig, logger.WithValues("DATABASE", tmpDbconfig.Name))
		})
	}
	if err := errorGroup.Wait(); err != nil {
		logger.Error(err, "Failed to reconcile db user for User resource")
		return err
	}
	return nil
}

func (r *UserReconciler) getConfigResource(ctx context.Context, dbConfig authv1alpha1.DatabaseConfig, logger logr.Logger) (*authv1alpha1.Config, error) {
	configResource := &authv1alpha1.Config{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      dbConfig.Name,
		Namespace: dbConfig.Namespace,
	}, configResource); err != nil {
		logger.Error(err, "unable to fetch Config'"+dbConfig.Name+"' in namespace '"+dbConfig.Namespace+"'")
		return nil, err
	}
	return configResource, nil
}

func (r *UserReconciler) processUserWithConfig(ctx context.Context, userResource *authv1alpha1.User, dbConfig authv1alpha1.DatabaseConfig, logger logr.Logger) error {
	logger.Info("Creating DB user for User resource in Database")
	db, err := r.getNewDbInterface(ctx, userResource, dbConfig, logger)
	if err != nil {
		return err
	}
	return db.ProcessUser(ctx)
}

func (r *UserReconciler) deleteUserWithConfig(ctx context.Context, userResource *authv1alpha1.User, dbConfig authv1alpha1.DatabaseConfig, logger logr.Logger) error {
	db, err := r.getNewDbInterface(ctx, userResource, dbConfig, logger)
	if err != nil {
		return err
	}
	return db.DeleteUser(ctx)
}

func (r *UserReconciler) getNewDbInterface(ctx context.Context, userResource *authv1alpha1.User, dbConfig authv1alpha1.DatabaseConfig, logger logr.Logger) (common.DatabaseInterface, error) {
	configResource, err := r.getConfigResource(ctx, dbConfig, logger)
	if err != nil {
		return nil, err
	}
	var db common.DatabaseInterface
	switch configResource.Spec.DatabaseType {
	case authv1alpha1.PostgreSQL:
		db = postgresql.NewPostgresFromConfig(configResource, userResource, r, logger)
	default:
		return nil, fmt.Errorf("no Such database type")
	}
	return db, nil
}

func (r *UserReconciler) setUserDeleteDBUserFinalizer(ctx context.Context, userResource *authv1alpha1.User, logger logr.Logger) error {
	if !controllerutil.ContainsFinalizer(userResource, userFinalizer) {
		// Add finalizer for this CR
		logger.Info("Setting finalizer for user resource")
		controllerutil.AddFinalizer(userResource, userFinalizer)
		if err := r.Update(ctx, userResource); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.User{}).
		Complete(r)
}
