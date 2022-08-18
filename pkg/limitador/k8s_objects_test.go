package limitador

import (
	"testing"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// TODO: Test individual k8s objects.
func newTestLimitadorObj(name, namespace string, limits []limitadorv1alpha1.RateLimit) *limitadorv1alpha1.Limitador {
	var (
		replicas = 1
		version  = "1.0"
		httpPort = int32(8000)
		grpcPort = int32(8001)
	)
	return &limitadorv1alpha1.Limitador{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Limitador",
			APIVersion: "limitador.kuadrant.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: limitadorv1alpha1.LimitadorSpec{
			Replicas: &replicas,
			Version:  &version,
			Listener: &limitadorv1alpha1.Listener{
				HTTP: &limitadorv1alpha1.TransportProtocol{Port: &httpPort},
				GRPC: &limitadorv1alpha1.TransportProtocol{Port: &grpcPort},
			},
			Limits: limits,
		},
	}

}

func TestServiceName(t *testing.T) {
	name := ServiceName(newTestLimitadorObj("my-limitador-instance", "default", nil))
	assert.Equal(t, name, "limitador-my-limitador-instance")
}
