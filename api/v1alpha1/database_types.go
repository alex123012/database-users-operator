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

// DatabaseSpec defines the desired state of Database.
type DatabaseSpec struct {
	// Type of database to connect, required
	Type DatabaseType `json:"databaseType"`

	// Config for connecting for PostgreSQL compatible databases, not required.
	// required if DatabaseType equals to "PostgreSQL"
	PostgreSQL PostgreSQLConfig `json:"postgreSql,omitempty"`
}

// Database types that are currently supported.
type DatabaseType string

const (
	PostgreSQL DatabaseType = "PostgreSQL"
)

type PostgresSSLMode string

const (
	SSLModeDISABLE    PostgresSSLMode = "disable"
	SSLModeALLOW      PostgresSSLMode = "allow"
	SSLModePREFER     PostgresSSLMode = "prefer"
	SSLModeREQUIRE    PostgresSSLMode = "require"
	SSLModeVERIFYCA   PostgresSSLMode = "verify-ca"
	SSLModeVERIFYFULL PostgresSSLMode = "verify-full"
)

// PostgreSQLConfig is config that will be used by operator to connect to PostgreSQL compatible databases.
type PostgreSQLConfig struct {
	// Full DNS name/ip for database to use, required.
	// If K8S service is used to connect - provide host
	// as <db-service-name>.<db-service-namespace>.svc.cluster.local
	// refer to --host flag in https://www.postgresql.org/docs/current/app-psql.html
	Host string `json:"host"`

	// k8s-service/database port to connect to execute queries, defaults to 5432.
	// refer to --port flag in https://www.postgresql.org/docs/current/app-psql.html
	Port int `json:"port"`

	// User that will be used to connect to database, defaults to "postgres".
	// It must have at least CREATEROLE privilege (if you won't provide superuser acess to users)
	// or database superuser role if you think you'll be needed to give some users database superuser privileges
	// refer to --username flag in https://www.postgresql.org/docs/current/app-psql.html
	// and https://www.postgresql.org/docs/current/sql-grant.html "GRANT on Roles"
	User string `json:"user"`

	// SSL mode that will be used to connect to PostgreSQL, defaults to "disable".
	// Posssible values: "disable", "allow", "prefer", "require", "verify-ca", "verify-full".
	// If SSL mode is "require", "verify-ca", "verify-full" - operator will generate K8S secret with
	// SSL bundle (CA certificate, user certificate and user key) for User CR with same name as User CR.
	// see https://www.postgresql.org/docs/current/libpq-ssl.html
	SSLMode PostgresSSLMode `json:"sslMode"`

	// Database name that will be used to connect to database, not required
	// refer to --dbname flag in https://www.postgresql.org/docs/current/app-psql.html
	DatabaseName string `json:"databaseName,omitempty"`

	// Secret with SSL CA certificate ("ca.crt" key), user certificate ("tls.crt" key) and user key ("tls.key" key).
	// If SSL Mode equals to "disable", "allow" or "prefer" field is not required.
	// If SSL Mode equals to "require", "verify-ca" or "verify-full" - required.
	SSLCredentialsSecret NamespacedName `json:"sslSecrets,omitempty"`

	// Secret with CA key for creating users certificates
	// If SSL Mode equals to "disable", "allow" or "prefer" field is not required.
	// If SSL Mode equals to "require", "verify-ca" or "verify-full" - required.
	SSLCAKey Secret `json:"sslCaKey,omitempty"`

	// Secret with password for User to connect to database
	// If SSL Mode equals to "disable", "allow" or "prefer" field is required.
	// If SSL Mode equals to "require", "verify-ca" or "verify-full" - not required.
	PasswordSecret Secret `json:"passwordSecret,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// Database is the Schema for the databases API.
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DatabaseSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// DatabaseList contains a list of Database.
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}
