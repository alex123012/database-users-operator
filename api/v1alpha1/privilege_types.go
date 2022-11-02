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

import "regexp"

type PrivilegeType string

func (p PrivilegeType) IsEmpty() bool {
	return p == ""
}

func (p PrivilegeType) IsTablePrivilegeType() bool {
	_, f := privilegeTypeTableMap[p]
	return f
}

func (p PrivilegeType) IsDatabasePrivilegeType() bool {
	_, f := privilegeTypeDatabaseMap[p]
	return f
}

func (p PrivilegeType) IsAllTableSchemaPrivilegeType() bool {
	return AllTablesSchemaRegex.MatchString(string(p))
}

func (p PrivilegeType) IsAllPrivilegeType() bool {
	_, f := privilegeTypeAllMap[p]
	return f
}

// func (p PrivilegeType) IsSequencePrivilegeType() bool {
// 	_, f := privilegeTypeTableMap[p]
// 	return f
// }

// func (p PrivilegeType) IsProcedurePrivilegeType() bool {
// 	_, f := privilegeTypeTableMap[p]
// 	return f
// }

const (
	SELECT          PrivilegeType = "SELECT"
	INSERT          PrivilegeType = "INSERT"
	UPDATE          PrivilegeType = "UPDATE"
	DELETE          PrivilegeType = "DELETE"
	TRUNCATE        PrivilegeType = "TRUNCATE"
	REFERENCES      PrivilegeType = "REFERENCES"
	TRIGGER         PrivilegeType = "TRIGGER"
	CREATE          PrivilegeType = "CREATE"
	CONNECT         PrivilegeType = "CONNECT"
	TEMPORARY       PrivilegeType = "TEMPORARY"
	TEMP            PrivilegeType = "TEMP"
	EXECUTE         PrivilegeType = "EXECUTE"
	USAGE           PrivilegeType = "USAGE"
	SET             PrivilegeType = "SET"
	ALTERSYSTEM     PrivilegeType = "ALTER SYSTEM"
	ALLPRIVILEGES   PrivilegeType = "ALL PRIVILEGES"
	ALL             PrivilegeType = "ALL"
	ALLTABLESSCHEMA PrivilegeType = "ALL TABLES IN SCHEMA.*"
)

var (
	AllTablesSchemaRegex = regexp.MustCompile(string(ALLTABLESSCHEMA))
)

var privilegeTypeDatabaseMap map[PrivilegeType]struct{} = map[PrivilegeType]struct{}{
	CREATE:    {},
	CONNECT:   {},
	TEMPORARY: {},
	TEMP:      {},
}

var privilegeTypeTableMap map[PrivilegeType]struct{} = map[PrivilegeType]struct{}{
	SELECT:     {},
	INSERT:     {},
	UPDATE:     {},
	DELETE:     {},
	TRUNCATE:   {},
	REFERENCES: {},
	TRIGGER:    {},
}

var privilegeTypeAllMap map[PrivilegeType]struct{} = map[PrivilegeType]struct{}{
	ALL:           {},
	ALLPRIVILEGES: {},
}
