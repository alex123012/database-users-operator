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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrivilegesBindingSpec defines the desired state of PrivilegesBinding.
type PrivilegesBindingSpec struct {
	// DatabaseBinding references to the DatabaseBinding that will be
	// used to apply privileges to user in a particular database
	DatabaseBindings []NamespacedName `json:"databaseBindings"`

	// List of database privileges that will be applied to user.
	// If user already exists in database - all it privileges will be
	// synchronized with this list (all privileges that are not defined will be revoked).
	Privileges NamespacedName `json:"privileges"`
}

// PrivilegesBindingStatus defines the observed state of PrivilegesBinding.
type PrivilegesBindingStatus struct {
	Summary StatusSummary `json:"summary,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PrivilegesBinding is the Schema for the privilegesbindings API.
type PrivilegesBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrivilegesBindingSpec   `json:"spec,omitempty"`
	Status PrivilegesBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PrivilegesBindingList contains a list of PrivilegesBinding.
type PrivilegesBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrivilegesBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PrivilegesBinding{}, &PrivilegesBindingList{})
}
