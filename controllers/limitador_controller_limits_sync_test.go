package controllers

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

var _ = Describe("Limitador controller syncs limits to pod", func() {
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

	Context("Updating a Limitador object - multiple replicas", func() {
		var (
			limitadorObj *limitadorv1alpha1.Limitador
			replicas     = 3
		)

		limit1 := limitadorv1alpha1.RateLimit{
			Conditions: []string{"req.method == 'GET'"},
			MaxValue:   10,
			Namespace:  "test-namespace",
			Seconds:    60,
			Variables:  []string{"user_id"},
			Name:       "useless",
		}
		limits := []limitadorv1alpha1.RateLimit{limit1}
		updatedLimits := []limitadorv1alpha1.RateLimit{
			limit1,
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
			limitadorObj.Spec.Replicas = &replicas
			limitadorObj.Spec.Limits = limits
			Expect(k8sClient.Create(ctx, limitadorObj)).Should(Succeed())
			Eventually(testLimitadorIsReady(ctx, limitadorObj)).WithContext(ctx).Should(Succeed())
		})

		It("Should sync config map changes to pod automatically via Limitador's file watcher", func(ctx SpecContext) {
			// Get pods
			podList := &corev1.PodList{}
			options := &client.ListOptions{
				LabelSelector: labels.SelectorFromSet(limitador.Labels(limitadorObj)),
				Namespace:     limitadorObj.Namespace,
			}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, podList, options)).To(Succeed())
				g.Expect(podList.Items).To(HaveLen(replicas))
			}).WithContext(ctx).Should(Succeed())

			// Update limitador with new limits
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNamespace,
					Name:      limitadorObj.Name,
				}, &updatedLimitador)).To(Succeed())

				updatedLimitador.Spec.Limits = updatedLimits

				g.Expect(k8sClient.Update(ctx, &updatedLimitador)).To(Succeed())
			}).WithContext(ctx).Should(Succeed())

			// Wait for ConfigMap to be updated by the operator
			cm := &corev1.ConfigMap{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      limitador.LimitsConfigMapName(limitadorObj),
					Namespace: limitadorObj.Namespace,
				}, cm)).To(Succeed())

				// Verify ConfigMap has updated content
				g.Expect(cm.Data[limitador.LimitadorConfigFileName]).To(ContainSubstring("req.method == 'POST'"))
			}).WithContext(ctx).Should(Succeed())

			// Verify that Limitador automatically reloads the config from the ConfigMap
			// Limitador watches the config file and detects symlink changes made by Kubernetes
			// when the ConfigMap is updated
			config, err := config.GetConfig()
			Expect(err).To(BeNil())
			clientSet, err := kubernetes.NewForConfig(config)
			Expect(err).To(BeNil())

			// Name of the pod where the function will be executed.
			podName := podList.Items[0].Name

			// Command to read the config file inside the pod
			command := []string{"cat", fmt.Sprintf("%s/%s", limitador.LimitadorCMMountPath, limitador.LimitadorConfigFileName)}

			// Verify the config file in the pod matches the updated ConfigMap
			// This happens automatically via Kubernetes ConfigMap mount (symlink update)
			// and Limitador's file watcher detects and reloads it
			Eventually(func(g Gomega) {
				req := clientSet.CoreV1().
					RESTClient().
					Post().
					Resource("pods").
					Name(podName).
					Namespace(limitadorObj.Namespace).
					SubResource("exec")

				option := &corev1.PodExecOptions{
					Command:   command,
					Stdin:     false,
					Stdout:    true,
					Stderr:    true,
					TTY:       true,
					Container: "limitador",
				}
				req.VersionedParams(
					option,
					scheme.ParameterCodec,
				)

				// Create an executor.
				executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
				g.Expect(err).To(BeNil())

				// Create buffers to capture stdout and stderr.
				var stdout, stderr bytes.Buffer

				// Create a StreamOptions struct.
				streamOptions := remotecommand.StreamOptions{
					Stdout: &stdout,
					Stderr: &stderr,
				}

				// Execute the function within the pod.
				err = executor.StreamWithContext(ctx, streamOptions)
				g.Expect(err).To(BeNil())
				g.Expect(stderr.String()).To(BeEmpty())

				// Get the config map
				configmap := corev1.ConfigMap{}
				if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: limitadorObj.Namespace, Name: limitador.LimitsConfigMapName(limitadorObj)}, &configmap); err != nil {
					g.Expect(err).To(BeNil())
				}

				configmapData := configmap.Data[limitador.LimitadorConfigFileName]
				// There might be line break differences
				configmapInPod := strings.ReplaceAll(stdout.String(), "\r\n", "\n")
				g.Expect(configmapData).To(Equal(configmapInPod))
			}).WithContext(ctx).Should(Succeed())
		}, specTimeOut)
	})
})
