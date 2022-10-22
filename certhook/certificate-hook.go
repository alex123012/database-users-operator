package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/alex123012/k8s-database-users-operator/pkg/database/postgresql"
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
		data, err := generateCertificate(user, caPrivKeyBytes, caCertBytes)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Generated cert-key pair")

		if err := writeClientSecret(ctx, clientset, user, clusterName, data); err != nil {
			log.Fatal(err)
		}
	}
}
func writeClientSecret(ctx context.Context, clientset coreV1Types.SecretInterface, username, clusterName string, data map[string][]byte) error {
	secret := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", clusterName, username),
		},
		Data: data,
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

	secretCaCert, err := clientset.Get(ctx, fmt.Sprintf("%s-root", clusterName), metav1.GetOptions{})
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

func generateCertificate(username string, caPrivKeyBytes, caCertBytes []byte) (map[string][]byte, error) {
	return postgresql.GenPostgresCertFromCA(username, map[string][]byte{"ca.key": caPrivKeyBytes, "ca.crt": caCertBytes})
}
