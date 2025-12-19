package limitador

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
)

type DeploymentOptions struct {
	Args               []string
	VolumeMounts       []corev1.VolumeMount
	Volumes            []corev1.Volume
	DeploymentStrategy appsv1.DeploymentStrategy
	EnvVar             []corev1.EnvVar
	ImagePullSecrets   []corev1.LocalObjectReference
}

type DeploymentStorageOptions struct {
	Args               []string
	VolumeMounts       []corev1.VolumeMount
	Volumes            []corev1.Volume
	DeploymentStrategy appsv1.DeploymentStrategy
}

const (
	LimitadorConfigFileName = "limitador-config.yaml"
	LimitadorCMMountPath    = "/home/limitador/etc"
	LimitsCMVolumeName      = "config-file"
)

func DeploymentArgs(limObj *limitadorv1alpha1.Limitador, storageOptions DeploymentStorageOptions) []string {
	args := []string{}

	// stick to the same default as Limitador
	if limObj.Spec.RateLimitHeaders != nil {
		args = append(args, "--rate-limit-headers", string(*limObj.Spec.RateLimitHeaders))
	}

	if limObj.Spec.Telemetry != nil && *limObj.Spec.Telemetry == "exhaustive" {
		args = append(args, "--limit-name-in-labels")
	}

	if limObj.Spec.Tracing != nil {
		args = append(args, "--tracing-endpoint", limObj.Spec.Tracing.Endpoint)
	}

	if limObj.Spec.Verbosity != nil {
		args = append(args, fmt.Sprintf("-%s", strings.Repeat("v", int(*limObj.Spec.Verbosity))))
	}

	// let's set explicitly the HTTP port,
	// as it is being set in the readiness and liveness probe and in the service
	args = append(args, "--http-port", strconv.Itoa(int(limObj.HTTPPort())))

	// let's set explicitly the GRPC port,
	// as it is being set in the service
	args = append(args, "--rls-port", strconv.Itoa(int(limObj.GRPCPort())))

	// sets the metrics-label-default
	if limObj.Spec.MetricLabelsDefault != nil {
		args = append(args, "--metric-labels-default", *limObj.Spec.MetricLabelsDefault)
	}

	args = append(args, filepath.Join(LimitadorCMMountPath, LimitadorConfigFileName))
	args = append(args, storageOptions.Args...)

	return args
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
