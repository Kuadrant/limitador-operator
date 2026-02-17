package limitador

import (
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"github.com/kuadrant/limitador-operator/pkg/helpers"
)

var intStrOne = &intstr.IntOrString{
	Type:   0,
	IntVal: 1,
}

func TestConstants(t *testing.T) {
	assert.Check(t, LimitadorRepository == "quay.io/kuadrant/limitador")
	assert.Check(t, StatusEndpoint == "/status")
	assert.Check(t, LimitadorConfigFileName == "limitador-config.yaml")
	assert.Check(t, LimitadorCMMountPath == "/home/limitador/etc")
}

// TODO: Test individual k8s objects.
func newTestLimitadorObj(name, namespace string, limits []limitadorv1alpha1.RateLimit) *limitadorv1alpha1.Limitador {
	var (
		replicas = 1
		version  = "1.0"
		httpPort = int32(8000)
		grpcPort = int32(8001)
	)
	return &limitadorv1alpha1.Limitador{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Limitador",
			APIVersion: "limitador.kuadrant.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: limitadorv1alpha1.LimitadorSpec{
			Replicas: &replicas,
			Version:  &version,
			Listener: &limitadorv1alpha1.Listener{
				HTTP: &limitadorv1alpha1.TransportProtocol{Port: &httpPort},
				GRPC: &limitadorv1alpha1.TransportProtocol{Port: &grpcPort},
			},
			Limits: limits,
			PodDisruptionBudget: &limitadorv1alpha1.PodDisruptionBudgetType{
				MaxUnavailable: intStrOne,
			},
		},
	}
}

func TestServiceName(t *testing.T) {
	name := ServiceName(newTestLimitadorObj("my-limitador-instance", "default", nil))
	assert.Equal(t, name, "limitador-my-limitador-instance")
}

func TestDeployment(t *testing.T) {
	t.Run("default replicas", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		limObj.Spec.Replicas = nil
		deployment := Deployment(limObj, DeploymentOptions{})
		assert.Assert(subT, deployment.Spec.Replicas != nil)
		assert.Assert(subT, *deployment.Spec.Replicas == 1)
	})

	t.Run("replicas", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		limObj.Spec.Replicas = ptr.To(2)
		deployment := Deployment(limObj, DeploymentOptions{})
		assert.Assert(subT, deployment.Spec.Replicas != nil)
		assert.Assert(subT, *deployment.Spec.Replicas == 2)
	})

	t.Run("labels", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		deployment := Deployment(limObj, DeploymentOptions{})
		assert.DeepEqual(subT, deployment.Labels,
			map[string]string{
				"app":                          "limitador",
				"limitador-resource":           "some-name",
				"app.kubernetes.io/name":       helpers.LimitadorAppName,
				"app.kubernetes.io/instance":   "some-name",
				"app.kubernetes.io/component":  helpers.LimitadorAppName,
				"app.kubernetes.io/managed-by": "limitador-operator",
				"app.kubernetes.io/part-of":    "kuadrant",
			})
		assert.DeepEqual(subT, deployment.Spec.Template.Labels,
			map[string]string{
				"app":                          "limitador",
				"limitador-resource":           "some-name",
				"app.kubernetes.io/name":       helpers.LimitadorAppName,
				"app.kubernetes.io/instance":   "some-name",
				"app.kubernetes.io/component":  helpers.LimitadorAppName,
				"app.kubernetes.io/managed-by": "limitador-operator",
				"app.kubernetes.io/part-of":    "kuadrant",
			})
		assert.DeepEqual(subT, deployment.Spec.Selector.MatchLabels,
			map[string]string{
				"app":                "limitador",
				"limitador-resource": "some-name",
			})
	})

	t.Run("args", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		deployment := Deployment(limObj, DeploymentOptions{
			Args: []string{"a", "b", "c"},
		})
		assert.Assert(subT, len(deployment.Spec.Template.Spec.Containers) == 1)
		assert.DeepEqual(subT, deployment.Spec.Template.Spec.Containers[0].Args,
			[]string{"a", "b", "c"},
		)
	})

	t.Run("volumeMounts", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		deployment := Deployment(limObj, DeploymentOptions{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "A",
					MountPath: "/path/A",
				},
				{
					Name:      "B",
					MountPath: "/path/B",
				},
			},
		})
		assert.Assert(subT, len(deployment.Spec.Template.Spec.Containers) == 1)
		assert.DeepEqual(subT, deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
			[]corev1.VolumeMount{
				{
					Name:      "A",
					MountPath: "/path/A",
				},
				{
					Name:      "B",
					MountPath: "/path/B",
				},
			},
		)
	})

	t.Run("volumes", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		deployment := Deployment(limObj, DeploymentOptions{
			Volumes: []corev1.Volume{
				{
					Name: "A",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "secretA",
							},
						},
					},
				},
				{
					Name: "B",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "secretB",
							},
						},
					},
				},
			},
		})
		assert.DeepEqual(subT, deployment.Spec.Template.Spec.Volumes,
			[]corev1.Volume{
				{
					Name: "A",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "secretA",
							},
						},
					},
				},
				{
					Name: "B",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "secretB",
							},
						},
					},
				},
			},
		)
	})

	t.Run("imagePullSecrets", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		deployment := Deployment(limObj, DeploymentOptions{
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "regcred"}},
		})

		assert.DeepEqual(subT, deployment.Spec.Template.Spec.ImagePullSecrets,
			[]corev1.LocalObjectReference{{Name: "regcred"}},
		)
	})
}

