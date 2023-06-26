package controllers_test

import (
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alex123012/database-users-operator/pkg/database"
)

func resetCLusterAndDB(db *database.FakeDatabase, objects ...client.Object) {
	deleteObject := func(o client.Object) bool {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}, o)
		return apierrors.IsNotFound(err)
	}

	for _, o := range objects {
		Expect(k8sClient.Delete(ctx, o)).To(Succeed())
		Eventually(deleteObject, 5).WithArguments(o).Should(BeTrue())
	}
	db.ResetDB()
}

func createObjects(objects ...client.Object) {
	for _, o := range objects {
		Expect(k8sClient.Create(ctx, o)).To(Succeed())
	}
}
