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

	redisURL, err := getURLFromRedisSecret(ctx, cl, defSecretNamespace, *redisCachedObj.ConfigSecretRef)
	if err != nil {
		return DeploymentStorageOptions{}, err
	}

	command := []string{"redis_cached", redisURL}
	if redisCachedObj.Options != nil {
		if redisCachedObj.Options.TTL != nil {
			command = append(command, "--ttl", strconv.Itoa(*redisCachedObj.Options.TTL))
		}
		if redisCachedObj.Options.Ratio != nil {
			command = append(command, "--ratio", strconv.Itoa(*redisCachedObj.Options.Ratio))
		}
		if redisCachedObj.Options.FlushPeriod != nil {
			command = append(command, "--flush-period", strconv.Itoa(*redisCachedObj.Options.FlushPeriod))
		}
		if redisCachedObj.Options.MaxCached != nil {
			command = append(command, "--max-cached", strconv.Itoa(*redisCachedObj.Options.MaxCached))
		}
	}

	return DeploymentStorageOptions{
		Command: command,
	}, nil
}
