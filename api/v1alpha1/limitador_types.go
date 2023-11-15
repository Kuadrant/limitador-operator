/*
Copyright 2020 Red Hat.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"reflect"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kuadrant/limitador-operator/pkg/helpers"
)

const (
	DefaultServiceHTTPPort int32 = 8080
	DefaultServiceGRPCPort int32 = 8081

	// Status conditions
	StatusConditionReady string = "Ready"
)

var (
	defaultResourceRequirements = &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		},
	}
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LimitadorSpec defines the desired state of Limitador
type LimitadorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// +optional
	Replicas *int `json:"replicas,omitempty"`

	// +optional
	Version *string `json:"version,omitempty"`

	// +optional
	Listener *Listener `json:"listener,omitempty"`

	// +optional
	Storage *Storage `json:"storage,omitempty"`

	// +optional
	RateLimitHeaders *RateLimitHeadersType `json:"rateLimitHeaders,omitempty"`

	// +optional
	Telemetry *Telemetry `json:"telemetry,omitempty"`

	// +optional
	Limits []RateLimit `json:"limits,omitempty"`

	// +optional
	PodDisruptionBudget *PodDisruptionBudgetType `json:"pdb,omitempty"`

	// +optional
	ResourceRequirements *corev1.ResourceRequirements `json:"resourceRequirements,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Limitador is the Schema for the limitadors API
type Limitador struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="(!has(self.storage) || !has(self.storage.disk)) || (!has(self.replicas) || self.replicas < 2)",message="disk storage does not allow multiple replicas"
	Spec   LimitadorSpec   `json:"spec,omitempty"`
	Status LimitadorStatus `json:"status,omitempty"`
}

func (l *Limitador) GRPCPort() int32 {
	if l.Spec.Listener == nil ||
		l.Spec.Listener.GRPC == nil ||
		l.Spec.Listener.GRPC.Port == nil {
		return DefaultServiceGRPCPort
	}

	return *l.Spec.Listener.GRPC.Port
}

func (l *Limitador) HTTPPort() int32 {
	if l.Spec.Listener == nil ||
		l.Spec.Listener.HTTP == nil ||
		l.Spec.Listener.HTTP.Port == nil {
		return DefaultServiceHTTPPort
	}

	return *l.Spec.Listener.HTTP.Port
}

func (l *Limitador) Limits() []RateLimit {
	if l.Spec.Limits == nil {
		return make([]RateLimit, 0)
	}

	return l.Spec.Limits
}

func (l *Limitador) GetResourceRequirements() *corev1.ResourceRequirements {
	if l.Spec.ResourceRequirements == nil {
		return defaultResourceRequirements
	}

	return l.Spec.ResourceRequirements
}

//+kubebuilder:object:root=true

// LimitadorList contains a list of Limitador
type LimitadorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Limitador `json:"items"`
}

// RateLimitHeadersType defines the valid options for the --rate-limit-headers arg
// +kubebuilder:validation:Enum=NONE;DRAFT_VERSION_03
type RateLimitHeadersType string

const (
	RateLimitHeadersTypeNONE    RateLimitHeadersType = "NONE"
	RateLimitHeadersTypeDraft03 RateLimitHeadersType = "DRAFT_VERSION_03"
)

// Telemetry defines the level of metrics Limitador will expose to the user
// +kubebuilder:validation:Enum=basic;exhaustive
type Telemetry string

const (
	TelemetryBasic      Telemetry = "basic"
	TelemetryExhaustive Telemetry = "exhaustive"
)

// Storage contains the options for Limitador counters database or in-memory data storage
type Storage struct {
	// +optional
	Redis *Redis `json:"redis,omitempty"`

	// +optional
	RedisCached *RedisCached `json:"redis-cached,omitempty"`

	// +optional
	Disk *DiskSpec `json:"disk,omitempty"`
}

type Redis struct {
	// +ConfigSecretRef refers to the secret holding the URL for Redis.
	// +optional
	ConfigSecretRef *corev1.LocalObjectReference `json:"configSecretRef,omitempty"`
}

type RedisCachedOptions struct {
	// +optional
	// TTL for cached counters in milliseconds [default: 5000]
	TTL *int `json:"ttl,omitempty"`

	// +optional
	// Ratio to apply to the TTL from Redis on cached counters [default: 10]
	Ratio *int `json:"ratio,omitempty"`

	// +optional
	// FlushPeriod for counters in milliseconds [default: 1000]
	FlushPeriod *int `json:"flush-period,omitempty"`

	// +optional
	// MaxCached refers to the maximum amount of counters cached [default: 10000]
	MaxCached *int `json:"max-cached,omitempty"`
}

type RedisCached struct {
	// +ConfigSecretRef refers to the secret holding the URL for Redis.
	// +optional
	ConfigSecretRef *corev1.LocalObjectReference `json:"configSecretRef,omitempty"`

	// +optional
	Options *RedisCachedOptions `json:"options,omitempty"`
}

// PersistentVolumeClaimResources defines the resources configuration
// of the backup data destination PersistentVolumeClaim
type PersistentVolumeClaimResources struct {
	// Storage Resource requests to be used on the PersistentVolumeClaim.
	// To learn more about resource requests see:
	// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	Requests resource.Quantity `json:"requests"` // Should this be a string or a resoure.Quantity? it seems it is serialized as a string
}

type PVCGenericSpec struct {
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// Resources represents the minimum resources the volume should have.
	// Ignored when VolumeName field is set
	// +optional
	Resources *PersistentVolumeClaimResources `json:"resources,omitempty"`
	// VolumeName is the binding reference to the PersistentVolume backing this claim.
	// +optional
	VolumeName *string `json:"volumeName,omitempty"`
}

// DiskOptimizeType defines the valid options for "optimize" option of the disk persistence type
// +kubebuilder:validation:Enum=throughput;disk
type DiskOptimizeType string

const (
	DiskOptimizeTypeThroughput DiskOptimizeType = "throughput"
	DiskOptimizeTypeDisk       DiskOptimizeType = "disk"
)

type DiskSpec struct {
	// +optional
	PVC *PVCGenericSpec `json:"persistentVolumeClaim,omitempty"`

	// +optional
	Optimize *DiskOptimizeType `json:"optimize,omitempty"`
}

type Listener struct {
	// +optional
	HTTP *TransportProtocol `json:"http,omitempty"`
	// +optional
	GRPC *TransportProtocol `json:"grpc,omitempty"`
}

type TransportProtocol struct {
	// +optional
	Port *int32 `json:"port,omitempty"`
	// We could describe TLS within this type
}

// RateLimit defines the desired Limitador limit
type RateLimit struct {
	Conditions []string `json:"conditions"`
	MaxValue   int      `json:"max_value"`
	Namespace  string   `json:"namespace"`
	Seconds    int      `json:"seconds"`
	Variables  []string `json:"variables"`
	Name       string   `json:"name,omitempty"`
}

// LimitadorStatus defines the observed state of Limitador
type LimitadorStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed spec.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Represents the observations of a foo's current state.
	// Known .status.conditions.type are: "Available"
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Service provides information about the service exposing limitador API
	// +optional
	Service *LimitadorService `json:"service,omitempty"`
}

type LimitadorService struct {
	Host  string `json:"host,omitempty"`
	Ports Ports  `json:"ports,omitempty"`
}

type Ports struct {
	HTTP int32 `json:"http,omitempty"`
	GRPC int32 `json:"grpc,omitempty"`
}

type PodDisruptionBudgetType struct {
	// An eviction is allowed if at most "maxUnavailable" limitador pods
	// are unavailable after the eviction, i.e. even in absence of
	// the evicted pod. For example, one can prevent all voluntary evictions
	// by specifying 0. This is a mutually exclusive setting with "minAvailable".
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
	// An eviction is allowed if at least "minAvailable" limitador pods will
	// still be available after the eviction, i.e. even in the absence of
	// the evicted pod.  So for example you can prevent all voluntary
	// evictions by specifying "100%".
	// +optional
	MinAvailable *intstr.IntOrString `json:"minAvailable,omitempty"`
}

func (s *LimitadorStatus) Equals(other *LimitadorStatus, logger logr.Logger) bool {
	if s.ObservedGeneration != other.ObservedGeneration {
		diff := cmp.Diff(s.ObservedGeneration, other.ObservedGeneration)
		logger.V(1).Info("status observedGeneration not equal", "difference", diff)
		return false
	}

	// Marshalling sorts by condition type
	currentMarshaledJSON, _ := helpers.ConditionMarshal(s.Conditions)
	otherMarshaledJSON, _ := helpers.ConditionMarshal(other.Conditions)
	if string(currentMarshaledJSON) != string(otherMarshaledJSON) {
		diff := cmp.Diff(string(currentMarshaledJSON), string(otherMarshaledJSON))
		logger.V(1).Info("status conditions not equal", "difference", diff)
		return false
	}

	if !reflect.DeepEqual(s.Service, other.Service) {
		diff := cmp.Diff(s.Service, other.Service)
		logger.V(1).Info("status service not equal", "difference", diff)
		return false
	}

	return true
}

func init() {
	SchemeBuilder.Register(&Limitador{}, &LimitadorList{})
}
