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

// +kubebuilder:validation:Required
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserSpec defines the desired state of User.
type UserSpec struct {
	// List of databases, where user needs to be created with configs for it.
	Databases []DatabaseRef `json:"databases"`
}

type DatabaseRef struct {
	// The name of the Database CR to create user in, required.
	Name string `json:"name"`

	// Reference to secret with password for user in the database, not required.
	PasswordSecret Secret `json:"passwordSecret,omitempty"`

	// If operator would create data for user (for example for postgres with sslMode=="verify-full"),
	// it is reference to non-existed Secret, that will be created during user creation in the database, not required.
	CreatedSecret NamespacedName `json:"createdSecret,omitempty"`

	// List of references to Privileges CR, that will be applied to created user in the database, required.
	Privileges []Name `json:"privileges"`
}

// UserStatus defines the observed state of User.
type UserStatus struct {
	Summary StatusSummary `json:"summary,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// User is the Schema for the users API.
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UserList contains a list of User.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
