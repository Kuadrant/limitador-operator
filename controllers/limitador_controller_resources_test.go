package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller manages resource requirements", func() {
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

	Context("Creating a new Limitador object with specific resource requirements", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		// empty resources, means no resource requirements,
		// which is different from the default resource requirements
		resourceRequirements := v1.ResourceRequirements{}

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.ResourceRequirements = &resourceRequirements
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should create a new deployment with the custom resource requirements", func(ctx SpecContext) {
			deployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, &deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources).To(
				Equal(resourceRequirements))
		}, specTimeOut)
	})

	Context("Updating limitador object with new resource requirements", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		// empty resources, means no resource requirements,
		// which is different from the default resource requirements
		resourceRequirements := v1.ResourceRequirements{}

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should modify deployment resource requirements", func(ctx SpecContext) {
			deployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, &deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			expectedDefaultResourceRequirements := v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("250m"),
					v1.ResourceMemory: resource.MustParse("32Mi"),
				},
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("500m"),
					v1.ResourceMemory: resource.MustParse("64Mi"),
				},
			}

			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources).To(Equal(
				expectedDefaultResourceRequirements,
			))

			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)).To(Succeed())

				updatedLimitador.Spec.ResourceRequirements = &resourceRequirements

				g.Expect(k8sClient.Update(ctx, &updatedLimitador)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Eventually(func(g Gomega) {
				newDeployment := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, newDeployment)).Should(Succeed())
				g.Expect(newDeployment.Spec.Template.Spec.Containers[0].Resources).To(Equal(resourceRequirements))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})
})
