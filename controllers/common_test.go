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
	"github.com/alex123012/database-users-operator/pkg/utils"
)

const (
	defaultMysqlConnString    = "mysql:test-user:mysupersecretpass@tcp(test-mysql:3306)/?interpolateParams=true"
	defaultPostgresConnString = "pgx:host=test-postgres user=test-user port=5432 password=mysupersecretpass"
)

func defaultPostgresConfig() *v1alpha1.PostgreSQLConfig {
	return &v1alpha1.PostgreSQLConfig{
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

func defaultMysqlConfig() *v1alpha1.MySQLConfig {
	return &v1alpha1.MySQLConfig{
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
	dbType            v1alpha1.DatabaseType
	dbConfig          interface{}
	fakeDB            *database.FakeDatabase
	connectionStrings []string
	queries           []string
	removeQueries     []string
	creteUserSecret   bool
}

func newTestDatabase(dbType v1alpha1.DatabaseType, dbConfig interface{}, fakeDB *database.FakeDatabase, connStrings, queries, removeQueries []string, creteUserSecret bool) testDatabase {
	return testDatabase{
		dbType:            dbType,
		dbConfig:          dbConfig,
		fakeDB:            fakeDB,
		connectionStrings: connStrings,
		queries:           queries,
		removeQueries:     removeQueries,
		creteUserSecret:   creteUserSecret,
	}
}

func (t testDatabase) run(additionalObjects ...client.Object) {
	var (
		user       *v1alpha1.User
		secret     *v1.Secret
		database   *v1alpha1.Database
		privileges *v1alpha1.Privileges
	)

	BeforeEach(func() {
		user, secret, database, privileges = bundle(namespace, t.dbType)
		switch t.dbType {
		case v1alpha1.PostgreSQL:
			database.Spec.PostgreSQL = t.dbConfig.(*v1alpha1.PostgreSQLConfig)
		case v1alpha1.MySQL:
			database.Spec.MySQL = t.dbConfig.(*v1alpha1.MySQLConfig)
		default:
			Fail("not supported db")
		}

		createObjects(additionalObjects...)

		t.fakeDB.Conn.ResetDB()
		createObjects(secret, database, privileges, user)
		waitForUsersReadiness(user)
	})

	AfterEach(func() {
		t.fakeDB.Conn.ResetDB()
		deleteObjects(user, secret, database, privileges)
		deleteObjects(additionalObjects...)

		time.Sleep(userCreationTimeout)
		checkQueries(t.fakeDB, t.removeQueries)

		By("Deleting users secret", func() {
			if !t.creteUserSecret {
				return
			}

			_, err := utils.Secret(ctx, types.NamespacedName{Namespace: namespace, Name: uniqueName("created-secret", t.dbType)}, k8sClient)
			Expect(err).To(HaveOccurred())
		})
	})

	It("works", func() {
		checkConnectionStrings(fakeDB, t.connectionStrings)

		checkQueries(fakeDB, t.queries)

		By("Creating users secret", func() {
			if !t.creteUserSecret {
				return
			}
			_, err := utils.Secret(ctx, types.NamespacedName{Namespace: namespace, Name: uniqueName("created-secret", t.dbType)}, k8sClient)
			Expect(err).ToNot(HaveOccurred())
		})

		By("setting proper status", func() {
			fetchedUser := &v1alpha1.User{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: user.GetName()}, fetchedUser)).To(Succeed())
			Expect(fetchedUser.Status).To(Equal(v1alpha1.UserStatus{Summary: v1alpha1.StatusSummary{Message: "Successfully created user in all specified databases", Ready: true}}))
		})

		By("adding event", func() {
			events := &v1.EventList{}
			Expect(k8sClient.List(ctx, events, &client.ListOptions{Namespace: namespace})).To(Succeed())
			checkEvents(events, user, "SuccessfullyCreatedUser", "Successfully created user in all specified databases")
		})
	})
}

func checkConnectionStrings(fakeDB *database.FakeDatabase, expected []string) {
	connections := fakeDB.Conn.Connections()

	By("connection strings len", func() {
		Expect(connections).To(HaveLen(len(expected)))
	})

	for _, connString := range expected {
		By("Connection string: "+connString, func() {
			Expect(connections[connString]).To(BeTrue())
		})
	}
}

func checkQueries(fakeDB *database.FakeDatabase, expected []string) {
	queries := fakeDB.Conn.Queries()
	By("Executed queries len", func() {
		Expect(queries).To(HaveLen(len(expected)))
	})

	var previousQueryOrder int
	for _, query := range expected {
		currentQueryOrder := queries[query]
		By("Executed query: "+query, func() {
			Expect(currentQueryOrder).NotTo(Equal(0))
			Expect(currentQueryOrder).To(BeNumerically(">", previousQueryOrder))
			previousQueryOrder = currentQueryOrder
		})
	}
}

func checkEvents(events *v1.EventList, obj client.Object, reason, msg string) {
	Expect(events.Items).NotTo(BeEmpty())
	for _, event := range events.Items {
		if event.InvolvedObject.Name == obj.GetName() && event.Reason == reason {
			Expect(event.Message).To(Equal(msg))
			return
		}
	}
	failMsg := fmt.Sprintf("No event found for name=%s reason=%s", obj.GetName(), reason)
	Fail(failMsg)
}

func waitForUsersReadiness(user *v1alpha1.User) {
	EventuallyWithOffset(1, func() string {
		userCreated := v1alpha1.User{}
		if err := k8sClient.Get(
			ctx,
			types.NamespacedName{Name: user.Name, Namespace: user.Namespace},
			&userCreated,
		); err != nil {
			return fmt.Sprintf("%v+", err)
		}

		if !userCreated.Status.Summary.Ready {
			return "not ready"
		}
		time.Sleep(userCreationTimeout)
		return "ready"
	}, userCreationTimeout, 1*time.Second).Should(Equal("ready"))
}

func objectNotFound(o client.Object) bool {
	err := k8sClient.Get(ctx, types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}, o)
	return apierrors.IsNotFound(err)
}

