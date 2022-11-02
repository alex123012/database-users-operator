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
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	nameField            = "name"
	namespaceField       = "namespace"
	passwordSecretField  = "passwordSecret"
	databaseConfigsField = "databaseConfigs"
	privilegesField      = "privileges"

	privilegeField = "privilege"
	onField        = "on"
	databaseField  = "database"
)

var (
	specField = field.NewPath("spec")
)

// log is for logging in this package.
var userlog = logf.Log.WithName("user-resource")

func (r *User) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-auth-alex123012-com-v1alpha1-user,mutating=true,failurePolicy=fail,sideEffects=None,groups=auth.alex123012.com,resources=users,verbs=create;update,versions=v1alpha1,name=muser.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &User{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *User) Default() {
	userlog.Info("default", "name", r.Name)
	for i := range r.Spec.DatabaseConfigs {
		if dbconfig := &r.Spec.DatabaseConfigs[i]; dbconfig.Name != "" && dbconfig.Namespace == "" {
			dbconfig.Namespace = r.GetNamespace()
		}
	}
	if r.Spec.PasswordSecret.Name != "" && r.Spec.PasswordSecret.Namespace == "" {
		r.Spec.PasswordSecret.Namespace = r.GetNamespace()
	}
}

//+kubebuilder:webhook:path=/validate-auth-alex123012-com-v1alpha1-user,mutating=false,failurePolicy=fail,sideEffects=None,groups=auth.alex123012.com,resources=users,verbs=create;update,versions=v1alpha1,name=vuser.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &User{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *User) ValidateCreate() error {
	userlog.Info("validate create", "name", r.Name)
	return r.validateUser()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *User) ValidateUpdate(old runtime.Object) error {
	userlog.Info("validate update", "name", r.Name)
	return r.validateUser()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *User) ValidateDelete() error {
	userlog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *User) validateUser() error {
	var allErrs field.ErrorList
	if err := r.validateUserPasswordSecret(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := r.validateUserDatabaseConfigs(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := r.validateUserPrivileges(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		GroupVersion.WithKind("User").GroupKind(),
		r.Name, allErrs)
}

func (r *User) validateUserPasswordSecret() *field.Error {
	// if r.Spec.PasswordSecret.Name != "" && r.Spec.PasswordSecret.Namespace == "" {
	// 	return field.Required(specField.Child(passwordSecretField).Child(namespaceField),
	// 		"password secret namespace can't be empty when secret name is set")
	// }
	return nil
}

func (r *User) validateUserDatabaseConfigs() *field.Error {
	if len(r.Spec.DatabaseConfigs) == 0 {
		return field.Required(specField.Child(databaseConfigsField),
			"database configs can't be empty")
	}
	for i, dbconfig := range r.Spec.DatabaseConfigs {
		if err := validateUserDatabaseConfig(dbconfig, i); err != nil {
			return err
		}
	}
	return nil
}

func (r *User) validateUserPrivileges() *field.Error {
	for i, privilege := range r.Spec.Privileges {
		if err := validateUserPrivilege(privilege, i); err != nil {
			return err
		}
	}
	return nil
}

func validateUserDatabaseConfig(dbconfig DatabaseConfig, index int) *field.Error {
	if dbconfig.Name == "" {
		return field.Required(specField.Child(databaseConfigsField).Index(index).Child(nameField),
			"database config name can't be empty")
	}
	return nil
}

func validateUserPrivilege(privilege Privilege, index int) *field.Error {
	priv := privilege.Privilege
	if priv.IsEmpty() {
		return field.Required(specField.Child(privilegesField).Index(index).Child(privilegeField),
			"privilege can't be empty field")

	} else if !PrivilegeType(privilege.On).IsAllTableSchemaPrivilegeType() && PrivilegeType(strings.ToUpper(privilege.On)).IsAllTableSchemaPrivilegeType() {
		return field.Forbidden(specField.Child(privilegesField).Index(index).Child(onField),
			"privilege on all schema must be upper case")

	} else if priv.IsTablePrivilegeType() {
		if privilege.Database == "" {
			return field.Required(specField.Child(privilegesField).Index(index).Child(databaseField),
				"'database' can't be empty field when 'privilege' is table scoped")
		}
		if privilege.On == "" {
			return field.Required(specField.Child(privilegesField).Index(index).Child(onField),
				"'on' can't be empty field when 'privilege' is table scoped")
		}

	} else if priv.IsDatabasePrivilegeType() {
		if privilege.Database == "" {
			return field.Required(specField.Child(privilegesField).Index(index).Child(databaseField),
				"'database' can't be empty field when 'privilege' is database scoped")
		}
		if privilege.On != "" {
			return field.Required(specField.Child(privilegesField).Index(index).Child(databaseField),
				"'on' field can't be set when 'privilege' is database scoped")
		}

	} else if priv.IsAllPrivilegeType() {
		if privilege.Database == "" && privilege.On == "" {
			return field.Required(specField.Child(privilegesField).Index(index).Child(onField),
				"'database' or 'on' fields can't be empty both when 'privilege' is set to 'ALL' or 'ALL PRIVILEGES'")
		}
		if privilege.Database == "" && privilege.On != "" {
			return field.Required(specField.Child(privilegesField).Index(index).Child(onField),
				"'database' can't be empty field when 'privilege' is set to 'ALL' or 'ALL PRIVILEGES' and 'on' field is set")
		}

	} else {
		upperPriv := PrivilegeType(strings.ToUpper(string(privilege.Privilege)))
		if upperPriv.IsAllPrivilegeType() ||
			// PrivilegeType(privilege.On).IsAllTableSchemaPrivilegeType() ||
			upperPriv.IsDatabasePrivilegeType() ||
			upperPriv.IsTablePrivilegeType() {
			return field.Forbidden(specField.Child(privilegesField).Index(index).Child(onField),
				"privilege must be upper case")
		}
		if privilege.Database != "" || privilege.On != "" {
			return field.Forbidden(specField.Child(privilegesField).Index(index).Child(onField),
				"'database' or 'on' field can't be set when 'privilege' is role name")
		}

	}

	return nil
}
