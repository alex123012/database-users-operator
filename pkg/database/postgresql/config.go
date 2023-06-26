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

package postgresql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/pkg/utils"
)

type Config struct {
	Host                               string
	User                               string
	Password                           string
	DatabaseName                       string
	Port                               int
	SSLMode                            v1alpha1.PostgresSSLMode
	SSLCACert, SSLUserCert, SSLUserKey string
	sslCAKey                           string

	createCertificates bool
}

func NewConfig(host string, port int, user, pass, dbname string, sslmode v1alpha1.PostgresSSLMode, sslCaCert, sslUserCert, sslUserKey, sslCAKey string) *Config {
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
		sslCACertFile   = utils.PathFromHome(fmt.Sprintf("postgres-certs/%s/%s_%s.ca", c.Host, c.DatabaseName, c.User))
		sslUserCertFile = utils.PathFromHome(fmt.Sprintf("postgres-certs/%s/%s_%s.crt", c.Host, c.DatabaseName, c.User))
		sslUserKeyFile  = utils.PathFromHome(fmt.Sprintf("postgres-certs/%s/%s_%s.key", c.Host, c.DatabaseName, c.User))
	)

	if c.SSLCACert != "" {
		if err := createCertFile(sslCACertFile, c.SSLCACert); err != nil {
			return "", err
		}
		connSlice = append(connSlice, fmt.Sprintf("sslrootcert=%s", sslCACertFile))
	}

	if c.SSLUserCert != "" {
		c.createCertificates = true
		if err := createCertFile(sslUserCertFile, c.SSLUserCert); err != nil {
			return "", err
		}
		connSlice = append(connSlice, fmt.Sprintf("sslcert=%s", sslUserCertFile))
	}

	if c.SSLUserKey != "" {
		c.createCertificates = true
		if err := createCertFile(sslUserKeyFile, c.SSLUserKey); err != nil {
			return "", err
		}
		connSlice = append(connSlice, fmt.Sprintf("sslkey=%s", sslUserKeyFile))
	}

	go func() {
		defer os.Remove(sslCACertFile)
		defer os.Remove(sslUserCertFile)
		defer os.Remove(sslUserKeyFile)
		<-deleteFilesSigChan
	}()

	return strings.Join(connSlice, " "), nil
}

func (c *Config) Copy() *Config {
	newconf := *c
	return &newconf
}

func (c *Config) CreateCerts() bool {
	return c.createCertificates
}

func createCertFile(path string, data string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(data), 0o600)
}
