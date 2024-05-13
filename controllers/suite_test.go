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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/log"
	"github.com/kuadrant/limitador-operator/pkg/reconcilers"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

// SharedConfig contains minimum cluster connection config that can be safely marshalled as rest.Config is unsafe to marshall
type SharedConfig struct {
	Host            string          `json:"host"`
	TLSClientConfig TLSClientConfig `json:"tlsClientConfig"`
}

type TLSClientConfig struct {
	Insecure bool    `json:"insecure"`
	CertData []uint8 `json:"certData,omitempty"`
	KeyData  []uint8 `json:"keyData,omitempty"`
	CAData   []uint8 `json:"caData,omitempty"`
}

var _ = SynchronizedBeforeSuite(func() []byte {
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    ptr.To(true),
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	Expect(limitadorv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	limitadorBaseReconciler := reconcilers.NewBaseReconciler(
		mgr.GetClient(), mgr.GetScheme(), mgr.GetAPIReader(),
		ctrl.Log.WithName("controllers").WithName("limitador"),
		mgr.GetEventRecorderFor("Limitador"),
	)

	err = (&LimitadorReconciler{
		BaseReconciler: limitadorBaseReconciler,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
		gexec.KillAndWait(4 * time.Second)

		// Teardown the test environment once controller is finished.
		// Otherwise from Kubernetes 1.21+, teardown timeouts waiting on
		// kube-apiserver to return
		err := testEnv.Stop()
		Expect(err).ToNot(HaveOccurred())
	}()

	// Create a shared configuration struct to pass Config information to all sub processes
	sharedCfg := SharedConfig{
		Host: cfg.Host,
		TLSClientConfig: TLSClientConfig{
			Insecure: cfg.TLSClientConfig.Insecure,
			CertData: cfg.TLSClientConfig.CertData,
			KeyData:  cfg.TLSClientConfig.KeyData,
			CAData:   cfg.TLSClientConfig.CAData,
		},
	}

	// Marshal the shared configuration struct
	data, err := json.Marshal(sharedCfg)
	Expect(err).NotTo(HaveOccurred())

	return data
}, func(data []byte) {
	// Unmarshal the shared configuration struct
	var sharedCfg SharedConfig
	Expect(json.Unmarshal(data, &sharedCfg)).To(Succeed())

	// Create the rest.Config object from the shared configuration
	cfg := &rest.Config{
		Host: sharedCfg.Host,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: sharedCfg.TLSClientConfig.Insecure,
			CertData: sharedCfg.TLSClientConfig.CertData,
			KeyData:  sharedCfg.TLSClientConfig.KeyData,
			CAData:   sharedCfg.TLSClientConfig.CAData,
		},
	}

	// Create new scheme for each client
	s := runtime.NewScheme()
	Expect(scheme.AddToScheme(s)).To(Succeed())
	err := limitadorv1alpha1.AddToScheme(s)
	Expect(err).NotTo(HaveOccurred())

	// Set the shared configuration
	k8sClient, err = client.New(cfg, client.Options{Scheme: s})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	By("tearing down the test environment")
})

func TestMain(m *testing.M) {
	logger := log.NewLogger(
		log.SetLevel(log.DebugLevel),
		log.SetMode(log.ModeDev),
		log.WriteTo(GinkgoWriter),
	).WithName("controller_test")
	log.SetLogger(logger)

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	os.Exit(m.Run())
}
