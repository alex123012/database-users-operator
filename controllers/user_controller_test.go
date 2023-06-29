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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/pkg/utils"
	testsutils "github.com/alex123012/database-users-operator/pkg/utils/tests_utils"
)

const (
	userCreationTimeout = 10 * time.Second
)

var _ = Describe("UserController", Ordered, func() {
	Context("PostgreSQL with Cerificates", Ordered, func() {
		cfg := defaultPostgresConfig()

		cfg.SSLMode = "verify-full"
		cfg.SSLCredentialsSecret = v1alpha1.NamespacedName{
			Name:      "ssl-postgresql",
			Namespace: namespace,
		}
		cfg.SSLCAKey = v1alpha1.Secret{
			Key: "ca.key",
			Secret: v1alpha1.NamespacedName{
				Name:      "cakey-postgresql",
				Namespace: namespace,
			},
		}

		caKeySecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cakey-postgresql",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"ca.key": []byte(testsutils.SSLCAKey),
			},
		}

		sslUserSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ssl-postgresql",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"ca.crt":  []byte(testsutils.SSLCACert),
				"tls.key": []byte(testsutils.SSLJohnKey),
				"tls.crt": []byte(testsutils.SSLJohnCert),
			},
		}

		sslConnString := func(db string) string {
			p := func(p string) string { return utils.PathFromHome("postgres-certs/test-postgres", p) }
			return fmt.Sprintf("sslmode=verify-full sslrootcert=%s sslcert=%s sslkey=%s", p(db+"_test-user.ca"), p(db+"_test-user.crt"), p(db+"_test-user.key"))
		}

		connStrings := []string{
			fmt.Sprintf("%s %s", defaultPostgresConnString, sslConnString("")),
			fmt.Sprintf("%s %s", `pgx:host=test-postgres user=test-user port=5432 dbname=DB password=mysupersecretpass`, sslConnString("DB")),
		}

		queries := []string{
			`CREATE USER "user-postgresql" WITH PASSWORD 'mysupersecretpass'`,
			`GRANT MY PRIVILEGE ON "CUSTOM ON" TO "user-postgresql"`,
			`GRANT MY PRIVILEGE ON DATABASE "DB" TO "user-postgresql"`,
			`GRANT MY PRIVILEGE TO "user-postgresql"`,
		}

		removeQueries := []string{
			`REVOKE MY PRIVILEGE ON "CUSTOM ON" FROM "user-postgresql"`,
			`REVOKE MY PRIVILEGE ON DATABASE "DB" FROM "user-postgresql"`,
			`REVOKE MY PRIVILEGE FROM "user-postgresql"`,
			`DROP USER "user-postgresql"`,
		}

		tester := newTestDatabase(v1alpha1.PostgreSQL, cfg, fakeDB, connStrings, queries, removeQueries, true)
		tester.run(caKeySecret, sslUserSecret)
	})

	Context("MySQL", Ordered, func() {
		cfg := defaultMysqlConfig()

		connStrings := []string{defaultMysqlConnString}

		queries := []string{
			`CREATE USER ?@? IDENTIFIED BY ?user-mysql*mysupersecretpass`,
			`GRANT ? ON ?.? TO ?MY PRIVILEGEDBCUSTOM ONuser-mysql`,
			`GRANT ? ON ?.* TO ?MY PRIVILEGEDBuser-mysql`,
			`GRANT ? TO ?MY PRIVILEGEuser-mysql`,
		}

		removeQueries := []string{
			`REVOKE ? ON ?.? FROM ?MY PRIVILEGEDBCUSTOM ONuser-mysql`,
			`REVOKE ? ON ?.* FROM ?MY PRIVILEGEDBuser-mysql`,
			`REVOKE ? FROM ?MY PRIVILEGEuser-mysql`,
			`DROP USER ?@?user-mysql*`,
		}

		tester := newTestDatabase(v1alpha1.MySQL, cfg, fakeDB, connStrings, queries, removeQueries, false)
		tester.run()
	})
})
