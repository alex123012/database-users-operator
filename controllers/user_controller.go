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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authv1alpha1 "github.com/alex123012/k8s-database-users-operator/api/v1alpha1"
	"github.com/go-logr/logr"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=auth.alex123012.com,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=auth.alex123012.com,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=auth.alex123012.com,resources=users/finalizers,verbs=update
//+kubebuilder:rbac:groups=v1,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the User object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	userResource := &authv1alpha1.User{}
	if err := client.IgnoreNotFound(r.Get(ctx, req.NamespacedName, userResource)); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		logger.Error(err, "unable to fetch Job")
		return ctrl.Result{}, err
	}
	if _, err := r.getV1Secret(ctx, userResource.Name, userResource.Namespace, logger); err == nil && errors.IsNotFound(err) {

		err := r.createV1Secret(ctx, userResource, map[string]string{}, logger)
		if err != nil {
			logger.Error(err, "Failed to create new v1.Secret '"+userResource.Name+"' in namespace '"+userResource.Namespace+"'")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *UserReconciler) getV1Secret(ctx context.Context, name, namespace string, logger logr.Logger) (*v1.Secret, error) {
	secretV1Resource := &v1.Secret{}

	logger.Info("Trying to get v1.Secret with name '" + name + "' in namespace '" + namespace + "'")
	// Check if the job already exists, if not create a new one
	err := r.Client.Get(
		ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		secretV1Resource)
	return secretV1Resource, err
}

func (r *UserReconciler) createV1Secret(ctx context.Context, userResource *authv1alpha1.User, data map[string]string, logger logr.Logger) error {
	logger.Info("Creating a new v1.Secret '" + userResource.Name + "' in namespace '" + userResource.Namespace + "'")
	secretV1Resource := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userResource.Name,
			Namespace: userResource.Namespace,
		},
		StringData: data,
		Type:       v1.SecretTypeOpaque,
	}
	if err := ctrl.SetControllerReference(userResource, secretV1Resource, r.Scheme); err != nil {
		logger.Error(err, "Error setting reference for v1.Secret from'"+userResource.Name+"' in namespace '"+userResource.Namespace+"'")
		return err
	}
	return r.Client.Create(ctx, secretV1Resource)
}

func (r *UserReconciler) createDBUser(ctx context.Context, userResource *authv1alpha1.User, logger logr.Logger) error {
	switch userResource.Spec.Cluster.Type {
	case authv1alpha1.CockroachDB:
		return r.createCockroachDBUser(ctx, userResource, logger)
	default:
		return fmt.Errorf("no Such ClusterType")
	}
}

func (r *UserReconciler) createCockroachDBUser(ctx context.Context, userResource *authv1alpha1.User, logger logr.Logger) error {
	cockroachUserSecret, err := r.getV1Secret(
		ctx,
		userResource.Spec.Cluster.Credentials.Secret.Name,
		userResource.Spec.Cluster.Credentials.Secret.Namespace,
		logger,
	)
	if err != nil {
		return err
	}

	caFile := FilePathFromHome("cockroach-certs/ca.crt")
	clientCert := FilePathFromHome(fmt.Sprintf("cockroach-certs/client.%s.crt", userResource.Spec.Cluster.Credentials.Username))
	clientKey := FilePathFromHome(fmt.Sprintf("cockroach-certs/client.%s.key", userResource.Spec.Cluster.Credentials.Username))

	if err := CreateFileFromBytes(caFile, cockroachUserSecret.Data["ca.crt"]); err != nil {
		return err
	}
	if err := CreateFileFromBytes(clientCert, cockroachUserSecret.Data["tls.key"]); err != nil {
		return err
	}
	if err := CreateFileFromBytes(clientKey, cockroachUserSecret.Data["tls.crt"]); err != nil {
		return err
	}
	postgresConfig := NewPostgresConfig(
		fmt.Sprintf("%s-public", userResource.Spec.Cluster.Name),
		26257,
		userResource.Spec.Cluster.Credentials.Username,
		"", "", SSLModeVERIFYFULL, caFile, clientCert, clientKey,
	)
	db, err := NewDBConnection(postgresConfig, DBDriverPostgres)
	if err != nil {
		return err
	}
	defer db.Close()
	sqlStatement := `
INSERT INTO users (age, email, first_name, last_name)
VALUES ($1, $2, $3, $4)
RETURNING id`
	id := 0
	if err := db.QueryRow(sqlStatement, 30, "jon@calhoun.io", "Jonathan", "Calhoun").Scan(&id); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.User{}).
		Complete(r)
}
