package limitador

import (
	"testing"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConstants(t *testing.T) {
	assert.Check(t, DefaultReplicas == 1)
	assert.Check(t, LimitadorRepository == "quay.io/kuadrant/limitador")
	assert.Check(t, StatusEndpoint == "/status")
	assert.Check(t, LimitadorConfigFileName == "limitador-config.yaml")
	assert.Check(t, LimitsCMNamePrefix == "limits-config-")
	assert.Check(t, LimitadorCMMountPath == "/home/limitador/etc/")
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
