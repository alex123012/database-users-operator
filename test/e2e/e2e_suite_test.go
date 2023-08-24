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

package e2e_test

import (
	"context"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	defaultscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
)

func TestSystemTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Tests Suite")
}

const (
	namespace = "test-database-users-operator"
)

var (
	k8sClient  client.Client
	clientSet  *kubernetes.Clientset
	restConfig *restclient.Config
)

var _ = BeforeSuite(func() {
	if os.Getenv("E2E_TESTS") != "yes" {
		return
	}

	scheme := runtime.NewScheme()
	Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(defaultscheme.AddToScheme(scheme)).To(Succeed())

	restConfig = controllerruntime.GetConfigOrDie()

	var err error
	k8sClient, err = client.New(restConfig, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	clientSet, err = createClientSet()
	Expect(err).NotTo(HaveOccurred())

	operatorNamespace := mustHaveEnv("K8S_OPERATOR_NAMESPACE")

	ctx := context.Background()

	Eventually(func() int32 {
		operatorDeployment, err := clientSet.AppsV1().Deployments(operatorNamespace).Get(ctx, "database-users-operator-controller-manager", metav1.GetOptions{})
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		return operatorDeployment.Status.ReadyReplicas
	}, 10, 1).Should(BeNumerically("==", 1), "Expected to have Operator Pod Ready")

	// _, err = clientSet.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{})
	// Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	// err := clientSet.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
	// Expect(err).NotTo(HaveOccurred())
})
