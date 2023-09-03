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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/pkg/database/postgresql"
	testsutils "github.com/alex123012/database-users-operator/pkg/utils/tests_utils"
)

func TestConfig_ConnString(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Error(err, "can't get user home dir")
	}
	type fields struct {
		Host         string
		User         string
		Password     string
		DatabaseName string
		Port         int
		SSLMode      v1alpha1.PostgresSSLMode
		SSLCACert    string
		SSLUserCert  string
		SSLUserKey   string
	}
	type args struct {
	}
	tests := []struct {
		name                   string
		fields                 fields
		args                   args
		want                   string
		wantErr                bool
		wantCreateCertificates bool
	}{
		{
			name:                   "SSL config",
			want:                   fmt.Sprintf("host=postgres user=user port=5432 dbname=dbname password=password sslmode=verify-full sslrootcert=%s/postgres-certs/postgres/dbname_user.ca sslcert=%s/postgres-certs/postgres/dbname_user.crt sslkey=%s/postgres-certs/postgres/dbname_user.key", home, home, home),
			wantCreateCertificates: true,
			args:                   args{},
			fields: fields{
				Host:         "postgres",
				User:         "user",
				Password:     "password",
				DatabaseName: "dbname",
				Port:         5432,
				SSLMode:      "verify-full",
				SSLCACert:    testsutils.SSLCACert,
				SSLUserCert:  testsutils.SSLJohnCert,
				SSLUserKey:   testsutils.SSLJohnKey,
			},
		},
	}
	for _, tt := range tests {
		fs := []struct {
			ext     string
			content string
		}{
			{
				ext:     "ca",
				content: tt.fields.SSLCACert,
			},
			{
				ext:     "crt",
				content: tt.fields.SSLUserCert,
			},
			{
				ext:     "key",
				content: tt.fields.SSLUserKey,
			},
		}

		t.Run(tt.name, func(t *testing.T) {
			c := postgresql.NewConfig(tt.fields.Host, tt.fields.Port, tt.fields.User, tt.fields.Password, tt.fields.DatabaseName,
				tt.fields.SSLMode, tt.fields.SSLCACert, tt.fields.SSLUserCert, tt.fields.SSLUserKey, "")
			defer c.Close()

			got, err := c.ConnString()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.ConnString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Config.ConnString() = %v, want %v", got, tt.want)
			}
			if c.CreateCerts() != tt.wantCreateCertificates {
				t.Errorf("Config.CreateCerts() = %v, want %v", c.CreateCerts(), tt.wantCreateCertificates)
			}

			if tt.wantCreateCertificates {
				for _, f := range fs {
					if err := compareFileContent(f.content, f.ext, tt.fields.Host, tt.fields.DatabaseName, tt.fields.User); err != nil {
						t.Error(err)
					}
				}
			}
		})

		for _, f := range fs {
			f := certPath(tt.fields.Host, tt.fields.DatabaseName, tt.fields.User, f.ext)
			if _, err := os.Stat(f); !os.IsNotExist(err) {
				t.Errorf("file not deleted: %s", f)
			}
		}
	}
}

func certPath(host, dbname, user, ext string) string {
	return filepath.Join(os.Getenv("HOME"), fmt.Sprintf("postgres-certs/%s/%s_%s.%s", host, dbname, user, ext))
}

func compareFileContent(content, ext, host, dbname, user string) error {
	fullPath := certPath(host, dbname, user, ext)
	fileContent, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	if string(fileContent) != content {
		return fmt.Errorf("file content not equal to wanted: %s", fullPath)
	}
	return nil
}
