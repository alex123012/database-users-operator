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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ConfigSpec defines the desired state of Config
type ConfigSpec struct {
	DatabaseType DatabaseType `json:"databaseType"`
	// CockroachDB  PostgreSQLConfig `json:"cockroachDB"`
	PostgreSQL PostgreSQLConfig `json:"postgreSQL"`
}

type DatabaseType string

const (
	PostgreSQL DatabaseType = "PostgreSQL"
	// CockroachDB DatabaseType = "CockroachDB"
)

type PostgreSQLConfig struct {
	Host           string                   `json:"host"`
	Port           int                      `json:"port"`
	Namespace      string                   `json:"namespace"`
	User           string                   `json:"user"`
	SSLMode        database.PostgresSSLMode `json:"sslMode,omitempty"`
	ClusterName    string                   `json:"clusterName,omitempty"`
	DatabaseName   string                   `json:"databaseName,omitempty"`
	SSLCredentials SSLSecrets               `json:"sslSecrets,omitempty"`
	PasswordSecret Secret                   `json:"passwordSecret,omitempty"`
}

type SSLSecrets struct {
	UserSecret Secret `json:"userSecret,omitempty"`
	CASecret   Secret `json:"caSecret,omitempty"`
}

type Secret struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// ConfigStatus defines the observed state of Config
type ConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Config is the Schema for the configs API
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigSpec   `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConfigList contains a list of Config
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Config `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Config{}, &ConfigList{})
}
