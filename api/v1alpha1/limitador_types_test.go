package v1alpha1

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestLimitador_ResourceRequirements(t *testing.T) {
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

	type fields struct {
		Spec LimitadorSpec
	}
	tests := []struct {
		name   string
		fields fields
		want   *corev1.ResourceRequirements
	}{
		{
			name:   "test default is returned when not specified in spec",
			fields: fields{Spec: LimitadorSpec{}},
			want:   defaultResourceRequirements,
		},
		{
			name:   "test value in spec is returned when value is not nil",
			fields: fields{Spec: LimitadorSpec{ResourceRequirements: resourceRequirements}},
			want:   resourceRequirements,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Limitador{
				Spec: tt.fields.Spec,
			}
			if got := l.ResourceRequirements(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResourceRequirements() = %v, want %v", got, tt.want)
			}
		})
	}
}
