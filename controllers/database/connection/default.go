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

func (d *DefaultConnector) Copy() interface{} {
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

func (d *DefaultConnector) infoLog(disableLog LogInfo, query string) {
	if disableLog == DisableLogger {
		return
	}
	d.logger.Info(fmt.Sprintf("Executing statement '%s'", query))
}

func (d *DefaultConnector) Exec(ctx context.Context, disableLog LogInfo, query string) error {
	d.infoLog(disableLog, query)
	_, err := d.db.ExecContext(ctx, query)
	return err
}
