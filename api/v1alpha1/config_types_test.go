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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Config", func() {

	Context("ConfigSpec", func() {
		It("can be created", func() {
			created := generatePostgresConfigObject("config1")
			Expect(k8sClient.Create(context.Background(), created)).To(Succeed())

			fetched := &Config{}
			Expect(k8sClient.Get(context.Background(), getConfigKey(created), fetched)).To(Succeed())
			Expect(fetched).To(Equal(created))
		})

		It("can be deleted", func() {
			created := generatePostgresConfigObject("config2")
			Expect(k8sClient.Create(context.Background(), created)).To(Succeed())

			Expect(k8sClient.Delete(context.Background(), created)).To(Succeed())
			Expect(k8sClient.Get(context.Background(), getConfigKey(created), created)).ToNot(Succeed())
		})

		Context("Default settings for Postgres", func() {
			var (
				configObject         Config
				expectedConfigObject Config
			)

			BeforeEach(func() {
				expectedConfigObject = *generatePostgresConfigObject("foo")
			})

			When("Namespace not set", func() {
				It("Sets namespace for passwordSecret", func() {
					configObject = *generatePostgresConfigObject("config5")
					configObject.Spec.PostgreSQL.PasswordSecret.Namespace = ""

					Expect(k8sClient.Create(context.Background(), &configObject)).To(Succeed())
					fetchedConfig := &Config{}
					Expect(k8sClient.Get(context.Background(), getConfigKey(&configObject), fetchedConfig)).To(Succeed())
					Expect(fetchedConfig.Spec).To(Equal(expectedConfigObject.Spec))
				})

				It("Sets namespace for SSL configs", func() {
					expectedConfigObject.Spec.PostgreSQL.SSLMode = database.SSLModeREQUIRE
					expectedConfigObject.Spec.PostgreSQL.PasswordSecret = Secret{}
					expectedConfigObject.Spec.PostgreSQL.SSLCredentials = SSLSecrets{
						UserSecret: Secret{Name: "foo", Namespace: "default"},
						CASecret:   Secret{Name: "bar", Namespace: "default"},
					}

					configObject = *generatePostgresConfigObject("config6")
					configObject.Spec.PostgreSQL.SSLMode = database.SSLModeREQUIRE
					configObject.Spec.PostgreSQL.PasswordSecret = Secret{}
					configObject.Spec.PostgreSQL.SSLCredentials = SSLSecrets{
						UserSecret: Secret{Name: "foo"},
						CASecret:   Secret{Name: "bar"},
					}

					Expect(k8sClient.Create(context.Background(), &configObject)).To(Succeed())
					fetchedConfig := &Config{}
					Expect(k8sClient.Get(context.Background(), getConfigKey(&configObject), fetchedConfig)).To(Succeed())
					Expect(fetchedConfig.Spec).To(Equal(expectedConfigObject.Spec))
				})
			})
		})
	})
})

func getConfigKey(config *Config) types.NamespacedName {
	return types.NamespacedName{
		Name:      config.GetName(),
		Namespace: config.GetNamespace(),
	}
}

func generatePostgresConfigObject(name string) *Config {
	namespace := "default"
	return &Config{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: ConfigSpec{
			DatabaseType: PostgreSQL,
			PostgreSQL: PostgreSQLConfig{
				Host:    "postgres",
				Port:    5432,
				User:    "postgres",
				SSLMode: database.SSLModeDISABLE,
				PasswordSecret: Secret{
					Name:      "foo",
					Namespace: namespace,
				},
			},
		},
	}
}
