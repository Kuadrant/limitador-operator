package limitador

import (
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
)

func RedisDeploymentOptions(ctx context.Context, cl client.Client, defSecretNamespace string, redisObj limitadorv1alpha1.Redis) (DeploymentStorageOptions, error) {
	if redisObj.ConfigSecretRef == nil {
		return DeploymentStorageOptions{}, errors.New("there's no ConfigSecretRef set")
	}

	err := validateRedisSecret(ctx, cl, defSecretNamespace, *redisObj.ConfigSecretRef)
	if err != nil {
		return DeploymentStorageOptions{}, err
	}

	return DeploymentStorageOptions{
		Command: []string{"redis", "$(LIMITADOR_OPERATOR_REDIS_URL)"},
	}, nil
}

func DeploymentEnvVar(configSecretRef *v1.LocalObjectReference) ([]v1.EnvVar, error) {
	if configSecretRef == nil {
		return nil, errors.New("there's no ConfigSecretRef set")
	}

	env := []v1.EnvVar{
		{
			Name: "LIMITADOR_OPERATOR_REDIS_URL",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					Key: "URL",
					LocalObjectReference: v1.LocalObjectReference{
						Name: configSecretRef.Name,
					},
				},
			},
		},
	}
	return env, nil
}

func validateRedisSecret(ctx context.Context, cl client.Client, defSecretNamespace string, secretRef v1.LocalObjectReference) error {
	secret := &v1.Secret{}
	if err := cl.Get(
		ctx,
		types.NamespacedName{
			Name:      secretRef.Name,
			Namespace: defSecretNamespace,
		},
		secret,
	); err != nil {
		// Must exist, so if it does not, also return err
		return err
	}

	// nil map behaves as empty map when reading
	if _, ok := secret.Data["URL"]; ok {
		return nil
	}

	return errors.New("the storage config Secret doesn't have the `URL` field")
}
