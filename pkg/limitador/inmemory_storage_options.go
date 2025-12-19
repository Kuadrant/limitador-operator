package limitador

func InMemoryDeploymentOptions() (DeploymentStorageOptions, error) {
	return DeploymentStorageOptions{
		Args: []string{"memory"},
	}, nil
}
