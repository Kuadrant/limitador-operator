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
	"encoding/json"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
	"github.com/kuadrant/limitador-operator/pkg/observability"
	"github.com/kuadrant/limitador-operator/pkg/reconcilers"
)

// LimitadorReconciler reconciles a Limitador object
type LimitadorReconciler struct {
	*reconcilers.BaseReconciler
}

//+kubebuilder:rbac:groups=limitador.kuadrant.io,resources=limitadors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=limitador.kuadrant.io,resources=limitadors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=limitador.kuadrant.io,resources=limitadors/finalizers,verbs=update
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services;configmaps;secrets;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=list;watch;update;patch

func (r *LimitadorReconciler) Reconcile(eventCtx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger().WithValues("limitador", req.NamespacedName)
	logger.V(1).Info("Reconciling Limitador")

	// Get Limitador object first to extract trace context from annotations
	limitadorObj := &limitadorv1alpha1.Limitador{}
	if err := r.Client().Get(eventCtx, req.NamespacedName, limitadorObj); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("no object found")
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Failed to get Limitador object.")
		return ctrl.Result{}, err
	}

	// Extract trace context from CR annotations as a LINK (not parent)
	// Since reconciliation is event-driven and asynchronous, we use a link rather than
	// a parent-child relationship to connect the operator trace with the operation that
	// created/updated the CR (e.g., kubectl apply, GitOps controller)
	var spanOpts []trace.SpanStartOption
	carrier := propagation.MapCarrier(limitadorObj.Annotations)
	linkCtx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	linkSpanCtx := trace.SpanFromContext(linkCtx).SpanContext()

	if linkSpanCtx.IsValid() {
		spanOpts = append(spanOpts, trace.WithLinks(trace.Link{
			SpanContext: linkSpanCtx,
		}))
	}

	// Store base logger (without trace context) in context
	ctx := observability.StoreBaseLogger(eventCtx, logger)

	// Start reconcile span with link to the original operation (if traceparent exists)
	// This automatically refreshes the logger in context with the span's trace context
	ctx, span := r.Tracer().StartReconcileSpan(ctx, req, spanOpts...)
	defer span.End()

	// Add Limitador-specific attributes to span
	storageType := "memory" // default
	if limitadorObj.Spec.Storage != nil {
		if limitadorObj.Spec.Storage.RedisCached != nil {
			storageType = "redis-cached"
		} else if limitadorObj.Spec.Storage.Redis != nil {
			storageType = "redis"
		} else if limitadorObj.Spec.Storage.Disk != nil {
			storageType = "disk"
		}
	}

	observability.AddLimitadorAttributes(span, limitadorObj.Namespace, limitadorObj.Name, limitadorObj.GetReplicas(), storageType)

	if logger.V(1).Enabled() {
		jsonData, err := json.MarshalIndent(limitadorObj, "", "  ")
		if err != nil {
			observability.RecordError(span, err, "failed to marshal Limitador object to JSON")
			return ctrl.Result{}, err
		}
		logger.V(1).Info(string(jsonData))
	}

	if limitadorObj.GetDeletionTimestamp() != nil {
		logger.Info("marked to be deleted")
		observability.RecordReconcileResult(span, ctrl.Result{}, nil)
		return ctrl.Result{}, nil
	}

	specResult, specErr := r.reconcileSpec(ctx, limitadorObj)

	statusResult, statusErr := r.reconcileStatus(ctx, limitadorObj, specErr)

	if specErr != nil {
		observability.RecordError(span, specErr, "spec reconciliation failed")
		return ctrl.Result{}, specErr
	}

	if statusErr != nil {
		observability.RecordError(span, statusErr, "status reconciliation failed")
		return ctrl.Result{}, statusErr
	}

	if specResult.RequeueAfter > 0 {
		logger.V(1).Info("Reconciling spec not finished. Requeueing.")
		observability.RecordReconcileResult(span, specResult, nil)
		return specResult, nil
	}

	if statusResult.RequeueAfter > 0 {
		logger.V(1).Info("Reconciling status not finished. Requeueing.")
		observability.RecordReconcileResult(span, statusResult, nil)
		return statusResult, nil
	}

	logger.Info("successfully reconciled")
	observability.RecordReconcileResult(span, ctrl.Result{}, nil)
	return ctrl.Result{}, nil
}

