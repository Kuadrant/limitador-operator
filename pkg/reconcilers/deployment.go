package reconcilers

import (
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentMutateFn is a function which mutates the existing Deployment into it's desired state.
type DeploymentMutateFn func(desired, existing *appsv1.Deployment) bool

func DeploymentMutator(opts ...DeploymentMutateFn) MutateFn {
	return func(existingObj, desiredObj client.Object) (bool, error) {
		existing, ok := existingObj.(*appsv1.Deployment)
		if !ok {
			return false, fmt.Errorf("%T is not a *appsv1.Deployment", existingObj)
		}
		desired, ok := desiredObj.(*appsv1.Deployment)
		if !ok {
			return false, fmt.Errorf("%T is not a *appsv1.Deployment", desiredObj)
		}

		update := false

		// Loop through each option
		for _, opt := range opts {
			tmpUpdate := opt(desired, existing)
			update = update || tmpUpdate
		}

		return update, nil
	}
}

func DeploymentAffinityMutator(desired, existing *appsv1.Deployment) bool {
	update := false
	if !reflect.DeepEqual(existing.Spec.Template.Spec.Affinity, desired.Spec.Template.Spec.Affinity) {
		existing.Spec.Template.Spec.Affinity = desired.Spec.Template.Spec.Affinity
		update = true
	}
	return update
}

func DeploymentReplicasMutator(desired, existing *appsv1.Deployment) bool {
	update := false

	var existingReplicas int32 = 1
	if existing.Spec.Replicas != nil {
		existingReplicas = *existing.Spec.Replicas
	}

	var desiredReplicas int32 = 1
	if desired.Spec.Replicas != nil {
		desiredReplicas = *desired.Spec.Replicas
	}

	if desiredReplicas != existingReplicas {
		existing.Spec.Replicas = &desiredReplicas
		update = true
	}

	return update
}

func DeploymentContainerListMutator(desired, existing *appsv1.Deployment) bool {
	update := false

	if len(existing.Spec.Template.Spec.Containers) != len(desired.Spec.Template.Spec.Containers) {
		existing.Spec.Template.Spec.Containers = desired.Spec.Template.Spec.Containers
		update = true
	}

	return update
}

func DeploymentImageMutator(desired, existing *appsv1.Deployment) bool {
	update := false

	if existing.Spec.Template.Spec.Containers[0].Image != desired.Spec.Template.Spec.Containers[0].Image {
		existing.Spec.Template.Spec.Containers[0].Image = desired.Spec.Template.Spec.Containers[0].Image
		update = true
	}

	return update
}

func DeploymentCommandMutator(desired, existing *appsv1.Deployment) bool {
	update := false

	if !reflect.DeepEqual(existing.Spec.Template.Spec.Containers[0].Command, desired.Spec.Template.Spec.Containers[0].Command) {
		existing.Spec.Template.Spec.Containers[0].Command = desired.Spec.Template.Spec.Containers[0].Command
		update = true
	}

	return update
}

func DeploymentResourcesMutator(desired, existing *appsv1.Deployment) bool {
	update := false

	if !reflect.DeepEqual(existing.Spec.Template.Spec.Containers[0].Resources, desired.Spec.Template.Spec.Containers[0].Resources) {
		existing.Spec.Template.Spec.Containers[0].Resources = desired.Spec.Template.Spec.Containers[0].Resources
		update = true
	}

	return update
}
