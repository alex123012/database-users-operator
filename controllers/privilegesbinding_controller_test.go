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
	"time"

	. "github.com/onsi/ginkgo/v2"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
)

const (
	privilegesBindingCreationTimeout = 3 * time.Second
	// privilegesBindingDeletionTimeout = 3 * time.Second.
)

var _ = Describe("PrivilegeBindingController", Ordered, func() {
	Context("PostgreSQL", Ordered, func() {
		cfg := defaultPostgresConfig()

		connStrings := []string{
			defaultPostgresConnString,
			`pgx:host=test-postgres user=test-user port=5432 dbname=DB password=mysupersecretpass`,
		}

		queries := []string{
			`GRANT MY PRIVILEGE ON "CUSTOM ON" TO "user-postgresql"`,
			`GRANT MY PRIVILEGE ON DATABASE "DB" TO "user-postgresql"`,
			`GRANT MY PRIVILEGE TO "user-postgresql"`,
		}

		tester := newTestDatabase(namespace, v1alpha1.PostgreSQL, cfg, fakeDBCreatorPrivileges, connStrings, queries)
		tester.run()
	})

	Context("MySQL", Ordered, func() {
		cfg := defaultMysqlConfig()

		connStrings := []string{defaultMysqlConnString}

		queries := []string{
			`GRANT ? ON ?.* TO ?MY PRIVILEGEDBuser-mysql`,
			`GRANT ? ON ?.? TO ?MY PRIVILEGEDBCUSTOM ONuser-mysql`,
			`GRANT ? TO ?MY PRIVILEGEuser-mysql`,
		}

		tester := newTestDatabase(namespace, v1alpha1.MySQL, cfg, fakeDBCreatorPrivileges, connStrings, queries)
		tester.run()
	})
})
