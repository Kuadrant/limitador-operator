# Tracing

Limitador offers distributed tracing enablement using the `.spec.tracing` CR configuration:

```yaml
---
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-sample
spec:
  listener:
    http:
      port: 8080
    grpc:
      port: 8081
  limits:
    - conditions: ["get_toy == 'yes'"]
      max_value: 2
      namespace: toystore-app
      seconds: 30
      variables: []
  verbosity: 3
  tracing:
    endpoint: rpc://my-otlp-collector:4317
```

Currently limitador only supports collectors using the OpenTelemetry Protocol with TLS disabled. The `endpoint` configuration option should contain the scheme, host and port of the service. The quantity and level of the information provided by the spans is configured via the `verbosity` argument.

![Limitador tracing example](https://github.com/Kuadrant/limitador-operator/assets/6575004/7bdc7c17-37a5-4dfe-ac56-432efa1070c4)
