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

package controllers_test

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/pkg/database"
)

const (
	defaultMysqlConnString    = "mysql:test-user:mysupersecretpass@tcp(test-mysql:3306)/?interpolateParams=true"
	defaultPostgresConnString = "pgx:host=test-postgres user=test-user port=5432 password=mysupersecretpass"
)

func defaultPostgresConfig() v1alpha1.PostgreSQLConfig {
	return v1alpha1.PostgreSQLConfig{
		Host: "test-postgres",
		Port: 5432,
		User: "test-user",
		PasswordSecret: v1alpha1.Secret{
			Key: "pass",
			Secret: v1alpha1.NamespacedName{
				Namespace: namespace,
				Name:      uniqueName("user-password", v1alpha1.PostgreSQL),
			},
		},
	}
}

func defaultMysqlConfig() v1alpha1.MySQLConfig {
	return v1alpha1.MySQLConfig{
		Host: "test-mysql",
		Port: 3306,
		User: "test-user",
		PasswordSecret: v1alpha1.Secret{
			Key: "pass",
			Secret: v1alpha1.NamespacedName{
				Namespace: namespace,
				Name:      uniqueName("user-password", v1alpha1.MySQL),
			},
		},
	}
}

type testDatabase struct {
	namespace         string
	dbType            v1alpha1.DatabaseType
	dbConfig          interface{}
	fakeDB            *database.FakeDatabase
	connectionStrings []string
	queries           []string
	removeQueries     []string
}

func newTestDatabase(namespace string, dbType v1alpha1.DatabaseType, dbConfig interface{}, fakeDB *database.FakeDatabase, connStrings, queries, removeQueries []string) testDatabase {
	return testDatabase{
		namespace:         namespace,
		dbType:            dbType,
		dbConfig:          dbConfig,
		fakeDB:            fakeDB,
		connectionStrings: connStrings,
		queries:           queries,
		removeQueries:     removeQueries,
	}
}

func (t testDatabase) run() {
	var (
		user              *v1alpha1.User
		secret            *v1.Secret
		database          *v1alpha1.Database
		databaseBinding   *v1alpha1.DatabaseBinding
		privileges        *v1alpha1.Privileges
		privilegesBinding *v1alpha1.PrivilegesBinding
	)

	BeforeEach(func() {
		t.fakeDB.Conn.ResetDB()
		user, secret, database, databaseBinding, privileges, privilegesBinding = bundle(t.namespace, t.dbType)
		switch t.dbType {
		case v1alpha1.PostgreSQL:
			database.Spec.PostgreSQL = t.dbConfig.(v1alpha1.PostgreSQLConfig)
		case v1alpha1.MySQL:
			database.Spec.MySQL = t.dbConfig.(v1alpha1.MySQLConfig)
		default:
			Expect(t.dbType).To(Equal("not supported db"))
		}

		createObjects(user, secret, database, databaseBinding, privileges, privilegesBinding)
		waitForDatabaseBindingReady(databaseBinding)
		waitForPrivilegesBindingReady(privilegesBinding)
	})

	AfterEach(func() {
		t.fakeDB.Conn.ResetDB()
		resetCLuster(t.fakeDB, privilegesBinding, databaseBinding, user, secret, database, privileges)
		f := checkQueries(t.fakeDB, t.removeQueries)
		f()
	})

	It("works", func() {
		By("Connection strings", checkConnectionStrings(t.fakeDB, t.connectionStrings))

		By("Executed queries", checkQueries(t.fakeDB, t.queries))
	})
}

func checkConnectionStrings(fakeDB *database.FakeDatabase, expected []string) func() {
	return func() {
		connections := fakeDB.Conn.Connections()
		Expect(connections).To(HaveLen(len(expected)))
		for _, connString := range expected {
			Expect(connections[connString]).To(BeTrue())
		}
	}
}

func checkQueries(fakeDB *database.FakeDatabase, expected []string) func() {
	return func() {
		queries := fakeDB.Conn.Queries()
		Expect(queries).To(HaveLen(len(expected)))
		for _, query := range expected {
			Expect(queries[query]).NotTo(Equal(0))
		}
	}
}