func deleteObjects(objects ...client.Object) {
	for _, o := range objects {
		By(fmt.Sprintf("Deleting %s from namespace %s", o.GetName(), o.GetNamespace()), func() {
			Expect(k8sClient.Delete(ctx, o)).To(Succeed())
			Eventually(objectNotFound, 5).WithArguments(o).Should(BeTrue())
		})
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

func bundle(namespace string, dbType v1alpha1.DatabaseType) (*v1alpha1.User, *v1.Secret, *v1alpha1.Database, *v1alpha1.Privileges) {
	name := func(s string) string {
		return uniqueName(s, dbType)
	}

	database := &v1alpha1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name: name("database"),
		},
		Spec: v1alpha1.DatabaseSpec{
			Type: dbType,
		},
	}

	privileges := &v1alpha1.Privileges{
		ObjectMeta: metav1.ObjectMeta{
			Name: name("privileges"),
		},
		Privileges: []v1alpha1.PrivilegeSpec{
			{Privilege: "MY PRIVILEGE", On: "CUSTOM ON", Database: "DB"},
			{Privilege: "MY PRIVILEGE", Database: "DB"},
			{Privilege: "MY PRIVILEGE"},
		},
	}

	user := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: name("user"),
		},
		Spec: v1alpha1.UserSpec{
			Databases: []v1alpha1.DatabaseRef{
				{
					Name: name("database"),
					PasswordSecret: v1alpha1.Secret{
						Key: "pass",
						Secret: v1alpha1.NamespacedName{
							Name:      name("user-password"),
							Namespace: namespace,
						},
					},
					CreatedSecret: v1alpha1.NamespacedName{
						Name:      name("created-secret"),
						Namespace: namespace,
					},
					Privileges: []v1alpha1.Name{
						{Name: name("privileges")},
					},
				},
			},
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

	return user, secret, database, privileges
}
