package limitador

import (
	"testing"

	"gotest.tools/assert"
	appsv1 "k8s.io/api/apps/v1"
)

func TestInMemoryDeploymentOptions(t *testing.T) {
	t.Run("basic inmemory deployment options", func(subT *testing.T) {
		options, err := InMemoryDeploymentOptions()
		assert.NilError(subT, err)
		assert.DeepEqual(subT, options,
			DeploymentStorageOptions{
				Args: []string{"memory"},
				DeploymentStrategy: appsv1.DeploymentStrategy{
					Type:          appsv1.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &appsv1.RollingUpdateDeployment{},
				}})
	})
}
