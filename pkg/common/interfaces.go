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

package common

import (
	"context"

	authv1alpha1 "github.com/alex123012/database-users-operator/api/v1alpha1"
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
