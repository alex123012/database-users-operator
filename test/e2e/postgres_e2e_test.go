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
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
)

var _ = Describe("PostgreSQL::E2E", Ordered, func() {
	if os.Getenv("E2E_TESTS") != "yes" {
		return
	}

	const (
		postgresSTS       = "postgresql-db"
		postgresPod       = "postgresql-db-0"
		postgresContainer = "postgresql-db"
	)

	var (
		ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
		execer      *PodExecer
	)

	BeforeAll(func() {
		createObjectsFromYaml(ctx, "../../docs/examples/postgresql/01-statefulset.yaml")

		Eventually(func() bool {
			for {
				sts, err := clientSet.AppsV1().StatefulSets(namespace).Get(ctx, postgresSTS, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				if v := sts.Status.ReadyReplicas > 0; v {
					return v
				}
			}
		}, 30, 1).Should(BeTrue(), "Expected to have PostgreSQL Pod Ready")

		Eventually(func() bool {
			for {
				sts, err := clientSet.BatchV1().Jobs(namespace).Get(ctx, "prepare-example", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				if v := sts.Status.Succeeded > 0; v {
					return v
				}
			}
		}, 30, 1).Should(BeTrue(), "Expected to have prepare job Ready")

		execer = NewPodExecer(postgresSTS+"-0", namespace, postgresContainer)
	})

	AfterAll(func() {
		defer cancel()
		deleteObjectsFromYaml(ctx, "../../docs/examples/postgresql/04-user.yaml")
		Eventually(func() bool {
			for {
				user := &v1alpha1.User{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "john"}, user)
				if v := errors.IsNotFound(err); v {
					return v
				}
			}
		}, "30s", 1).Should(BeTrue(), "Expected User to be deleted")

		// deleteObjectsFromYaml(ctx, "../../docs/examples/postgresql/02-database.yaml", "../../docs/examples/postgresql/03-privileges.yaml", "../../docs/examples/postgresql/01-statefulset.yaml")
	})

	It("Creates postgres pod", func() {
		_, err := clientSet.CoreV1().Pods(namespace).Get(ctx, postgresPod, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Postgres is ready", func() {
		Eventually(func() bool {
			for {
				_, err := execer.execCmd(ctx, []string{"pg_isready"})
				if v := err == nil; v {
					return v
				}
			}
		}, "30s", 1).Should(BeTrue(), "Expected to have PostgreSQL instance Ready")
	})

	It("Connects to postgres", func() {
		Eventually(func() bool {
			for {
				d := runPostgresCommand(ctx, execer, "SELECT 1", false)
				if v := d == "1"; v {
					return v
				}
			}
		}, "30s", 1).Should(BeTrue(), "Expected to have PostgreSQL instance connectable")
	})

	Context("Creates user in database", Ordered, func() {
		BeforeAll(func() {
			createObjectsFromYaml(ctx, "../../docs/examples/postgresql/02-database.yaml", "../../docs/examples/postgresql/03-privileges.yaml", "../../docs/examples/postgresql/04-user.yaml")
			Eventually(func() bool {
				for {
					user := &v1alpha1.User{}
					err := k8sClient.Get(ctx, types.NamespacedName{Name: "john"}, user)
					Expect(err).NotTo(HaveOccurred())

					if v := user.Status.Summary.Ready; v {
						return v
					}
				}
			}, "30s", 1).Should(BeTrue(), "Expected to have User ready")
		})

		It("Creates user in database", func() {
			Expect(runPostgresCommand(ctx, execer, `SELECT 1 FROM pg_roles WHERE rolname='john';`, true)).To(Equal("1"))
		})

		It("Creates user in database", func() {
			Expect(runPostgresCommand(ctx, execer, `SELECT 1 FROM pg_roles WHERE rolname='john';`, true)).To(Equal("1"))
		})

		It("Creates inserts proper privileges", func() {
			Expect(runPostgresCommand(ctx, execer, `SELECT rolname FROM pg_roles WHERE pg_has_role('john', oid, 'member') ORDER BY rolname;`, true)).To(Equal("john\nsome_role"))
		})
	})
})

func runPostgresCommand(ctx context.Context, execer *PodExecer, sql string, checkErr bool) string {
	Expect(sql).ToNot(BeEmpty())
	u := "postgres"
	// if len(username) > 0 {
	// 	u = username[0]
	// }

	commands := []string{"psql", "-h", "127.0.0.1", "-U", u, "-tXAc", sql}
	data, err := execer.execCmd(ctx, commands)
	fmt.Printf("\n\n%s\n\n", data)
	if checkErr {
		Expect(err).To(Succeed())
	}
	return string(data)
}

func createObjectsFromYaml(ctx context.Context, filenames ...string) {
	funcObjectsFromYaml(ctx, createObjects, filenames...)
}

func deleteObjectsFromYaml(ctx context.Context, filenames ...string) {
	funcObjectsFromYaml(ctx, deleteObjects, filenames...)
}

func funcObjectsFromYaml(ctx context.Context, f func(ctx context.Context, objects ...client.Object) error, filenames ...string) {
	for _, filename := range filenames {
		objects, err := decodeYamlFileWithNamespace(filename, namespace)
		Expect(err).NotTo(HaveOccurred())
		err = f(ctx, objects...)
		if errors.IsAlreadyExists(err) || errors.IsNotFound(err) {
			continue
		}
		Expect(err).ShouldNot(HaveOccurred())
	}
}
