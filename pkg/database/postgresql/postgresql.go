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
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/go-logr/logr"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/pkg/database/connection"
)

type Postgresql struct {
	db         connection.Connection
	config     *Config
	logger     logr.Logger
	cfgSigChan chan struct{}
}

func NewPostgresql(c connection.Connection, config *Config, logger logr.Logger) *Postgresql {
	return &Postgresql{
		config: config,
		db:     c,
		logger: logger,
	}
}

func (p *Postgresql) Connect(ctx context.Context) error {
	p.cfgSigChan = make(chan struct{})
	connString, err := p.config.ConnString(p.cfgSigChan)
	if err != nil {
		return err
	}

	return p.db.Connect(ctx, "pgx", connString)
}

func (p *Postgresql) Close(ctx context.Context) error {
	defer close(p.cfgSigChan)
	return p.db.Close(ctx)
}

func (p *Postgresql) CreateUser(ctx context.Context, username, password string) (map[string]string, error) {
	// TODO (alex123012): use gorm.Statement, refer to https://gorm.io/docs/sql_builder.html#Clauses
	query, logInfo := createUserQuery(username, password)
	err := p.db.Exec(ctx, logInfo, query)

	var sslCertificates map[string]string
	if p.config.CreateCerts() && !isAlreadyExists(err) {
		var err error
		sslCertificates, err = p.genPostgresCertFromCA(username)
		if err != nil {
			return nil, err
		}
	}

	return sslCertificates, ignoreAlreadyExists(err)
}

func createUserQuery(username, password string) (string, connection.LogInfo) {
	logInfo := connection.EnableLogger
	stmtBuilder := &strings.Builder{}
	stmtBuilder.WriteString("CREATE USER ")
	stmtBuilder.WriteString(escapeLiteral(username))
	if password != "" {
		stmtBuilder.WriteString(" WITH PASSWORD ")
		stmtBuilder.WriteString(escapeString(password))
		logInfo = connection.DisableLogger
	}
	return stmtBuilder.String(), logInfo
}

func (p *Postgresql) DeleteUser(ctx context.Context, username string) error {
	// TODO (alex123012): use gorm.Statement, refer to https://gorm.io/docs/sql_builder.html#Clauses
	query := deleteUserQuery(username)
	return p.db.Exec(ctx, connection.EnableLogger, query)
}

func deleteUserQuery(username string) string {
	stmtBuilder := &strings.Builder{}
	stmtBuilder.WriteString("DROP USER ")
	stmtBuilder.WriteString(escapeLiteral(username))
	return stmtBuilder.String()
}

func (p *Postgresql) ApplyPrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error {
	return p.privilegesProcessor(ctx, username, privileges, "GRANT", "TO")
}

func (p *Postgresql) RevokePrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error {
	return p.privilegesProcessor(ctx, username, privileges, "REVOKE", "FROM")
}

func (p *Postgresql) privilegesProcessor(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec, statement, arg string) error {
	for _, privilege := range privileges {
		var err error
		switch {
		case privilege.Database != "" && privilege.On != "" && privilege.Privilege != "":
			err = p.inDatabasePrivilege(ctx, username, privilege.Database, privilege.On, privilege.Privilege, statement, arg)

		case privilege.Database != "" && privilege.Privilege != "":
			err = p.databasePrivilege(ctx, username, privilege.Database, privilege.Privilege, statement, arg)

		case privilege.Privilege != "":
			err = p.privilege(ctx, username, privilege.Privilege, statement, arg)

		default:
			err = errors.New("can't use this type of privilege")
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Postgresql) inDatabasePrivilege(ctx context.Context, username, dbname, on string, privilege v1alpha1.PrivilegeType, statement, arg string) error {
	newconf := p.config.Copy()
	newconf.DatabaseName = dbname
	conn := p.db.Copy()
	newP := NewPostgresql(conn, newconf, p.logger)
	if err := newP.Connect(ctx); err != nil {
		return err
	}
	defer newP.Close(ctx)
	query := prepareStatementForPrivilege(statement, arg, username, dbname, on, privilege)
	return newP.db.Exec(ctx, connection.EnableLogger, query)
}

func (p *Postgresql) databasePrivilege(ctx context.Context, username, dbname string, privilege v1alpha1.PrivilegeType, statement, arg string) error {
	query := prepareStatementForPrivilege(statement, arg, username, dbname, "", privilege)
	return p.db.Exec(ctx, connection.EnableLogger, query)
}

func (p *Postgresql) privilege(ctx context.Context, username string, privilege v1alpha1.PrivilegeType, statement, arg string) error {
	query := prepareStatementForPrivilege(statement, arg, username, "", "", privilege)
	return p.db.Exec(ctx, connection.EnableLogger, query)
}

func prepareStatementForPrivilege(statement, arg, username, dbname, on string, privilege v1alpha1.PrivilegeType) string {
	stmtBuilder := &strings.Builder{}
	stmtBuilder.WriteString(statement)
	stmtBuilder.WriteString(" ")
	stmtBuilder.WriteString(escapeLiteralWithoutQuotes(string(privilege)))

	if on != "" {
		stmtBuilder.WriteString(" ON ")
		stmtBuilder.WriteString(escapeLiteral(on))
	} else if dbname != "" {
		stmtBuilder.WriteString(" ON DATABASE ")
		stmtBuilder.WriteString(escapeLiteral(dbname))
	}

	stmtBuilder.WriteString(" ")
	stmtBuilder.WriteString(arg)
	stmtBuilder.WriteString(" ")
	stmtBuilder.WriteString(escapeLiteral(username))
	return stmtBuilder.String()
}

func (p *Postgresql) genPostgresCertFromCA(userName string) (map[string]string, error) {
	caKeyBlock, _ := pem.Decode([]byte(p.config.sslCAKey))
	caPrivKey, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	caCertBlock, _ := pem.Decode([]byte(p.config.SSLCACert))
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, err
	}

	// user cert config
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63()),
		Subject: pkix.Name{
			CommonName: userName,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0),
	}

	// user private key
	privKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// sign the user cert
	certBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, caCert, &privKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	// PEM encode the user cert, key and ca cert
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	return map[string]string{"tls.crt": string(certPEM), "tls.key": string(privKeyPEM), "ca.crt": p.config.SSLCACert}, nil
}
