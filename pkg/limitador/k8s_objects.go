package limitador

import (
	"crypto/md5"
	"fmt"
	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/helpers"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

const (
	DefaultVersion             = "latest"
	DefaultReplicas            = 1
	Image                      = "quay.io/3scale/limitador"
	StatusEndpoint             = "/status"
	DefaultServiceHTTPPort     = 8080
	DefaultServiceGRPCPort     = 8081
	EnvLimitadorConfigFileName = "LIMITADOR_CONFIG_FILE_NAME"
	LimitadorCMHash            = "hash"
	LimitsCMNamePrefix         = "limits-config-"
	LimitadorCMMountPath       = "/"
	LimitadorLimitsFileEnv     = "LIMITS_FILE"
)

var (
	LimitadorConfigFileName = helpers.FetchEnv(EnvLimitadorConfigFileName, "limitador-config.yaml")
)

func LimitadorService(limitador *limitadorv1alpha1.Limitador) *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            limitador.Name,
			Namespace:       limitador.ObjectMeta.Namespace, // TODO: revisit later. For now assume same.
			Labels:          labels(),
			OwnerReferences: []metav1.OwnerReference{ownerRefToLimitador(limitador)},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "http",
					Protocol:   v1.ProtocolTCP,
					Port:       helpers.GetValueOrDefault(*limitador.Spec.Listener.HTTP.Port, DefaultServiceHTTPPort).(int32),
					TargetPort: intstr.FromString("http"),
				},
				{
					Name:       "grpc",
					Protocol:   v1.ProtocolTCP,
					Port:       helpers.GetValueOrDefault(*limitador.Spec.Listener.GRPC.Port, DefaultServiceGRPCPort).(int32),
					TargetPort: intstr.FromString("grpc"),
				},
			},
			Selector:  labels(),
			ClusterIP: v1.ClusterIPNone,
			Type:      v1.ServiceTypeClusterIP,
		},
	}
}

func LimitadorDeployment(limitador *limitadorv1alpha1.Limitador) *appsv1.Deployment {
	var replicas int32 = DefaultReplicas
	if limitador.Spec.Replicas != nil {
		replicas = int32(*limitador.Spec.Replicas)
	}

	version := DefaultVersion
	if limitador.Spec.Version != nil {
		version = *limitador.Spec.Version
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            limitador.ObjectMeta.Name,      // TODO: revisit later. For now assume same.
			Namespace:       limitador.ObjectMeta.Namespace, // TODO: revisit later. For now assume same.
			Labels:          labels(),
			OwnerReferences: []metav1.OwnerReference{ownerRefToLimitador(limitador)},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels(),
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels(),
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "limitador",
							Image: Image + ":" + version,
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: helpers.GetValueOrDefault(*limitador.Spec.Listener.HTTP.Port, DefaultServiceHTTPPort).(int32),
									Protocol:      v1.ProtocolTCP,
								},
								{
									Name:          "grpc",
									ContainerPort: helpers.GetValueOrDefault(*limitador.Spec.Listener.GRPC.Port, DefaultServiceGRPCPort).(int32),
									Protocol:      v1.ProtocolTCP,
								},
							},
							Env: []v1.EnvVar{
								{
									Name:  "RUST_LOG",
									Value: "info",
								},
								{
									Name:  LimitadorLimitsFileEnv,
									Value: LimitadorCMMountPath + LimitadorConfigFileName,
								},
							},
							LivenessProbe: &v1.Probe{
								Handler: v1.Handler{
									HTTPGet: &v1.HTTPGetAction{
										Path:   StatusEndpoint,
										Port:   intstr.FromInt(int(helpers.GetValueOrDefault(*limitador.Spec.Listener.HTTP.Port, DefaultServiceHTTPPort).(int32))),
										Scheme: v1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      2,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							ReadinessProbe: &v1.Probe{
								Handler: v1.Handler{
									HTTPGet: &v1.HTTPGetAction{
										Path:   StatusEndpoint,
										Port:   intstr.FromInt(int(helpers.GetValueOrDefault(*limitador.Spec.Listener.HTTP.Port, DefaultServiceHTTPPort).(int32))),
										Scheme: v1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "config-file",
									MountPath: LimitadorCMMountPath,
								},
							},
							ImagePullPolicy: v1.PullIfNotPresent,
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "config-file",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: LimitsCMNamePrefix + limitador.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func LimitsConfigMap(limitador *limitadorv1alpha1.Limitador) (*v1.ConfigMap, error) {
	limitsMarshalled, marshallErr := yaml.Marshal(limitador.Spec.Limits)
	if marshallErr != nil {
		return nil, marshallErr
	}

	return &v1.ConfigMap{
		Data: map[string]string{
			LimitadorConfigFileName: string(limitsMarshalled),
			LimitadorCMHash:         fmt.Sprintf("%x", md5.Sum(limitsMarshalled)),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      LimitsCMNamePrefix + limitador.Name,
			Namespace: limitador.Namespace,
			Labels:    map[string]string{"app": "limitador"},
		},
	}, nil
}

func labels() map[string]string {
	return map[string]string{"app": "limitador"}
}

func ownerRefToLimitador(limitador *limitadorv1alpha1.Limitador) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: limitador.APIVersion,
		Kind:       limitador.Kind,
		Name:       limitador.Name,
		UID:        limitador.UID,
	}
}
