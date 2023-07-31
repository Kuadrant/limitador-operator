package v1alpha1

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestLimitador_GRPCPort(t *testing.T) {
	var port = int32(8080)

	type fields struct {
		Spec LimitadorSpec
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{
			name: "test default is returned if spec listener is nil",
			fields: fields{
				Spec: LimitadorSpec{},
			},
			want: DefaultServiceGRPCPort,
		},
		{
			name: "test default is returned if spec GRPC is nil",
			fields: fields{
				Spec: LimitadorSpec{Listener: &Listener{}},
			},
			want: DefaultServiceGRPCPort,
		},
		{
			name: "test default is returned if spec GRPC Port is nil",
			fields: fields{
				Spec: LimitadorSpec{Listener: &Listener{GRPC: &TransportProtocol{}}},
			},
			want: DefaultServiceGRPCPort,
		},
		{
			name: "test value in spec is returned when specified",
			fields: fields{
				Spec: LimitadorSpec{Listener: &Listener{GRPC: &TransportProtocol{Port: &port}}},
			},
			want: port,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Limitador{
				Spec: tt.fields.Spec,
			}
			if got := l.GRPCPort(); got != tt.want {
				t.Errorf("GRPCPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimitador_HTTPPort(t *testing.T) {
	var port = int32(8080)

	type fields struct {
		Spec LimitadorSpec
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{
			name: "test default is returned if spec listener is nil",
			fields: fields{
				Spec: LimitadorSpec{},
			},
			want: DefaultServiceHTTPPort,
		},
		{
			name: "test default is returned if spec HTTP is nil",
			fields: fields{
				Spec: LimitadorSpec{Listener: &Listener{}},
			},
			want: DefaultServiceHTTPPort,
		},
		{
			name: "test default is returned if spec HTTP Port is nil",
			fields: fields{
				Spec: LimitadorSpec{Listener: &Listener{HTTP: &TransportProtocol{}}},
			},
			want: DefaultServiceHTTPPort,
		},
		{
			name: "test value in spec is returned when specified",
			fields: fields{
				Spec: LimitadorSpec{Listener: &Listener{HTTP: &TransportProtocol{Port: &port}}},
			},
			want: port,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Limitador{
				Spec: tt.fields.Spec,
			}
			if got := l.HTTPPort(); got != tt.want {
				t.Errorf("HTTPPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimitador_Limits(t *testing.T) {
	limits := []RateLimit{
		{
			Conditions: []string{"test"},
		},
	}

	type fields struct {
		Spec LimitadorSpec
	}
	tests := []struct {
		name   string
		fields fields
		want   []RateLimit
	}{
		{
			name:   "test default is returned if limits in spec is nil",
			fields: fields{Spec: LimitadorSpec{}},
			want:   make([]RateLimit, 0),
		},
		{
			name:   "test value in spec is returned if specified",
			fields: fields{Spec: LimitadorSpec{Limits: limits}},
			want:   limits,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Limitador{
				Spec: tt.fields.Spec,
			}
			if got := l.Limits(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Limits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStorage_SecretRef(t *testing.T) {
	var (
		redisSecretRef       = &corev1.ObjectReference{Name: "redis"}
		redisCachedSecretRef = &corev1.ObjectReference{Name: "redisCached"}
	)

	type fields struct {
		Redis       *Redis
		RedisCached *RedisCached
	}
	tests := []struct {
		name   string
		fields fields
		want   *corev1.ObjectReference
	}{
		{
			name:   "test redis secret ref is returned if not nil",
			fields: fields{Redis: &Redis{ConfigSecretRef: redisSecretRef}},
			want:   redisSecretRef,
		},
		{
			name:   "test redis cached ref is returned if redis nil",
			fields: fields{RedisCached: &RedisCached{ConfigSecretRef: redisCachedSecretRef}},
			want:   redisCachedSecretRef,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				Redis:       tt.fields.Redis,
				RedisCached: tt.fields.RedisCached,
			}
			if got := s.SecretRef(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SecretRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStorage_Validate(t *testing.T) {
	type fields struct {
		Redis       *Redis
		RedisCached *RedisCached
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "test false if redis is nil",
			fields: fields{},
			want:   false,
		},
		{
			name:   "test false if redis secret ref is nil",
			fields: fields{Redis: &Redis{}},
			want:   false,
		},
		{
			name:   "test true if redis secret ref is not nil",
			fields: fields{Redis: &Redis{ConfigSecretRef: &corev1.ObjectReference{}}},
			want:   true,
		},
		{
			name:   "test false if redis cached is nil",
			fields: fields{Redis: &Redis{}},
			want:   false,
		},
		{
			name:   "test false if redis cached secret ref is nil",
			fields: fields{RedisCached: &RedisCached{}},
			want:   false,
		},
		{
			name:   "test true if redis secret ref is not nil",
			fields: fields{RedisCached: &RedisCached{ConfigSecretRef: &corev1.ObjectReference{}}},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				Redis:       tt.fields.Redis,
				RedisCached: tt.fields.RedisCached,
			}
			if got := s.Validate(); got != tt.want {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStorage_Config(t *testing.T) {
	const url = "test"
	var option = 4040

	type fields struct {
		Redis       *Redis
		RedisCached *RedisCached
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "test redis storage type returned if redis is not nil",
			fields: fields{Redis: &Redis{}},
			want:   []string{string(StorageTypeRedis), url},
		},
		{
			name:   "test redis cached storage type returned if redis cached is not nil",
			fields: fields{RedisCached: &RedisCached{}},
			want:   []string{string(StorageTypeRedisCached), url},
		},
		{
			name: "test redis cached storage type with options returned",
			fields: fields{RedisCached: &RedisCached{Options: &RedisCachedOptions{
				TTL:         &option,
				Ratio:       &option,
				FlushPeriod: &option,
				MaxCached:   &option,
			}}},
			want: []string{string(StorageTypeRedisCached), url, fmt.Sprintf("--ttl %s", strconv.Itoa(option)),
				fmt.Sprintf("--ratio %s", strconv.Itoa(option)), fmt.Sprintf("--flush-period %s", strconv.Itoa(option)),
				fmt.Sprintf("--max-cached %s", strconv.Itoa(option))},
		},
		{
			name:   "test memory storage type if redis and redis cached is nil",
			fields: fields{},
			want:   []string{string(StorageTypeInMemory)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				Redis:       tt.fields.Redis,
				RedisCached: tt.fields.RedisCached,
			}
			if got := s.Config(url); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Config() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimitadorStatus_Equals(t *testing.T) {
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

	type fields struct {
		ObservedGeneration int64
		Conditions         []metav1.Condition
		Service            *LimitadorService
	}
	type args struct {
		other *LimitadorStatus
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "test false if observed generation are different",
			fields: fields{ObservedGeneration: int64(1)},
			args:   args{other: status},
			want:   false,
		},
		{
			name:   "test false if condition are different",
			fields: fields{ObservedGeneration: status.ObservedGeneration},
			args:   args{other: status},
			want:   false,
		},
		{
			name:   "test false if service are different",
			fields: fields{ObservedGeneration: status.ObservedGeneration, Conditions: status.Conditions},
			args:   args{other: status},
			want:   false,
		},
		{
			name:   "test true if status are the same",
			fields: fields{ObservedGeneration: status.ObservedGeneration, Conditions: status.Conditions, Service: status.Service},
			args:   args{other: status},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LimitadorStatus{
				ObservedGeneration: tt.fields.ObservedGeneration,
				Conditions:         tt.fields.Conditions,
				Service:            tt.fields.Service,
			}
			if got := s.Equals(tt.args.other, logr.Logger{}); got != tt.want {
				t.Errorf("Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}
