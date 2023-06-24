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

// DatabaseBindingSpec defines the desired state of DatabaseBinding
type DatabaseBindingSpec struct {
	// DatabaseConfig references to the DatabaseConfig that will be used to connect to DB
	DatabaseConfig NamespacedName `json:"databaseConfig"`

	// Users holds references to the objects the privileges applies to.
	User NamespacedName `json:"user"`
}

// DatabaseBindingStatus defines the observed state of DatabaseBinding
type DatabaseBindingStatus struct {
	Summary StatusSummary `json:"summary,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DatabaseBinding is the Schema for the databasebindings API
type DatabaseBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseBindingSpec   `json:"spec,omitempty"`
	Status DatabaseBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatabaseBindingList contains a list of DatabaseBinding
type DatabaseBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatabaseBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatabaseBinding{}, &DatabaseBindingList{})
}
