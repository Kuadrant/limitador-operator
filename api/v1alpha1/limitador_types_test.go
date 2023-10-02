package v1alpha1

import (
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLimitadorGetResourceRequirements(t *testing.T) {
	var resourceRequirements = &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1m"),
			corev1.ResourceMemory: resource.MustParse("1Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2m"),
			corev1.ResourceMemory: resource.MustParse("2Mi"),
		},
	}

	t.Run("test default is returned when not specified in spec", func(subT *testing.T) {
		l := &Limitador{Spec: LimitadorSpec{}}
		assert.DeepEqual(subT, l.GetResourceRequirements(), defaultResourceRequirements)
	})

	t.Run("test value in spec is returned when value is not nil", func(subT *testing.T) {
		l := &Limitador{Spec: LimitadorSpec{ResourceRequirements: resourceRequirements}}
		assert.DeepEqual(subT, l.GetResourceRequirements(), resourceRequirements)
	})
}

func TestLimitadorGRPCPort(t *testing.T) {
	t.Run("test default is returned if spec listener is nil", func(subT *testing.T) {
		l := Limitador{}
		assert.Equal(subT, l.GRPCPort(), DefaultServiceGRPCPort)
	})

	t.Run("test default is returned if spec listener is nil", func(subT *testing.T) {
		l := Limitador{Spec: LimitadorSpec{Listener: &Listener{}}}
		assert.Equal(subT, l.GRPCPort(), DefaultServiceGRPCPort)
	})

	t.Run("test default is returned if spec GRPC Port is nil", func(subT *testing.T) {
		l := Limitador{Spec: LimitadorSpec{Listener: &Listener{GRPC: &TransportProtocol{}}}}
		assert.Equal(subT, l.GRPCPort(), DefaultServiceGRPCPort)
	})

	t.Run("test value in spec is returned when specified", func(subT *testing.T) {
		var port = int32(8080)
		l := Limitador{Spec: LimitadorSpec{Listener: &Listener{GRPC: &TransportProtocol{Port: &port}}}}
		assert.Equal(subT, l.GRPCPort(), port)
	})
}

func TestLimitadorHTTPPort(t *testing.T) {
	t.Run("test default is returned if spec listener is nil", func(subT *testing.T) {
		l := Limitador{}
		assert.Equal(subT, l.HTTPPort(), DefaultServiceHTTPPort)
	})

	t.Run("test default is returned if spec HTTP is nil", func(subT *testing.T) {
		l := Limitador{Spec: LimitadorSpec{Listener: &Listener{}}}
		assert.Equal(subT, l.HTTPPort(), DefaultServiceHTTPPort)
	})

	t.Run("test default is returned if spec HTTP Port is nil", func(subT *testing.T) {
		l := Limitador{Spec: LimitadorSpec{Listener: &Listener{HTTP: &TransportProtocol{}}}}
		assert.Equal(subT, l.HTTPPort(), DefaultServiceHTTPPort)
	})

	t.Run("test value in spec is returned when specified", func(subT *testing.T) {
		var port = int32(8080)
		l := Limitador{Spec: LimitadorSpec{Listener: &Listener{HTTP: &TransportProtocol{Port: &port}}}}
		assert.Equal(subT, l.HTTPPort(), port)
	})
}

func TestLimitadorLimits(t *testing.T) {
	t.Run("test default is returned if limits in spec is nil", func(subT *testing.T) {
		l := Limitador{}
		assert.DeepEqual(subT, l.Limits(), make([]RateLimit, 0))
	})

	t.Run("test value in spec is returned if specified", func(subT *testing.T) {
		limits := []RateLimit{{Conditions: []string{"test"}}}
		l := Limitador{Spec: LimitadorSpec{Limits: limits}}
		assert.DeepEqual(subT, l.Limits(), limits)
	})
}

func TestLimitadorStatusEquals(t *testing.T) {
	var (
		conditions = []metav1.Condition{
			{
				Type: StatusConditionReady,
			},
		}
		service = &LimitadorService{
			Host:  "test",
			Ports: Ports{},
		}
		status = &LimitadorStatus{
			ObservedGeneration: 0,
			Conditions:         conditions,
			Service:            service,
		}
	)

	t.Run("test false if observed generation are different", func(subT *testing.T) {
		l := LimitadorStatus{ObservedGeneration: int64(1)}
		assert.Equal(subT, l.Equals(status, logr.Logger{}), false)
	})

	t.Run("test false if condition are different", func(subT *testing.T) {
		l := LimitadorStatus{ObservedGeneration: status.ObservedGeneration}
		assert.Equal(subT, l.Equals(status, logr.Logger{}), false)
	})

	t.Run("test false if service are different", func(subT *testing.T) {
		l := LimitadorStatus{ObservedGeneration: status.ObservedGeneration, Conditions: status.Conditions}
		assert.Equal(subT, l.Equals(status, logr.Logger{}), false)
	})

	t.Run("test true if status are the same", func(subT *testing.T) {
		l := LimitadorStatus{ObservedGeneration: status.ObservedGeneration, Conditions: status.Conditions, Service: status.Service}
		assert.Equal(subT, l.Equals(status, logr.Logger{}), true)
	})
}
