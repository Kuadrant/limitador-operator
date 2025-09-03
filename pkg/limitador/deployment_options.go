package limitador

import (
	"fmt"
	"k8s.io/utils/env"
	"path/filepath"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
)

type DeploymentOptions struct {
	Command            []string
	VolumeMounts       []corev1.VolumeMount
	Volumes            []corev1.Volume
	DeploymentStrategy appsv1.DeploymentStrategy
	EnvVar             []corev1.EnvVar
	ImagePullSecrets   []corev1.LocalObjectReference
}

type DeploymentStorageOptions struct {
	Command            []string
	VolumeMounts       []corev1.VolumeMount
	Volumes            []corev1.Volume
	DeploymentStrategy appsv1.DeploymentStrategy
}

const (
	LimitadorConfigFileName            = "limitador-config.yaml"
	LimitadorCMMountPath               = "/home/limitador/etc"
	LimitsCMVolumeName                 = "config-file"
	MetricsLabelDefaultEnvName         = "LIMITADOR_METRIC_LABELS_DEFAULT"
	MetricsLabelDefaultEnvDefaultValue = "descriptors[1]"
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

	if limObj.Spec.Tracing != nil {
		command = append(command, "--tracing-endpoint", limObj.Spec.Tracing.Endpoint)
	}

	if limObj.Spec.Verbosity != nil {
		command = append(command, fmt.Sprintf("-%s", strings.Repeat("v", int(*limObj.Spec.Verbosity))))
	}

	// let's set explicitly the HTTP port,
	// as it is being set in the readiness and liveness probe and in the service
	command = append(command, "--http-port", strconv.Itoa(int(limObj.HTTPPort())))

	// let's set explicitly the GRPC port,
	// as it is being set in the service
	command = append(command, "--rls-port", strconv.Itoa(int(limObj.GRPCPort())))

	// sets the metrics-label-default
	command = append(command, "--metric-labels-default", env.GetString(MetricsLabelDefaultEnvName, MetricsLabelDefaultEnvDefaultValue))

	command = append(command, filepath.Join(LimitadorCMMountPath, LimitadorConfigFileName))
	command = append(command, storageOptions.Command...)

	return command
}

func DeploymentVolumeMounts(storageOptions DeploymentStorageOptions) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      LimitsCMVolumeName,
			MountPath: LimitadorCMMountPath,
		},
	}
	volumeMounts = append(volumeMounts, storageOptions.VolumeMounts...)
	return volumeMounts
}

func DeploymentVolumes(limObj *limitadorv1alpha1.Limitador, storageOptions DeploymentStorageOptions) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: LimitsCMVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: LimitsConfigMapName(limObj),
					},
				},
			},
		},
	}
	volumes = append(volumes, storageOptions.Volumes...)
	return volumes
}