func TestPodDisruptionBudgetName(t *testing.T) {
	name := PodDisruptionBudgetName(newTestLimitadorObj("my-limitador-instance", "default", nil))
	assert.Equal(t, name, "limitador-my-limitador-instance")
}

func TestPodDisruptionBudget(t *testing.T) {
	limitadorObj := newTestLimitadorObj("my-limitador-instance", "default", nil)
	pdb := PodDisruptionBudget(limitadorObj)
	assert.DeepEqual(t, pdb.Spec.MaxUnavailable, intStrOne)
	assert.DeepEqual(t, pdb.Spec.Selector.MatchLabels, SelectorLabels(limitadorObj))
}

func TestLimitsConfigMap(t *testing.T) {
	t.Run("config map name", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		configMap, err := LimitsConfigMap(limObj)
		assert.NilError(subT, err)
		assert.Assert(subT, configMap != nil)
		assert.Assert(subT, configMap.Name == LimitsConfigMapName(limObj))
	})

	t.Run("config map namespace", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		configMap, err := LimitsConfigMap(limObj)
		assert.NilError(subT, err)
		assert.Assert(subT, configMap != nil)
		assert.Assert(subT, configMap.Namespace == "some-ns")
	})

	t.Run("config map labels", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		configMap, err := LimitsConfigMap(limObj)
		assert.NilError(subT, err)
		assert.Assert(subT, configMap != nil)
		assert.DeepEqual(subT, configMap.Labels,
			map[string]string{
				"app":                          "limitador",
				"limitador-resource":           "some-name",
				"app.kubernetes.io/name":       helpers.LimitadorAppName,
				"app.kubernetes.io/instance":   "some-name",
				"app.kubernetes.io/component":  helpers.LimitadorAppName,
				"app.kubernetes.io/managed-by": "limitador-operator",
				"app.kubernetes.io/part-of":    "kuadrant",
			})
	})

	t.Run("config map limits", func(subT *testing.T) {
		limits := []limitadorv1alpha1.RateLimit{
			{
				Conditions: []string{"cond == '1'"},
				Variables:  []string{"var1", "var2"},
				MaxValue:   1000,
				Namespace:  "my-ns",
				Seconds:    60,
				Name:       "useless",
			},
			{
				Conditions: []string{"cond == '1'"},
				Variables:  []string{"var1", "var2"},
				MaxValue:   100000,
				Namespace:  "my-ns",
				Seconds:    3600,
			},
		}

		limObj := newTestLimitadorObj("some-name", "some-ns", limits)
		configMap, err := LimitsConfigMap(limObj)
		assert.NilError(subT, err)
		assert.Assert(subT, configMap != nil)
		serializedLimts, ok := configMap.Data[LimitadorConfigFileName]
		assert.Assert(subT, ok)

		// Compare unmarshalled structs to avoid serialization issues
		var limitsUnMarshalled []limitadorv1alpha1.RateLimit
		unmarshallErr := yaml.Unmarshal([]byte(serializedLimts), &limitsUnMarshalled)
		assert.NilError(subT, unmarshallErr)
		assert.DeepEqual(subT, limits, limitsUnMarshalled)
	})

	t.Run("config map nil limits", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		configMap, err := LimitsConfigMap(limObj)
		assert.NilError(subT, err)
		assert.Assert(subT, configMap != nil)

		// when limits are nil, limitadorObj.Limits() returns empty slice
		// Thus, the expected value is "limitador-config.yaml": "[]\n"
		serializedLimts, ok := configMap.Data[LimitadorConfigFileName]
		assert.Assert(subT, ok)

		// Compare unmarshalled structs to avoid serialization issues
		var limitsUnMarshalled []limitadorv1alpha1.RateLimit
		unmarshallErr := yaml.Unmarshal([]byte(serializedLimts), &limitsUnMarshalled)
		assert.NilError(subT, unmarshallErr)
		assert.DeepEqual(subT, make([]limitadorv1alpha1.RateLimit, 0), limitsUnMarshalled)
	})

	t.Run("config map empty limits", func(subT *testing.T) {
		limits := make([]limitadorv1alpha1.RateLimit, 0)
		limObj := newTestLimitadorObj("some-name", "some-ns", limits)
		configMap, err := LimitsConfigMap(limObj)
		assert.NilError(subT, err)
		assert.Assert(subT, configMap != nil)
		serializedLimts, ok := configMap.Data[LimitadorConfigFileName]
		assert.Assert(subT, ok)

		// Compare unmarshalled structs to avoid serialization issues
		var limitsUnMarshalled []limitadorv1alpha1.RateLimit
		unmarshallErr := yaml.Unmarshal([]byte(serializedLimts), &limitsUnMarshalled)
		assert.NilError(subT, unmarshallErr)
		assert.DeepEqual(subT, limits, limitsUnMarshalled)
	})
}

