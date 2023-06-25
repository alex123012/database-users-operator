package database

import (
	"context"
	"fmt"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/controllers/database/postgresql"
	"github.com/alex123012/database-users-operator/controllers/internal"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Interface interface {
	Close(cxt context.Context) error
	CreateUser(ctx context.Context, username, password string) (map[string]string, error)
	DeleteUser(ctx context.Context, username string) error
	ApplyPrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error
	RevokePrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error
}

func NewDatabase(ctx context.Context, s v1alpha1.DatabaseSpec, kClient client.Client, logger logr.Logger) (Interface, error) {
	switch s.Type {
	case v1alpha1.PostgreSQL:
		return newPostgresql(ctx, s.PostgreSQL, kClient, logger)
	}
	return nil, fmt.Errorf("can't find supported DB type '%s'", s.Type)
}

func newPostgresql(ctx context.Context, c v1alpha1.PostgreSQLConfig, kClient client.Client, logger logr.Logger) (*postgresql.Postgresql, error) {
	sslData := make(map[string]string, 0)
	var sslCAKey string
	if c.SSLMode == v1alpha1.SSLModeREQUIRE || c.SSLMode == v1alpha1.SSLModeVERIFYCA || c.SSLMode == v1alpha1.SSLModeVERIFYFULL {
		var err error
		sslData, err = internal.DecodeSecretData(ctx, types.NamespacedName(c.SSLCredentialsSecret), kClient)
		if err != nil {
			return nil, err
		}
		sslCAData, err := internal.DecodeSecretData(ctx, types.NamespacedName(c.SSLCAKey.Secret), kClient)
		if err != nil {
			return nil, err
		}
		sslCAKey = sslCAData[c.SSLCAKey.Key]
	}

	var password string
	if c.PasswordSecret.Key != "" && c.PasswordSecret.Secret.Name != "" && c.PasswordSecret.Secret.Namespace != "" {
		data, err := internal.DecodeSecretData(ctx, types.NamespacedName(c.PasswordSecret.Secret), kClient)
		if err != nil {
			return nil, err
		}
		password = data[c.PasswordSecret.Key]
	}

	cfg := postgresql.NewConfig(c.Host, c.Port, c.User, password, c.DatabaseName,
		string(c.SSLMode), sslData["ca.crt"], sslData["tls.crt"], sslData["tls.key"], sslCAKey)

	return postgresql.NewPostgresql(cfg, logger), nil
}
