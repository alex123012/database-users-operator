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

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/controllers/database/connection"
	"github.com/go-logr/logr"
)

type Connection interface {
	Copy() interface{}
	Close(ctx context.Context) error
	Connect(ctx context.Context, driver string, connString string) error
	Exec(ctx context.Context, disableLog connection.LogInfo, query string) error
}

type Postgresql struct {
	db         Connection
	config     *Config
	logger     logr.Logger
	cfgSigChan chan struct{}
}

func NewPostgresql(c Connection, config *Config, logger logr.Logger) *Postgresql {
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
	//TODO (alex123012): use gorm.Statement, refer to https://gorm.io/docs/sql_builder.html#Clauses
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
	//TODO (alex123012): use gorm.Statement, refer to https://gorm.io/docs/sql_builder.html#Clauses
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
	for _, privelege := range privileges {
		var err error
		switch {
		case privelege.Database != "" && privelege.On != "" && privelege.Privilege != "":
			err = p.inDatabasePrivilege(ctx, username, privelege.Database, privelege.On, privelege.Privilege, statement, arg)

		case privelege.Database != "" && privelege.Privilege != "":
			err = p.databasePrivilege(ctx, username, privelege.Database, privelege.Privilege, statement, arg)

		case privelege.Privilege != "":
			err = p.privilege(ctx, username, privelege.Privilege, statement, arg)

		default:
			err = errors.New("can't use this type of privelege")
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Postgresql) inDatabasePrivilege(ctx context.Context, username, dbname, on string, privelege v1alpha1.PrivilegeType, statement, arg string) error {
	newconf := p.config.Copy()
	newconf.DatabaseName = dbname
	conn := p.db.Copy()
	newP := NewPostgresql(conn.(Connection), newconf, p.logger)
	if err := newP.Connect(ctx); err != nil {
		return err
	}
	defer newP.Close(ctx)
	query := prepareStatementForPrivilege(statement, arg, username, dbname, on, privelege)
	return newP.db.Exec(ctx, connection.EnableLogger, query)
}

func (p *Postgresql) databasePrivilege(ctx context.Context, username, dbname string, privelege v1alpha1.PrivilegeType, statement, arg string) error {
	query := prepareStatementForPrivilege(statement, arg, username, dbname, "", privelege)
	return p.db.Exec(ctx, connection.EnableLogger, query)
}

func (p *Postgresql) privilege(ctx context.Context, username string, privelege v1alpha1.PrivilegeType, statement, arg string) error {
	query := prepareStatementForPrivilege(statement, arg, username, "", "", privelege)
	return p.db.Exec(ctx, connection.EnableLogger, query)
}

func prepareStatementForPrivilege(statement, arg, username, dbname, on string, privelege v1alpha1.PrivilegeType) string {
	stmtBuilder := &strings.Builder{}
	stmtBuilder.WriteString(statement)
	stmtBuilder.WriteString(" ")
	stmtBuilder.WriteString(escapeLiteralWithoutQuotes(string(privelege)))

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