func (r *LimitadorReconciler) reconcileSpec(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) (ctrl.Result, error) {
	ctx, span := r.Tracer().StartReconcileSpecSpan(ctx)
	defer span.End()

	if err := r.reconcileService(ctx, limitadorObj); err != nil {
		observability.RecordError(span, err, "failed to reconcile service")
		return ctrl.Result{}, err
	}

	if err := r.reconcilePVC(ctx, limitadorObj); err != nil {
		observability.RecordError(span, err, "failed to reconcile PVC")
		return ctrl.Result{}, err
	}

	if err := r.reconcileDeployment(ctx, limitadorObj); err != nil {
		observability.RecordError(span, err, "failed to reconcile deployment")
		return ctrl.Result{}, err
	}

	if err := r.reconcileLimitsConfigMap(ctx, limitadorObj); err != nil {
		observability.RecordError(span, err, "failed to reconcile limits ConfigMap")
		return ctrl.Result{}, err
	}

	if err := r.reconcilePdb(ctx, limitadorObj); err != nil {
		observability.RecordError(span, err, "failed to reconcile PodDisruptionBudget")
		return ctrl.Result{}, err
	}

	result, err := r.reconcilePodLimitsHashAnnotation(ctx, limitadorObj)
	if err != nil {
		observability.RecordError(span, err, "failed to reconcile pod annotations")
	} else {
		observability.RecordSpecCompleted(span)
	}
	return result, err
}

func (r *LimitadorReconciler) reconcilePodLimitsHashAnnotation(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) (ctrl.Result, error) {
	ctx, span := r.Tracer().StartResourceSpan(ctx, "PodAnnotation", limitadorObj.Namespace, limitador.DeploymentName(limitadorObj))
	defer span.End()

	podList := &corev1.PodList{}
	options := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(limitador.Labels(limitadorObj)),
		Namespace:     limitadorObj.Namespace,
	}
	if err := r.Client().List(ctx, podList, options); err != nil {
		observability.RecordError(span, err, "failed to list pods")
		return ctrl.Result{}, err
	}

	if len(podList.Items) == 0 {
		span.SetStatus(codes.Ok, "")
		return ctrl.Result{Requeue: true}, nil
	}

	// Replicas won't change if spec.Replicas goes from value to nil
	if limitadorObj.Spec.Replicas != nil && len(podList.Items) != int(limitadorObj.GetReplicas()) {
		span.SetStatus(codes.Ok, "")
		return ctrl.Result{Requeue: true}, nil
	}

	// Use CM resource version to track limits changes
	cm := &corev1.ConfigMap{}
	if err := r.Client().Get(ctx, types.NamespacedName{Name: limitador.LimitsConfigMapName(limitadorObj), Namespace: limitadorObj.Namespace}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			span.SetStatus(codes.Ok, "")
			return ctrl.Result{Requeue: true}, nil
		}
		observability.RecordError(span, err, "failed to get limits ConfigMap")
		return ctrl.Result{}, err
	}

	for idx := range podList.Items {
		pod := &podList.Items[idx]
		annotations := pod.GetAnnotations()
		// Update only if there is a change in resource version value
		if annotations[limitadorv1alpha1.PodAnnotationConfigMapResourceVersion] != cm.ResourceVersion {
			if err := r.ReconcilePodAnnotation(ctx, pod.Name, pod.Namespace,
				limitadorv1alpha1.PodAnnotationConfigMapResourceVersion, cm.ResourceVersion); err != nil {
				observability.RecordError(span, err, "failed to update pod annotation")
				return ctrl.Result{}, err
			}
		}
	}

	span.SetStatus(codes.Ok, "")
	return ctrl.Result{}, nil
}

