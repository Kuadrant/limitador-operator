package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller manages image version", func() {

	var testNamespace string

	BeforeEach(func() {
		CreateNamespace(&testNamespace)
	})

	AfterEach(DeleteNamespaceCallback(&testNamespace))

	Context("Creating a new Limitador object with specific image version", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.Version = &[]string{"otherversion"}[0]
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			// Do not expect to have limitador ready
		})

		It("Should create a new deployment with the custom image", func() {
			deployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			expectedImage := fmt.Sprintf("%s:%s", limitador.LimitadorRepository, "otherversion")
			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedImage))
		})
	})

	Context("Updating limitador object with a new image version", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should modify the deployment with the custom image", func() {
			deployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(), types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.DeploymentName(limitadorObj),
				}, &deployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(deployment.Spec.Template.Spec.Containers[0].Image).Should(
				Equal(ExpectedDefaultImage),
			)

			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(), types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)

				if err != nil {
					return false
				}

				updatedLimitador.Spec.Version = &[]string{"otherversion"}[0]

				// the new deployment very likely will not be available (image does not exist)
				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				newDeployment := appsv1.Deployment{}
				err := k8sClient.Get(context.TODO(), types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.DeploymentName(limitadorObj),
				}, &newDeployment)

				if err != nil {
					return false
				}

				expectedImage := fmt.Sprintf("%s:%s", limitador.LimitadorRepository, "otherversion")
				return expectedImage == newDeployment.Spec.Template.Spec.Containers[0].Image
			}, timeout, interval).Should(BeTrue())
		})
	})
})
