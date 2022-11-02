/*
Copyright 2022.

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

package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-logr/logr"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

type disableLoggerType int

const (
	DisableLogger disableLoggerType = 1
	EnableLogger  disableLoggerType = 0
)

type DBconnector struct {
	db         *sqlx.DB
	connString string
	dbDriver   DBDriver
	logger     logr.Logger
}

func NewDBConnector(connString string, dbDriver DBDriver, logger logr.Logger) *DBconnector {
	return &DBconnector{
		connString: connString,
		dbDriver:   dbDriver,
		logger:     logger,
	}
}
func (d *DBconnector) Connect(ctx context.Context) error {
	db, err := sqlx.ConnectContext(ctx, string(d.dbDriver), d.connString)
	if err != nil {
		return err
	}
	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(1)
	d.db = db
	return nil
}

func (d *DBconnector) Close(ctx context.Context) error {
	return d.db.Close()
}

func (d *DBconnector) infoLog(q string, args []any) []any {
	if len(args) > 0 && args[0] == DisableLogger {
		return args[1:]
	}
	d.logger.Info(fmt.Sprintf("Executing statement '%s' with values: %v", q, args))
	return args
}

func (d *DBconnector) Exec(ctx context.Context, q string, args ...any) error {
	args = d.infoLog(q, args)
	return exec(ctx, d.db, q, args...)
}

func (d *DBconnector) Query(ctx context.Context, q string, args ...any) (*sqlx.Rows, error) {
	args = d.infoLog(q, args)
	return query(ctx, d.db, q, args...)
}

func (d *DBconnector) NamedExec(ctx context.Context, q NamedQuery, disableArg ...any) error {
	d.infoLog(q.Query, disableArg)
	return namedExec(ctx, d.db, q.Query, q.Arg)
}

func (d *DBconnector) NamedQuery(ctx context.Context, q NamedQuery, disableArg ...any) (*sqlx.Rows, error) {
	d.infoLog(q.Query, disableArg)
	return namedQuery(ctx, d.db, q.Query, q.Arg)
}

func (d *DBconnector) Select(ctx context.Context, dest any, quer string, args ...any) error {
	args = d.infoLog(quer, args)
	return selectx(ctx, d.db, dest, quer, args...)
}

func (d *DBconnector) Get(ctx context.Context, dest any, quer string, args ...any) error {
	args = d.infoLog(quer, args)
	return getx(ctx, d.db, dest, quer, args...)
}

func (d *DBconnector) ExecTx(ctx context.Context, queryList []Query, namedQuery []NamedQuery, disableArg ...any) error {
	tx, err := d.db.BeginTxx(ctx, &sql.TxOptions{})
	d.logger.Info("Executing transaction...")
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			d.logger.Error(err, "Rollback tx error")
		}
	}()
	for _, q := range queryList {
		args := d.infoLog(q.Query, q.Args)
		if err := exec(ctx, tx, q.Query, args...); err != nil {
			return err
		}
	}
	for _, namedQuery := range namedQuery {
		d.infoLog(namedQuery.Query, disableArg)
		if err := namedExec(ctx, tx, namedQuery.Query, namedQuery.Arg); err != nil {
			return err
		}
	}
	d.logger.Info("End transaction.")
	return tx.Commit()
}

func (d *DBconnector) MapperFunc(tagName string, f func(string) string) {
	d.db.Mapper = reflectx.NewMapperFunc(tagName, f)
}

func selectx(ctx context.Context, q sqlx.QueryerContext, dest any, quer string, args ...any) error {
	return sqlx.SelectContext(ctx, q, dest, quer, args...)
}

func getx(ctx context.Context, q sqlx.QueryerContext, dest any, quer string, args ...any) error {
	return sqlx.GetContext(ctx, q, dest, quer, args...)
}

func exec(ctx context.Context, db sqlx.ExtContext, query string, args ...any) error {
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func query(ctx context.Context, db sqlx.ExtContext, q string, args ...any) (*sqlx.Rows, error) {
	return db.QueryxContext(ctx, q, args...)
}

func namedExec(ctx context.Context, db sqlx.ExtContext, q string, args any) error {
	_, err := sqlx.NamedExecContext(ctx, db, q, args)
	return err
}

func namedQuery(ctx context.Context, db sqlx.ExtContext, q string, args any) (*sqlx.Rows, error) {
	return sqlx.NamedQueryContext(ctx, db, q, args)
}
