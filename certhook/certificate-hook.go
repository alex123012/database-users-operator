package certhook

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"time"

	coreV1 "k8s.io/api/core/v1"

	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/kubernetes"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type userArray []string

func (i *userArray) String() string {
	return strings.Join(*i, ", ")
}

func (i *userArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var users userArray
	var clusterName, clusterNamespace string
	flag.Var(&users, "user", "Username for creating client cert for CockroachDB (for multiple users - provide flag for each user).")
	flag.StringVar(&clusterName, "cockroach-cluster-name", "cockroachdb", "Name of cockroach cluster")
	flag.StringVar(&clusterNamespace, "cockroach-cluster-namespace", "default", "Namespace of cockroach cluster")
	flag.Parse()

	ctx := context.Background()
	clientset := initClient(clusterNamespace)
	caPrivKeyBytes, caCertBytes, err := getCaKeyAndcert(ctx, clientset, clusterName)
	if err != nil {
		log.Fatal(err)
	}

	for _, user := range users {
		if err := checkAlreadyExists(ctx, clientset, clusterName, user); err != nil {
			continue
		}
		cert, privKey, err := generateCertificate(user, caPrivKeyBytes, caCertBytes)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Generated cert-key pair")

		if err := writeClientSecret(ctx, clientset, user, clusterName, cert, privKey, caCertBytes); err != nil {
			log.Fatal(err)
		}
		// err = WriteBytesToFile(fmt.Sprintf("cockroachdb-certs/client.%s.crt", user), cert)
		// if err != nil {
		// 	return
		// }

		// err = WriteBytesToFile(fmt.Sprintf("cockroachdb-certs/client.%s.key", user), priv)
		// if err != nil {
		// 	return
		// }
	}
}
func writeClientSecret(ctx context.Context, clientset coreV1Types.SecretInterface, username, clusterName string, privKey, cert *bytes.Buffer, caCert []byte) error {
	secret := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-client-%s-user-tls", clusterName, username),
		},
		StringData: map[string]string{
			"ca.crt":  string(caCert),
			"tls.crt": cert.String(),
			"tls.key": privKey.String(),
		},
		Type: coreV1.SecretTypeOpaque,
	}
	if _, err := clientset.Create(ctx, secret, metav1.CreateOptions{}); errors.IsAlreadyExists(err) {
		return nil
	} else {
		return err
	}
}
func initClient(clusterNamespace string) coreV1Types.SecretInterface {
	kubeconfig := os.Getenv("HOME") + "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset.CoreV1().Secrets(clusterNamespace)
}

func getCaKeyAndcert(ctx context.Context, clientset coreV1Types.SecretInterface, clusterName string) ([]byte, []byte, error) { //(*rsa.PrivateKey, *x509.Certificate, error) {
	secretCaKey, err := clientset.Get(ctx, fmt.Sprintf("%s-ca", clusterName), metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	secretCaCert, err := clientset.Get(ctx, fmt.Sprintf("%s-node", clusterName), metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	return secretCaKey.Data["ca.key"], secretCaCert.Data["ca.crt"], nil
}

func checkAlreadyExists(ctx context.Context, clientset coreV1Types.SecretInterface, clusterName, username string) error {
	name := fmt.Sprintf("%s-client-%s-user-tls", clusterName, username)
	if secret, err := clientset.Get(ctx, name, metav1.GetOptions{}); err == nil {
		return errors.NewAlreadyExists(schema.GroupResource{Group: secret.GroupVersionKind().Group, Resource: secret.Kind}, name)
	}
	return nil
}
func generateCertificate(username string, caPrivKeyBytes, caCertBytes []byte) (*bytes.Buffer, *bytes.Buffer, error) {
	// user cert config
	caPrivKey, err := byteToCaPrivateKey(caPrivKeyBytes)
	if err != nil {
		return nil, nil, err
	}
	caCert, err := byteToCaCert(caCertBytes)
	if err != nil {
		return nil, nil, err
	}
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63()),
		Subject: pkix.Name{
			CommonName:   username,
			Organization: []string{"Cockroach"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0),
	}

	// user private key
	privKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	// sign the user cert
	CertBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, caCert, &privKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	// PEM encode the user cert and key, ca cert
	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: CertBytes,
	})

	if err != nil {
		return nil, nil, err
	}

	privKeyPEM := new(bytes.Buffer)
	err = pem.Encode(privKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})
	if err != nil {
		return nil, nil, err
	}

	return certPEM, privKeyPEM, nil
}

// WritePem writes data in the file at the given path
func WriteBytesToFile(filepath string, buffer *bytes.Buffer) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Chmod(0600)
	_, err = f.Write(buffer.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// Util funcs
func byteToCaPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func byteToCaCert(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}
