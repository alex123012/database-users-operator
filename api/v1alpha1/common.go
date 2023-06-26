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

// Secret is a reference for kubernetes secret.
type Secret struct {
	// Secret is secret name and namespace
	Secret NamespacedName `json:"secret"`
	// Kubernetes secret key with data
	Key string `json:"key"`
}

type NamespacedName struct {
	// resource namespace
	Namespace string `json:"namespace"`

	// resource name
	Name string `json:"name"`
}

type StatusSummary struct {
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}
