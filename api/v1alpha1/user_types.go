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
	// Name is appropriate database user name.
	// If not provided - operator automatically substitutes this parameter with CR name
	Name string `json:"name,omitempty"`

	DatabaseConfig DatabaseConfig `json:"databaseConfig"`
	// Privileges
	Privileges []Privilege `json:"privileges,omitempty"`
}

type DatabaseConfig struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type Privilege struct {
	// Privilege is role name or PrivilegeType
	Privilege PrivilegeType `json:"privilege"`

	// if used PrivilegeType from PrivilegeTypeMap in Privilege specify object to give Privilege to
	On string `json:"on,omitempty"`
}

type PrivilegeType string

func (p PrivilegeType) IsPrivilege() bool {
	_, f := PrivilegeTypeMap[p]
	return f
}

const (
	SELECT        PrivilegeType = "SELECT"
	INSERT        PrivilegeType = "INSERT"
	UPDATE        PrivilegeType = "UPDATE"
	DELETE        PrivilegeType = "DELETE"
	TRUNCATE      PrivilegeType = "TRUNCATE"
	REFERENCES    PrivilegeType = "REFERENCES"
	TRIGGER       PrivilegeType = "TRIGGER"
	CREATE        PrivilegeType = "CREATE"
	CONNECT       PrivilegeType = "CONNECT"
	TEMPORARY     PrivilegeType = "TEMPORARY"
	EXECUTE       PrivilegeType = "EXECUTE"
	USAGE         PrivilegeType = "USAGE"
	SET           PrivilegeType = "SET"
	ALTERSYSTEM   PrivilegeType = "ALTERSYSTEM"
	ALLPRIVILEGES PrivilegeType = "ALL PRIVILEGES"
)

var PrivilegeTypeMap map[PrivilegeType]struct{} = map[PrivilegeType]struct{}{
	SELECT:        {},
	INSERT:        {},
	UPDATE:        {},
	DELETE:        {},
	TRUNCATE:      {},
	REFERENCES:    {},
	TRIGGER:       {},
	CREATE:        {},
	CONNECT:       {},
	TEMPORARY:     {},
	EXECUTE:       {},
	USAGE:         {},
	SET:           {},
	ALTERSYSTEM:   {},
	ALLPRIVILEGES: {},
}

// UserStatus defines the observed state of User
type UserStatus struct {
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
