# Storage

The default storage for _**Limitador**_ _limits counter_ is in memory, which there's no configuration needed.
In order to configure a Redis data structure store, currently there are 2 alternatives:

* Redis
* Redis Cached

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

It's also required to setup `Spec.Storage`

## Redis

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
  limits:
    - conditions: ["get_toy == 'yes'"]
      max_value: 2
      namespace: toystore-app
      seconds: 30
      variables: []
```

## Redis Cached

### Options

| Option       | Description                                                       |
|--------------|-------------------------------------------------------------------|
| ttl          | TTL for cached counters in milliseconds [default: 5000]           |
| ratio        | Ratio to apply to the TTL from Redis on cached counters [default: |
| flush-period | Flushing period for counters in milliseconds [default: 1000]      |
| max-cached   | Maximum amount of counters cached [default: 10000]                |


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
     options: # Every option is optional
        ttl: 1000

  limits:
    - conditions: ["get_toy == 'yes'"]
      max_value: 2
      namespace: toystore-app
      seconds: 30
      variables: []
```
