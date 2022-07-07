package limitador

import (
	"testing"

	"gotest.tools/assert"
)

func TestConstants(t *testing.T) {
	assert.Check(t, "latest" == DefaultVersion)
	assert.Check(t, 1 == DefaultReplicas)
	assert.Check(t, "quay.io/3scale/limitador" == Image)
	assert.Check(t, "/status" == StatusEndpoint)
	assert.Check(t, "limitador-config.yaml" == LimitadorConfigFileName)
	assert.Check(t, "hash" == LimitadorCMHash)
	assert.Check(t, "limits-config-" == LimitsCMNamePrefix)
	assert.Check(t, "/home/limitador/etc/" == LimitadorCMMountPath)
	assert.Check(t, "LIMITS_FILE" == LimitadorLimitsFileEnv)
}

//TODO: Test individual k8s objects. Extract limitadorObj creation from controller_test
