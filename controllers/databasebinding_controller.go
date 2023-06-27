/*
Copyright 2023.

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
	"strings"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/pkg/utils"
)

const (
	dbBindingFinalizer = "databasebinding.databaseusersoperator.com/finalizer"
)

// DatabaseBindingReconciler reconciles a DatabaseBinding object.
type DatabaseBindingReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	DatabaseCreator databaseCreator
}

//+kubebuilder:rbac:groups=databaseusersoperator.com,resources=databasebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=databaseusersoperator.com,resources=databasebindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=databaseusersoperator.com,resources=databasebindings/finalizers,verbs=update
// +kubebuilder:rbac:groups=databaseusersoperator.com,resources=users,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=databaseusersoperator.com,resources=databases,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *DatabaseBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("NAME", req.NamespacedName.Name, "NAMESPACE", req.NamespacedName.Namespace)

	dbBinding, err := r.databaseBinding(ctx, req.NamespacedName, logger)
	if err != nil || dbBinding == nil {
		return ctrl.Result{}, err
	}

	user, err := r.user(ctx, dbBinding.Spec.User, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	dbConfig, err := r.database(ctx, dbBinding.Spec.Database, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check if the resource is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if dbBinding.GetDeletionTimestamp() != nil {
		logger.Info("Received deletion event")
		if !controllerutil.ContainsFinalizer(dbBinding, dbBindingFinalizer) {
			return ctrl.Result{}, nil
		}

		if err := r.deleteUserInDatabase(ctx, dbBinding, user, dbConfig, logger); err != nil {
			return ctrl.Result{}, err
		}

		// Remove finalizer. Once all finalizers have been
		// removed, the object will be deleted.
		controllerutil.RemoveFinalizer(dbBinding, dbBindingFinalizer)
		return ctrl.Result{}, r.Update(ctx, dbBinding)
	}

	if err := r.createUserInDatabase(ctx, dbBinding, user, dbConfig, logger); err != nil {
		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(dbBinding, dbBindingFinalizer) {
		// Add finalizer for this CR
		logger.Info("Setting finalizer for resource")
		controllerutil.AddFinalizer(dbBinding, dbBindingFinalizer)
		if err := r.Update(ctx, dbBinding); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully created user")
	dbBinding.Status.Summary = v1alpha1.StatusSummary{
		Ready:   true,
		Message: "",
	}

	return ctrl.Result{}, r.Status().Update(ctx, dbBinding)
}

func (r *DatabaseBindingReconciler) databaseBinding(ctx context.Context, nn types.NamespacedName, logger logr.Logger) (*v1alpha1.DatabaseBinding, error) {
	dbBinding := &v1alpha1.DatabaseBinding{}
	if err := r.Get(ctx, nn, dbBinding); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("DatabaseBinding resource not found. Ignoring since object must be deleted.")
			return nil, nil
		}
		logger.Error(err, "unable to fetch DatabaseBinding resource")
		return nil, err
	}

	return dbBinding, nil
}

func (r *DatabaseBindingReconciler) user(ctx context.Context, nn v1alpha1.NamespacedName, _ logr.Logger) (*v1alpha1.User, error) {
	user := &v1alpha1.User{}
	if err := r.Get(ctx, types.NamespacedName(nn), user); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *DatabaseBindingReconciler) database(ctx context.Context, nn v1alpha1.NamespacedName, _ logr.Logger) (*v1alpha1.Database, error) {
	db := &v1alpha1.Database{}
	if err := r.Get(ctx, types.NamespacedName(nn), db); err != nil {
		return nil, err
	}
	return db, nil
}

func (r *DatabaseBindingReconciler) createUserInDatabase(ctx context.Context, dbBinding *v1alpha1.DatabaseBinding, user *v1alpha1.User, dbConfig *v1alpha1.Database, logger logr.Logger) error {
	db, err := r.DatabaseCreator(ctx, dbConfig.Spec, r.Client, logger)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	password, err := r.userPassword(ctx, user)
	if err != nil {
		return err
	}

	secretData, err := db.CreateUser(ctx, user.Name, password)
	if err != nil || len(secretData) < 1 {
		return err
	}

	secretName := strings.Join([]string{user.Name, dbConfig.Name, "data"}, "-")
	secret := newSecret(secretName, user.Namespace, secretData)

	if err := ctrl.SetControllerReference(dbBinding, secret, r.Scheme); err != nil {
		return err
	}
	return r.Create(ctx, secret)
}

func (r *DatabaseBindingReconciler) deleteUserInDatabase(ctx context.Context, _ *v1alpha1.DatabaseBinding, user *v1alpha1.User, dbConfig *v1alpha1.Database, logger logr.Logger) error {
	db, err := r.DatabaseCreator(ctx, dbConfig.Spec, r.Client, logger)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	return db.DeleteUser(ctx, user.Name)
}

func (r *DatabaseBindingReconciler) userPassword(ctx context.Context, user *v1alpha1.User) (string, error) {
	secretCfg := user.PasswordSecret
	if secretCfg.Key == "" || secretCfg.Secret.Name == "" || secretCfg.Secret.Namespace == "" {
		return "", nil
	}

	data, err := utils.DecodeSecretData(ctx, types.NamespacedName(secretCfg.Secret), r.Client)
	if err != nil {
		return "", err
	}

	password, ok := data[secretCfg.Key]
	if !ok {
		grSecret := schema.ParseGroupKind("v1.Secret")
		return "", errors.NewInvalid(
			grSecret, secretCfg.Secret.Name,
			field.ErrorList{
				field.NotFound(field.NewPath("data", secretCfg.Key), secretCfg.Secret.Name),
			},
		)
	}
	return password, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DatabaseBinding{}).
		Owns(&v1alpha1.Database{}).
		Owns(&v1alpha1.User{}).
		Complete(r)
}

func newSecret(name, namespace string, stringData map[string]string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: stringData,
	}
}
