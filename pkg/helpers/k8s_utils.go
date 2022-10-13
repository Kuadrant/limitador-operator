package helpers

import (
	appsv1 "k8s.io/api/apps/v1"
)

func FindDeploymentStatusCondition(conditions []appsv1.DeploymentCondition, conditionType string) *appsv1.DeploymentCondition {
	for i := range conditions {
		if conditions[i].Type == appsv1.DeploymentConditionType(conditionType) {
			return &conditions[i]
		}
	}

	return nil
}
