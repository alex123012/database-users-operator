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
	"bytes"
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PostgreSQL::E2E", Ordered, func() {
	if os.Getenv("E2E_TESTS") != "yes" {
		return
	}

	const (
		postgresPod       = "postgresql-db"
		postgresContainer = "postgresql-db"
	)

	var (
		ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
		// postgresInstance []client.Object
		execer *PodExecer
	)

	BeforeAll(func() {
		// var err error
		// postgresInstance, err = decodeYamlFileWithNamespace("../../docs/examples/postgresql/01-statefulset.yaml", namespace)
		// Expect(err).NotTo(HaveOccurred())
		// Expect(createObjects(ctx, postgresInstance...)).To(Succeed())

		Eventually(func() int32 {
			postgresSTS, err := clientSet.AppsV1().StatefulSets(namespace).Get(ctx, postgresPod, metav1.GetOptions{})
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			return postgresSTS.Status.ReadyReplicas
		}, 30, 1).Should(BeNumerically("==", 1), "Expected to have PostgreSQL Pod Ready")

		execer = NewPodExecer(clientSet, restConfig, postgresPod+"-0", namespace, postgresContainer)
	})

	AfterAll(func() {
		defer cancel()
		// Expect(deleteObjects(ctx, postgresInstance...)).To(Succeed())
	})

	It("Creates postgres", func() {
		clientSet.CoreV1().Pods("")
		_, err := clientSet.AppsV1().StatefulSets(namespace).Get(ctx, postgresPod, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
	It("Postgres is ready", func() {
		stdout := bytes.NewBuffer(nil)
		stderr := bytes.NewBuffer(nil)
		Eventually(func() bool {
			err := execer.execCmd(ctx, []string{"pg_isready"}, nil, stdout, stderr)
			return err == nil
		}, "30s", 1).Should(BeTrue(), "Expected to have PostgreSQL instance Ready")
	})
	It("Connects to postgres", func() {
		runPostgresCommand(ctx, execer, "/bin/bash", "-c", "psql -U postgres -t -h 127.0.0.1 ")
	})
})

func runPostgresCommand(ctx context.Context, execer *PodExecer, commands ...string) {
	Expect(len(commands)).To(BeNumerically(">", 0))
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	err := execer.execCmd(ctx, commands, nil, stdout, stderr)
	Expect(stdout.String()).To(Equal(""))
	Expect(stderr.String()).To(Equal(""))
	Expect(err).To(Succeed())
}
