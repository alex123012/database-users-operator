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

package utils

import (
	"context"
	"os"
	"path/filepath"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DecodeSecretData(ctx context.Context, nn types.NamespacedName, client client.Client) (map[string]string, error) {
	secret, err := Secret(ctx, nn, client)
	if err != nil {
		return nil, err
	}

	data := make(map[string]string)
	for key, value := range secret.Data {
		data[key] = string(value)
	}

	return data, nil
}

func Secret(ctx context.Context, nn types.NamespacedName, client client.Client) (*v1.Secret, error) {
	secret := &v1.Secret{}
	err := client.Get(ctx, nn, secret)
	return secret, err
}

func PathFromHome(paths ...string) string {
	paths = append([]string{os.Getenv("HOME")}, paths...)
	return filepath.Join(paths...)
}
