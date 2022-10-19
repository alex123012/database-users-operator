package database

import (
	"errors"
	"fmt"
	"strings"
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

var ErrAlreadyExists = errors.New("already exists")

func ProcessPostgresError(err error) error {
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return ErrAlreadyExists
	}
	return err
}
func IgnoreAlreadyExists(err error) error {
	if errors.Is(err, ErrAlreadyExists) {
		return nil
	}
	return err
}
