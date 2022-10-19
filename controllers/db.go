package controllers

import (
	// "database/sql"
	"database/sql"
	"fmt"
	"strings"
	// _ "github.com/lib/pq"
)

type Config interface {
	String() string
}
type DBDriver string

const (
	DBDriverPostgres DBDriver = "postgres"
)

type SSLMode string

const (
	SSLModeDISABLE    SSLMode = "disable"
	SSLModeALLOW      SSLMode = "allow"
	SSLModePREFER     SSLMode = "prefer"
	SSLModeREQUIRE    SSLMode = "require"
	SSLModeVERIFYCA   SSLMode = "verify-ca"
	SSLModeVERIFYFULL SSLMode = "verify-full"
)

type PostgresConfig struct {
	Host         string
	User         string
	Password     string
	Dbname       string
	Port         int
	SSLMode      SSLMode
	SSLCACert    string
	SSLlUserCert string
	SSLUserkey   string
}

func NewPostgresConfig(host string, port int, user, pass, dbname string, sslmode SSLMode,
	sslCaCert, sslUserCert, sslUserKey string) *PostgresConfig {
	return &PostgresConfig{
		Host:         host,
		User:         user,
		Password:     pass,
		Dbname:       dbname,
		Port:         port,
		SSLMode:      sslmode,
		SSLCACert:    sslCaCert,
		SSLlUserCert: sslUserCert,
		SSLUserkey:   sslUserKey,
	}
}

func (d *PostgresConfig) String() string {
	connSlice := []string{
		fmt.Sprintf("host=%s", d.Host),
		fmt.Sprintf("user=%s", d.User),
		fmt.Sprintf("port=%d", d.Port),
		fmt.Sprintf("sslmode=%s", d.SSLMode),
	}
	if d.SSLMode != "disable" {
		connSlice = append(connSlice, fmt.Sprintf("sslrootcert=%s", d.SSLCACert))
		connSlice = append(connSlice, fmt.Sprintf("sslcert=%s", d.SSLlUserCert))
		connSlice = append(connSlice, fmt.Sprintf("sslkey=%s", d.SSLUserkey))
	}
	if d.Dbname != "" {
		connSlice = append(connSlice, fmt.Sprintf("dbname=%s", d.Dbname))
	}
	if d.Password != "" {
		connSlice = append(connSlice, fmt.Sprintf("password=%s", d.Password))
	}
	return strings.Join(connSlice, " ")
}

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

// func main() {
// 	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
// 		"password=%s dbname=%s sslmode=disable",
// 		host, port, user, password, dbname)
// 	db, err := sql.Open("postgres", psqlInfo)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer db.Close()

// 	err = db.Ping()
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println("Successfully connected!")
// }
