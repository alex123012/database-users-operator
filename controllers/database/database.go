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

package database

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/controllers/database/connection"
	"github.com/alex123012/database-users-operator/controllers/database/postgresql"
	"github.com/alex123012/database-users-operator/controllers/internal"
)

type Database interface {
	Close(cxt context.Context) error
	CreateUser(ctx context.Context, username, password string) (map[string]string, error)
	DeleteUser(ctx context.Context, username string) error
	ApplyPrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error
	RevokePrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error
}

func NewDatabase(ctx context.Context, s v1alpha1.DatabaseSpec, client client.Client, logger logr.Logger) (Database, error) {
	conn := connection.NewDefaultConnector(logger)
	return newDatabase(ctx, conn, s, client, logger)
}

func newDatabase(ctx context.Context, conn connection.Connection, s v1alpha1.DatabaseSpec, client client.Client, logger logr.Logger) (Database, error) {
	var db Database
	var err error
	switch s.Type {
	case v1alpha1.PostgreSQL:
		db, err = newPostgresql(ctx, conn, s.PostgreSQL, client, logger)
	default:
		err = fmt.Errorf("can't find supported DB type '%s'", s.Type)
	}
	return db, err
}

func newPostgresql(ctx context.Context, conn connection.Connection, c v1alpha1.PostgreSQLConfig, client client.Client, logger logr.Logger) (*postgresql.Postgresql, error) {
	sslData := make(map[string]string, 0)
	var sslCAKey string
	if c.SSLMode == v1alpha1.SSLModeREQUIRE || c.SSLMode == v1alpha1.SSLModeVERIFYCA || c.SSLMode == v1alpha1.SSLModeVERIFYFULL {
		var err error
		sslData, err = internal.DecodeSecretData(ctx, types.NamespacedName(c.SSLCredentialsSecret), client)
		if err != nil {
			return nil, err
		}
		sslCAData, err := internal.DecodeSecretData(ctx, types.NamespacedName(c.SSLCAKey.Secret), client)
		if err != nil {
			return nil, err
		}
		sslCAKey = sslCAData[c.SSLCAKey.Key]
	}

	password, err := passwordFromSecret(ctx, client, c.PasswordSecret)
		if err != nil {
			return nil, err
		}

	cfg := postgresql.NewConfig(c.Host, c.Port, c.User, password, c.DatabaseName,
		c.SSLMode, sslData["ca.crt"], sslData["tls.crt"], sslData["tls.key"], sslCAKey)

	p := postgresql.NewPostgresql(conn, cfg, logger)
	return p, p.Connect(ctx)
}

func passwordFromSecret(ctx context.Context, client client.Client, secretNN v1alpha1.Secret) (string, error) {
	var password string
	if secretNN.Key != "" && secretNN.Secret.Name != "" && secretNN.Secret.Namespace != "" {
		data, err := internal.DecodeSecretData(ctx, types.NamespacedName(secretNN.Secret), client)
		if err != nil {
			return "", err
		}
		password = data[secretNN.Key]
	}
	return password, nil
}
