package limitador

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
)

const (
	DiskVolumeName = "storage"
	DiskPath       = "/var/lib/limitador/data"
)

func DiskDeploymentOptions(limObj *limitadorv1alpha1.Limitador, diskObj limitadorv1alpha1.DiskSpec) (DeploymentStorageOptions, error) {
	command := []string{"disk"}

	if diskObj.Optimize != nil {
		command = append(command, "--optimize", string(*diskObj.Optimize))
	}

	command = append(command, DiskPath)

	return DeploymentStorageOptions{
		Command:      command,
		VolumeMounts: diskVolumeMounts(),
		Volumes:      diskVolumes(limObj),
		// Disk storage requires killing all existing pods before creating a new one, as the PV canont be shared.
		// Limitador adds a lock to the disk. If the lock is not released before the new pod is created,
		// the new pod will panic and never start. Seen error:
		// thread 'main' panicked at 'called `Result::unwrap()` on an `Err` value: Error { message: "IO error: While lock file: /var/lib/limitador/data/LOCK: Resource temporarily unavailable" }', /usr/src/limitador/limitador/src/storage/disk/rocksdb_storage.rs:155:40
		DeploymentStrategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
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
