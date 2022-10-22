package common

import (
	"context"

	authv1alpha1 "github.com/alex123012/k8s-database-users-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
)

type KubeInterface interface {
	CreateV1Secret(ctx context.Context, resourceReference *authv1alpha1.User, data map[string][]byte, logger logr.Logger) error
	GetV1Secret(ctx context.Context, name, namespace string, logger logr.Logger) (*v1.Secret, error)
	DeleteV1Secret(ctx context.Context, secretResource v1.Secret, logger logr.Logger) error
}

type DatabaseInterface interface {
	ProcessUser(ctx context.Context) error
	DeleteUser(ctx context.Context) error
}

func SimpleMapper(str string) string {
	return str
}
