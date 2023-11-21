package controllers

import (
	"context"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller manages replicas", func() {

	var testNamespace string

	BeforeEach(func() {
		CreateNamespace(&testNamespace)
	})

	AfterEach(DeleteNamespaceCallback(&testNamespace))

	Context("Creating a new Limitador object with specific replicas", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		var replicas int32 = 2

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.Replicas = &[]int{int(replicas)}[0]
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should create a new deployment with the custom replicas", func() {
			deployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, &deployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(*deployment.Spec.Replicas).To(Equal(replicas))
		})
	})

	Context("Updating limitador object with new replicas", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		var replicas int32 = 2

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should modify deployment replicas", func() {
			deployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, &deployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(*deployment.Spec.Replicas).To(Equal(int32(1)))

			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(), types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)

				if err != nil {
					return false
				}

				updatedLimitador.Spec.Replicas = &[]int{int(replicas)}[0]

				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				newDeployment := &appsv1.Deployment{}
				err := k8sClient.Get(context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, newDeployment)

				if err != nil {
					return false
				}

				return reflect.DeepEqual(*newDeployment.Spec.Replicas, replicas)
			}, timeout, interval).Should(BeTrue())
		})
	})
})
