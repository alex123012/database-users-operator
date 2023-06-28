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
