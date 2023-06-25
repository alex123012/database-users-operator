package postgresql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alex123012/database-users-operator/controllers/internal"
)

type Config struct {
	Host                               string
	User                               string
	Password                           string
	DatabaseName                       string
	Port                               int
	SSLMode                            string
	SSLCACert, SSLUserCert, SSLUserKey string
	sslCAKey                           string

	createCertificates bool
}

func NewConfig(host string, port int, user, pass, dbname, sslmode, sslCaCert, sslUserCert, sslUserKey, sslCAKey string) *Config {

	return &Config{
		Host:         host,
		User:         user,
		Password:     pass,
		DatabaseName: dbname,
		Port:         port,
		SSLMode:      sslmode,
		SSLCACert:    sslCaCert,
		SSLUserCert:  sslUserCert,
		SSLUserKey:   sslUserKey,
		sslCAKey:     sslCAKey,
	}
}

func (c *Config) ConnString(deleteFilesSigChan <-chan struct{}) (string, error) {
	connSlice := []string{
		fmt.Sprintf("host=%s", c.Host),
		fmt.Sprintf("user=%s", c.User),
		fmt.Sprintf("port=%d", c.Port),
	}

	if c.SSLMode != "" {
		connSlice = append(connSlice, fmt.Sprintf("sslmode=%s", c.SSLMode))
	}
	if c.DatabaseName != "" {
		connSlice = append(connSlice, fmt.Sprintf("dbname=%s", c.DatabaseName))
	}
	if c.Password != "" {
		connSlice = append(connSlice, fmt.Sprintf("password=%s", c.Password))
	}

	var (
		sslCACertFile   = fmt.Sprintf("postgres-certs/%s/%s_%s.ca", c.Host, c.DatabaseName, c.User)
		sslUserCertFile = fmt.Sprintf("postgres-certs/%s/%s_%s.crt", c.Host, c.DatabaseName, c.User)
		sslUserKeyFile  = fmt.Sprintf("postgres-certs/%s/%s_%s.key", c.Host, c.DatabaseName, c.User)
	)

	if c.SSLCACert != "" {
		f, err := createFileAndReturnName(sslCACertFile, c.SSLCACert)
		if err != nil {
			return "", err
		}
		connSlice = append(connSlice, fmt.Sprintf("sslrootcert=%s", f))
	}

	if c.SSLUserCert != "" {
		c.createCertificates = true
		f, err := createFileAndReturnName(sslUserCertFile, c.SSLUserCert)
		if err != nil {
			return "", err
		}
		connSlice = append(connSlice, fmt.Sprintf("sslcert=%s", f))
	}

	if c.SSLUserKey != "" {
		c.createCertificates = true
		f, err := createFileAndReturnName(sslUserKeyFile, c.SSLUserKey)
		if err != nil {
			return "", err
		}
		connSlice = append(connSlice, fmt.Sprintf("sslkey=%s", f))
	}

	go func() {
		defer os.Remove(sslCACertFile)
		defer os.Remove(sslUserCertFile)
		defer os.Remove(sslUserKeyFile)
		<-deleteFilesSigChan
	}()

	return strings.Join(connSlice, " "), nil
}

func (in *Config) Copy() *Config {
	newconf := *in
	return &newconf
}

func createFileAndReturnName(filename, certData string) (string, error) {
	f := internal.PathFromHome(filename)
	if err := createCertFile(f, []byte(certData)); err != nil {
		return "", err
	}
	return f, nil
}

func createCertFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
