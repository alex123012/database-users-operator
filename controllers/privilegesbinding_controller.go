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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
)

const (
	privBindingFinalizer = "privilegebinding.databaseusersoperator.com/finalizer"
)

// PrivilegesBindingReconciler reconciles a PrivilegesBinding object.
type PrivilegesBindingReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	DatabaseCreator databaseCreator
}

//+kubebuilder:rbac:groups=databaseusersoperator.com,resources=privilegesbindings,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=databaseusersoperator.com,resources=privilegesbindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=databaseusersoperator.com,resources=privilegesbindings/finalizers,verbs=update
// +kubebuilder:rbac:groups=databaseusersoperator.com,resources=databasebindings,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=databaseusersoperator.com,resources=privileges,verbs=get;list;watch
// +kubebuilder:rbac:groups=databaseusersoperator.com,resources=users,verbs=get;list;watch
// +kubebuilder:rbac:groups=databaseusersoperator.com,resources=databases,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *PrivilegesBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("NAME", req.NamespacedName.Name, "NAMESPACE", req.NamespacedName.Namespace)

	privBinding, err := r.privilegesBinding(ctx, req.NamespacedName, logger)
	if err != nil || privBinding == nil {
		return ctrl.Result{}, err
	}

	privileges, err := r.privileges(ctx, privBinding.Spec.Privileges, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	dbBindings, err := r.databaseUsers(ctx, privBinding.Spec.DatabaseBindings, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check if the resource is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if privBinding.GetDeletionTimestamp() != nil {
		logger.Info("Received deletion event")
		if !controllerutil.ContainsFinalizer(privBinding, privBindingFinalizer) {
			return ctrl.Result{}, nil
		}

		for _, dbBinding := range dbBindings {
			if err := r.revokePrivileges(ctx, privBinding, dbBinding, privileges.Privileges, logger); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Remove finalizer. Once all finalizers have been
		// removed, the object will be deleted.
		controllerutil.RemoveFinalizer(privBinding, privBindingFinalizer)
		return ctrl.Result{}, r.Update(ctx, privBinding)
	}

	if !controllerutil.ContainsFinalizer(privBinding, privBindingFinalizer) {
		// Add finalizer for this CR
		logger.Info("Setting finalizer for resource")
		controllerutil.AddFinalizer(privBinding, privBindingFinalizer)
		if err := r.Update(ctx, privBinding); err != nil {
			return ctrl.Result{}, err
		}
	}

	for _, dbBinding := range dbBindings {
		if err := r.applyPrivileges(ctx, privBinding, dbBinding, privileges.Privileges, logger); err != nil {
			return ctrl.Result{}, err
		}
	}

	privBinding.Status.Summary = v1alpha1.StatusSummary{
		Ready:   true,
		Message: "",
	}

	return ctrl.Result{}, r.Status().Update(ctx, privBinding)
}

type databaseUserBinding struct {
	dbConfig *v1alpha1.Database
	user     *v1alpha1.User
}

func (r *PrivilegesBindingReconciler) privilegesBinding(ctx context.Context, nn types.NamespacedName, logger logr.Logger) (*v1alpha1.PrivilegesBinding, error) {
	privBinding := &v1alpha1.PrivilegesBinding{}
	if err := r.Get(ctx, nn, privBinding); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("PrivilegesBinding resource not found. Ignoring since object must be deleted.")
			return nil, nil
		}
		logger.Error(err, "unable to fetch PrivilegesBinding resource")
		return nil, err
	}

	return privBinding, nil
}

func (r *PrivilegesBindingReconciler) privileges(ctx context.Context, nn v1alpha1.NamespacedName, _ logr.Logger) (*v1alpha1.Privileges, error) {
	privileges := &v1alpha1.Privileges{}
	if err := r.Get(ctx, types.NamespacedName(nn), privileges); err != nil {
		return nil, err
	}
	return privileges, nil
}

func (r *PrivilegesBindingReconciler) databaseUsers(ctx context.Context, nns []v1alpha1.NamespacedName, _ logr.Logger) ([]databaseUserBinding, error) {
	dbBindings := make([]databaseUserBinding, 0, len(nns))
	for _, nn := range nns {
		dbBinding := &v1alpha1.DatabaseBinding{}
		if err := r.Get(ctx, types.NamespacedName(nn), dbBinding); err != nil {
			// TODO (alex123012): ????Don't fail all reconciliation because of one deleted DatabaseBinding????
			// logger.Error(err, "can't get DatabaseBinding")
			// continue
			return nil, err
		}

		user := &v1alpha1.User{}
		if err := r.Get(ctx, types.NamespacedName(dbBinding.Spec.User), user); err != nil {
			return nil, err
		}

		dbConfig := &v1alpha1.Database{}
		if err := r.Get(ctx, types.NamespacedName(dbBinding.Spec.Database), dbConfig); err != nil {
			return nil, err
		}
		dbBindings = append(dbBindings, databaseUserBinding{user: user, dbConfig: dbConfig})
	}
	return dbBindings, nil
}

func (r *PrivilegesBindingReconciler) revokePrivileges(ctx context.Context, _ *v1alpha1.PrivilegesBinding, dbBinding databaseUserBinding, privileges []v1alpha1.PrivilegeSpec, logger logr.Logger) error {
	db, err := r.DatabaseCreator(ctx, dbBinding.dbConfig.Spec, r.Client, logger)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	return db.RevokePrivileges(ctx, dbBinding.user.Name, privileges)
}

func (r *PrivilegesBindingReconciler) applyPrivileges(ctx context.Context, _ *v1alpha1.PrivilegesBinding, dbBinding databaseUserBinding, privileges []v1alpha1.PrivilegeSpec, logger logr.Logger) error {
	db, err := r.DatabaseCreator(ctx, dbBinding.dbConfig.Spec, r.Client, logger)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	return db.ApplyPrivileges(ctx, dbBinding.user.Name, privileges)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrivilegesBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PrivilegesBinding{}).
		Complete(r)
}
