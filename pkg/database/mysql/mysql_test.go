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

package mysql_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/pkg/database/connection"
	"github.com/alex123012/database-users-operator/pkg/database/mysql"
)

func TestPostgresql(t *testing.T) {
	type fields struct {
		config *mysql.Config
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
		queryList func(args, fields) []string
	}{
		{
			name: "Create user with password, apply privileges, revoke privileges, delete user",
			fields: fields{
				config: mysql.NewConfig("mysql", 3306, "user", "password", "dbname", ""),
				logger: logr.Discard(),
			},
			args: args{
				ctx:      context.Background(),
				username: "john",
				password: "mysupersecretpass",
				privileges: []v1alpha1.PrivilegeSpec{
					{Privilege: "ALL PRIVILEGES", On: "table", Database: "dat"},
					{Privilege: "CONNECT", Database: "conn_dat"},
					{Privilege: "rolename"},
				},
			},
			queryList: func(a args, f fields) []string {
				return []string{
					fmt.Sprint(`CREATE USER ?@? IDENTIFIED BY ?`, a.username, f.config.UsersHostname(), a.password),

					fmt.Sprint(`GRANT ? ON ?.? TO ?`, a.privileges[0].Privilege, a.privileges[0].Database, a.privileges[0].On, a.username),
					fmt.Sprint(`GRANT ? ON ?.* TO ?`, a.privileges[1].Privilege, a.privileges[1].Database, a.username),
					fmt.Sprint(`GRANT ? TO ?`, a.privileges[2].Privilege, a.username),

					fmt.Sprint(`REVOKE ? ON ?.? FROM ?`, a.privileges[0].Privilege, a.privileges[0].Database, a.privileges[0].On, a.username),
					fmt.Sprint(`REVOKE ? ON ?.* FROM ?`, a.privileges[1].Privilege, a.privileges[1].Database, a.username),
					fmt.Sprint(`REVOKE ? FROM ?`, a.privileges[2].Privilege, a.username),

					fmt.Sprint(`DROP USER ?@?`, a.username, f.config.UsersHostname()),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := connection.NewFakeConnection()
			p := mysql.NewMysql(mockDB, tt.fields.config, tt.fields.logger)
			if err := p.Connect(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Postgresql.Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if _, err := p.CreateUser(tt.args.ctx, tt.args.username, tt.args.password); (err != nil) != tt.wantErr {
				t.Errorf("Postgresql.Connect() error = %v, wantErr %v", err, tt.wantErr)
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

			expectedQueries := tt.queryList(tt.args, tt.fields)
			actualQueries := mockDB.Queries()
			for i, query := range expectedQueries {
				if actualQueries[query] != i+1 {
					t.Errorf("Query not executed or executed out of order: '%s', %d", query, i+1)
				}
			}

			if len(expectedQueries) != len(actualQueries) {
				t.Errorf("Count of executed queries doesn't match: expected=%d, acrual=%d", len(expectedQueries), len(actualQueries))
			}
		})
	}
}
