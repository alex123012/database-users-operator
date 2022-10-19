package database

import (
	"database/sql"

	_ "github.com/lib/pq"
)

type Config interface {
	String() string
}
type DBDriver string

const (
	DBDriverPostgres DBDriver = "postgres"
)

func NewDBConnection(config Config, dbDriver DBDriver) (*sql.DB, error) {
	db, err := sql.Open(string(dbDriver), config.String())
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, err
}
