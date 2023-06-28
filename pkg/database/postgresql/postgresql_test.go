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

package postgresql_test

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/logr"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/pkg/database/connection"
	"github.com/alex123012/database-users-operator/pkg/database/postgresql"
	testsutils "github.com/alex123012/database-users-operator/pkg/utils/tests_utils"
)

func TestPostgresql(t *testing.T) {
	type fields struct {
		config *postgresql.Config
		logger logr.Logger
	}
	type args struct {
		ctx        context.Context
		username   string
		password   string
		privileges []v1alpha1.PrivilegeSpec
	}

	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		queryList func(args) []string
	}{
		{
			name: "Create and delete user with password",
			fields: fields{
				config: postgresql.NewConfig("postgres", 5432, "user", "password", "dbname", "disable", "", "", "", ""),
				logger: logr.Discard(),
			},
			args: args{
				ctx:      context.Background(),
				username: "john_only_passwd",
				password: "mysupersecretpass",
			},
			queryList: func(a args) []string {
				return []string{
					fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s'`, a.username, a.password),
					fmt.Sprintf(`DROP USER "%s"`, a.username),
				}
			},
		},

		{
			name: "Create and delete user without",
			fields: fields{
				config: postgresql.NewConfig("postgres", 5432, "user", "password", "dbname", "disable", "", "", "", ""),
				logger: logr.Discard(),
			},
			args: args{
				ctx:      context.Background(),
				username: "john_only_passwd",
			},
			queryList: func(a args) []string {
				return []string{
					fmt.Sprintf(`CREATE USER "%s"`, a.username),
					fmt.Sprintf(`DROP USER "%s"`, a.username),
				}
			},
		},

		{
			name: "Create user with password, apply privileges, revoke privileges, delete user",
			fields: fields{
				config: postgresql.NewConfig("postgres", 5432, "user", "password", "dbname", "verify-full", testsutils.SSLCACert, testsutils.SSLJohnCert, testsutils.SSLJohnKey, testsutils.SSLCAKey),
				logger: logr.Discard(),
			},
			args: args{
				ctx:      context.Background(),
				username: "John_Doe",
				password: "JohnDoePassword",
				privileges: []v1alpha1.PrivilegeSpec{
					{Privilege: "ALL PRIVILEGES", On: "table", Database: "dat"},
					{Privilege: "CONNECT", Database: "conn_dat"},
					{Privilege: "rolename"},
				},
			},
			queryList: func(a args) []string {
				return []string{
					fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s'`, a.username, a.password),

					fmt.Sprintf(`GRANT %s ON "%s" TO "%s"`, a.privileges[0].Privilege, a.privileges[0].On, a.username),
					fmt.Sprintf(`GRANT %s ON DATABASE "%s" TO "%s"`, a.privileges[1].Privilege, a.privileges[1].Database, a.username),
					fmt.Sprintf(`GRANT %s TO "%s"`, a.privileges[2].Privilege, a.username),

					fmt.Sprintf(`REVOKE %s ON "%s" FROM "%s"`, a.privileges[0].Privilege, a.privileges[0].On, a.username),
					fmt.Sprintf(`REVOKE %s ON DATABASE "%s" FROM "%s"`, a.privileges[1].Privilege, a.privileges[1].Database, a.username),
					fmt.Sprintf(`REVOKE %s FROM "%s"`, a.privileges[2].Privilege, a.username),

					fmt.Sprintf(`DROP USER "%s"`, a.username),
				}
			},
		},
	}

	if err := checkCertsValidity(map[string]string{"ca.crt": testsutils.SSLCACert, "tls.crt": invalidSSLCert}); err == nil {
		t.Error("Invalid func for checking certs validity")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := connection.NewFakeConnection()
			p := postgresql.NewPostgresql(mockDB, tt.fields.config, tt.fields.logger)
			if err := p.Connect(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Postgresql.Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			data, err := p.CreateUser(tt.args.ctx, tt.args.username, tt.args.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Postgresql.Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if data != nil {
				if err := checkCertsValidity(data); err != nil {
					t.Error(err)
				}
			}

			if err := p.ApplyPrivileges(tt.args.ctx, tt.args.username, tt.args.privileges); (err != nil) != tt.wantErr {
				t.Errorf("Postgresql.Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := p.RevokePrivileges(tt.args.ctx, tt.args.username, tt.args.privileges); (err != nil) != tt.wantErr {
				t.Errorf("Postgresql.Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := p.DeleteUser(tt.args.ctx, tt.args.username); (err != nil) != tt.wantErr {
				t.Errorf("Postgresql.Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := p.Close(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Postgresql.Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			expectedQueries := tt.queryList(tt.args)
			actualQueries := mockDB.Queries()
			for i, query := range expectedQueries {
				if actualQueries[query] != i+1 {
					t.Errorf("Query not executed or executed out of order: %s", query)
				}
			}

			if len(expectedQueries) != len(actualQueries) {
				t.Errorf("Count of executed queries doesn't match: expected=%d, acrual=%d", len(expectedQueries), len(actualQueries))
			}
		})
	}
}

func checkCertsValidity(data map[string]string) error {
	if data["ca.crt"] != testsutils.SSLCACert {
		return errors.New("CA cert doen't match expencted CA cert")
	}
	return verifyCert(testsutils.SSLCACert, data["tls.crt"])
}

func verifyCert(caCertData, clientCertData string) error {
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM([]byte(caCertData)) {
		return errors.New("cannot append ca cert")
	}

	clientBlocks, _ := pem.Decode([]byte(clientCertData))
	if clientBlocks == nil {
		return errors.New("cannot decode clients cert")
	}

	clientCert, err := x509.ParseCertificate(clientBlocks.Bytes)
	if err != nil {
		return errors.New("cannot parse clients cert: " + err.Error())
	}

	_, err = clientCert.Verify(x509.VerifyOptions{
		Roots:     caPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	return err
}

const invalidSSLCert = `
-----BEGIN CERTIFICATE-----
MIID3zCCAsegAwIBAgIIWBXu6yBRCGMwDQYJKoZIhvcNAQELBQAwKzESMBAGA1UE
ChMJQ29ja3JvYWNoMRUwEwYDVQQDEwxDb2Nrcm9hY2ggQ0EwHhcNMjIxMDIzMTMy
NzQ1WhcNMjMxMDIzMTMyNzQ1WjAPMQ0wCwYDVQQDEwRqb2huMIICIjANBgkqhkiG
9w0BAQEFAAOCAg8AMIICCgKCAgEAp2INJu9aIU3AT+wp+S6+DlOHCXXedgjJI5NX
3rkM6PxSq4O6hbcMJIbziWY4dn6MTLVC/l7jA+e9gvKZZnYUCIQfn55ORt45aGFZ
QgWTcYT4YoFOEFwyNAy0iOSNZhhaZTFrNwjxgPPZkGS+JTLu92mZiMmjDuIXxVoV
Wv0CxA1V4Au00n7Jy4EjoOH5n17u9Lg3WwdsQh5S/RLx08m1aRB3btzldG+qlIpD
aKbGYkuwMZXjH06Eg1O3uiHQaJBRgyESK4etwGZjHIeNeDIMaWKdeUvzXF4Vb28M
v0cljAgyaLRyltNV/BmpZNnol5Aq/tyTm8/CGm4f1lBLiKWU5BsW9iT8DprDZ8Ev
tOUwewlkxJfzaRwpOZ2379Wf6WakI3IJq9OsSCmMFgOmTY8vuQ0Vrh/V/bOkhZ10
svMsQaczjplSi9ic9UaKcT9Rt6+Y3RlrsQkv85WrTPtNQ9O4h+710VknjhmbXQ7G
NCXX7+ldFxuXCOqSda9eOFlsUblitKWdZLyzxApZG8RU6ksYCDVpow/s98s1Y+F4
pMCSplgoINugGEePXqJC9oSoULen1ISLAF+eu25CSL+wKGMKPV2n4Qel4rIXEcqt
Jd0E3nSuvlMyJl9bDWILZJuWM/w4A1TJBAZuPXwiNpADEhSBjsYtxqDQiNHA9oE3
kPWaLE0CAwEAAaMjMCEwHwYDVR0jBBgwFoAU72qqGu5Bb8xAZTgRrayTXGWBu7Qw
DQYJKoZIhvcNAQELBQADggEBANVK1QnedsdOHSgjNJL/UmNFetXoeIv6aE0muam3
nsT/Hq9/JTl0TteiNZ50OFszqCQRtVb9f+TFHbeOfRttts4aSq0XHXESqqU04/pR
ms10ln9KO1bEOYkXliL70U2SFJLCwyJl67b6Le36oawXr0klppV7kgdd0ym1fQnF
K08BzyVM5hHdrKK8lzqRqJFUkreCe+Ck8VcIc/1YYGz/8deriPd/QCHnvDMSnV+x
uTXsNUV23VCYmS/R1/KEb7ZX1bNiyokfzUgb76RykOsn1dVVkvohd+0mLWhN92uq
EzDdOSlyCRx0x1w+jrh1/Pogf9EMCkdFXmWC1tqow5gGttU=
-----END CERTIFICATE-----
`