func (r *LimitadorReconciler) reconcilePdb(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) error {
	ctx, span := r.Tracer().StartResourceSpan(ctx, "PodDisruptionBudget", limitadorObj.Namespace, limitador.PodDisruptionBudgetName(limitadorObj))
	defer span.End()

	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	pdb := limitador.PodDisruptionBudget(limitadorObj)
	if err := r.SetOwnerReference(limitadorObj, pdb); err != nil {
		observability.RecordError(span, err, "failed to set owner reference")
		return err
	}

	err = r.ReconcilePodDisruptionBudget(ctx, pdb)
	logger.V(1).Info("reconcile pdb", "error", err)
	if err != nil {
		observability.RecordError(span, err, "failed to reconcile PodDisruptionBudget")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

func (r *LimitadorReconciler) reconcileDeployment(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) error {
	ctx, span := r.Tracer().StartResourceSpan(ctx, "Deployment", limitadorObj.Namespace, limitador.DeploymentName(limitadorObj))
	defer span.End()

	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	deploymentOptions, err := r.getDeploymentOptions(ctx, limitadorObj)
	if err != nil {
		observability.RecordError(span, err, "failed to get deployment options")
		return err
	}

	deployment := limitador.Deployment(limitadorObj, deploymentOptions)
	if err := r.SetOwnerReference(limitadorObj, deployment); err != nil {
		observability.RecordError(span, err, "failed to set owner reference")
		return err
	}

	err = r.ReconcileDeployment(ctx, deployment)
	logger.V(1).Info("reconcile deployment", "error", err)
	if err != nil {
		observability.RecordError(span, err, "failed to reconcile deployment")
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (r *LimitadorReconciler) reconcileService(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) error {
	ctx, span := r.Tracer().StartResourceSpan(ctx, "Service", limitadorObj.Namespace, limitador.ServiceName(limitadorObj))
	defer span.End()

	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	limitadorService := limitador.Service(limitadorObj)
	if err := r.SetOwnerReference(limitadorObj, limitadorService); err != nil {
		observability.RecordError(span, err, "failed to set owner reference")
		return err
	}

	err = r.ReconcileService(ctx, limitadorService)
	logger.V(1).Info("reconcile service", "error", err)
	if err != nil {
		observability.RecordError(span, err, "failed to reconcile service")
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (r *LimitadorReconciler) reconcilePVC(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) error {
	ctx, span := r.Tracer().StartResourceSpan(ctx, "PersistentVolumeClaim", limitadorObj.Namespace, limitador.PVCName(limitadorObj))
	defer span.End()

	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	pvc := limitador.PVC(limitadorObj)
	if err := r.SetOwnerReference(limitadorObj, pvc); err != nil {
		observability.RecordError(span, err, "failed to set owner reference")
		return err
	}

	err = r.ReconcilePersistentVolumeClaim(ctx, pvc)
	logger.V(1).Info("reconcile pvc", "error", err)
	if err != nil {
		observability.RecordError(span, err, "failed to reconcile PVC")
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (r *LimitadorReconciler) reconcileLimitsConfigMap(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) error {
	ctx, span := r.Tracer().StartResourceSpan(ctx, "ConfigMap", limitadorObj.Namespace, limitador.LimitsConfigMapName(limitadorObj))
	defer span.End()

	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	limitsConfigMap, err := limitador.LimitsConfigMap(limitadorObj)
	if err != nil {
		observability.RecordError(span, err, "failed to create limits ConfigMap")
		return err
	}
	if err := r.SetOwnerReference(limitadorObj, limitsConfigMap); err != nil {
		observability.RecordError(span, err, "failed to set owner reference")
		return err
	}

	err = r.ReconcileConfigMap(ctx, limitsConfigMap)
	logger.V(1).Info("reconcile limits ConfigMap", "error", err)
	if err != nil {
		observability.RecordError(span, err, "failed to reconcile limits ConfigMap")
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (r *LimitadorReconciler) getDeploymentOptions(ctx context.Context, limObj *limitadorv1alpha1.Limitador) (limitador.DeploymentOptions, error) {
	deploymentOptions := limitador.DeploymentOptions{}

	deploymentStorageOptions, err := r.getDeploymentStorageOptions(ctx, limObj)
	if err != nil {
		return deploymentOptions, err
	}

	deploymentOptions.Args = limitador.DeploymentArgs(limObj, deploymentStorageOptions)
	deploymentOptions.VolumeMounts = limitador.DeploymentVolumeMounts(deploymentStorageOptions)
	deploymentOptions.Volumes = limitador.DeploymentVolumes(limObj, deploymentStorageOptions)
	deploymentOptions.DeploymentStrategy = deploymentStorageOptions.DeploymentStrategy
	deploymentOptions.EnvVar, err = r.getDeploymentEnvVar(limObj)
	if err != nil {
		return deploymentOptions, err
	}
	deploymentOptions.ImagePullSecrets = r.getDeploymentImagePullSecrets(limObj)

	return deploymentOptions, nil
}

func (r *LimitadorReconciler) getDeploymentStorageOptions(ctx context.Context, limObj *limitadorv1alpha1.Limitador) (limitador.DeploymentStorageOptions, error) {
	if limObj.Spec.Storage != nil {
		if limObj.Spec.Storage.Redis != nil {
			return limitador.RedisDeploymentOptions(ctx, r.Client(), limObj.Namespace, *limObj.Spec.Storage.Redis)
		}

		if limObj.Spec.Storage.RedisCached != nil {
			return limitador.RedisCachedDeploymentOptions(ctx, r.Client(), limObj.Namespace, *limObj.Spec.Storage.RedisCached)
		}

		if limObj.Spec.Storage.Disk != nil {
			return limitador.DiskDeploymentOptions(limObj, *limObj.Spec.Storage.Disk)
		}

		// if all of them are nil, fallback to InMemory
	}

	return limitador.InMemoryDeploymentOptions()
}

func (r *LimitadorReconciler) getDeploymentEnvVar(limObj *limitadorv1alpha1.Limitador) ([]corev1.EnvVar, error) {
	if limObj.Spec.Storage != nil {
		if limObj.Spec.Storage.Redis != nil {
			return limitador.DeploymentEnvVar(limObj.Spec.Storage.Redis.ConfigSecretRef)
		}

		if limObj.Spec.Storage.RedisCached != nil {
			return limitador.DeploymentEnvVar(limObj.Spec.Storage.RedisCached.ConfigSecretRef)
		}
	}

	return nil, nil
}

func (r *LimitadorReconciler) getDeploymentImagePullSecrets(limObj *limitadorv1alpha1.Limitador) []corev1.LocalObjectReference {
	return limObj.Spec.ImagePullSecrets
}

// SetupWithManager sets up the controller with the Manager.
func (r *LimitadorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&limitadorv1alpha1.Limitador{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Complete(r)
}
