package limitador

import (
	"gotest.tools/assert"
	"testing"
)

func TestConstants(t *testing.T) {
	assert.Check(t, "latest" == DefaultVersion)
	assert.Check(t, 1 == DefaultReplicas)
	assert.Check(t, "quay.io/3scale/limitador" == Image)
	assert.Check(t, "/status" == StatusEndpoint)
	assert.Check(t, 8080 == DefaultServiceHTTPPort)
	assert.Check(t, 8081 == DefaultServiceGRPCPort)
	assert.Check(t, "LIMITADOR_CONFIG_FILE_NAME" == EnvLimitadorConfigFileName)
	assert.Check(t, "hash" == LimitadorCMHash)
	assert.Check(t, "limits-config-" == LimitsCMNamePrefix)
	assert.Check(t, "/" == LimitadorCMMountPath)
	assert.Check(t, "LIMITS_FILE" == LimitadorLimitsFileEnv)
}

//TODO: Test individual k8s objects. Extract limitadorObj creation from controller_test
