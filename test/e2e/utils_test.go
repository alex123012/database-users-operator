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

package e2e_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func mustHaveEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		panic(fmt.Sprintf("Environment variable '%s' not found", name))
	}
	return value
}

func createClientSet() (*kubernetes.Clientset, error) {
	config, err := controllerruntime.GetConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("[error] %s \n", err)
	}

	return clientset, err
}

func createObjects(ctx context.Context, objects ...client.Object) error {
	for _, obj := range objects {
		if err := k8sClient.Create(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

func deleteObjects(ctx context.Context, objects ...client.Object) error {
	for _, obj := range objects {
		if err := k8sClient.Delete(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

func decodeYamlFileWithNamespace(filename, namespace string) ([]client.Object, error) {
	unstructs, err := decodeYamlFile(filename)
	if err != nil {
		return nil, err
	}

	objects := make([]client.Object, 0, len(unstructs))
	for _, obj := range unstructs {
		obj.SetNamespace(namespace)
		objects = append(objects, obj)
	}
	return objects, nil
}

func decodeYamlFile(filename string) ([]*unstructured.Unstructured, error) {
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return decodeYAML(fileContent)
}

func decodeYAML(data []byte) ([]*unstructured.Unstructured, error) {
	multidocReader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	objects := make([]*unstructured.Unstructured, 0)
	// Iterate over the data until Read returns io.EOF. Every successful
	// read returns a complete YAML document.
	for {
		buf, err := multidocReader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return objects, nil
			}

			return nil, err
		}

		// Do not use this YAML doc if it is unkind.
		var typeMeta runtime.TypeMeta
		if err := utilyaml.Unmarshal(buf, &typeMeta); err != nil {
			continue
		}
		if typeMeta.Kind == "" {
			continue
		}

		// Define the unstructured object into which the YAML document will be
		// unmarshaled.
		obj := &unstructured.Unstructured{
			Object: make(map[string]interface{}),
		}

		// Unmarshal the YAML document into the unstructured object.
		if err := utilyaml.Unmarshal(buf, &obj.Object); err != nil {
			return nil, err
		}

		objects = append(objects, obj)
	}
}
