package controllers

import (
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
	const (
		nodeTimeOut = NodeTimeout(time.Second * 30)
		specTimeOut = SpecTimeout(time.Minute * 2)
	)

	var testNamespace string

	BeforeEach(func(ctx SpecContext) {
		CreateNamespaceWithContext(ctx, &testNamespace)
	}, nodeTimeOut)

	AfterEach(func(ctx SpecContext) {
		DeleteNamespaceWithContext(ctx, &testNamespace)
	}, nodeTimeOut)

	Context("Creating a new Limitador object with specific pdb", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		maxUnavailable := &intstr.IntOrString{Type: 0, IntVal: 3}
		pdbType := &limitadorv1alpha1.PodDisruptionBudgetType{MaxUnavailable: maxUnavailable}

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.PodDisruptionBudget = pdbType
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		}, nodeTimeOut)

		It("Should create PodDisruptionBudget", func(ctx SpecContext) {
			pdb := &policyv1.PodDisruptionBudget{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.PodDisruptionBudgetName(limitadorObj),
					}, pdb)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(pdb.Spec.MaxUnavailable).To(Equal(maxUnavailable))
		})
	})

	Context("Updating limitador object with new pdb", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		maxUnavailable := &intstr.IntOrString{Type: 0, IntVal: 3}
		pdbType := &limitadorv1alpha1.PodDisruptionBudgetType{MaxUnavailable: maxUnavailable}

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should modify pdb object with the new limits", func(ctx SpecContext) {
			pdb := &policyv1.PodDisruptionBudget{}
			err := k8sClient.Get(ctx,
				types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.PodDisruptionBudgetName(limitadorObj),
				}, pdb)
			// returns false when err is nil
			Expect(errors.IsNotFound(err)).To(BeTrue())

			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)).To(Succeed())

				updatedLimitador.Spec.PodDisruptionBudget = pdbType

				g.Expect(k8sClient.Update(ctx, &updatedLimitador)).To(Succeed())
			}).Should(Succeed())

			Eventually(func(g Gomega) {
				newPDB := &policyv1.PodDisruptionBudget{}
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.PodDisruptionBudgetName(limitadorObj),
					}, newPDB)).To(Succeed())

				g.Expect(newPDB.Spec.MaxUnavailable).To(Equal(maxUnavailable))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})
})
