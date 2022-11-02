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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("User", func() {

	Context("ConfigSpec", func() {
		It("can be created without privileges", func() {
			created := generateUserObject("user1")
			Expect(k8sClient.Create(context.Background(), created)).To(Succeed())

			fetched := &User{}
			Expect(k8sClient.Get(context.Background(), getUserKey(created), fetched)).To(Succeed())
			Expect(fetched).To(Equal(created))
		})

		It("can be deleted", func() {
			created := generateUserObject("user2")
			Expect(k8sClient.Create(context.Background(), created)).To(Succeed())

			Expect(k8sClient.Delete(context.Background(), created)).To(Succeed())
			Expect(k8sClient.Get(context.Background(), getUserKey(created), created)).ToNot(Succeed())
		})

		It("is validated", func() {
			var (
				ErrRoleWithDbOrTable = "'database' or 'on' field can't be set when 'privilege' is role name"
				ErrMustBeUpperCase   = "privilege must be upper case"
			)

			type testCase struct {
				Description string
				Resource    *User
				Error       string
			}

			userObject := generateUserObject("user3")
			validationTestCases := []testCase{
				{
					Description: "checking the database configs count",
					Resource:    userObject.SetDbConfigs([]DatabaseConfig{}),
					Error:       "database configs can't be empty",
				},
				{
					Description: "checking the database configs name set",
					Resource:    userObject.SetDbConfigs([]DatabaseConfig{{}, {}}),
					Error:       "database config name can't be empty",
				},
				{
					Description: "checking the privilege rolename empty",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "admin"}, {}}),
					Error:       "privilege can't be empty field",
				},
				{
					Description: "checking the lowercase all table in schema privelege",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "SELECT", On: "all tables in schema public", Database: "db"}}),
					Error:       "privilege on all schema must be upper case",
				},
				{
					Description: "checking the table privilege without db",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "SELECT", On: "table"}}),
					Error:       "'database' can't be empty field when 'privilege' is table scoped",
				},
				{
					Description: "checking the table privilege without table",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "SELECT", Database: "db"}}),
					Error:       "'on' can't be empty field when 'privilege' is table scoped",
				},
				{
					Description: "checking the database privilege without db",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "CREATE"}}),
					Error:       "'database' can't be empty field when 'privilege' is database scoped",
				},
				{
					Description: "checking the database privilege with table",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "CREATE", Database: "db", On: "table"}}),
					Error:       "'on' field can't be set when 'privilege' is database scoped",
				},
				{
					Description: "checking the all privilege without db",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "ALL PRIVILEGES"}}),
					Error:       "'database' or 'on' fields can't be empty both when 'privilege' is set to 'ALL' or 'ALL PRIVILEGES'",
				},
				{
					Description: "checking the all privilege without db",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "ALL PRIVILEGES", On: "table"}}),
					Error:       "'database' can't be empty field when 'privilege' is set to 'ALL' or 'ALL PRIVILEGES' and 'on' field is set",
				},
				{
					Description: "checking the lowercase table privelege",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "select", On: "table", Database: "db"}}),
					Error:       ErrMustBeUpperCase,
				},
				{
					Description: "checking the lowercase db privelege",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "create", Database: "db"}}),
					Error:       ErrMustBeUpperCase,
				},
				{
					Description: "checking the privilege rolename set with table",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "admin", On: "table"}}),
					Error:       ErrRoleWithDbOrTable,
				},
				{
					Description: "checking the privilege rolename set with database",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "admin", Database: "db"}}),
					Error:       ErrRoleWithDbOrTable,
				},
				{
					Description: "checking the privilege rolename set with database and table",
					Resource:    userObject.SetPrivileges([]Privilege{{Privilege: "admin", On: "table", Database: "db"}}),
					Error:       ErrRoleWithDbOrTable,
				},
				{
					Description: "checking lower case all tables in schema privileges",
					Resource:    userObject.SetPrivileges([]Privilege{{On: "all TABLES IN SCHEMA public", Database: "db", Privilege: "SELECT"}}),
					Error:       "privilege on all schema must be upper case",
				},
			}

			for _, testcase := range validationTestCases {
				By(testcase.Description, func() {
					invalidUser := testcase.Resource
					Expect(apierrors.IsInvalid(k8sClient.Create(context.Background(), invalidUser))).To(BeTrue())
					Expect(k8sClient.Create(context.Background(), invalidUser)).To(MatchError(ContainSubstring(testcase.Error)))
				})
			}

			By("checking vaild User", func() {
				validUser := generateUserObject("user4").SetPrivileges([]Privilege{
					{Privilege: "admin"},
					{Privilege: "INSERT", On: "table", Database: "db"},
					{Privilege: "CREATE", Database: "db"},
					{Privilege: "SELECT", On: "ALL TABLES IN SCHEMA public", Database: "db"},
					{Privilege: "ALL PRIVILEGES", Database: "db", On: "table"},
				})

				Expect(k8sClient.Create(context.Background(), validUser)).To(Succeed())
				fetchedUser := &User{}
				Expect(k8sClient.Get(context.Background(), getUserKey(validUser), fetchedUser)).To(Succeed())
				Expect(fetchedUser.Spec).To(Equal(validUser.Spec))
			})

		})

		Context("Default settings", func() {
			var (
				userObject         User
				expectedUserObject User
			)

			BeforeEach(func() {
				expectedUserObject = *generateUserObject("foo")
			})

			When("Namespace not set", func() {
				It("Sets namespace for passwordSecret", func() {
					userObject = *generateUserObject("user5")
					userObject.Spec.PasswordSecret.Namespace = ""

					Expect(k8sClient.Create(context.Background(), &userObject)).To(Succeed())
					fetchedUser := &User{}
					Expect(k8sClient.Get(context.Background(), getUserKey(&userObject), fetchedUser)).To(Succeed())
					Expect(fetchedUser.Spec).To(Equal(expectedUserObject.Spec))
				})

				It("Sets namespace for database configs", func() {
					expectedUserObject.Spec.DatabaseConfigs = []DatabaseConfig{
						{Name: "test1", Namespace: "default"},
						{Name: "test1", Namespace: "Kek"},
						{Name: "test1", Namespace: "default"},
					}

					userObject = *generateUserObject("user6")
					userObject.Spec.DatabaseConfigs = []DatabaseConfig{
						{Name: "test1"},
						{Name: "test1", Namespace: "Kek"},
						{Name: "test1"},
					}

					Expect(k8sClient.Create(context.Background(), &userObject)).To(Succeed())
					fetchedUser := &User{}
					Expect(k8sClient.Get(context.Background(), getUserKey(&userObject), fetchedUser)).To(Succeed())
					Expect(fetchedUser.Spec).To(Equal(expectedUserObject.Spec))
				})
			})
		})
	})
})

func getUserKey(user *User) types.NamespacedName {
	return types.NamespacedName{
		Name:      user.GetName(),
		Namespace: user.GetNamespace(),
	}
}

func generateUserObject(username string) *User {
	namespace := "default"
	return &User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      username,
			Namespace: namespace,
		},
		Spec: UserSpec{
			DatabaseConfigs: []DatabaseConfig{
				{
					Name:      "test1",
					Namespace: namespace,
				},
			},
			PasswordSecret: Secret{
				Name:      "test",
				Namespace: namespace,
			},
			Privileges: []Privilege{},
		}}
}
