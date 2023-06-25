package connection

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type defaultConnector struct {
	db     *sqlx.DB
	logger logr.Logger
}

func NewDefaultConnector(logger logr.Logger) *defaultConnector {
	return &defaultConnector{
		logger: logger,
	}
}

func (d *defaultConnector) Copy() interface{} {
	return NewDefaultConnector(d.logger)
}

func (d *defaultConnector) Connect(ctx context.Context, driver, connString string) error {
	db, err := sqlx.ConnectContext(ctx, driver, connString)
	if err != nil {
		return err
	}
	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(1)
	d.db = db
	return nil
}

func (d *defaultConnector) Close(ctx context.Context) error {
	return d.db.Close()
}

func (d *defaultConnector) infoLog(disableLog LogInfo, q string, args []interface{}) {
	if disableLog == DisableLogger {
		return
	}
	d.logger.Info(fmt.Sprintf("Executing statement '%s' with values: %v", q, args))
}

func (d *defaultConnector) Exec(ctx context.Context, disableLog LogInfo, query string, args ...interface{}) error {
	d.infoLog(disableLog, query, args)
	_, err := d.db.ExecContext(ctx, query, args...)
	return err
}

// func (d *defaultConnector) Query(ctx context.Context, disableLog LogInfo, query string, args ...interface{}) (*sqlx.Rows, error) {
// 	d.infoLog(disableLog, query, args)
// 	return d.db.QueryxContext(ctx, query, args...)
// }

// func (d *defaultConnector) Select(ctx context.Context, disableLog LogInfo, dest interface{}, query string, args ...interface{}) error {
// 	d.infoLog(disableLog, query, args)
// 	return sqlx.SelectContext(ctx, d.db, dest, query, args...)
// }
