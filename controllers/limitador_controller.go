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
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

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
//+kubebuilder:rbac:groups="",resources=configmaps;secrets,verbs=get;list;watch;create;update;delete

func (r *LimitadorReconciler) Reconcile(eventCtx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger().WithValues("limitador", req.NamespacedName)
	logger.V(1).Info("Reconciling Limitador")
	ctx := logr.NewContext(eventCtx, logger)

	// Delete Limitador deployment and service if needed
	limitadorObj := &limitadorv1alpha1.Limitador{}
	if err := r.Client().Get(ctx, req.NamespacedName, limitadorObj); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("no object found")
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Failed to get Limitador object.")
		return ctrl.Result{}, err
	}

	if logger.V(1).Enabled() {
		jsonData, err := json.MarshalIndent(limitadorObj, "", "  ")
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.V(1).Info(string(jsonData))
	}

	if limitadorObj.GetDeletionTimestamp() != nil {
		logger.Info("marked to be deleted")
		return ctrl.Result{}, nil
	}

	specResult, specErr := r.reconcileSpec(ctx, limitadorObj)
	if specErr == nil && specResult.Requeue {
		logger.V(1).Info("Reconciling spec not finished. Requeueing.")
		return specResult, nil
	}

	statusResult, statusErr := r.reconcileStatus(ctx, limitadorObj, specErr)

	if specErr != nil {
		return ctrl.Result{}, specErr
	}

	if statusErr != nil {
		return ctrl.Result{}, statusErr
	}

	if statusResult.Requeue {
		logger.V(1).Info("Reconciling status not finished. Requeueing.")
		return statusResult, nil
	}

	logger.Info("successfully reconciled")
	return ctrl.Result{}, nil
}

func (r *LimitadorReconciler) reconcileSpec(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) (ctrl.Result, error) {
	if err := r.reconcileService(ctx, limitadorObj); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileDeployment(ctx, limitadorObj); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileLimitsConfigMap(ctx, limitadorObj); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *LimitadorReconciler) reconcileDeployment(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) error {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	limitadorStorage := limitadorObj.Spec.Storage
	var storageConfigSecret *v1.Secret
	if limitadorStorage != nil {
		if limitadorStorage.Validate() {
			if storageConfigSecret, err = getStorageConfigSecret(ctx, r.Client(), limitadorObj.Namespace, limitadorStorage.SecretRef()); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("there's no ConfigSecretRef set")
		}
	}

	deployment := limitador.LimitadorDeployment(limitadorObj, storageConfigSecret)
	// controller reference
	if err := r.SetOwnerReference(limitadorObj, deployment); err != nil {
		return err
	}
	err = r.ReconcileDeployment(ctx, deployment, mutateLimitadorDeployment)
	logger.V(1).Info("reconcile deployment", "error", err)
	if err != nil {
		return err
	}

	return nil
}

func (r *LimitadorReconciler) reconcileService(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) error {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	limitadorService := limitador.LimitadorService(limitadorObj)
	// controller reference
	if err := r.SetOwnerReference(limitadorObj, limitadorService); err != nil {
		return err
	}

	err = r.ReconcileService(ctx, limitadorService, reconcilers.CreateOnlyMutator)
	logger.V(1).Info("reconcile service", "error", err)
	if err != nil {
		return err
	}

	return nil
}

func (r *LimitadorReconciler) reconcileLimitsConfigMap(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) error {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	limitsConfigMap, err := limitador.LimitsConfigMap(limitadorObj)
	if err != nil {
		return err
	}
	// controller reference
	if err := r.SetOwnerReference(limitadorObj, limitsConfigMap); err != nil {
		return err
	}

	err = r.ReconcileConfigMap(ctx, limitsConfigMap, mutateLimitsConfigMap)
	logger.V(1).Info("reconcile limits ConfigMap", "error", err)
	if err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LimitadorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&limitadorv1alpha1.Limitador{}).
		Owns(&appsv1.Deployment{}).
		Owns(&v1.ConfigMap{}).
		Complete(r)
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

	// Limits in limitador.LimitadorConfigFileName field
	var desiredLimits []limitadorv1alpha1.RateLimit
	err := yaml.Unmarshal([]byte(desired.Data[limitador.LimitadorConfigFileName]), &desiredLimits)
	if err != nil {
		return false, err
	}

	var existingLimits []limitadorv1alpha1.RateLimit
	err = yaml.Unmarshal([]byte(existing.Data[limitador.LimitadorConfigFileName]), &existingLimits)
	if err != nil {
		return false, err
	}

	// TODO(eastizle): deepEqual returns false when the order in the list is not equal.
	// Improvement would be checking to equality of slices ignoring order
	if !reflect.DeepEqual(desiredLimits, existingLimits) {
		existing.Data[limitador.LimitadorConfigFileName] = desired.Data[limitador.LimitadorConfigFileName]
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

func getStorageConfigSecret(ctx context.Context, client client.Client, limitadorNamespace string, secretRef *v1.ObjectReference) (*v1.Secret, error) {
	storageConfigSecret := &v1.Secret{}
	if err := client.Get(
		ctx,
		types.NamespacedName{
			Name: secretRef.Name,
			Namespace: func() string {
				if secretRef.Namespace != "" {
					return secretRef.Namespace
				}
				return limitadorNamespace
			}(),
		},
		storageConfigSecret,
	); err != nil {
		return nil, err
	}

	if len(storageConfigSecret.Data) > 0 && storageConfigSecret.Data["URL"] != nil {
		return storageConfigSecret, nil
	}
	return nil, errors.NewBadRequest("the storage config Secret doesn't have the `URL` field")
}
