package limitador

import (
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
)

type DeploymentOptions struct {
	Command            []string
	VolumeMounts       []v1.VolumeMount
	Volumes            []v1.Volume
	DeploymentStrategy appsv1.DeploymentStrategy
	EnvVar             []v1.EnvVar
}

type DeploymentStorageOptions struct {
	Command            []string
	VolumeMounts       []v1.VolumeMount
	Volumes            []v1.Volume
	DeploymentStrategy appsv1.DeploymentStrategy
}

const (
	LimitadorConfigFileName = "limitador-config.yaml"
	LimitadorCMMountPath    = "/home/limitador/etc"
	LimitsCMVolumeName      = "config-file"
)

func DeploymentCommand(limObj *limitadorv1alpha1.Limitador, storageOptions DeploymentStorageOptions) []string {
	command := []string{"limitador-server"}

	// stick to the same default as Limitador
	if limObj.Spec.RateLimitHeaders != nil {
		command = append(command, "--rate-limit-headers", string(*limObj.Spec.RateLimitHeaders))
	}

	if limObj.Spec.Telemetry != nil && *limObj.Spec.Telemetry == "exhaustive" {
		command = append(command, "--limit-name-in-labels")
	}

	command = append(command, filepath.Join(LimitadorCMMountPath, LimitadorConfigFileName))
	command = append(command, storageOptions.Command...)

	return command
}

func DeploymentVolumeMounts(storageOptions DeploymentStorageOptions) []v1.VolumeMount {
	volumeMounts := []v1.VolumeMount{
		{
			Name:      LimitsCMVolumeName,
			MountPath: LimitadorCMMountPath,
		},
	}
	volumeMounts = append(volumeMounts, storageOptions.VolumeMounts...)
	return volumeMounts
}

func DeploymentVolumes(limObj *limitadorv1alpha1.Limitador, storageOptions DeploymentStorageOptions) []v1.Volume {
	volumes := []v1.Volume{
		{
			Name: LimitsCMVolumeName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: LimitsConfigMapName(limObj),
					},
				},
			},
		},
	}
	volumes = append(volumes, storageOptions.Volumes...)
	return volumes
}
