package limitador

import (
	"strconv"
	"testing"

	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
)

func TestDeploymentCommand(t *testing.T) {
	basicLimitador := func() *limitadorv1alpha1.Limitador {
		return &limitadorv1alpha1.Limitador{
			TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
			ObjectMeta: metav1.ObjectMeta{Name: "somename", Namespace: "somenamespace"},
			Spec:       limitadorv1alpha1.LimitadorSpec{},
		}
	}

	t.Run("when default spec", func(subT *testing.T) {
		limObj := basicLimitador()
		command := DeploymentCommand(limObj, DeploymentStorageOptions{Command: []string{"memory"}})
		assert.DeepEqual(subT, command,
			[]string{
				"limitador-server",
				"--http-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
				"--rls-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
				"/home/limitador/etc/limitador-config.yaml",
				"memory",
			})
	})

	t.Run("when rate limit headers set in the spec command line args includes --rate-limit-headers", func(subT *testing.T) {
		limObj := basicLimitador()
		limObj.Spec.RateLimitHeaders = &[]limitadorv1alpha1.RateLimitHeadersType{"DRAFT_VERSION_03"}[0]

		command := DeploymentCommand(limObj, DeploymentStorageOptions{Command: []string{"memory"}})
		assert.DeepEqual(subT, command,
			[]string{
				"limitador-server",
				"--rate-limit-headers",
				"DRAFT_VERSION_03",
				"--http-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
				"--rls-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
				"/home/limitador/etc/limitador-config.yaml",
				"memory",
			})
	})

	t.Run("hardcoded config file path included", func(subT *testing.T) {
		limObj := basicLimitador()
		command := DeploymentCommand(limObj, DeploymentStorageOptions{})
		assert.DeepEqual(subT, command,
			[]string{
				"limitador-server",
				"--http-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
				"--rls-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
				"/home/limitador/etc/limitador-config.yaml",
			})
	})

	t.Run("commands from storage option appended", func(subT *testing.T) {
		limObj := basicLimitador()
		command := DeploymentCommand(limObj, DeploymentStorageOptions{
			Command: []string{"a", "b", "c"},
		})
		assert.DeepEqual(subT, command,
			[]string{
				"limitador-server",
				"--http-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
				"--rls-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
				"/home/limitador/etc/limitador-config.yaml",
				"a", "b", "c",
			})
	})
}

func TestDeploymentVolumeMounts(t *testing.T) {
	t.Run("limits config volume mount included", func(subT *testing.T) {
		volumeMounts := DeploymentVolumeMounts(DeploymentStorageOptions{})
		assert.DeepEqual(subT, volumeMounts,
			[]v1.VolumeMount{
				{
					Name:      LimitsCMVolumeName,
					MountPath: LimitadorCMMountPath,
				},
			})
	})
	t.Run("storage volume mounts appended", func(subT *testing.T) {
		volumeMounts := DeploymentVolumeMounts(DeploymentStorageOptions{
			VolumeMounts: []v1.VolumeMount{
				{Name: "a", MountPath: "/a"},
				{Name: "b", MountPath: "/b"},
				{Name: "c", MountPath: "/c"},
			},
		})
		assert.DeepEqual(subT, volumeMounts,
			[]v1.VolumeMount{
				{Name: LimitsCMVolumeName, MountPath: LimitadorCMMountPath},
				{Name: "a", MountPath: "/a"},
				{Name: "b", MountPath: "/b"},
				{Name: "c", MountPath: "/c"},
			})
	})
}

func TestDeploymentVolumes(t *testing.T) {
	basicLimitador := func() *limitadorv1alpha1.Limitador {
		return &limitadorv1alpha1.Limitador{
			TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
			ObjectMeta: metav1.ObjectMeta{Name: "somename", Namespace: "somenamespace"},
			Spec:       limitadorv1alpha1.LimitadorSpec{},
		}
	}

	t.Run("limits config volume included", func(subT *testing.T) {
		limObj := basicLimitador()
		volumes := DeploymentVolumes(limObj, DeploymentStorageOptions{})
		assert.DeepEqual(subT, volumes,
			[]v1.Volume{
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
			})
	})

	t.Run("storage volumes appended", func(subT *testing.T) {
		limObj := basicLimitador()
		volumes := DeploymentVolumes(limObj, DeploymentStorageOptions{
			Volumes: []v1.Volume{
				{Name: "a"},
				{Name: "b"},
				{Name: "c"},
			},
		})
		assert.DeepEqual(subT, volumes,
			[]v1.Volume{
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
				{Name: "a"},
				{Name: "b"},
				{Name: "c"},
			})
	})
}
