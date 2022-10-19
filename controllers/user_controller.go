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
	standardErrors "errors"
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
	"github.com/alex123012/k8s-database-users-operator/pkg/database"
	"github.com/alex123012/k8s-database-users-operator/pkg/utils"
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
		logger.Error(err, "unable to fetch User'"+userResource.Name+"' in namespace '"+userResource.Namespace+"'")
		return ctrl.Result{}, err
	}

	configResource := &authv1alpha1.Config{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      userResource.Spec.DatabaseConfig.Name,
		Namespace: userResource.Spec.DatabaseConfig.Namespace,
	}, configResource); err != nil {
		logger.Error(err, "unable to fetch Config'"+userResource.Spec.DatabaseConfig.Name+"' in namespace '"+userResource.Spec.DatabaseConfig.Namespace+"'")
		return ctrl.Result{}, err
	}

	if _, err := r.getV1Secret(ctx, userResource.Name, userResource.Namespace, logger); errors.IsNotFound(err) {

		logger.Info("Creating DB user for User resource '" + userResource.Name + "' in namespace '" + userResource.Namespace + "'")
		if err := r.processUser(ctx, userResource, configResource, logger); standardErrors.Is(err, database.ErrAlreadyExists) || errors.IsAlreadyExists(err) {
			return ctrl.Result{}, nil
		} else if err != nil {
			logger.Error(err, "Failed to create db user for User resource '"+userResource.Name+"' in namespace '"+userResource.Namespace+"'")
			return ctrl.Result{}, err
		}

	} else if err != nil {
		logger.Error(err, "Failed to get new v1.Secret '"+userResource.Name+"' in namespace '"+userResource.Namespace+"'")
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

func (r *UserReconciler) createV1Secret(ctx context.Context, userResource *authv1alpha1.User, data map[string][]byte, logger logr.Logger) error {
	logger.Info("Creating a new v1.Secret '" + userResource.Name + "' in namespace '" + userResource.Namespace + "'")
	secretV1Resource := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userResource.Name,
			Namespace: userResource.Namespace,
		},
		Data: data,
		Type: v1.SecretTypeOpaque,
	}
	if err := ctrl.SetControllerReference(userResource, secretV1Resource, r.Scheme); err != nil {
		logger.Error(err, "Error setting reference for v1.Secret from'"+userResource.Name+"' in namespace '"+userResource.Namespace+"'")
		return err
	}
	return r.Client.Create(ctx, secretV1Resource)
}

func (r *UserReconciler) processUser(ctx context.Context, userResource *authv1alpha1.User, configResource *authv1alpha1.Config, logger logr.Logger) error {
	switch configResource.Spec.DatabaseType {
	case authv1alpha1.CockroachDB:
		cockroachConfig := &configResource.Spec.CockroachDB
		err := r.createCockroachDBUser(ctx, userResource, cockroachConfig, logger)
		if err != nil {
			return err
		}

		logger.Info("Generating certificates for User resource '" + userResource.Name + "' in namespace '" + userResource.Namespace + "'")
		data, err := r.generateCockroachDBCertificatesForUser(ctx, userResource, cockroachConfig, logger)
		if err != nil {
			logger.Error(err, "Failed to generate new certificates for User '"+userResource.Name+"' in namespace '"+userResource.Namespace+"'")
			return err
		}

		logger.Info("Creating v1.Secret for User resource '" + userResource.Name + "' in namespace '" + userResource.Namespace + "'")
		if err := r.createV1Secret(ctx, userResource, data, logger); err != nil {
			logger.Error(err, "Failed to create new v1.Secret '"+userResource.Name+"' in namespace '"+userResource.Namespace+"'")
			return err
		}
		return nil
	default:
		return fmt.Errorf("no Such database type")
	}

}

func (r *UserReconciler) createCockroachDBUser(ctx context.Context, userResource *authv1alpha1.User, configResource *authv1alpha1.PostgreSQLConfig, logger logr.Logger) error {
	cockroachRootSecret, err := r.getV1Secret(
		ctx,
		configResource.SSLCredentials.UserSecret.Name,
		configResource.SSLCredentials.UserSecret.Namespace,
		logger,
	)
	if err != nil {
		return err
	}

	caFile := utils.FilePathFromHome("cockroach-certs/ca.crt")
	clientCert := utils.FilePathFromHome(fmt.Sprintf("cockroach-certs/client.%s.crt", configResource.User))
	clientKey := utils.FilePathFromHome(fmt.Sprintf("cockroach-certs/client.%s.key", configResource.User))

	if err := utils.CreateFileFromBytes(caFile, cockroachRootSecret.Data["ca.crt"]); err != nil {
		return err
	}
	defer utils.DeleteFile(caFile)

	if err := utils.CreateFileFromBytes(clientCert, cockroachRootSecret.Data["tls.crt"]); err != nil {
		return err
	}
	defer utils.DeleteFile(clientCert)

	if err := utils.CreateFileFromBytes(clientKey, cockroachRootSecret.Data["tls.key"]); err != nil {
		return err
	}
	defer utils.DeleteFile(clientKey)

	postgresConfig := database.NewPostgresConfig(
		fmt.Sprintf("%s.%s.svc.cluster.local", configResource.Host, configResource.Namespace),
		26257,
		configResource.User,
		"", "", database.SSLModeVERIFYFULL, caFile, clientCert, clientKey,
	)
	fmt.Println(postgresConfig.String())
	db, err := database.NewDBConnection(postgresConfig, database.DBDriverPostgres)
	if err != nil {
		return err
	}
	defer db.Close()

	sqlCreateUser := fmt.Sprintf(`CREATE USER "%s";`, userResource.Spec.Name)
	_, err = db.ExecContext(ctx, sqlCreateUser)
	if database.IgnoreAlreadyExists(database.ProcessPostgresError(err)) != nil {
		return err
	}

	for _, priv := range userResource.Spec.Privileges {
		sqlGrant := fmt.Sprintf(`GRANT "%s" TO "%s";`, priv.Privilege, userResource.Spec.Name)
		if priv.On != "" {
			sqlGrant = fmt.Sprintf(`GRANT "%s" ON "%s" TO "%s";`, priv.Privilege, priv.On, userResource.Spec.Name)
		}

		if _, err := db.ExecContext(ctx, sqlGrant); database.ProcessPostgresError(err) != nil {
			return err
		}
	}
	return nil
}
func (r *UserReconciler) generateCockroachDBCertificatesForUser(ctx context.Context, userResource *authv1alpha1.User, configResource *authv1alpha1.PostgreSQLConfig, logger logr.Logger) (map[string][]byte, error) {

	cockroachRootSecret, err := r.getV1Secret(ctx, configResource.SSLCredentials.UserSecret.Name, configResource.SSLCredentials.CASecret.Namespace, logger)
	if err != nil {
		return nil, err
	}
	caCert, err := utils.ByteToCaCert(cockroachRootSecret.Data["ca.crt"])
	if err != nil {
		return nil, err
	}

	cockroachCAKeySecret, err := r.getV1Secret(ctx, configResource.SSLCredentials.CASecret.Name, configResource.SSLCredentials.CASecret.Namespace, logger)
	if err != nil {
		return nil, err
	}
	// user cert config
	caPrivKey, err := utils.ByteToCaPrivateKey(cockroachCAKeySecret.Data["ca.key"])
	if err != nil {
		return nil, err
	}

	return utils.GenCockroachCertFromCA(userResource.Spec.Name, caCert, caPrivKey)
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.User{}).
		Complete(r)
}
