package limitador

import appsv1 "k8s.io/api/apps/v1"

func InMemoryDeploymentOptions() (DeploymentStorageOptions, error) {
	return DeploymentStorageOptions{
		Args: []string{"memory"},
		DeploymentStrategy: appsv1.DeploymentStrategy{
			Type:          appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{},
		},
	}, nil
}
