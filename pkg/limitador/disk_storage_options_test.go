package limitador

import (
	"testing"

	"gotest.tools/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
)

func TestDiskDeploymentOptions(t *testing.T) {
	basicLimitador := func() *limitadorv1alpha1.Limitador {
		return &limitadorv1alpha1.Limitador{
			TypeMeta:   metav1.TypeMeta{Kind: "Limitador", APIVersion: "limitador.kuadrant.io/v1alpha1"},
			ObjectMeta: metav1.ObjectMeta{Name: "somename", Namespace: "somenamespace"},
			Spec:       limitadorv1alpha1.LimitadorSpec{},
		}
	}

	t.Run("basic disk deployment options", func(subT *testing.T) {
		limObj := basicLimitador()
		options, err := DiskDeploymentOptions(limObj, limitadorv1alpha1.DiskSpec{})
		assert.NilError(subT, err)
		assert.DeepEqual(subT, options,
			DeploymentStorageOptions{
				Command: []string{"disk", DiskPath},
				VolumeMounts: []v1.VolumeMount{
					v1.VolumeMount{ReadOnly: false, Name: DiskVolumeName, MountPath: DiskPath},
				},
				Volumes: []v1.Volume{
					v1.Volume{
						Name: DiskVolumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								ClaimName: PVCName(limObj),
								ReadOnly:  false,
							},
						},
					},
				},
				DeploymentStrategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			})
	})

	t.Run("disk optimize option", func(subT *testing.T) {
		limObj := basicLimitador()
		options, err := DiskDeploymentOptions(
			limObj,
			limitadorv1alpha1.DiskSpec{Optimize: &[]limitadorv1alpha1.DiskOptimizeType{limitadorv1alpha1.DiskOptimizeTypeDisk}[0]},
		)
		assert.NilError(subT, err)
		assert.DeepEqual(subT, options,
			DeploymentStorageOptions{
				Command: []string{"disk", "--optimize", string(limitadorv1alpha1.DiskOptimizeTypeDisk), DiskPath},
				VolumeMounts: []v1.VolumeMount{
					v1.VolumeMount{ReadOnly: false, Name: DiskVolumeName, MountPath: DiskPath},
				},
				Volumes: []v1.Volume{
					v1.Volume{
						Name: DiskVolumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								ClaimName: PVCName(limObj),
								ReadOnly:  false,
							},
						},
					},
				},
				DeploymentStrategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			})
	})
}