func newDiskStorageLimitador(name string) *limitadorv1alpha1.Limitador {
	limObj := newTestLimitadorObj(name, "some-ns", nil)
	limObj.Spec.Storage = &limitadorv1alpha1.Storage{
		Disk: &limitadorv1alpha1.DiskSpec{},
	}
	return limObj
}

func TestPVC(t *testing.T) {
	t.Run("limitador object with storage other than disk returns PVC to be deleted", func(subT *testing.T) {
		limObj := newTestLimitadorObj("some-name", "some-ns", nil)
		limObj.Spec.Storage = nil
		pvc := PVC(limObj)
		assert.Assert(subT, helpers.IsObjectTaggedToDelete(pvc))

		limObj = newTestLimitadorObj("some-name", "some-ns", nil)
		limObj.Spec.Storage = &limitadorv1alpha1.Storage{Disk: nil}
		pvc = PVC(limObj)
		assert.Assert(subT, helpers.IsObjectTaggedToDelete(pvc))
	})

	t.Run("labels", func(subT *testing.T) {
		limObj := newDiskStorageLimitador("this-is-resource-name")
		pvc := PVC(limObj)
		assert.DeepEqual(subT, pvc.Labels,
			map[string]string{
				"app":                          "limitador",
				"limitador-resource":           "this-is-resource-name",
				"app.kubernetes.io/name":       helpers.LimitadorAppName,
				"app.kubernetes.io/instance":   "this-is-resource-name",
				"app.kubernetes.io/component":  helpers.LimitadorAppName,
				"app.kubernetes.io/managed-by": "limitador-operator",
				"app.kubernetes.io/part-of":    "kuadrant",
			})
	})

	t.Run("RWO access mode", func(subT *testing.T) {
		limObj := newDiskStorageLimitador("some-name")
		pvc := PVC(limObj)
		assert.DeepEqual(subT, pvc.Spec.AccessModes,
			[]corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce})
	})

	t.Run("default resources", func(subT *testing.T) {
		limObj := newDiskStorageLimitador("some-name")
		pvc := PVC(limObj)
		assert.DeepEqual(subT, pvc.Spec.Resources.Requests,
			corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
		)
	})

	t.Run("custom resources", func(subT *testing.T) {
		limObj := newDiskStorageLimitador("some-name")
		limObj.Spec.Storage.Disk.PVC = &limitadorv1alpha1.PVCGenericSpec{
			Resources: &limitadorv1alpha1.PersistentVolumeClaimResources{
				Requests: resource.MustParse("100Gi"),
			},
		}
		pvc := PVC(limObj)
		assert.DeepEqual(subT, pvc.Spec.Resources.Requests,
			corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("100Gi")},
		)
	})

	t.Run("default storage class", func(subT *testing.T) {
		limObj := newDiskStorageLimitador("some-name")
		pvc := PVC(limObj)
		assert.Assert(subT, pvc.Spec.StorageClassName == nil)
	})

	t.Run("custom storage class", func(subT *testing.T) {
		limObj := newDiskStorageLimitador("some-name")
		limObj.Spec.Storage.Disk.PVC = &limitadorv1alpha1.PVCGenericSpec{
			StorageClassName: ptr.To("myCustomStorage"),
		}
		pvc := PVC(limObj)
		assert.DeepEqual(subT, pvc.Spec.StorageClassName, ptr.To("myCustomStorage"))
	})
}
