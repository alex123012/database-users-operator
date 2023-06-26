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

package connection

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	_ "github.com/jackc/pgx/v5/stdlib" // package for sqlx driver
	"github.com/jmoiron/sqlx"
)

type DefaultConnector struct {
	db     *sqlx.DB
	logger logr.Logger
}

func NewDefaultConnector(logger logr.Logger) *DefaultConnector {
	return &DefaultConnector{
		logger: logger,
	}
}

func (d *DefaultConnector) Copy() Connection {
	return NewDefaultConnector(d.logger)
}

func (d *DefaultConnector) Connect(ctx context.Context, driver, connString string) error {
	db, err := sqlx.ConnectContext(ctx, driver, connString)
	if err != nil {
		return err
	}
	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(1)
	d.db = db
	return nil
}

func (d *DefaultConnector) Close(_ context.Context) error {
	return d.db.Close()
}

func (d *DefaultConnector) infoLog(disableLog LogInfo, query string, args ...interface{}) {
	if disableLog == DisableLogger {
		return
	}
	d.logger.Info(fmt.Sprintf("Executing statement '%s' with values %v", query, args))
}

func (d *DefaultConnector) Exec(ctx context.Context, disableLog LogInfo, query string, args ...interface{}) error {
	d.infoLog(disableLog, query, args...)
	_, err := d.db.ExecContext(ctx, query, args...)
	return err
}
