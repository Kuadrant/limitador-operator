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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultServiceHTTPPort int32 = 8080
	DefaultServiceGRPCPort int32 = 8081

	// Status conditions
	StatusConditionReady string = "Ready"

	// Status reasons
	StatusReasonInstanceRunning   string = "LimitadorInstanceRunning"
	StatusReasonServiceNotRunning string = "LimitadorInstanceNotRunning"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LimitadorSpec defines the desired state of Limitador
type LimitadorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Replicas *int `json:"replicas,omitempty"`

	// +optional
	Version *string `json:"version,omitempty"`

	// +optional
	Listener *Listener `json:"listener,omitempty"`

	// +optional
	Limits []RateLimit `json:"limits,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Limitador is the Schema for the limitadors API
type Limitador struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

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

//+kubebuilder:object:root=true

// LimitadorList contains a list of Limitador
type LimitadorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Limitador `json:"items"`
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
}

// LimitadorStatus defines the observed state of Limitador
type LimitadorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	Service LimitadorService `json:"service,omitempty"`
}

type LimitadorService struct {
	Host  string `json:"host,omitempty"`
	Ports Ports  `json:"ports,omitempty"`
}

type Ports struct {
	HTTP int32 `json:"http,omitempty"`
	GRPC int32 `json:"grpc,omitempty"`
}

func (s *LimitadorStatus) Ready() bool {
	for _, condition := range s.Conditions {
		if condition.Type == StatusConditionReady {
			return condition.Status == metav1.ConditionTrue
		}
	}
	return false
}

func init() {
	SchemeBuilder.Register(&Limitador{}, &LimitadorList{})
}
