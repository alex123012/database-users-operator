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
	databaseBindingCreationTimeout = 3 * time.Second
	// databaseBindingDeletionTimeout = 3 * time.Second.
)

var _ = Describe("DatabaseBindingController", Ordered, func() {
	Context("PostgreSQL", Ordered, func() {
		cfg := defaultPostgresConfig()
		connStrings := []string{defaultPostgresConnString}

		queries := []string{
			`CREATE USER "user-postgresql" WITH PASSWORD 'mysupersecretpass'`,
		}

		tester := newTestDatabase(namespace, v1alpha1.PostgreSQL, cfg, fakeDBCreatorDB, connStrings, queries)
		tester.run()
	})

	Context("MySQL", Ordered, func() {
		cfg := defaultMysqlConfig()
		connStrings := []string{defaultMysqlConnString}

		queries := []string{
			`CREATE USER ?@? IDENTIFIED BY ?user-mysql*mysupersecretpass`,
		}

		tester := newTestDatabase(namespace, v1alpha1.MySQL, cfg, fakeDBCreatorDB, connStrings, queries)
		tester.run()
	})
})