func waitForDatabaseBindingReady(databaseBinding *v1alpha1.DatabaseBinding) {
	EventuallyWithOffset(1, func() string {
		databaseBindingCreated := v1alpha1.DatabaseBinding{}
		if err := k8sClient.Get(
			ctx,
			types.NamespacedName{Name: databaseBinding.Name, Namespace: databaseBinding.Namespace},
			&databaseBindingCreated,
		); err != nil {
			return fmt.Sprintf("%v+", err)
		}

		if !databaseBindingCreated.Status.Summary.Ready {
			return "not ready"
		}

		return "ready"
	}, databaseBindingCreationTimeout, 1*time.Second).Should(Equal("ready"))
}

func waitForPrivilegesBindingReady(privilegesBinding *v1alpha1.PrivilegesBinding) {
	EventuallyWithOffset(1, func() string {
		privilegesBindingCreated := v1alpha1.PrivilegesBinding{}
		if err := k8sClient.Get(
			ctx,
			types.NamespacedName{Name: privilegesBinding.Name, Namespace: privilegesBinding.Namespace},
			&privilegesBindingCreated,
		); err != nil {
			return fmt.Sprintf("%v+", err)
		}

		if !privilegesBindingCreated.Status.Summary.Ready {
			return "not ready"
		}

		return "ready"
	}, privilegesBindingCreationTimeout, 1*time.Second).Should(Equal("ready"))
}

func resetCLuster(_ *database.FakeDatabase, objects ...client.Object) {
	deleteObject := func(o client.Object) bool {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}, o)
		return apierrors.IsNotFound(err)
	}

	for _, o := range objects {
		Expect(k8sClient.Delete(ctx, o)).To(Succeed())
		Eventually(deleteObject, 5).WithArguments(o).Should(BeTrue())
	}
}

func createObjects(objects ...client.Object) {
	for _, o := range objects {
		Expect(k8sClient.Create(ctx, o)).To(Succeed())
	}
}

func uniqueName(s string, dbType v1alpha1.DatabaseType) string {
	return fmt.Sprintf("%s-%s", s, strings.ToLower(string(dbType)))
}

func bundle(namespace string, dbType v1alpha1.DatabaseType) (*v1alpha1.User, *v1.Secret, *v1alpha1.Database, *v1alpha1.DatabaseBinding, *v1alpha1.Privileges, *v1alpha1.PrivilegesBinding) {
	name := func(s string) string {
		return uniqueName(s, dbType)
	}

	database := &v1alpha1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name("database"),
			Namespace: namespace,
		},
		Spec: v1alpha1.DatabaseSpec{
			Type: dbType,
		},
	}

	user := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name("user"),
			Namespace: namespace,
		},
		PasswordSecret: v1alpha1.Secret{
			Secret: v1alpha1.NamespacedName{
				Name:      name("user-password"),
				Namespace: namespace,
			},
			Key: "pass",
		},
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name("user-password"),
			Namespace: namespace,
		},
		StringData: map[string]string{
			"pass": "mysupersecretpass",
		},
	}

	databaseBinding := &v1alpha1.DatabaseBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name("database-binding"),
			Namespace: namespace,
		},
		Spec: v1alpha1.DatabaseBindingSpec{
			Database: v1alpha1.NamespacedName{
				Namespace: namespace,
				Name:      name("database"),
			},
			User: v1alpha1.NamespacedName{
				Namespace: namespace,
				Name:      name("user"),
			},
		},
	}

	privileges := &v1alpha1.Privileges{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name("privileges"),
		},
		Privileges: []v1alpha1.PrivilegeSpec{
			{Privilege: "MY PRIVILEGE", On: "CUSTOM ON", Database: "DB"},
			{Privilege: "MY PRIVILEGE", Database: "DB"},
			{Privilege: "MY PRIVILEGE"},
		},
	}

	privilegesBinding := &v1alpha1.PrivilegesBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name("privileges-binding"),
		},
		Spec: v1alpha1.PrivilegesBindingSpec{
			DatabaseBindings: []v1alpha1.NamespacedName{
				{
					Namespace: namespace,
					Name:      name("database-binding"),
				},
			},
			Privileges: v1alpha1.NamespacedName{
				Namespace: namespace,
				Name:      name("privileges"),
			},
		},
	}

	return user, secret, database, databaseBinding, privileges, privilegesBinding
}
