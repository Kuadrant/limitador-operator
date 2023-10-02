package limitador

func InMemoryDeploymentOptions() (DeploymentStorageOptions, error) {
	return DeploymentStorageOptions{
		Command: []string{"memory"},
	}, nil
}
