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

package mysql

import (
	"context"
	"strings"

	"github.com/go-logr/logr"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/controllers/database/connection"
)

type Mysql struct {
	db     connection.Connection
	config *Config
	logger logr.Logger
}

func NewMysql(conn connection.Connection, config *Config, logger logr.Logger) *Mysql {
	return &Mysql{
		db:     conn,
		logger: logger,
		config: config,
	}
}

func (m *Mysql) Connect(ctx context.Context) error {
	connString, err := m.config.ConnString()
	if err != nil {
		return err
	}
	return m.db.Connect(ctx, "mysql", connString)
}

func (m *Mysql) Close(ctx context.Context) error {
	return m.db.Close(ctx)
}

func (m *Mysql) CreateUser(ctx context.Context, username, password string) (map[string]string, error) {
	query := "CREATE USER ?@? IDENTIFIED BY ?"
	return nil, m.db.Exec(ctx, connection.DisableLogger, query, username, m.config.UsersHostname(), password)
}

func (m *Mysql) DeleteUser(ctx context.Context, username string) error {
	query := "DROP USER ?@?"
	return m.db.Exec(ctx, connection.EnableLogger, query, username, m.config.UsersHostname())
}

func (m *Mysql) ApplyPrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error {
	return m.privilegesProcessor(ctx, username, privileges, "GRANT", "TO")
}

func (m *Mysql) RevokePrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error {
	return m.privilegesProcessor(ctx, username, privileges, "REVOKE", "FROM")
}

func (m *Mysql) privilegesProcessor(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec, statement, arg string) error {
	for _, privilege := range privileges {
		query, args := prepareStatementForPrivilege(statement, arg, username, privilege.Database, privilege.On, privilege.Privilege)
		if err := m.db.Exec(ctx, connection.EnableLogger, query, args...); err != nil {
			return err
		}
	}
	return nil
}

func prepareStatementForPrivilege(statement, arg, username, dbname, on string, privilege v1alpha1.PrivilegeType) (string, []interface{}) {
	stmtBuilder := &strings.Builder{}
	stmtBuilder.WriteString(statement)
	stmtBuilder.WriteString(" ")
	stmtBuilder.WriteString("?")
	args := []interface{}{privilege}
	if on != "" && dbname != "" {
		stmtBuilder.WriteString(" ON ")
		stmtBuilder.WriteString("?.?")
		args = append(args, dbname, on)
	} else if dbname != "" {
		stmtBuilder.WriteString(" ON ")
		stmtBuilder.WriteString("?.*")
		args = append(args, dbname)
	}
	stmtBuilder.WriteString(" ")
	stmtBuilder.WriteString(arg)
	stmtBuilder.WriteString(" ?")
	args = append(args, username)
	return stmtBuilder.String(), args
}
