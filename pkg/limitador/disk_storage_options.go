package limitador

import (
	"context"

	v1 "k8s.io/api/core/v1"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
)

const (
	DiskVolumeName = "storage"
	DiskPath       = "/var/lib/limitador/data"
)

func DiskDeploymentOptions(ctx context.Context, limObj *limitadorv1alpha1.Limitador, diskObj limitadorv1alpha1.DiskSpec) (DeploymentStorageOptions, error) {
	command := []string{"disk"}

	if diskObj.Optimize != nil {
		command = append(command, "--optimize", string(*diskObj.Optimize))
	}

	command = append(command, DiskPath)

	return DeploymentStorageOptions{
		Command:      command,
		VolumeMounts: diskVolumeMounts(),
		Volumes:      diskVolumes(limObj),
	}, nil
}

func diskVolumeMounts() []v1.VolumeMount {
	return []v1.VolumeMount{
		v1.VolumeMount{
			ReadOnly:  false,
			Name:      DiskVolumeName,
			MountPath: DiskPath,
		},
	}
}

func diskVolumes(limObj *limitadorv1alpha1.Limitador) []v1.Volume {
	return []v1.Volume{
		v1.Volume{
			Name: DiskVolumeName,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: PVCName(limObj),
					ReadOnly:  false,
				},
			},
		},
	}
}
