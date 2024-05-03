package limitador

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/helpers"
)

const (
	StatusEndpoint = "/status"
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
			Labels:    Labels(limitador),
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
			Selector:  Labels(limitador),
			ClusterIP: v1.ClusterIPNone,
			Type:      v1.ServiceTypeClusterIP,
		},
	}
}

func Deployment(limitador *limitadorv1alpha1.Limitador, deploymentOptions DeploymentOptions) *appsv1.Deployment {
	replicas := limitador.GetReplicas()

	image := GetLimitadorImage()

	// deprecated
	if limitador.Spec.Version != nil {
		image = fmt.Sprintf("%s:%s", LimitadorRepository, *limitador.Spec.Version)
	}

	if limitador.Spec.Image != nil {
		image = *limitador.Spec.Image
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DeploymentName(limitador),
			Namespace: limitador.ObjectMeta.Namespace, // TODO: revisit later. For now assume same.
			Labels:    Labels(limitador),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: deploymentOptions.DeploymentStrategy,
			Selector: &metav1.LabelSelector{
				MatchLabels: Labels(limitador),
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: Labels(limitador),
				},
				Spec: v1.PodSpec{
					Affinity: limitador.Spec.Affinity,
					Containers: []v1.Container{
						{
							Name:    "limitador",
							Image:   image,
							Command: deploymentOptions.Command,
							Env:     deploymentOptions.EnvVar,
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
							Resources:       *limitador.GetResourceRequirements(),
							VolumeMounts:    deploymentOptions.VolumeMounts,
							ImagePullPolicy: v1.PullIfNotPresent,
						},
					},
					Volumes: deploymentOptions.Volumes,
				},
			},
		},
	}
}

func LimitsConfigMap(limitadorObj *limitadorv1alpha1.Limitador) (*v1.ConfigMap, error) {
	limitsMarshalled, marshallErr := yaml.Marshal(limitadorObj.Limits())
	if marshallErr != nil {
		return nil, marshallErr
	}

	return &v1.ConfigMap{
		Data: map[string]string{
			LimitadorConfigFileName: string(limitsMarshalled),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      LimitsConfigMapName(limitadorObj),
			Namespace: limitadorObj.Namespace,
			Labels:    Labels(limitadorObj),
		},
	}, nil
}

func LimitsConfigMapName(limitadorObj *limitadorv1alpha1.Limitador) string {
	return fmt.Sprintf("limitador-limits-config-%s", limitadorObj.Name)
}

func ServiceName(limitadorObj *limitadorv1alpha1.Limitador) string {
	return fmt.Sprintf("limitador-%s", limitadorObj.Name)
}

func PVCName(limitadorObj *limitadorv1alpha1.Limitador) string {
	return fmt.Sprintf("limitador-%s", limitadorObj.Name)
}

func DeploymentName(limitadorObj *limitadorv1alpha1.Limitador) string {
	return fmt.Sprintf("limitador-%s", limitadorObj.Name)
}

func PodDisruptionBudget(limitadorObj *limitadorv1alpha1.Limitador) *policyv1.PodDisruptionBudget {
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PodDisruptionBudgetName(limitadorObj),
			Namespace: limitadorObj.ObjectMeta.Namespace,
			Labels:    Labels(limitadorObj),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: Labels(limitadorObj),
			},
		},
	}

	if limitadorObj.Spec.PodDisruptionBudget == nil {
		helpers.TagObjectToDelete(pdb)
		return pdb
	}

	pdb.Spec.MaxUnavailable = limitadorObj.Spec.PodDisruptionBudget.MaxUnavailable
	pdb.Spec.MinAvailable = limitadorObj.Spec.PodDisruptionBudget.MinAvailable

	return pdb
}

func PodDisruptionBudgetName(limitadorObj *limitadorv1alpha1.Limitador) string {
	return fmt.Sprintf("limitador-%s", limitadorObj.Name)
}

func ValidatePDB(pdb *policyv1.PodDisruptionBudget) error {
	if pdb.Spec.MaxUnavailable != nil && pdb.Spec.MinAvailable != nil {
		return fmt.Errorf("pdb spec invalid, maxunavailable and minavailable are mutually exclusive")
	}
	return nil
}

func Labels(limitador *limitadorv1alpha1.Limitador) map[string]string {
	return map[string]string{
		"app":                "limitador",
		"limitador-resource": limitador.ObjectMeta.Name,
	}
}

func PVC(limitador *limitadorv1alpha1.Limitador) *v1.PersistentVolumeClaim {
	pvc := &v1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      PVCName(limitador),
			Namespace: limitador.ObjectMeta.Namespace,
			Labels:    Labels(limitador),
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					// Default value for resources
					v1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	if limitador.Spec.Storage == nil || limitador.Spec.Storage.Disk == nil {
		helpers.TagObjectToDelete(pvc)
		return pvc
	}

	if limitador.Spec.Storage.Disk.PVC != nil {
		pvc.Spec.StorageClassName = limitador.Spec.Storage.Disk.PVC.StorageClassName
		if limitador.Spec.Storage.Disk.PVC.VolumeName != nil {
			pvc.Spec.VolumeName = *limitador.Spec.Storage.Disk.PVC.VolumeName
		}

		// Default value for resources
		if limitador.Spec.Storage.Disk.PVC.Resources != nil {
			pvc.Spec.Resources = v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: limitador.Spec.Storage.Disk.PVC.Resources.Requests,
				},
			}
		}
	}

	return pvc
}
