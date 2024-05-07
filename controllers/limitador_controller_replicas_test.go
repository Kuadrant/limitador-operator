package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller manages replicas", func() {
	const (
		nodeTimeOut = NodeTimeout(time.Second * 30)
		specTimeOut = SpecTimeout(time.Minute * 2)
	)
	var testNamespace string

	BeforeEach(func(ctx SpecContext) {
		CreateNamespaceWithContext(ctx, &testNamespace)
	})

	AfterEach(func(ctx SpecContext) {
		DeleteNamespaceWithContext(ctx, &testNamespace)
	}, nodeTimeOut)

	Context("Creating a new Limitador object with specific replicas", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		var replicas int32 = 2

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.Replicas = ptr.To(int(replicas))
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should create a new deployment with the custom replicas", func(ctx SpecContext) {
			deployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, &deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(*deployment.Spec.Replicas).To(Equal(replicas))
		}, specTimeOut)
	})

	Context("Updating limitador object with new replicas", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		var replicas int32 = 2

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should modify deployment replicas", func(ctx SpecContext) {
			deployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, &deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(*deployment.Spec.Replicas).To(Equal(int32(1)))

			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)).To(Succeed())

				updatedLimitador.Spec.Replicas = ptr.To(int(replicas))

				g.Expect(k8sClient.Update(ctx, &updatedLimitador)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Eventually(func(g Gomega) {
				newDeployment := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, newDeployment)).To(Succeed())

				g.Expect(*newDeployment.Spec.Replicas).To(Equal(replicas))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})
})
