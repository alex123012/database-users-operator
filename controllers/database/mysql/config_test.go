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
	"testing"

	"github.com/alex123012/database-users-operator/controllers/database/mysql"
)

func TestConfig_ConnString(t *testing.T) {
	type fields struct {
		Host         string
		Port         int
		User         string
		Password     string
		DatabaseName string

		usersHostname string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "default conn string",
			want: "john:MyPass@tcp(mysql:3306)/default?interpolateParams=true",
			fields: fields{
				Host:         "mysql",
				Port:         3306,
				User:         "john",
				Password:     "MyPass",
				DatabaseName: "default",
			},
		},
		{
			name: "without password, db and user",
			want: "tcp(mysql:3306)/?interpolateParams=true",
			fields: fields{
				Host: "mysql",
				Port: 3306,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mysql.NewConfig(tt.fields.Host, tt.fields.Port, tt.fields.User, tt.fields.Password, tt.fields.DatabaseName, tt.fields.usersHostname)
			got, err := c.ConnString()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.ConnString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Config.ConnString() = %v, want %v", got, tt.want)
			}
		})
	}
}
