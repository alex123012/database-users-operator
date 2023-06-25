package postgresql

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v5"
)

type Postgresql struct {
	db         *pgx.Conn
	config     *Config
	logger     logr.Logger
	cfgSigChan chan struct{}
}

func NewPostgresql(config *Config, logger logr.Logger) *Postgresql {
	return &Postgresql{
		config: config,
		logger: logger,
	}
}

func (p *Postgresql) Connect(ctx context.Context) error {
	p.cfgSigChan = make(chan struct{})
	connString, err := p.config.ConnString(p.cfgSigChan)
	if err != nil {
		return err
	}

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return err
	}

	p.db = conn
	return nil
}

func (p *Postgresql) Close(ctx context.Context) error {
	defer close(p.cfgSigChan)
	return p.db.Close(ctx)
}

func (p *Postgresql) CreateUser(ctx context.Context, username, password string) (map[string]string, error) {
	//TODO (alex123012): use gorm.Statement, refer to https://gorm.io/docs/sql_builder.html#Clauses
	query := createUserQuery(username, password)

	_, err := p.db.Exec(ctx, query)

	var sslCertificates map[string]string
	if p.config.createCertificates && !isAlreadyExists(err) {
		var err error
		sslCertificates, err = p.genPostgresCertFromCA(username)
		if err != nil {
			return nil, err
		}
	}

	return sslCertificates, ignoreAlreadyExists(err)
}

func createUserQuery(username, password string) string {
	stmtBuilder := &strings.Builder{}
	stmtBuilder.WriteString("CREATE USER ")
	stmtBuilder.WriteString(escapeLiteral(username))
	if password != "" {
		stmtBuilder.WriteString(" WITH PASSWORD ")
		stmtBuilder.WriteString(escapeString(password))
	}
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

func (p *Postgresql) DeleteUser(ctx context.Context, username string) error {
	return nil
}

func (p *Postgresql) ApplyPrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error {
	return nil
}

func (p *Postgresql) RevokePrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error {
	return nil
}
