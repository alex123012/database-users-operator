package utils

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"math/rand"
	"time"
)

func GenCockroachCertFromCA(userName string, caCert *x509.Certificate, caPrivKey *rsa.PrivateKey) (map[string][]byte, error) {

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63()),
		Subject: pkix.Name{
			CommonName:   userName,
			Organization: []string{"Cockroach"},
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
