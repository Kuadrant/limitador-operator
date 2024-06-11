package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/helpers"
	"github.com/kuadrant/limitador-operator/pkg/limitador"
)

func (r *LimitadorReconciler) reconcileStatus(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador, specErr error) (ctrl.Result, error) {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	newStatus, err := r.calculateStatus(ctx, limitadorObj, specErr)
	if err != nil {
		return reconcile.Result{}, err
	}

	equalStatus := limitadorObj.Status.Equals(newStatus, logger)
	logger.V(1).Info("Status", "status is different", !equalStatus)
	logger.V(1).Info("Status", "generation is different", limitadorObj.Generation != limitadorObj.Status.ObservedGeneration)
	if equalStatus && limitadorObj.Generation == limitadorObj.Status.ObservedGeneration {
		// Steady state
		logger.V(1).Info("Status was not updated")
		return reconcile.Result{}, nil
	}

	logger.V(1).Info("Updating Status", "sequence no:", fmt.Sprintf("sequence No: %v->%v", limitadorObj.Status.ObservedGeneration, newStatus.ObservedGeneration))

	limitadorObj.Status = *newStatus
	updateErr := r.Client().Status().Update(ctx, limitadorObj)
	if updateErr != nil {
		// Ignore conflicts, resource might just be outdated.
		if apierrors.IsConflict(updateErr) {
			logger.Info("Failed to update status: resource might just be outdated")
			return reconcile.Result{Requeue: true}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to update status: %w", updateErr)
	}
	return ctrl.Result{}, nil
}

func (r *LimitadorReconciler) calculateStatus(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador, specErr error) (*limitadorv1alpha1.LimitadorStatus, error) {
	newStatus := &limitadorv1alpha1.LimitadorStatus{
		ObservedGeneration: limitadorObj.Generation,
		// Copy initial conditions. Otherwise, status will always be updated
		Conditions: helpers.DeepCopyConditions(limitadorObj.Status.Conditions),
		Service: &limitadorv1alpha1.LimitadorService{
			Host: buildServiceHost(limitadorObj),
			Ports: limitadorv1alpha1.Ports{
				HTTP: limitadorObj.HTTPPort(),
				GRPC: limitadorObj.GRPCPort(),
			},
		},
	}

	availableCond, err := r.readyCondition(ctx, limitadorObj, specErr)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&newStatus.Conditions, *availableCond)

	return newStatus, nil
}

func (r *LimitadorReconciler) readyCondition(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador, specErr error) (*metav1.Condition, error) {
	cond := &metav1.Condition{
		Type:    limitadorv1alpha1.StatusConditionReady,
		Status:  metav1.ConditionTrue,
		Reason:  "Ready",
		Message: "Limitador is ready",
	}

	if specErr != nil {
		cond.Status = metav1.ConditionFalse
		cond.Reason = "ReconcilliationError"
		cond.Message = specErr.Error()
		return cond, nil
	}

	reason, err := r.checkLimitadorAvailable(ctx, limitadorObj)
	if err != nil {
		return nil, err
	}
	if reason != nil {
		cond.Status = metav1.ConditionFalse
		cond.Reason = "LimitadorNotAvailable"
		cond.Message = *reason
		return cond, nil
	}

	return cond, nil
}

func (r *LimitadorReconciler) checkLimitadorAvailable(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) (*string, error) {
	deployment := &appsv1.Deployment{}
	dKey := client.ObjectKey{ // Its deployment is built after the same name and namespace
		Namespace: limitadorObj.Namespace,
		Name:      limitador.DeploymentName(limitadorObj),
	}
	err := r.Client().Get(ctx, dKey, deployment)
	if client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	if apierrors.IsNotFound(err) {
		return ptr.To(err.Error()), nil
	}

	if deployment.Status.ObservedGeneration != deployment.Generation {
		return ptr.To("Deployment still in progress"), nil
	}

	availableCondition := helpers.FindDeploymentStatusCondition(deployment.Status.Conditions, string(appsv1.DeploymentAvailable))
	if availableCondition == nil {
		return ptr.To("Available condition not found"), nil
	}

	if availableCondition.Status != corev1.ConditionTrue {
		return &availableCondition.Message, nil
	}

	if deployment.Status.UnavailableReplicas != 0 {
		return ptr.To("Deployment has unavailable replicas"), nil
	}

	if deployment.Status.ReadyReplicas != deployment.Status.Replicas {
		return ptr.To("Deployment has replicas not ready yet"), nil
	}

	return nil, nil
}

func buildServiceHost(limitadorObj *limitadorv1alpha1.Limitador) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", limitador.ServiceName(limitadorObj), limitadorObj.Namespace)
}
