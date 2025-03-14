/*
Copyright 2021 Red Hat, Inc.

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

package reconcilers

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kuadrant/limitador-operator/pkg/helpers"
)

// MutateFn is a function which mutates the existing object into it's desired state.
type MutateFn func(desired, existing client.Object) (bool, error)

func CreateOnlyMutator(_, _ client.Object) (bool, error) {
	return false, nil
}

type BaseReconciler struct {
	client          client.Client
	scheme          *runtime.Scheme
	apiClientReader client.Reader
	logger          logr.Logger
	recorder        record.EventRecorder
}

// blank assignment to verify that BaseReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &BaseReconciler{}

func NewBaseReconciler(
	client client.Client, scheme *runtime.Scheme, apiClientReader client.Reader,
	logger logr.Logger, recorder record.EventRecorder) *BaseReconciler {
	return &BaseReconciler{
		client:          client,
		scheme:          scheme,
		apiClientReader: apiClientReader,
		logger:          logger,
		recorder:        recorder,
	}
}

func (b *BaseReconciler) Reconcile(context.Context, ctrl.Request) (ctrl.Result, error) {
	return reconcile.Result{}, nil
}

// Client returns a split client that reads objects from
// the cache and writes to the Kubernetes APIServer
func (b *BaseReconciler) Client() client.Client {
	return b.client
}

// APIClientReader return a client that directly reads objects
// from the Kubernetes APIServer
func (b *BaseReconciler) APIClientReader() client.Reader {
	return b.apiClientReader
}

func (b *BaseReconciler) Scheme() *runtime.Scheme {
	return b.scheme
}

func (b *BaseReconciler) Logger() logr.Logger {
	return b.logger
}

func (b *BaseReconciler) EventRecorder() record.EventRecorder {
	return b.recorder
}

// ReconcileResource attempts to mutate the existing state
// in order to match the desired state. The object's desired state must be reconciled
// with the existing state inside the passed in callback MutateFn.
//
// obj: Object of the same type as the 'desired' object.
//
//	Used to read the resource from the kubernetes cluster.
//	Could be zero-valued initialized object.
//
// desired: Object representing the desired state
//
// It returns an error.
func (b *BaseReconciler) ReconcileResource(ctx context.Context, obj, desired client.Object, mutateFn MutateFn) error {
	key := client.ObjectKeyFromObject(desired)

	if err := b.Client().Get(ctx, key, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		// Not found
		if !helpers.IsObjectTaggedToDelete(desired) {
			return b.CreateResource(ctx, desired)
		}

		// Marked for deletion and not found. Nothing to do.
		return nil
	}

	// item found successfully
	if helpers.IsObjectTaggedToDelete(desired) {
		return b.DeleteResource(ctx, desired)
	}

	update, err := mutateFn(desired, obj)
	if err != nil {
		return err
	}

	if update {
		return b.UpdateResource(ctx, obj)
	}

	return nil
}

func (b *BaseReconciler) ReconcileService(ctx context.Context, desired *corev1.Service, mutatefn MutateFn) error {
	return b.ReconcileResource(ctx, &corev1.Service{}, desired, mutatefn)
}

func (b *BaseReconciler) ReconcileDeployment(ctx context.Context, desired *appsv1.Deployment, mutatefn MutateFn) error {
	return b.ReconcileResource(ctx, &appsv1.Deployment{}, desired, mutatefn)
}

func (b *BaseReconciler) ReconcileConfigMap(ctx context.Context, desired *corev1.ConfigMap, mutatefn MutateFn) error {
	return b.ReconcileResource(ctx, &corev1.ConfigMap{}, desired, mutatefn)
}

func (b *BaseReconciler) ReconcilePodDisruptionBudget(ctx context.Context, desired *policyv1.PodDisruptionBudget, mutatefn MutateFn) error {
	return b.ReconcileResource(ctx, &policyv1.PodDisruptionBudget{}, desired, mutatefn)
}

func (b *BaseReconciler) ReconcilePersistentVolumeClaim(ctx context.Context, desired *corev1.PersistentVolumeClaim, mutatefn MutateFn) error {
	return b.ReconcileResource(ctx, &corev1.PersistentVolumeClaim{}, desired, mutatefn)
}

func (b *BaseReconciler) GetResource(ctx context.Context, objKey types.NamespacedName, obj client.Object) error {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	logger.Info("get object", "GKV", obj.GetObjectKind().GroupVersionKind(), "name", obj.GetName(), "namespace", obj.GetNamespace())
	return b.Client().Get(ctx, objKey, obj)
}

func (b *BaseReconciler) CreateResource(ctx context.Context, obj client.Object) error {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	logger.Info("create object", "GKV", obj.GetObjectKind().GroupVersionKind(), "name", obj.GetName(), "namespace", obj.GetNamespace())
	return b.Client().Create(ctx, obj)
}

func (b *BaseReconciler) UpdateResource(ctx context.Context, obj client.Object) error {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	logger.Info("update object", "GKV", obj.GetObjectKind().GroupVersionKind(), "name", obj.GetName(), "namespace", obj.GetNamespace())
	return b.Client().Update(ctx, obj)
}

func (b *BaseReconciler) DeleteResource(ctx context.Context, obj client.Object, options ...client.DeleteOption) error {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	logger.Info("delete object", "GKV", obj.GetObjectKind().GroupVersionKind(), "name", obj.GetName(), "namespace", obj.GetNamespace())
	return b.Client().Delete(ctx, obj, options...)
}

func (b *BaseReconciler) UpdateResourceStatus(ctx context.Context, obj client.Object) error {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	logger.Info("update object status", "GKV", obj.GetObjectKind().GroupVersionKind(), "name", obj.GetName(), "namespace", obj.GetNamespace())
	return b.Client().Status().Update(ctx, obj)
}

// SetOwnerReference sets owner as a Controller OwnerReference on owned
func (b *BaseReconciler) SetOwnerReference(owner, obj client.Object) error {
	err := controllerutil.SetControllerReference(owner, obj, b.Scheme())
	if err != nil {
		b.Logger().Error(err, "Error setting OwnerReference on object",
			"Kind", obj.GetObjectKind().GroupVersionKind().String(),
			"Namespace", obj.GetNamespace(),
			"Name", obj.GetName(),
		)
	}
	return err
}

// EnsureOwnerReference sets owner as a Controller OwnerReference on owned
// returns boolean to notify when the object has been updated
func (b *BaseReconciler) EnsureOwnerReference(owner, obj client.Object) (bool, error) {
	changed := false

	originalSize := len(obj.GetOwnerReferences())
	err := b.SetOwnerReference(owner, obj)
	if err != nil {
		return false, err
	}

	newSize := len(obj.GetOwnerReferences())
	if originalSize != newSize {
		changed = true
	}

	return changed, nil
}
