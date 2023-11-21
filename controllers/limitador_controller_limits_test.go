package controllers

import (
	"context"
	"reflect"
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

	var testNamespace string

	BeforeEach(func() {
		CreateNamespace(&testNamespace)
	})

	AfterEach(DeleteNamespaceCallback(&testNamespace))

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

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			limitadorObj.Spec.Limits = limits
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should create configmap with the custom limits", func() {
			cm := &v1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.LimitsConfigMapName(limitadorObj),
					}, cm)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			var cmLimits []limitadorv1alpha1.RateLimit
			err := yaml.Unmarshal([]byte(cm.Data[limitador.LimitadorConfigFileName]), &cmLimits)
			Expect(err).To(BeNil())
			Expect(cmLimits).To(Equal(limits))
		})
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

		BeforeEach(func() {
			limitadorObj = basicLimitador(testNamespace)
			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(limitadorObj), time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("Should modify configmap with the new limits", func() {
			cm := &v1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.LimitsConfigMapName(limitadorObj),
					}, cm)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			var cmLimits []limitadorv1alpha1.RateLimit
			err := yaml.Unmarshal([]byte(cm.Data[limitador.LimitadorConfigFileName]), &cmLimits)
			Expect(err).To(BeNil())
			Expect(cmLimits).To(BeEmpty())

			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(context.TODO(), types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)

				if err != nil {
					return false
				}

				updatedLimitador.Spec.Limits = limits

				return k8sClient.Update(context.TODO(), &updatedLimitador) == nil
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				newCM := &v1.ConfigMap{}
				err := k8sClient.Get(context.TODO(),
					types.NamespacedName{
						Namespace: testNamespace,
						Name:      limitador.LimitsConfigMapName(limitadorObj),
					}, newCM)

				if err != nil {
					return false
				}

				var cmLimits []limitadorv1alpha1.RateLimit
				err = yaml.Unmarshal([]byte(newCM.Data[limitador.LimitadorConfigFileName]), &cmLimits)
				if err != nil {
					return false
				}

				return reflect.DeepEqual(cmLimits, limits)
			}, timeout, interval).Should(BeTrue())
		})
	})
})
