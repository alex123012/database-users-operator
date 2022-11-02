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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserSpec defines the desired state of User
type UserSpec struct {

	// K8S secret with key "password" for user password to assign, not required
	PasswordSecret Secret `json:"passwordSecret,omitempty"`

	// List of Configs that will be used to create users
	DatabaseConfigs []DatabaseConfig `json:"databaseConfigs"`

	// List of database privileges that will be applied to user.
	// If user already exists in database - all it privileges will be
	// synchronized with this list (all privileges that are not defined in the lis will be revoked).
	Privileges []Privilege `json:"privileges"`
}

// Utility struct for Config CR specification
type DatabaseConfig struct {
	// Name of Config resource
	Name string `json:"name"`

	// Namespace of config resource
	Namespace string `json:"namespace"`
}

type Privilege struct {
	// Privilege is role name or PrivilegeType
	Privilege PrivilegeType `json:"privilege" postgres:"privilege_type"`

	// if used PrivilegeType from PrivilegeTypeTable in Privilege field
	// specify object to give Privilege to in database
	On string `json:"on,omitempty" postgres:"table_name"`

	// If Privilege is database specific - this field will be used to determine which db to use
	// (used PrivilegeType from PrivilegeTypeDatabase or PrivilegeTypeTable)
	Database string `json:"database,omitempty" postgres:"table_catalog"`
}

// UserStatus defines the observed state of User
type UserStatus struct {
	// TODO
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// User is the Schema for the users API
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}

func (r *User) SetDbConfigs(dbConfigs []DatabaseConfig) *User {
	newUser := r.DeepCopy()
	newUser.Spec.DatabaseConfigs = dbConfigs
	return newUser
}

func (r *User) SetPasswordSecret(name, namespace string) *User {
	newUser := r.DeepCopy()
	newUser.Spec.PasswordSecret = Secret{Name: name, Namespace: namespace}
	return newUser
}

func (r *User) SetPrivileges(privileges []Privilege) *User {
	newUser := r.DeepCopy()
	newUser.Spec.Privileges = privileges
	return newUser
}
