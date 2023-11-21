package controllers

import (
	"context"
	"reflect"
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

	var testNamespace string

	BeforeEach(func() {
		CreateNamespace(&testNamespace)
	})

	AfterEach(DeleteNamespaceCallback(&testNamespace))

	Context("Creating a new Limitador object with specific resource requirements", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		// empty resources, means no resource requirements,
		// which is different from the default resource requirements
		resourceRequirements := v1.ResourceRequirements{}

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.ResourceRequirements = &resourceRequirements
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should create a new deployment with the custom resource requirements", func() {
			deployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, &deployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources).To(
				Equal(resourceRequirements))
		})
	})

	Context("Updating limitador object with new resource requirements", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		// empty resources, means no resource requirements,
		// which is different from the default resource requirements
		resourceRequirements := v1.ResourceRequirements{}

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should modify deployment resource requirements", func() {
			deployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					}, &deployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

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
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(), types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)

				if err != nil {
					return false
				}

				updatedLimitador.Spec.ResourceRequirements = &resourceRequirements

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

				if len(newDeployment.Spec.Template.Spec.Containers) < 1 {
					return false
				}

				return reflect.DeepEqual(newDeployment.Spec.Template.Spec.Containers[0].Resources, resourceRequirements)
			}, timeout, interval).Should(BeTrue())
		})
	})
})
