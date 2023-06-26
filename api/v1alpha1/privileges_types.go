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

type PrivilegeType string

// PrivilegesSpec defines the desired state of Privileges.
type PrivilegeSpec struct {
	// Privilege is role name or PrivilegeType
	Privilege PrivilegeType `json:"privilege"`

	// if used PrivilegeType from PrivilegeTypeTable in Privilege field
	// specify object to give Privilege to in database
	On string `json:"on,omitempty"`

	// If Privilege is database specific - this field will be used to determine which db to use
	Database string `json:"database,omitempty"`
}

//+kubebuilder:object:root=true

// Privileges is the Schema for the privileges API.
type Privileges struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Privileges []PrivilegeSpec `json:"privileges,omitempty"`
}

//+kubebuilder:object:root=true

// PrivilegesList contains a list of Privileges.
type PrivilegesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Privileges `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Privileges{}, &PrivilegesList{})
}

// func (p PrivilegeType) isEmpty() bool {
// 	return p == ""
// }

// func (p PrivilegeType) isTablePrivilegeType() bool {
// 	return privilegeTypeTableMap[p]
// }

// func (p PrivilegeType) isDatabasePrivilegeType() bool {
// 	return privilegeTypeDatabaseMap[p]
// }

// func (p PrivilegeType) isAllTableSchemaPrivilegeType() bool {
// 	return allTablesSchemaRegex.MatchString(string(p))
// }

// func (p PrivilegeType) isAllPrivilegeType() bool {
// 	return privilegeTypeAllMap[p]
// }

// const (
// 	SELECT          PrivilegeType = "SELECT"
// 	INSERT          PrivilegeType = "INSERT"
// 	UPDATE          PrivilegeType = "UPDATE"
// 	DELETE          PrivilegeType = "DELETE"
// 	TRUNCATE        PrivilegeType = "TRUNCATE"
// 	REFERENCES      PrivilegeType = "REFERENCES"
// 	TRIGGER         PrivilegeType = "TRIGGER"
// 	CREATE          PrivilegeType = "CREATE"
// 	CONNECT         PrivilegeType = "CONNECT"
// 	TEMPORARY       PrivilegeType = "TEMPORARY"
// 	TEMP            PrivilegeType = "TEMP"
// 	EXECUTE         PrivilegeType = "EXECUTE"
// 	USAGE           PrivilegeType = "USAGE"
// 	SET             PrivilegeType = "SET"
// 	ALTERSYSTEM     PrivilegeType = "ALTER SYSTEM"
// 	ALLPRIVILEGES   PrivilegeType = "ALL PRIVILEGES"
// 	ALL             PrivilegeType = "ALL"
// 	ALLTABLESSCHEMA PrivilegeType = "ALL TABLES IN SCHEMA.*"
// )

// var (
// allTablesSchemaRegex = regexp.MustCompile(string(ALLTABLESSCHEMA))

// privilegeTypeDatabaseMap = map[PrivilegeType]bool{
// 	CREATE:    true,
// 	CONNECT:   true,
// 	TEMPORARY: true,
// 	TEMP:      true,
// }

// privilegeTypeTableMap = map[PrivilegeType]bool{
// 	SELECT:     true,
// 	INSERT:     true,
// 	UPDATE:     true,
// 	DELETE:     true,
// 	TRUNCATE:   true,
// 	REFERENCES: true,
// 	TRIGGER:    true,
// }

//	privilegeTypeAllMap = map[PrivilegeType]bool{
//		ALL:           true,
//		ALLPRIVILEGES: true,
//	}
// )
