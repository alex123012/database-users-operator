package internal

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DecodeSecretData(ctx context.Context, nn types.NamespacedName, kClient client.Client) (map[string]string, error) {
	secret := &v1.Secret{}
	if err := kClient.Get(ctx, nn, secret); err != nil {
		return nil, err
	}

	data := make(map[string]string)
	for key, value := range secret.Data {
		decodedValue, err := base64.StdEncoding.DecodeString(string(value))
		if err != nil {
			return nil, err
		}
		data[key] = string(decodedValue)
	}

	return data, nil
}

func PathFromHome(path string) string {
	return filepath.Join(os.Getenv("HOME"), path)
}
