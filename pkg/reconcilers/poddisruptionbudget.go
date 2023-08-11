package reconcilers

import (
	"fmt"
	"reflect"

	policyv1 "k8s.io/api/policy/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PodDisruptionBudgetMutator(existingObj, desiredObj client.Object) (bool, error) {
	update := false

	existing, ok := existingObj.(*policyv1.PodDisruptionBudget)
	if !ok {
		return false, fmt.Errorf("%T is not a *policyv1.PodDisruptionBudget", existingObj)
	}
	desired, ok := desiredObj.(*policyv1.PodDisruptionBudget)
	if !ok {
		return false, fmt.Errorf("%T is not a *policyv1.PodDisruptionBudget", desiredObj)
	}

	if !reflect.DeepEqual(existing.Spec.MaxUnavailable, desired.Spec.MaxUnavailable) {
		existing.Spec.MaxUnavailable = desired.Spec.MaxUnavailable
		update = true
	}

	if !reflect.DeepEqual(existing.Spec.MinAvailable, desired.Spec.MinAvailable) {
		existing.Spec.MinAvailable = desired.Spec.MinAvailable
		update = true
	}

	return update, nil
}
