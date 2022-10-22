package postgresql

import (
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

	authv1alpha1 "github.com/alex123012/k8s-database-users-operator/api/v1alpha1"
	"github.com/alex123012/k8s-database-users-operator/pkg/utils"
	"github.com/jackc/pgx/v5/pgconn"
)

func GenPostgresCertFromCA(userName string, secretData map[string][]byte) (map[string][]byte, error) {

	// user cert config
	caPrivKey, err := utils.ByteToCaPrivateKey(secretData["ca.key"])
	if err != nil {
		return nil, err
	}
	caCert, err := utils.ByteToCaCert(secretData["ca.crt"])
	if err != nil {
		return nil, err
	}

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
	CertBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, caCert, &privKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	// PEM encode the user cert, key and ca cert
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: CertBytes,
	})

	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert.Raw,
	})

	return map[string][]byte{"tls.crt": certPEM, "tls.key": privKeyPEM, "ca.crt": caCertPEM}, nil
}

func IgnoreAlreadyExists(err error) error {
	if IsAlreadyExists(err) {
		return nil
	}
	return err
}

func IsAlreadyExists(err error) bool {
	return ProcessToPostgressError(err) == "42710"
}

const NotAPostgresError string = "Not a postgres error"

func ProcessToPostgressError(err error) string {
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return pgErr.SQLState()
		}
		return NotAPostgresError
	}
	return ""
}

func EscapeLiteral(str string) string {
	ident := strings.Split(str, ".")
	parts := make([]string, len(ident))
	for i := range ident {
		parts[i] = strings.ReplaceAll(
			strings.ReplaceAll(ident[i], string([]byte{0}), ""),
			`"`, `""`)
		if parts[i] != "*" {
			parts[i] = `"` + parts[i] + `"`
		}
	}
	return strings.Join(parts, ".")
}

func EscapeLiteralWithoutQuotes(str string) string {
	ident := strings.Split(str, ".")
	parts := make([]string, len(ident))
	for i := range ident {
		tmp := strings.ReplaceAll(ident[i], string([]byte{0}), "")
		tmp = strings.ReplaceAll(tmp, `"`, `""`)
		parts[i] = strings.ReplaceAll(tmp, `#`, ``)
		parts[i] = strings.ReplaceAll(tmp, `;`, ``)
	}
	return strings.Join(parts, ".")
}

func EscapeString(str string) string {
	return "'" + strings.ReplaceAll(str, "'", "''") + "'"
}

func Intersect(set1, set2 []authv1alpha1.Privilege) []authv1alpha1.Privilege {
	hashSet1 := make(map[authv1alpha1.Privilege]struct{})
	resultMap := make(map[authv1alpha1.Privilege]struct{})

	for _, v := range set1 {
		hashSet1[v] = struct{}{}
	}

	for _, v := range set2 {
		if _, ok := hashSet1[v]; ok {
			resultMap[v] = struct{}{}
		}
	}

	set := make([]authv1alpha1.Privilege, 0)
	for key := range resultMap {
		set = append(set, key)
	}
	return set
}

func IntersectDefinedPrivsWithDB(definedPrivs, dbPrivsMap map[authv1alpha1.Privilege]struct{}) ([]authv1alpha1.Privilege, []authv1alpha1.Privilege) {

	toRevoke := make([]authv1alpha1.Privilege, 0)
	toCreate := make([]authv1alpha1.Privilege, 0)
	for key := range dbPrivsMap {
		if _, f := definedPrivs[key]; !f {
			toRevoke = append(toRevoke, key)
		}
	}
	for key := range definedPrivs {
		if _, f := dbPrivsMap[key]; !f {
			toCreate = append(toCreate, key)
		}
	}
	return toCreate, toRevoke
}
