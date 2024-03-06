package controllers

import (
	"context"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller manages PodDisruptionBudget", func() {

	var testNamespace string

	BeforeEach(func() {
		CreateNamespace(&testNamespace)
	})

	AfterEach(DeleteNamespaceCallback(&testNamespace))

	Context("Creating a new Limitador object with specific pdb", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		maxUnavailable := &intstr.IntOrString{Type: 0, IntVal: 3}
		pdbType := &limitadorv1alpha1.PodDisruptionBudgetType{MaxUnavailable: maxUnavailable}

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.PodDisruptionBudget = pdbType
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReadyAndAvailable(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should create PodDisruptionBudget", func() {
			pdb := &policyv1.PodDisruptionBudget{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.PodDisruptionBudgetName(limitadorObj),
					}, pdb)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(pdb.Spec.MaxUnavailable).To(Equal(maxUnavailable))
		})
	})

	Context("Updating limitador object with new pdb", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		maxUnavailable := &intstr.IntOrString{Type: 0, IntVal: 3}
		pdbType := &limitadorv1alpha1.PodDisruptionBudgetType{MaxUnavailable: maxUnavailable}

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReadyAndAvailable(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should modify pdb object with the new limits", func() {
			pdb := &policyv1.PodDisruptionBudget{}
			err := k8sClient.Get(context.TODO(),
				types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.PodDisruptionBudgetName(limitadorObj),
				}, pdb)
			// returns false when err is nil
			Expect(errors.IsNotFound(err)).To(BeTrue())

			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(), types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)

				if err != nil {
					return false
				}

				updatedLimitador.Spec.PodDisruptionBudget = pdbType

				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				newPDB := &policyv1.PodDisruptionBudget{}
				err := k8sClient.Get(context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.PodDisruptionBudgetName(limitadorObj),
					}, newPDB)

				if err != nil {
					return false
				}

				return reflect.DeepEqual(newPDB.Spec.MaxUnavailable, maxUnavailable)
			}, timeout, interval).Should(BeTrue())
		})
	})
})
