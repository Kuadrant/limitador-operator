package controllers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
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

	newStatus, requeue, err := r.calculateStatus(ctx, limitadorObj, specErr)
	if err != nil {
		return reconcile.Result{}, err
	}

	equalStatus := limitadorObj.Status.Equals(newStatus, logger)
	logger.V(1).Info("Status", "status is different", !equalStatus)
	logger.V(1).Info("Status", "generation is different", limitadorObj.Generation != limitadorObj.Status.ObservedGeneration)
	if equalStatus && limitadorObj.Generation == limitadorObj.Status.ObservedGeneration {
		// Steady state
		logger.V(1).Info("Status was not updated")
		return reconcile.Result{Requeue: requeue}, nil
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
	return ctrl.Result{Requeue: requeue}, nil
}

func (r *LimitadorReconciler) calculateStatus(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador, specErr error) (*limitadorv1alpha1.LimitadorStatus, bool, error) {
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

	readyCond, err := r.readyCondition(ctx, limitadorObj, specErr)
	if err != nil {
		return nil, true, err
	}

	meta.SetStatusCondition(&newStatus.Conditions, *readyCond)

	if meta.IsStatusConditionFalse(newStatus.Conditions, limitadorv1alpha1.StatusConditionReady) {
		return newStatus, true, nil
	}

	availableCond, requeue, err := r.availableCondition(ctx, limitadorObj)
	if err != nil {
		return nil, true, err
	}
	meta.SetStatusCondition(&newStatus.Conditions, *availableCond)

	return newStatus, requeue, nil
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

func (r *LimitadorReconciler) availableCondition(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) (*metav1.Condition, bool, error) {
	cond := &metav1.Condition{
		Type:    limitadorv1alpha1.StatusConditionAvailable,
		Status:  metav1.ConditionTrue,
		Reason:  "Available",
		Message: "Limitador limits synced",
	}

	reason, err := r.checkLimitadorSynced(ctx, limitadorObj)
	if err != nil {
		return nil, true, err
	}
	if reason != nil {
		cond.Status = metav1.ConditionFalse
		cond.Reason = "LimitadorNotAvailable"
		cond.Message = *reason
		return cond, true, nil
	}

	return cond, false, nil
}

func (r *LimitadorReconciler) checkLimitadorSynced(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) (*string, error) {
	// Create Kubernetes clientset.
	clientset, err := kubernetes.NewForConfig(r.RestConfig)
	if err != nil {
		return nil, err
	}

	podList := &corev1.PodList{}
	options := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(limitador.Labels(limitadorObj)),
		Namespace:     limitadorObj.Namespace,
	}
	if err := r.Client().List(ctx, podList, options); err != nil {
		return nil, err
	}

	if len(podList.Items) == 0 {
		return nil, errors.New("found 0 pods")
	}

	// Name of the pod where the function will be executed.
	podName := podList.Items[0].Name

	// Command to execute your function.
	command := []string{"cat", fmt.Sprintf("%s/%s", limitador.LimitadorCMMountPath, limitador.LimitadorConfigFileName)}

	req := clientset.CoreV1().
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
	executor, err := remotecommand.NewSPDYExecutor(r.RestConfig, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	// Create buffers to capture stdout and stderr.
	var stdout, stderr bytes.Buffer

	// Create a StreamOptions struct.
	streamOptions := remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	// Execute the function within the pod.
	err = executor.StreamWithContext(ctx, streamOptions)
	if err != nil {
		return nil, err
	}

	if stderr.String() != "" {
		return nil, errors.New(stderr.String())
	}

	// Get the config map
	configmap := corev1.ConfigMap{}
	if err := r.Client().Get(ctx, client.ObjectKey{Namespace: limitadorObj.Namespace, Name: limitador.LimitsConfigMapName(limitadorObj)}, &configmap); err != nil {
		return nil, err
	}

	configmapData := configmap.Data[limitador.LimitadorConfigFileName]
	// There might be line break differences
	configmapInPod := strings.ReplaceAll(stdout.String(), "\r\n", "\n")
	if configmapData != configmapInPod {
		tmp := "Limit difference detected"
		return &tmp, nil
	}

	return nil, nil
}

func (r *LimitadorReconciler) checkLimitadorAvailable(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) (*string, error) {
	deployment := &appsv1.Deployment{}
	dKey := client.ObjectKey{ // Its deployment is built after the same name and namespace
		Namespace: limitadorObj.Namespace,
		Name:      limitador.DeploymentName(limitadorObj),
	}
	err := r.Client().Get(ctx, dKey, deployment)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && apierrors.IsNotFound(err) {
		tmp := err.Error()
		return &tmp, nil
	}

	availableCondition := helpers.FindDeploymentStatusCondition(deployment.Status.Conditions, "Available")
	if availableCondition == nil {
		tmp := "Available condition not found"
		return &tmp, nil
	}

	if availableCondition.Status != corev1.ConditionTrue {
		return &availableCondition.Message, nil
	}

	return nil, nil
}

func buildServiceHost(limitadorObj *limitadorv1alpha1.Limitador) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", limitador.ServiceName(limitadorObj), limitadorObj.Namespace)
}
