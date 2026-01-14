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

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/utils/env"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/controllers"
	"github.com/kuadrant/limitador-operator/pkg/helpers"
	"github.com/kuadrant/limitador-operator/pkg/log"
	"github.com/kuadrant/limitador-operator/pkg/reconcilers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = k8sruntime.NewScheme()
	logLevel = env.GetString("LOG_LEVEL", "info")
	logMode  = env.GetString("LOG_MODE", "production")
	gitSHA   string // pass ldflag here to display gitSHA hash
	dirty    string // must be string as passed in by ldflag to determine display .
	version  string // must be string as passed in by ldflag to determine display .
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(limitadorv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	logger := log.NewLogger(
		log.SetLevel(log.ToLevel(logLevel)),
		log.SetMode(log.ToMode(logMode)),
		log.WriteTo(os.Stdout),
	)
	log.SetLogger(logger)
}

func printControllerMetaInfo() {
	setupLog := log.Log

	setupLog.Info(fmt.Sprintf("go version: %s", runtime.Version()))
	setupLog.Info(fmt.Sprintf("go os/arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	setupLog.Info("base logger", "log level", logLevel, "log mode", logMode)
	setupLog.Info("build information", "version", version, "commit", gitSHA, "dirty", dirty)
}

func main() {
	printControllerMetaInfo()
	setupLog := log.Log

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	cacheOptions := cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&corev1.ConfigMap{}: {
				Label: labels.SelectorFromSet(labels.Set{
					helpers.LabelKeyApp: helpers.LimitadorAppName,
				}),
			},
			&appsv1.Deployment{}: {
				Label: labels.SelectorFromSet(labels.Set{
					helpers.LabelKeyApp: helpers.LimitadorAppName,
				}),
			},
			&policyv1.PodDisruptionBudget{}: {
				Label: labels.SelectorFromSet(labels.Set{
					helpers.LabelKeyApp: helpers.LimitadorAppName,
				}),
			},
			&corev1.Pod{}: {
				Label: labels.SelectorFromSet(labels.Set{
					helpers.LabelKeyApp: helpers.LimitadorAppName,
				}),
			},
			&corev1.PersistentVolumeClaim{}: {
				Label: labels.SelectorFromSet(labels.Set{
					helpers.LabelKeyApp: helpers.LimitadorAppName,
				}),
			},
		},
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&corev1.Secret{},
				},
			},
		},
		Cache:                  cacheOptions,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		WebhookServer:          webhook.NewServer(webhook.Options{Port: 9443}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "3745a16e.kuadrant.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	limitadorBaseReconciler := reconcilers.NewBaseReconciler(
		mgr.GetClient(), mgr.GetScheme(), mgr.GetAPIReader(),
		log.Log.WithName("limitador"),
		mgr.GetEventRecorderFor("Limitador"),
	)

	if err = (&controllers.LimitadorReconciler{
		BaseReconciler: limitadorBaseReconciler,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create Limitador controller")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
