package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller manages affinity", func() {
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

	Context("Creating a new Limitador object with specific affinity", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		affinity := &v1.Affinity{
			PodAntiAffinity: &v1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
					{
						Weight: 100,
						PodAffinityTerm: v1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"pod": "label",
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			},
		}

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.Affinity = affinity
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should create a new deployment with the custom affinity", func(ctx SpecContext) {
			deployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.DeploymentName(limitadorObj),
					},
					&deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deployment.Spec.Template.Spec.Affinity).To(Equal(affinity))
		}, specTimeOut)
	})

	Context("Updating limitador object with new affinity settings", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		affinity := &v1.Affinity{
			PodAntiAffinity: &v1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
					{
						Weight: 100,
						PodAffinityTerm: v1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"pod": "label",
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			},
		}

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should modify the deployment with the affinity custom settings", func(ctx SpecContext) {
			deployment := appsv1.Deployment{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.DeploymentName(limitadorObj),
				}, &deployment)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Expect(deployment.Spec.Template.Spec.Affinity).To(BeNil())

			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)).Should(Succeed())

				updatedLimitador.Spec.Affinity = affinity.DeepCopy()

				g.Expect(k8sClient.Update(ctx, &updatedLimitador)).Should(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Eventually(func(g Gomega) {
				newDeployment := appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitador.DeploymentName(limitadorObj),
				}, &newDeployment)).To(Succeed())
				g.Expect(newDeployment.Spec.Template.Spec.Affinity).Should(Equal(affinity))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})
})
