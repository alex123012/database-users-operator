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

package v1alpha1

import (
	"github.com/alex123012/database-users-operator/pkg/database"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var (
	configlog         = logf.Log.WithName("config-resource")
	postgreSQLField   = "postgreSQL"
	databaseTypeField = "databaseType"
	sslSecretsField   = "sslSecrets"
)

func (r *Config) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-auth-alex123012-com-v1alpha1-config,mutating=true,failurePolicy=fail,sideEffects=None,groups=auth.alex123012.com,resources=configs,verbs=create;update,versions=v1alpha1,name=mconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Config{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Config) Default() {
	configlog.Info("default", "name", r.Name)
	if r.Spec.DatabaseType == PostgreSQL {
		if r.Spec.PostgreSQL.PasswordSecret.Name != "" && r.Spec.PostgreSQL.PasswordSecret.Namespace == "" {
			r.Spec.PostgreSQL.PasswordSecret.Namespace = r.GetNamespace()
		}
		if r.Spec.PostgreSQL.SSLCredentials.UserSecret.Name != "" && r.Spec.PostgreSQL.SSLCredentials.UserSecret.Namespace == "" {
			r.Spec.PostgreSQL.SSLCredentials.UserSecret.Namespace = r.GetNamespace()
		}
		if r.Spec.PostgreSQL.SSLCredentials.CASecret.Name != "" && r.Spec.PostgreSQL.SSLCredentials.CASecret.Namespace == "" {
			r.Spec.PostgreSQL.SSLCredentials.CASecret.Namespace = r.GetNamespace()
		}
		if r.Spec.PostgreSQL.Port == 0 {
			r.Spec.PostgreSQL.Port = 5432
		}
		if r.Spec.PostgreSQL.User == "" {
			r.Spec.PostgreSQL.User = "postgres"
		}
		if r.Spec.PostgreSQL.SSLMode == "" {
			r.Spec.PostgreSQL.SSLMode = database.SSLModeDISABLE
		}
	}
}

//+kubebuilder:webhook:path=/validate-auth-alex123012-com-v1alpha1-config,mutating=false,failurePolicy=fail,sideEffects=None,groups=auth.alex123012.com,resources=configs,verbs=create;update,versions=v1alpha1,name=vconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Config{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Config) ValidateCreate() error {
	configlog.Info("validate create", "name", r.Name)
	return r.validateConfig()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Config) ValidateUpdate(old runtime.Object) error {
	configlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Config) ValidateDelete() error {
	configlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *Config) validateConfig() error {
	var allErrs field.ErrorList
	if err := r.validatePostgresConfig(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		GroupVersion.WithKind("User").GroupKind(),
		r.Name, allErrs)
}

func (r *Config) validatePostgresConfig() *field.Error {
	if r.Spec.DatabaseType == PostgreSQL {

		switch r.Spec.PostgreSQL.SSLMode {
		case database.SSLModeVERIFYCA, database.SSLModeREQUIRE, database.SSLModeVERIFYFULL:
			if r.Spec.PostgreSQL.SSLCredentials == (SSLSecrets{}) ||
				r.Spec.PostgreSQL.SSLCredentials.CASecret.Name == "" ||
				r.Spec.PostgreSQL.SSLCredentials.UserSecret.Name == "" {
				return field.Required(specField.Child(postgreSQLField).Child(sslSecretsField),
					"SSL credentials must be specified when using SSL for connecting to db")
			}
		case database.SSLModeDISABLE, database.SSLModePREFER, database.SSLModeALLOW:
			if r.Spec.PostgreSQL.PasswordSecret == (Secret{}) ||
				r.Spec.PostgreSQL.PasswordSecret.Name == "" {

				return field.Required(specField.Child(postgreSQLField).Child(passwordSecretField),
					"password secret must be provided when using no ssl for connecting to db")

			}
		}
	} else {
		return field.Invalid(specField.Child(databaseTypeField), r.Spec.DatabaseType,
			"Not a valid database type")
	}
	return nil
}
