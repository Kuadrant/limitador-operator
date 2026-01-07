package limitador

import (
	"context"
	"errors"
	"strconv"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RedisCachedDeploymentOptions(ctx context.Context, cl client.Client, defSecretNamespace string, redisCachedObj limitadorv1alpha1.RedisCached) (DeploymentStorageOptions, error) {
	if redisCachedObj.ConfigSecretRef == nil {
		return DeploymentStorageOptions{}, errors.New("there's no ConfigSecretRef set")
	}

	err := validateRedisSecret(ctx, cl, defSecretNamespace, *redisCachedObj.ConfigSecretRef)
	if err != nil {
		return DeploymentStorageOptions{}, err
	}

	command := []string{"redis_cached", "$(LIMITADOR_OPERATOR_REDIS_URL)"}
	if redisCachedObj.Options != nil {
		if redisCachedObj.Options.FlushPeriod != nil {
			command = append(command, "--flush-period", strconv.Itoa(*redisCachedObj.Options.FlushPeriod))
		}
		if redisCachedObj.Options.MaxCached != nil {
			command = append(command, "--max-cached", strconv.Itoa(*redisCachedObj.Options.MaxCached))
		}
		if redisCachedObj.Options.ResponseTimeout != nil {
			command = append(command, "--response-timeout", strconv.Itoa(*redisCachedObj.Options.ResponseTimeout))
		}
		if redisCachedObj.Options.BatchSize != nil {
			command = append(command, "--batch-size", strconv.Itoa(*redisCachedObj.Options.BatchSize))
		}
	}

	return DeploymentStorageOptions{
		Args: command,
	}, nil
}
