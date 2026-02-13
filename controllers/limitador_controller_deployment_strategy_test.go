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

var _ = Describe("Limitador controller deployment strategy", func() {
	const (
		nodeTimeOut = NodeTimeout(time.Second * 40)
		specTimeOut = SpecTimeout(time.Minute * 2)
	)
	var testNamespace string

	BeforeEach(func(ctx SpecContext) {
		CreateNamespaceWithContext(ctx, &testNamespace)
	})

	AfterEach(func(ctx SpecContext) {
		DeleteNamespaceWithContext(ctx, &testNamespace)
	}, nodeTimeOut)

	Context("Storage type transitions", func() {
		It("Should handle in-memory to disk storage transition", func(ctx SpecContext) {
			// Create limitador with in-memory storage (default)
			limitadorObj := basicLimitador(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())

			// Verify initial deployment has RollingUpdate strategy
			deployment := &appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deployment.Spec.Strategy.Type).To(Equal(appsv1.RollingUpdateDeploymentStrategyType))
			Expect(deployment.Spec.Strategy.RollingUpdate).NotTo(BeNil())

			// Update to disk storage
			updatedLimitador := &limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitadorObj.Name,
					},
					updatedLimitador)).To(Succeed())

				updatedLimitador.Spec.Storage = &limitadorv1alpha1.Storage{
					Disk: &limitadorv1alpha1.DiskSpec{},
				}
				g.Expect(k8sClient.Update(ctx, updatedLimitador)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			// Verify deployment strategy changed to Recreate and RollingUpdate is nil
			Eventually(func(g Gomega) {
				updatedDeployment := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					updatedDeployment)).To(Succeed())

				g.Expect(updatedDeployment.Spec.Strategy.Type).To(Equal(appsv1.RecreateDeploymentStrategyType))
				g.Expect(updatedDeployment.Spec.Strategy.RollingUpdate).To(BeNil(),
					"RollingUpdate field should be nil when using Recreate strategy")
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)

		It("Should handle disk to in-memory storage transition", func(ctx SpecContext) {
			// Create limitador with disk storage
			limitadorObj := limitadorWithDiskStorage(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())

			// Verify initial deployment has Recreate strategy
			deployment := &appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deployment.Spec.Strategy.Type).To(Equal(appsv1.RecreateDeploymentStrategyType))
			Expect(deployment.Spec.Strategy.RollingUpdate).To(BeNil())

			// Update to in-memory storage
			updatedLimitador := &limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitadorObj.Name,
					},
					updatedLimitador)).To(Succeed())

				updatedLimitador.Spec.Storage = nil // in-memory is the default
				g.Expect(k8sClient.Update(ctx, updatedLimitador)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			// Verify deployment strategy changed to RollingUpdate
			Eventually(func(g Gomega) {
				updatedDeployment := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					updatedDeployment)).To(Succeed())

				g.Expect(updatedDeployment.Spec.Strategy.Type).To(Equal(appsv1.RollingUpdateDeploymentStrategyType))
				g.Expect(updatedDeployment.Spec.Strategy.RollingUpdate).NotTo(BeNil(),
					"RollingUpdate field should be set when using RollingUpdate strategy")
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)

		It("Should maintain RollingUpdate strategy when switching between compatible storage types", func(ctx SpecContext) {
			// Create limitador with in-memory storage
			limitadorObj := basicLimitador(testNamespace)
			limitadorObj.Spec.Replicas = ptr.To(2)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())

			// Verify initial deployment has RollingUpdate strategy
			deployment := &appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deployment.Spec.Strategy.Type).To(Equal(appsv1.RollingUpdateDeploymentStrategyType))
			Expect(deployment.Spec.Strategy.RollingUpdate).NotTo(BeNil())

			// Update spec to trigger reconciliation (but keep in-memory storage)
			updatedLimitador := &limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitadorObj.Name,
					},
					updatedLimitador)).To(Succeed())

				updatedLimitador.Spec.Replicas = ptr.To(3)
				g.Expect(k8sClient.Update(ctx, updatedLimitador)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			// Verify deployment still has RollingUpdate strategy
			Eventually(func(g Gomega) {
				updatedDeployment := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					updatedDeployment)).To(Succeed())

				g.Expect(updatedDeployment.Spec.Strategy.Type).To(Equal(appsv1.RollingUpdateDeploymentStrategyType))
				g.Expect(updatedDeployment.Spec.Strategy.RollingUpdate).NotTo(BeNil())
				g.Expect(*updatedDeployment.Spec.Replicas).To(Equal(int32(3)))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})
})
