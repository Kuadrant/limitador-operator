package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller manages limits", func() {
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

	Context("Creating a new Limitador object with specific limits", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		limits := []limitadorv1alpha1.RateLimit{
			{
				Conditions: []string{"req.method == 'GET'"},
				MaxValue:   10,
				Namespace:  "test-namespace",
				Seconds:    60,
				Variables:  []string{"user_id"},
				Name:       "useless",
			},
			{
				Conditions: []string{"req.method == 'POST'"},
				MaxValue:   5,
				Namespace:  "test-namespace",
				Seconds:    60,
				Variables:  []string{"user_id"},
			},
		}

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.Limits = limits
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		}, nodeTimeOut)

		It("Should create configmap with the custom limits", func(ctx SpecContext) {
			cm := &v1.ConfigMap{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.LimitsConfigMapName(limitadorObj),
					}, cm)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			var cmLimits []limitadorv1alpha1.RateLimit
			err := yaml.Unmarshal([]byte(cm.Data[limitador.LimitadorConfigFileName]), &cmLimits)
			Expect(err).To(BeNil())
			Expect(cmLimits).To(Equal(limits))
		}, specTimeOut)
	})

	Context("Updating limitador object with new limits", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		limits := []limitadorv1alpha1.RateLimit{
			{
				Conditions: []string{"req.method == 'GET'"},
				MaxValue:   10,
				Namespace:  "test-namespace",
				Seconds:    60,
				Variables:  []string{"user_id"},
				Name:       "useless",
			},
			{
				Conditions: []string{"req.method == 'POST'"},
				MaxValue:   5,
				Namespace:  "test-namespace",
				Seconds:    60,
				Variables:  []string{"user_id"},
			},
		}

		BeforeEach(func(ctx SpecContext) {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		}, nodeTimeOut)

		It("Should modify configmap with the new limits", func(ctx SpecContext) {
			cm := &v1.ConfigMap{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.LimitsConfigMapName(limitadorObj),
					}, cm)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			var cmLimits []limitadorv1alpha1.RateLimit
			err := yaml.Unmarshal([]byte(cm.Data[limitador.LimitadorConfigFileName]), &cmLimits)
			Expect(err).To(BeNil())
			Expect(cmLimits).To(BeEmpty())

			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)).To(Succeed())

				updatedLimitador.Spec.Limits = limits

				g.Expect(k8sClient.Update(ctx, &updatedLimitador)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			Eventually(func(g Gomega) {
				newCM := &v1.ConfigMap{}
				g.Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.LimitsConfigMapName(limitadorObj),
					}, newCM)).To(Succeed())

				var cmLimits []limitadorv1alpha1.RateLimit
				g.Expect(yaml.Unmarshal([]byte(newCM.Data[limitador.LimitadorConfigFileName]), &cmLimits)).To(Succeed())
				g.Expect(cmLimits).To(Equal(limits))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})
})
