package limitador

import (
	"strconv"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
)

func TestDeploymentArgs(t *testing.T) {
	basicLimitador := func() *limitadorv1alpha1.Limitador {
		return &limitadorv1alpha1.Limitador{
			TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
			ObjectMeta: metav1.ObjectMeta{Name: "somename", Namespace: "somenamespace"},
			Spec:       limitadorv1alpha1.LimitadorSpec{},
		}
	}

	t.Run("when default spec", func(subT *testing.T) {
		limObj := basicLimitador()
		args := DeploymentArgs(limObj, DeploymentStorageOptions{Args: []string{"memory"}})
		assert.DeepEqual(subT, args,
			[]string{
				"--http-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
				"--rls-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
				"/home/limitador/etc/limitador-config.yaml",
				"memory",
			})
	})
	t.Run("when metric labels default is set", func(subT *testing.T) {
		limObj := basicLimitador()
		metricLabelsDefault := "descriptors[1][\"metrics-labels\"]"
		limObj.Spec.MetricLabelsDefault = &metricLabelsDefault
		args := DeploymentArgs(limObj, DeploymentStorageOptions{Args: []string{"memory"}})
		assert.DeepEqual(subT, args,
			[]string{
				"--http-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
				"--rls-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
				"--metric-labels-default",
				"descriptors[1][\"metrics-labels\"]",
				"/home/limitador/etc/limitador-config.yaml",
				"memory",
			})
	})

	t.Run("when rate limit headers set in the spec command line args includes --rate-limit-headers", func(subT *testing.T) {
		limObj := basicLimitador()
		limObj.Spec.RateLimitHeaders = ptr.To(limitadorv1alpha1.RateLimitHeadersType("DRAFT_VERSION_03"))

		args := DeploymentArgs(limObj, DeploymentStorageOptions{Args: []string{"memory"}})
		assert.DeepEqual(subT, args,
			[]string{
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
		args := DeploymentArgs(limObj, DeploymentStorageOptions{})
		assert.DeepEqual(subT, args,
			[]string{
				"--http-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
				"--rls-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
				"/home/limitador/etc/limitador-config.yaml",
			})
	})

	t.Run("commands from storage option appended", func(subT *testing.T) {
		limObj := basicLimitador()
		args := DeploymentArgs(limObj, DeploymentStorageOptions{
			Args: []string{"a", "b", "c"},
		})
		assert.DeepEqual(subT, args,
			[]string{
				"--http-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
				"--rls-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
				"/home/limitador/etc/limitador-config.yaml",
				"a", "b", "c",
			})
	})

	t.Run("when verbosity is set in the spec command line args includes -v*", func(subT *testing.T) {
		tests := []struct {
			Name           string
			VerbosityLevel limitadorv1alpha1.VerbosityLevel
			ExpectedArg    string
		}{
			{"log level 1", 1, "-v"},
			{"log level 2", 2, "-vv"},
			{"log level 3", 3, "-vvv"},
			{"log level 4", 4, "-vvvv"},
		}
		for _, tt := range tests {
			subT.Run(tt.Name, func(subTest *testing.T) {
				limObj := basicLimitador()
				limObj.Spec.Verbosity = ptr.To(tt.VerbosityLevel)
				args := DeploymentArgs(limObj, DeploymentStorageOptions{})
				assert.Assert(subTest, is.Contains(args, tt.ExpectedArg))
			})
		}
	})

	t.Run("command from tracing endpoint appended", func(subT *testing.T) {
		testEndpoint := "rpc://tracing-endpoint:4317"
		limObj := basicLimitador()
		limObj.Spec.Tracing = &limitadorv1alpha1.Tracing{
			Endpoint: testEndpoint,
		}
		args := DeploymentArgs(limObj, DeploymentStorageOptions{})
		assert.DeepEqual(subT, args,
			[]string{
				"--tracing-endpoint",
				testEndpoint,
				"--http-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceHTTPPort)),
				"--rls-port",
				strconv.Itoa(int(limitadorv1alpha1.DefaultServiceGRPCPort)),
				"/home/limitador/etc/limitador-config.yaml",
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
