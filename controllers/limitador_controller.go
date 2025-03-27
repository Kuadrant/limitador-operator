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
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
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
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=services;configmaps;secrets;persistentvolumeclaims,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=list;watch;update

func (r *LimitadorReconciler) Reconcile(eventCtx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger().WithValues("limitador", req.NamespacedName)
	logger.V(1).Info("Reconciling Limitador")
	ctx := logr.NewContext(eventCtx, logger)

	// Delete Limitador deployment and service if needed
	limitadorObj := &limitadorv1alpha1.Limitador{}
	if err := r.Client().Get(ctx, req.NamespacedName, limitadorObj); err != nil {
		if apierrors.IsNotFound(err) {
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

	statusResult, statusErr := r.reconcileStatus(ctx, limitadorObj, specErr)

	if specErr != nil {
		return ctrl.Result{}, specErr
	}

	if statusErr != nil {
		return ctrl.Result{}, statusErr
	}

	if specResult.Requeue {
		logger.V(1).Info("Reconciling spec not finished. Requeueing.")
		return specResult, nil
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

	if err := r.reconcilePVC(ctx, limitadorObj); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileDeployment(ctx, limitadorObj); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileLimitsConfigMap(ctx, limitadorObj); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcilePdb(ctx, limitadorObj); err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcilePodLimitsHashAnnotation(ctx, limitadorObj)
}

func (r *LimitadorReconciler) reconcilePodLimitsHashAnnotation(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) (ctrl.Result, error) {
	podList := &corev1.PodList{}
	options := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(limitador.Labels(limitadorObj)),
		Namespace:     limitadorObj.Namespace,
	}
	if err := r.Client().List(ctx, podList, options); err != nil {
		return ctrl.Result{}, err
	}

	if len(podList.Items) == 0 {
		return ctrl.Result{Requeue: true}, nil
	}

	// Replicas won't change if spec.Replicas goes from value to nil
	if limitadorObj.Spec.Replicas != nil && len(podList.Items) != int(limitadorObj.GetReplicas()) {
		return ctrl.Result{Requeue: true}, nil
	}

	// Use CM resource version to track limits changes
	cm := &corev1.ConfigMap{}
	if err := r.Client().Get(ctx, types.NamespacedName{Name: limitador.LimitsConfigMapName(limitadorObj), Namespace: limitadorObj.Namespace}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	for idx := range podList.Items {
		pod := podList.Items[idx]
		annotations := pod.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		// Update only if there is a change in resource version value
		if annotations[limitadorv1alpha1.PodAnnotationConfigMapResourceVersion] != cm.ResourceVersion {
			annotations[limitadorv1alpha1.PodAnnotationConfigMapResourceVersion] = cm.ResourceVersion
			pod.SetAnnotations(annotations)
			if err := r.Client().Update(ctx, &pod); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *LimitadorReconciler) reconcilePdb(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) error {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	pdb := limitador.PodDisruptionBudget(limitadorObj)
	if err := limitador.ValidatePDB(pdb); err != nil {
		return err
	}

	// controller reference
	if err := r.SetOwnerReference(limitadorObj, pdb); err != nil {
		return err
	}
	err = r.ReconcilePodDisruptionBudget(ctx, pdb, reconcilers.PodDisruptionBudgetMutator)
	logger.V(1).Info("reconcile pdb", "error", err)
	if err != nil {
		return err
	}
	return nil
}

func (r *LimitadorReconciler) reconcileDeployment(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) error {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	deploymentOptions, err := r.getDeploymentOptions(ctx, limitadorObj)
	if err != nil {
		return err
	}

	deploymentMutators := make([]reconcilers.DeploymentMutateFn, 0)
	if limitadorObj.Spec.Replicas != nil {
		deploymentMutators = append(deploymentMutators, reconcilers.DeploymentReplicasMutator)
	}

	deploymentMutators = append(deploymentMutators,
		reconcilers.DeploymentContainerListMutator,
		reconcilers.DeploymentImageMutator,
		reconcilers.DeploymentCommandMutator,
		reconcilers.DeploymentAffinityMutator,
		reconcilers.DeploymentResourcesMutator,
		reconcilers.DeploymentVolumesMutator,
		reconcilers.DeploymentVolumeMountsMutator,
		reconcilers.DeploymentEnvMutator,
		reconcilers.DeploymentPortsMutator,
		reconcilers.DeploymentLivenessProbeMutator,
		reconcilers.DeploymentReadinessProbeMutator,
		reconcilers.DeploymentTemplateLabelMutator,
		reconcilers.DeploymentObjectLabelMutator,
	)

	// reconcile imagepullsecrets only when set in limitador CR
	// if not set in limitador CR, the user will be able to add them manually and the operator
	// will not revert the changes.
	if len(deploymentOptions.ImagePullSecrets) > 0 {
		deploymentMutators = append(deploymentMutators, reconcilers.DeploymentImagePullSecretsMutator)
	}

	deployment := limitador.Deployment(limitadorObj, deploymentOptions)
	// controller reference
	if err := r.SetOwnerReference(limitadorObj, deployment); err != nil {
		return err
	}
	err = r.ReconcileDeployment(ctx, deployment, reconcilers.DeploymentMutator(deploymentMutators...))
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

	limitadorService := limitador.Service(limitadorObj)
	// controller reference
	if err := r.SetOwnerReference(limitadorObj, limitadorService); err != nil {
		return err
	}

	serviceMutator := reconcilers.ServiceMutator(reconcilers.ServicePortsMutator)

	err = r.ReconcileService(ctx, limitadorService, serviceMutator)
	logger.V(1).Info("reconcile service", "error", err)
	if err != nil {
		return err
	}

	return nil
}

func (r *LimitadorReconciler) reconcilePVC(ctx context.Context, limitadorObj *limitadorv1alpha1.Limitador) error {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return err
	}

	pvc := limitador.PVC(limitadorObj)
	// controller reference
	if err := r.SetOwnerReference(limitadorObj, pvc); err != nil {
		return err
	}

	// Not reconciling updates PVCs for now.
	err = r.ReconcilePersistentVolumeClaim(ctx, pvc, reconcilers.CreateOnlyMutator)
	logger.V(1).Info("reconcile pvc", "error", err)
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

func mutateLimitsConfigMap(desiredObj, existingObj client.Object) (bool, error) {
	existing, ok := existingObj.(*corev1.ConfigMap)
	if !ok {
		return false, fmt.Errorf("%T is not a *corev1.ConfigMap", existingObj)
	}
	desired, ok := desiredObj.(*corev1.ConfigMap)
	if !ok {
		return false, fmt.Errorf("%T is not a *corev1.ConfigMap", desiredObj)
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
		// if existing content cannot be parsed, leave existingLimits as nil, so the operator will
		// enforce desired content.
		existingLimits = nil
	}

	// TODO(eastizle): deepEqual returns false when the order in the list is not equal.
	// Improvement would be checking to equality of slices ignoring order
	if !reflect.DeepEqual(desiredLimits, existingLimits) {
		existing.Data[limitador.LimitadorConfigFileName] = desired.Data[limitador.LimitadorConfigFileName]
		updated = true
	}
	return updated, nil
}

func (r *LimitadorReconciler) getDeploymentOptions(ctx context.Context, limObj *limitadorv1alpha1.Limitador) (limitador.DeploymentOptions, error) {
	deploymentOptions := limitador.DeploymentOptions{}

	deploymentStorageOptions, err := r.getDeploymentStorageOptions(ctx, limObj)
	if err != nil {
		return deploymentOptions, err
	}

	deploymentOptions.Command = limitador.DeploymentCommand(limObj, deploymentStorageOptions)
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
