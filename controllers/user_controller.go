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
	"errors"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	"github.com/alex123012/database-users-operator/pkg/database"
	"github.com/alex123012/database-users-operator/pkg/utils"
)

const (
	userFinalizer = "user.databaseusersoperator.com/finalizer"
)

var ErrDatabaseConnect = errors.New("can't connect to database")

type databaseCreator func(ctx context.Context, s v1alpha1.DatabaseSpec, client client.Client, logger logr.Logger) (database.Database, error)

// UserReconciler reconciles a User object.
type UserReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	DatabaseCreator databaseCreator
}

//+kubebuilder:rbac:groups=databaseusersoperator.com,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=databaseusersoperator.com,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=databaseusersoperator.com,resources=users/finalizers,verbs=update
// +kubebuilder:rbac:groups=databaseusersoperator.com,resources=databases,verbs=get;list;watch
// +kubebuilder:rbac:groups=databaseusersoperator.com,resources=privileges,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("NAME", req.NamespacedName.Name, "NAMESPACE", req.NamespacedName.Namespace)

	user, err := r.user(ctx, req.NamespacedName, logger)
	if err != nil || user == nil {
		return ctrl.Result{}, err
	}

	// Check if the resource is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if user.GetDeletionTimestamp() != nil {
		logger.Info("Received deletion event")
		if !controllerutil.ContainsFinalizer(user, userFinalizer) {
			return ctrl.Result{}, nil
		}

		// Process deletetion user logic
		rec := r.databaseReconciler(user, true, logger)
		for _, dbRef := range user.Spec.Databases {
			if err := rec(ctx, dbRef); err != nil {
				return ctrl.Result{}, err
			}
		}
		logger.Info("Successfully deleted user from all specified databases")

		// Remove finalizer. Once all finalizers have been
		// removed, the object will be deleted.
		controllerutil.RemoveFinalizer(user, userFinalizer)
		return ctrl.Result{}, r.Update(ctx, user)
	}

	if !controllerutil.ContainsFinalizer(user, userFinalizer) {
		// Add finalizer for this CR
		logger.Info("Setting finalizer for resource")
		controllerutil.AddFinalizer(user, userFinalizer)
		if err := r.Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Process reconcile user logic
	rec := r.databaseReconciler(user, false, logger)
	for _, dbRef := range user.Spec.Databases {
		if err := rec(ctx, dbRef); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully created user in all specified databases")

	if !user.Status.Summary.Ready {
		user.Status.Summary = v1alpha1.StatusSummary{
			Ready:   true,
			Message: "",
		}
		err = r.Status().Update(ctx, user)
	}

	return ctrl.Result{}, err
}

func (r *UserReconciler) databaseReconciler(user *v1alpha1.User, deleteRequest bool, logger logr.Logger) func(ctx context.Context, dbRef v1alpha1.DatabaseRef) error {
	return func(ctx context.Context, dbRef v1alpha1.DatabaseRef) error {
		dbConfig, err := r.database(ctx, types.NamespacedName{Name: dbRef.Name}, logger)
		if err != nil {
			return err
		}

		privileges, err := r.privileges(ctx, dbRef.Privileges, logger)
		if err != nil {
			return err
		}

		db, err := r.DatabaseCreator(ctx, dbConfig.Spec, r.Client, logger)
		if err != nil {
			return errors.Join(ErrDatabaseConnect, err)
		}
		defer db.Close(ctx)

		f := r.databaseUserApply
		if deleteRequest {
			f = r.databaseUserDelete
		}
		return f(ctx, db, user, dbRef, privileges, logger)
	}
}

func (r *UserReconciler) user(ctx context.Context, nn types.NamespacedName, logger logr.Logger) (*v1alpha1.User, error) {
	user := &v1alpha1.User{}
	if err := r.Get(ctx, nn, user); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("User resource not found. Ignoring since object must be deleted.")
			return nil, nil
		}
		logger.Error(err, "unable to fetch User resource")
		return nil, err
	}

	return user, nil
}

func (r *UserReconciler) database(ctx context.Context, nn types.NamespacedName, _ logr.Logger) (*v1alpha1.Database, error) {
	db := &v1alpha1.Database{}
	if err := r.Get(ctx, nn, db); err != nil {
		return nil, err
	}
	return db, nil
}

func (r *UserReconciler) privileges(ctx context.Context, nns []v1alpha1.Name, _ logr.Logger) ([]v1alpha1.PrivilegeSpec, error) {
	var privileges []v1alpha1.PrivilegeSpec
	for _, nn := range nns {
		p := &v1alpha1.Privileges{}
		if err := r.Get(ctx, nn.ToNamespacedName(), p); err != nil {
			return nil, err
		}
		privileges = append(privileges, p.Privileges...)
	}
	return privileges, nil
}

func (r *UserReconciler) databaseUserApply(ctx context.Context, db database.Database, user *v1alpha1.User, dbRef v1alpha1.DatabaseRef, privileges []v1alpha1.PrivilegeSpec, logger logr.Logger) error {
	if err := r.createUserInDatabase(ctx, db, user, dbRef, logger); err != nil {
		return err
	}
	return db.ApplyPrivileges(ctx, user.Name, privileges)
}

func (r *UserReconciler) databaseUserDelete(ctx context.Context, db database.Database, user *v1alpha1.User, dbRef v1alpha1.DatabaseRef, privileges []v1alpha1.PrivilegeSpec, _ logr.Logger) error {
	defer func() {
		_ = r.Delete(ctx, newSecret(dbRef.CreatedSecret.ToNamespacedName(), nil))
	}()

	if err := db.RevokePrivileges(ctx, user.Name, privileges); err != nil {
		return err
	}

	return db.DeleteUser(ctx, user.Name)
}

func (r *UserReconciler) createUserInDatabase(ctx context.Context, db database.Database, user *v1alpha1.User, dbRef v1alpha1.DatabaseRef, _ logr.Logger) error {
	userPassword, err := r.userPassword(ctx, dbRef.PasswordSecret)
	if err != nil {
		return err
	}

	secretData, err := db.CreateUser(ctx, user.Name, userPassword)
	if err != nil || len(secretData) < 1 {
		return err
	}

	secretNN := types.NamespacedName{Namespace: dbRef.CreatedSecret.Namespace, Name: dbRef.CreatedSecret.Name}
	if _, err := utils.Secret(ctx, secretNN, r.Client); err == nil {
		return nil
	}

	secret := newSecret(secretNN, secretData)

	// TODO (alex123012): doesn't work GC (WHY???)
	if err := ctrl.SetControllerReference(user, secret, r.Scheme); err != nil {
		return err
	}

	return r.Create(ctx, secret)
}

func (r *UserReconciler) userPassword(ctx context.Context, secretCfg v1alpha1.Secret) (string, error) {
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
		return "", apierrors.NewInvalid(
			grSecret, secretCfg.Secret.Name,
			field.ErrorList{
				field.NotFound(field.NewPath("data", secretCfg.Key), secretCfg.Secret.Name),
			},
		)
	}
	return password, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.User{}).
		Owns(&v1.Secret{}).
		// Owns(&v1alpha1.Privileges{}).
		// Owns(&v1alpha1.Database{}).
		// Owns(&v1alpha1.User{}).
		// Watches(&source.Kind{Type: &v1alpha1.User{}}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}

func newSecret(nn types.NamespacedName, stringData map[string]string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nn.Name,
			Namespace: nn.Namespace,
		},
		StringData: stringData,
	}
}
