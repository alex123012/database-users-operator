package postgresql_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/controllers/database/postgresql"
)

func TestConfig_ConnString(t *testing.T) {
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
		deleteFilesSigChan chan struct{}
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
			want:                   "host=postgres user=user port=5432 sslmode=verify-full dbname=dbname password=password sslrootcert=/Users/alexmakh/postgres-certs/postgres/dbname_user.ca sslcert=/Users/alexmakh/postgres-certs/postgres/dbname_user.crt sslkey=/Users/alexmakh/postgres-certs/postgres/dbname_user.key",
			wantCreateCertificates: true,
			args: args{
				deleteFilesSigChan: make(chan struct{}),
			},
			fields: fields{
				Host:         "postgres",
				User:         "user",
				Password:     "password",
				DatabaseName: "dbname",
				Port:         5432,
				SSLMode:      "verify-full",
				SSLCACert:    sslCACert,
				SSLUserCert:  sslJohnCert,
				SSLUserKey:   sslJohnKey,
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
			defer close(tt.args.deleteFilesSigChan)

			c := postgresql.NewConfig(tt.fields.Host, tt.fields.Port, tt.fields.User, tt.fields.Password, tt.fields.DatabaseName,
				tt.fields.SSLMode, tt.fields.SSLCACert, tt.fields.SSLUserCert, tt.fields.SSLUserKey, "")
			got, err := c.ConnString(tt.args.deleteFilesSigChan)
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
		<-tt.args.deleteFilesSigChan
		time.Sleep(time.Second) // sleep for defer function invoke

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
