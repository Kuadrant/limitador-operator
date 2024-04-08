package controllers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller manages image version", func() {
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

	Context("Creating a new Limitador object with specific image version", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.Version = ptr.To("otherversion")
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			// Do not expect to have limitador ready
		}, nodeTimeOut)

		It("Should create a new deployment with the custom image", func(ctx SpecContext) {
			deployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.DeploymentName(limitadorObj),
				}, &deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			expectedImage := fmt.Sprintf("%s:%s", limitador.LimitadorRepository, "otherversion")
			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedImage))
		}, specTimeOut)
	})

	Context("Updating limitador object with a new image version", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)

			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		}, nodeTimeOut)

		It("Should modify the deployment with the custom image", func(ctx SpecContext) {
			deployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.DeploymentName(limitadorObj),
				}, &deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deployment.Spec.Template.Spec.Containers[0].Image).Should(
				Equal(ExpectedDefaultImage),
			)

			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)).Should(Succeed())

				updatedLimitador.Spec.Version = ptr.To("otherversion")

				// the new deployment very likely will not be available (image does not exist)
				g.Expect(k8sClient.Update(ctx, &updatedLimitador)).Should(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Eventually(func(g Gomega) {
				newDeployment := appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.DeploymentName(limitadorObj),
				}, &newDeployment)).To(Succeed())

				expectedImage := fmt.Sprintf("%s:%s", limitador.LimitadorRepository, "otherversion")
				g.Expect(expectedImage).Should(Equal(newDeployment.Spec.Template.Spec.Containers[0].Image))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})
})
