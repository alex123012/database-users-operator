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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
)

const (
	privilegesBindingCreationTimeout = 3 * time.Second
	// privilegesBindingDeletionTimeout = 3 * time.Second.
)

var _ = Describe("PrivilegeBindingController", func() {
	var (
		user              *v1alpha1.User
		database          *v1alpha1.Database
		databaseBinding   *v1alpha1.DatabaseBinding
		privileges        *v1alpha1.Privileges
		privilegesBinding *v1alpha1.PrivilegesBinding
		namespace         = "default"
	)
	Context("default behaviour", func() {
		BeforeEach(func() {
			user, database, databaseBinding, privileges, privilegesBinding = privilegesBindingBundle(namespace, v1alpha1.PostgreSQL)
			Expect(k8sClient.Create(ctx, user)).To(Succeed())
			Expect(k8sClient.Create(ctx, database)).To(Succeed())
			Expect(k8sClient.Create(ctx, databaseBinding)).To(Succeed())
			Expect(k8sClient.Create(ctx, privileges)).To(Succeed())
			Expect(k8sClient.Create(ctx, privilegesBinding)).To(Succeed())
			waitForPrivilegesBindingReady(privilegesBinding)
		})

		AfterEach(func() {
			for _, o := range []client.Object{privilegesBinding, databaseBinding, user, database, privileges} {
				Expect(k8sClient.Delete(ctx, o)).To(Succeed())
				Eventually(func(o client.Object) bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}, o)
					return apierrors.IsNotFound(err)
				}, 5).WithArguments(o).Should(BeTrue())
			}
			fakeDBCreatorPrivileges.ResetDB()
		})

		It("works", func() {
			queries := fakeDBCreatorPrivileges.Conn.Queries()
			By("Applied privileges to user in DB", func() {
				Expect(queries[`GRANT MY PRIVILEGE ON "CUSTOM ON" TO "user-2"`]).NotTo(Equal(0))
				Expect(queries[`GRANT MY PRIVILEGE ON DATABASE "DB" TO "user-2"`]).NotTo(Equal(0))
				Expect(queries[`GRANT MY PRIVILEGE TO "user-2"`]).NotTo(Equal(0))
			})
		})
	})
})

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

func privilegesBindingBundle(namespace string, dbType v1alpha1.DatabaseType) (*v1alpha1.User, *v1alpha1.Database, *v1alpha1.DatabaseBinding, *v1alpha1.Privileges, *v1alpha1.PrivilegesBinding) {
	database := &v1alpha1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "database-2",
		},
		Spec: v1alpha1.DatabaseSpec{
			Type: dbType,
		},
	}

	user := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "user-2",
		},
	}

	databaseBinding := &v1alpha1.DatabaseBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "database-binding-2",
		},
		Spec: v1alpha1.DatabaseBindingSpec{
			Database: v1alpha1.NamespacedName{
				Namespace: namespace,
				Name:      "database-2",
			},
			User: v1alpha1.NamespacedName{
				Namespace: namespace,
				Name:      "user-2",
			},
		},
	}

	privileges := &v1alpha1.Privileges{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "privileges-2",
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
			Name:      "privileges-binding-2",
		},
		Spec: v1alpha1.PrivilegesBindingSpec{
			DatabaseBindings: []v1alpha1.NamespacedName{
				{Namespace: namespace, Name: "database-binding-2"},
			},
			Privileges: v1alpha1.NamespacedName{
				Namespace: namespace,
				Name:      "privileges-2",
			},
		},
	}

	return user, database, databaseBinding, privileges, privilegesBinding
}
