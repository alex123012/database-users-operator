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
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("PostgreSQL", Ordered, func() {
	if os.Getenv("E2E_TESTS") != "yes" {
		return
	}

	var (
		ctx              = context.Background()
		postgresInstance []client.Object
	)

	BeforeAll(func() {
		var err error
		postgresInstance, err = decodeYamlFileWithNamespace("../../docs/examples/postgresql/01-statefulset.yaml", namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(createObjects(ctx, postgresInstance...)).To(Succeed())

		Eventually(func() int32 {
			postgresSTS, err := clientSet.AppsV1().StatefulSets(namespace).Get(ctx, "postgresql-db", metav1.GetOptions{})
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			return postgresSTS.Status.ReadyReplicas
		}, 30, 1).Should(BeNumerically("==", 1), "Expected to have PostgreSQL Pod Ready")
	})

	AfterAll(func() {
		Expect(deleteObjects(ctx, postgresInstance...)).To(Succeed())
	})

	It("Creates postgres", func() {
		_, err := clientSet.AppsV1().StatefulSets(namespace).Get(ctx, "postgresql-db", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
})
