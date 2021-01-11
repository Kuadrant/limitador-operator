/*
Copyright 2020 Red Hat.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	limitadorv1alpha1 "github.com/3scale/limitador-operator/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = limitadorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&LimitadorReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("limitador"),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Limitador controller", func() {
	const (
		LimitadorName      = "limitador-test"
		LimitadorNamespace = "default"
		LimitadorReplicas  = 2
		LimitadorImage     = "quay.io/3scale/limitador"
		LimitadorVersion   = "0.3.0"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	replicas := LimitadorReplicas
	version := LimitadorVersion
	limitador := limitadorv1alpha1.Limitador{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Limitador",
			APIVersion: "limitador.3scale.net/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      LimitadorName,
			Namespace: LimitadorNamespace,
		},
		Spec: limitadorv1alpha1.LimitadorSpec{
			Replicas: &replicas,
			Version:  &version,
		},
	}

	Context("Creating a new Limitador object", func() {
		BeforeEach(func() {
			err := k8sClient.Delete(context.TODO(), limitador.DeepCopy())
			Expect(err == nil || errors.IsNotFound(err))

			Expect(k8sClient.Create(context.TODO(), limitador.DeepCopy())).Should(Succeed())
		})

		It("Should create a new deployment with the right number of replicas and version", func() {
			createdLimitadorDeployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      LimitadorName,
					},
					&createdLimitadorDeployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(*createdLimitadorDeployment.Spec.Replicas).Should(
				Equal((int32)(LimitadorReplicas)),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Image).Should(
				Equal(LimitadorImage + ":" + LimitadorVersion),
			)
		})

		It("Should create a Limitador service", func() {
			createdLimitadorService := v1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: "default",   // Hardcoded for now
						Name:      "limitador", // Hardcoded for now
					},
					&createdLimitadorService)

				return err == nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Deleting a Limitador object", func() {
		BeforeEach(func() {
			err := k8sClient.Create(context.TODO(), limitador.DeepCopy())
			Expect(err == nil || errors.IsAlreadyExists(err))

			Expect(k8sClient.Delete(context.TODO(), limitador.DeepCopy())).Should(Succeed())
		})

		It("Should delete the limitador deployment", func() {
			createdLimitadorDeployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      LimitadorName,
					},
					&createdLimitadorDeployment)

				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("Should delete the limitador service", func() {
			createdLimitadorService := v1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: "default",   // Hardcoded for now
						Name:      "limitador", // Hardcoded for now
					},
					&createdLimitadorService)

				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Updating a limitador object", func() {
		BeforeEach(func() {
			err := k8sClient.Delete(context.TODO(), limitador.DeepCopy())
			Expect(err == nil || errors.IsNotFound(err))

			Expect(k8sClient.Create(context.TODO(), limitador.DeepCopy())).Should(Succeed())
		})

		It("Should modify the limitador deployment", func() {
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      LimitadorName,
					},
					&updatedLimitador)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			replicas = LimitadorReplicas + 1
			updatedLimitador.Spec.Replicas = &replicas
			version = "latest"
			updatedLimitador.Spec.Version = &version

			Expect(k8sClient.Update(context.TODO(), &updatedLimitador)).Should(Succeed())
			updatedLimitadorDeployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      LimitadorName,
					},
					&updatedLimitadorDeployment)

				if err != nil {
					return false
				}

				correctReplicas := *updatedLimitadorDeployment.Spec.Replicas == LimitadorReplicas+1
				correctImage := updatedLimitadorDeployment.Spec.Template.Spec.Containers[0].Image == LimitadorImage+":latest"

				return correctReplicas && correctImage
			}, timeout, interval).Should(BeTrue())
		})
	})
})
