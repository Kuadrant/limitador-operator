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
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
	"github.com/kuadrant/limitador-operator/pkg/reconcilers"
)

// LimitadorReconciler reconciles a Limitador object
type LimitadorReconciler struct {
	*reconcilers.BaseReconciler
}

//+kubebuilder:rbac:groups=limitador.kuadrant.io,resources=limitadors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=limitador.kuadrant.io,resources=limitadors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=limitador.kuadrant.io,resources=limitadors/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete

func (r *LimitadorReconciler) Reconcile(eventCtx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger().WithValues("limitador", req.NamespacedName)
	logger.V(1).Info("Reconciling Limitador")
	ctx := logr.NewContext(eventCtx, logger)

	// Delete Limitador deployment and service if needed
	limitadorObj := &limitadorv1alpha1.Limitador{}
	if err := r.Client().Get(ctx, req.NamespacedName, limitadorObj); err != nil {
		if errors.IsNotFound(err) {
			// The deployment and the service should be deleted automatically
			// because they have an owner ref to Limitador
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Failed to get Limitador object.")
		return ctrl.Result{}, err
	}

	if limitadorObj.GetDeletionTimestamp() != nil { // Marked to be deleted
		logger.V(1).Info("marked to be deleted")
		return ctrl.Result{}, nil
	}

	limitadorService := limitador.LimitadorService(limitadorObj)
	err := r.ReconcileService(ctx, limitadorService, reconcilers.CreateOnlyMutator)
	logger.V(1).Info("reconcile service", "error", err)
	if err != nil {
		return ctrl.Result{}, err
	}

	deployment := limitador.LimitadorDeployment(limitadorObj)
	err = r.ReconcileDeployment(ctx, deployment, mutateLimitadorDeployment)
	logger.V(1).Info("reconcile deployment", "error", err)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile Limits ConfigMap
	limitsConfigMap, err := limitador.LimitsConfigMap(limitadorObj)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.ReconcileConfigMap(ctx, limitsConfigMap, mutateLimitsConfigMap)
	logger.V(1).Info("reconcile limits ConfigMap", "error", err)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile Status
	if err := r.reconcileStatus(ctx, limitadorObj); err != nil {
		switch err.Error() {
		case "resource not ready":
			return ctrl.Result{Requeue: true}, nil
		default:
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LimitadorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&limitadorv1alpha1.Limitador{}).
		Complete(r)
}

func (r *LimitadorReconciler) reconcileStatus(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) (err error) {
	logger := logr.FromContext(ctx)

	isLimitadorRunning := r.checkLimitadorInstanceIsRunning(ctx, limitadorObj)
	changed := updateStatusReady(limitadorObj, isLimitadorRunning)

	changed = updateStatusService(limitadorObj) || changed

	if !limitadorObj.Status.Ready() {
		err = fmt.Errorf("resource not ready")
	}

	if !changed {
		logger.V(1).Info("resource status did not change")
		return // to save an update request
	}

	logger.V(1).Info("resource status changed", "limitador/status", limitadorObj.Status)

	if updateErr := r.Client().Status().Update(ctx, limitadorObj); updateErr != nil {
		logger.Error(updateErr, "failed to update the resource")
		err = updateErr
		return
	}

	logger.Info("status updated", "Name", limitadorObj.Name)
	return
}

func (r *LimitadorReconciler) checkLimitadorInstanceIsRunning(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) bool {
	logger := logr.FromContext(ctx)
	limitadorInstance := &appsv1.Deployment{}
	limitadorInstanceNamespacedName := client.ObjectKey{ // Its deployment is built after the same name and namespace
		Namespace: limitadorObj.Namespace,
		Name:      limitadorObj.Name,
	}
	if err := r.Client().Get(ctx, limitadorInstanceNamespacedName, limitadorInstance); err != nil {
		logger.Error(err, "Failed to get Limitador Instance.")
		return false
	}

	return limitadorInstance.Status.ReadyReplicas >= 1
}

func updateStatusService(limitadorObj *limitadorv1alpha1.Limitador) (changed bool) {
	changed = false
	serviceHost := buildServiceHost(limitadorObj)
	if serviceHost != limitadorObj.Status.Service.Host {
		limitadorObj.Status.Service.Host = serviceHost
		limitadorObj.Status.Service.Ports.GRPC = limitadorObj.GRPCPort()
		limitadorObj.Status.Service.Ports.HTTP = limitadorObj.HTTPPort()
		changed = true
	}
	return
}

func updateStatusReady(limitadorObj *limitadorv1alpha1.Limitador, ready bool) (changed bool) {
	status := metav1.ConditionFalse
	reason := limitadorv1alpha1.StatusReasonServiceNotRunning
	message := "There's no Limitador Pod running"

	if ready {
		status = metav1.ConditionTrue
		reason = limitadorv1alpha1.StatusReasonInstanceRunning
		message = ""
	}
	limitadorObj.Status.Conditions, changed = updateStatusConditions(limitadorObj.Status.Conditions, metav1.Condition{
		Type:    limitadorv1alpha1.StatusConditionReady,
		Status:  status,
		Reason:  reason,
		Message: message,
	})

	return
}

func updateStatusConditions(currentConditions []metav1.Condition, newCondition metav1.Condition) ([]metav1.Condition, bool) {
	newCondition.LastTransitionTime = metav1.Now()

	if currentConditions == nil {
		return []metav1.Condition{newCondition}, true
	}

	for i, condition := range currentConditions {
		if condition.Type == newCondition.Type {
			if condition.Status == newCondition.Status {
				if condition.Reason == newCondition.Reason && condition.Message == newCondition.Message {
					return currentConditions, false
				}

				newCondition.LastTransitionTime = condition.LastTransitionTime
			}

			res := make([]metav1.Condition, len(currentConditions))
			copy(res, currentConditions)
			res[i] = newCondition
			return res, true
		}
	}

	return append(currentConditions, newCondition), true
}

func buildServiceHost(limitadorObj *limitadorv1alpha1.Limitador) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", limitador.ServiceName(limitadorObj), limitadorObj.Namespace)
}

func mutateLimitsConfigMap(existingObj, desiredObj client.Object) (bool, error) {
	existing, ok := existingObj.(*v1.ConfigMap)
	if !ok {
		return false, fmt.Errorf("%T is not a *v1.ConfigMap", existingObj)
	}
	desired, ok := desiredObj.(*v1.ConfigMap)
	if !ok {
		return false, fmt.Errorf("%T is not a *v1.ConfigMap", desiredObj)
	}

	updated := false

	if existing.Data[limitador.LimitadorCMHash] != desired.Data[limitador.LimitadorCMHash] {
		for k, v := range map[string]string{
			limitador.LimitadorCMHash:         desired.Data[limitador.LimitadorCMHash],
			limitador.LimitadorConfigFileName: string(desired.Data[limitador.LimitadorConfigFileName]),
		} {
			existing.Data[k] = v
		}
		updated = true
	}
	return updated, nil
}

func mutateLimitadorDeployment(existingObj, desiredObj client.Object) (bool, error) {
	existing, ok := existingObj.(*appsv1.Deployment)
	if !ok {
		return false, fmt.Errorf("%T is not a *appsv1.Deployment", existingObj)
	}
	desired, ok := desiredObj.(*appsv1.Deployment)
	if !ok {
		return false, fmt.Errorf("%T is not a *appsv1.Deployment", desiredObj)
	}

	updated := false

	if existing.Spec.Replicas != desired.Spec.Replicas {
		existing.Spec.Replicas = desired.Spec.Replicas
		updated = true
	}

	if existing.Spec.Template.Spec.Containers[0].Image != desired.Spec.Template.Spec.Containers[0].Image {
		existing.Spec.Template.Spec.Containers[0].Image = desired.Spec.Template.Spec.Containers[0].Image
		updated = true
	}

	return updated, nil
}
