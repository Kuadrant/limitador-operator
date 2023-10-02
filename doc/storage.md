# Storage

_**Limitador**_ limits counters are stored in a backend storage. This is In contrast to the storage of
the limits themselves, which are always stored in ephemeral memory. Limitador's operator
supports several storage configurations:

* In-Memory: ephemeral and cannot be shared
* Redis: Persistent (depending on the redis storage configuration) and can be shared
* Redis Cached: Persistent (depending on the redis storage configuration) and can be shared
* Disk: Persistent (depending on the underlying disk persistence capabilities) and cannot be shared

## In-Memory

Counters are held in Limitador (ephemeral)

In-Memory is the default option defined by the Limitador's Operator.

```yaml
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-sample
spec:
  storage: null
```

For any of those, one should store the URL of the Redis service, inside a K8s opaque
[Secret](https://kubernetes.io/docs/concepts/configuration/secret/).

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: redisconfig
stringData:
  URL: redis://127.0.0.1/a # Redis URL of its running instance
type: Opaque
```

## Redis

Uses Redis to store counters.

Selected when `spec.storage.redis` is not `null`.

```yaml
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-sample
spec:
  storage:
    redis:
      configSecretRef: # The secret reference storing the URL for Redis
        name: redisconfig
        namespace: default # optional
```

The URL of the Redis service is provided inside a K8s opaque
[Secret](https://kubernetes.io/docs/concepts/configuration/secret/).

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: redisconfig
stringData:
  URL: redis://127.0.0.1/a # Redis URL of its running instance
type: Opaque
```

**Note**: Limitador's Operator will only read the `URL` field of the secret.

## Redis Cached

Uses Redis to store counters, with an in-memory cache.

Selected when `spec.storage.redis-cached` is not `null`.

```yaml
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-sample
spec:
  storage:
    redis-cached:
      configSecretRef: # The secret reference storing the URL for Redis
        name: redisconfig
        namespace: default # optional
```

The URL of the Redis service is provided inside a K8s opaque
[Secret](https://kubernetes.io/docs/concepts/configuration/secret/).

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: redisconfig
stringData:
  URL: redis://127.0.0.1/a # Redis URL of its running instance
type: Opaque
```

**Note**: Limitador's Operator will only read the `URL` field of the secret.

Additionally, caching options can be specified in the `spec.storage.redis-cached.options` field.

### Options

| Option         | Description                                                           |
|----------------|-----------------------------------------------------------------------|
| `ttl`          | TTL for cached counters in milliseconds [default: 5000]               |
| `ratio`        | Ratio to apply to the TTL from Redis on cached counters [default: 10] |
| `flush-period` | Flushing period for counters in milliseconds [default: 1000]          |
| `max-cached`   | Maximum amount of counters cached [default: 10000]                    |

For example:

```yaml
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-sample
spec:
  storage:
    redis-cached:
      configSecretRef: # The secret reference storing the URL for Redis
        name: redisconfig
      options: # Every option is optional
        ttl: 1000
        max-cached: 5000
```

## Disk

Counters are held on disk (persistent).
Kubernetes [Persistent Volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
will be used to store counters.

Selected when `spec.storage.disk` is not `null`.

```yaml
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-sample
spec:
  storage:
    disk: {}
```

Additionally, disk options can be specified in the `spec.storage.disk.persistentVolumeClaim`
and `spec.storage.disk.optimize` fields.

### Persistent Volume Claim Options

`spec.storage.disk.persistentVolumeClaim` field is an object with the following fields.

| Field                | Description                                                                                                                                                                                                                                                                                           |
|----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `storageClassName`   | [StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/) of the storage offered by cluster administrators [default: default storage class of the cluster]                                                                                                                         |
| `resources`          | The minimum [resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#quantity-resource-core) the volume should have. Resources will not take any effect when VolumeName is provided. This parameter is not updateable when the underlying PV is not resizable. [default: 1Gi] |
| `volumeName`         | The binding reference to the existing PersistentVolume backing this claim [default: *null*]                                                                                                                                                                                                           |

Example:

```yaml
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-sample
spec:
  storage:
    disk:
      persistentVolumeClaim:
        storageClassName: "customClass"
        resources:
          requests: 2Gi
```

### Optimize

Defines the valid optimization option of the disk persistence type.

`spec.storage.disk.optimize` field is a `string` type with the following valid values:

| Option         | Description                              |
|----------------|------------------------------------------|
| `throughput`   | Optimizes for higher throughput. **Default** |
| `disk`         | Optimizes for disk usage                 |

Example:

```yaml
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-sample
spec:
  storage:
    disk:
      optimize: disk
```
