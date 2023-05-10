package limitador

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
)

const (
	DefaultReplicas         = 1
	LimitadorRepository     = "quay.io/kuadrant/limitador"
	StatusEndpoint          = "/status"
	LimitadorConfigFileName = "limitador-config.yaml"
	LimitsCMNamePrefix      = "limits-config-"
	LimitadorCMMountPath    = "/home/limitador/etc/"
)

func Service(limitador *limitadorv1alpha1.Limitador) *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName(limitador),
			Namespace: limitador.ObjectMeta.Namespace, // TODO: revisit later. For now assume same.
			Labels:    labels(),
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "http",
					Protocol:   v1.ProtocolTCP,
					Port:       limitador.HTTPPort(),
					TargetPort: intstr.FromString("http"),
				},
				{
					Name:       "grpc",
					Protocol:   v1.ProtocolTCP,
					Port:       limitador.GRPCPort(),
					TargetPort: intstr.FromString("grpc"),
				},
			},
			Selector:  labels(),
			ClusterIP: v1.ClusterIPNone,
			Type:      v1.ServiceTypeClusterIP,
		},
	}
}

func Deployment(limitador *limitadorv1alpha1.Limitador, storageConfigSecret *v1.Secret) *appsv1.Deployment {
	var replicas int32 = DefaultReplicas
	if limitador.Spec.Replicas != nil {
		replicas = int32(*limitador.Spec.Replicas)
	}

	image := GetLimitadorImageVersion()
	if limitador.Spec.Version != nil {
		image = fmt.Sprintf("%s:%s", LimitadorRepository, *limitador.Spec.Version)
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      limitador.ObjectMeta.Name,      // TODO: revisit later. For now assume same.
			Namespace: limitador.ObjectMeta.Namespace, // TODO: revisit later. For now assume same.
			Labels:    labels(),
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
							Name:    "limitador",
							Image:   image,
							Command: deploymentContainerCommand(limitador.Spec.Storage, storageConfigSecret, limitador.Spec.RateLimitHeaders),
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: limitador.HTTPPort(),
									Protocol:      v1.ProtocolTCP,
								},
								{
									Name:          "grpc",
									ContainerPort: limitador.GRPCPort(),
									Protocol:      v1.ProtocolTCP,
								},
							},
							LivenessProbe: &v1.Probe{
								ProbeHandler: v1.ProbeHandler{
									HTTPGet: &v1.HTTPGetAction{
										Path:   StatusEndpoint,
										Port:   intstr.FromInt(int(limitador.HTTPPort())),
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
								ProbeHandler: v1.ProbeHandler{
									HTTPGet: &v1.HTTPGetAction{
										Path:   StatusEndpoint,
										Port:   intstr.FromInt(int(limitador.HTTPPort())),
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
					SecurityContext: &v1.PodSecurityContext{
						SupplementalGroups: []int64{1000},
					},
				},
			},
		},
	}
}

func LimitsConfigMap(limitador *limitadorv1alpha1.Limitador) (*v1.ConfigMap, error) {
	limitsMarshalled, marshallErr := yaml.Marshal(limitador.Limits())
	if marshallErr != nil {
		return nil, marshallErr
	}

	return &v1.ConfigMap{
		Data: map[string]string{
			LimitadorConfigFileName: string(limitsMarshalled),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      LimitsCMNamePrefix + limitador.Name,
			Namespace: limitador.Namespace,
			Labels:    map[string]string{"app": "limitador"},
		},
	}, nil
}

func ServiceName(limitadorObj *limitadorv1alpha1.Limitador) string {
	return fmt.Sprintf("limitador-%s", limitadorObj.Name)
}

func labels() map[string]string {
	return map[string]string{"app": "limitador"}
}

func deploymentContainerCommand(storage *limitadorv1alpha1.Storage, storageConfigSecret *v1.Secret, rateLimitHeaders *limitadorv1alpha1.RateLimitHeadersType) []string {
	command := []string{"limitador-server"}

	// stick to the same default as Limitador
	if rateLimitHeaders != nil {
		command = append(command, "--rate-limit-headers", string(*rateLimitHeaders))
	}

	command = append(command, fmt.Sprintf("%s%s", LimitadorCMMountPath, LimitadorConfigFileName))

	return append(command, storageConfig(storage, storageConfigSecret)...)
}

func storageConfig(storage *limitadorv1alpha1.Storage, storageConfigSecret *v1.Secret) []string {
	if storage == nil {
		return []string{string(limitadorv1alpha1.StorageTypeInMemory)}
	}
	return storage.Config(string(storageConfigSecret.Data["URL"]))
}
