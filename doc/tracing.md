# Tracing

## Data Plane Tracing (Limitador)

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

Currently limitador only supports collectors using the OpenTelemetry Protocol with TLS disabled. The `endpoint`
configuration option should contain the scheme, host and port of the service. The quantity and level of the information
provided by the spans is configured via the `verbosity` argument.

![Limitador tracing example](https://github.com/Kuadrant/limitador-operator/assets/6575004/7bdc7c17-37a5-4dfe-ac56-432efa1070c4)

## Control Plane Tracing (Limitador Operator)

The Limitador Operator supports distributed tracing of its reconciliation operations using OpenTelemetry. Control plane
tracing is configured via environment variables.

### Configuration

To enable control plane tracing, set the `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable to your OTLP collector
endpoint. If this variable is not set or is empty, tracing is disabled.

| Variable                      | Description                                                    | Default              | Required |
|-------------------------------|----------------------------------------------------------------|----------------------|----------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector endpoint URL (e.g., `rpc://localhost:4317`)     | `""` (disabled)      | **Yes**  |
| `OTEL_SERVICE_NAME`           | Service name for traces                                        | `limitador-operator` | No       |
| `OTEL_EXPORTER_OTLP_INSECURE` | Use insecure connection to collector                           | `false`              | No       |
| `OTEL_RESOURCE_ATTRIBUTES`    | Additional resource attributes (format: `key=value,key=value`) | `""`                 | No       |

### Endpoint URL Schemes

The endpoint scheme determines the protocol used:

- `rpc://host:port` → gRPC OTLP
- `http://host:port` → HTTP OTLP (insecure)
- `https://host:port` → HTTP OTLP (secure)

### Example: Deployment Configuration

To enable control plane tracing in a deployed operator, add the environment variables to the operator's deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: limitador-operator-controller-manager
spec:
  template:
    spec:
      containers:
        - name: manager
          env:
            # Required: OTLP endpoint to enable tracing
            - name: OTEL_EXPORTER_OTLP_ENDPOINT
              value: "rpc://jaeger-collector.observability.svc.cluster.local:4317"
            # Optional: Use insecure connection
            - name: OTEL_EXPORTER_OTLP_INSECURE
              value: "true"
            # Optional: Custom service name
            - name: OTEL_SERVICE_NAME
              value: "limitador-operator"
            # Optional: Additional resource attributes
            - name: OTEL_RESOURCE_ATTRIBUTES
              value: "environment=production,region=us-east-1"
```

### Example: Local Development

When running the operator locally with `make run`, you can enable tracing by setting the environment variables:

```bash
# Minimal configuration (gRPC OTLP with insecure connection)
OTEL_EXPORTER_OTLP_ENDPOINT=rpc://localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
make run

# Using HTTP OTLP
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
make run

# With additional resource attributes
OTEL_EXPORTER_OTLP_ENDPOINT=rpc://localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
OTEL_RESOURCE_ATTRIBUTES="environment=dev,developer=$(whoami)" \
make run
```

### Trace Context Propagation

The operator implements trace context propagation to connect operator reconciliation traces with the operations that
triggered them:

1. When a Limitador CR is created or updated with a `traceparent` annotation, the operator extracts the trace context
2. The reconciliation span is linked to the external trace context (not as a parent-child relationship, but as a link)
3. This allows correlating operator reconciliation with external tools (e.g., kubectl, GitOps controllers) that
   initiated the change

Example CR with trace context annotation:

```yaml
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-sample
  annotations:
    traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
spec:
# ... spec fields ...
```

### What Gets Traced

The operator traces:

- Complete reconciliation cycles with timing information
- Individual resource reconciliation (Deployment, Service, ConfigMap, PVC, PodDisruptionBudget)
- Status updates
- Error conditions and recovery attempts
- Limitador-specific attributes (namespace, name, replicas, storage type)
