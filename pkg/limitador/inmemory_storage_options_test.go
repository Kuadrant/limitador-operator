package limitador

import (
	"testing"

	"gotest.tools/assert"
)

func TestInMemoryDeploymentOptions(t *testing.T) {
	t.Run("basic inmemory deployment options", func(subT *testing.T) {
		options, err := InMemoryDeploymentOptions()
		assert.NilError(subT, err)
		assert.DeepEqual(subT, options,
			DeploymentStorageOptions{
				Args: []string{"memory"},
			})
	})
}
