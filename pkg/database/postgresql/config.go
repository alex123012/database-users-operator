package postgresql

import (
	"fmt"
	"strings"

	"github.com/alex123012/database-users-operator/pkg/database"
)

type DBType = string

const (
	PostgreSQL  DBType = "PostgreSQL"
	CockroachDB DBType = "CockroachDB"
)

type PostgresConfig struct {
	// connConfig *pgx.ConnConfig
	Host                               string
	User                               string
	Password                           string
	Dbname                             string
	Port                               int
	SSLMode                            database.PostgresSSLMode
	SSLCACert, SSLUserCert, SSLUserKey string
}

func NewPostgresConfig(host string, port int, user, pass, dbname string, sslmode database.PostgresSSLMode,
	sslCaCert, sslUserCert, sslUserKey string) *PostgresConfig {

	return &PostgresConfig{
		Host:        host,
		User:        user,
		Password:    pass,
		Dbname:      dbname,
		Port:        port,
		SSLMode:     sslmode,
		SSLCACert:   sslCaCert,
		SSLUserCert: sslUserCert,
		SSLUserKey:  sslUserKey,
	}
}

func (c *PostgresConfig) GetConfig() string {
	return c.connString()
}

func (c *PostgresConfig) connString() string {
	connSlice := []string{
		fmt.Sprintf("host=%s", c.Host),
		fmt.Sprintf("user=%s", c.User),
		fmt.Sprintf("port=%d", c.Port),
	}

	if c.SSLMode != "" {
		connSlice = append(connSlice, fmt.Sprintf("sslmode=%s", c.SSLMode))
	}
	if c.Dbname != "" {
		connSlice = append(connSlice, fmt.Sprintf("dbname=%s", c.Dbname))
	}
	if c.SSLCACert != "" {
		connSlice = append(connSlice, fmt.Sprintf("sslrootcert=%s", c.SSLCACert))
	}
	if c.SSLUserCert != "" {
		connSlice = append(connSlice, fmt.Sprintf("sslcert=%s", c.SSLUserCert))
	}
	if c.SSLUserKey != "" {
		connSlice = append(connSlice, fmt.Sprintf("sslkey=%s", c.SSLUserKey))
	}
	if c.Password != "" {
		connSlice = append(connSlice, fmt.Sprintf("password=%s", c.Password))
	}
	return strings.Join(connSlice, " ")
}

func (in *PostgresConfig) Copy() *PostgresConfig {
	newconf := *in
	return &newconf
}
