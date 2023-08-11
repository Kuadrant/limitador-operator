# Resource Requirements

The default resource requirement for _**Limitador**_ deployments is specified in [Limitador v1alpha1 API reference](../api/v1alpha1/limitador_types.go)
and will be applied if the resource requirement is not set in the spec.

```yaml
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
```

| **Field**            | **json/yaml field**    | **Type**                                                                                                                          | **Required** | **Default value**                                                                           | **Description**                            |
|----------------------|------------------------|-----------------------------------------------------------------------------------------------------------------------------------|--------------|---------------------------------------------------------------------------------------------|--------------------------------------------|
| ResourceRequirements | `resourceRequirements` | [*corev1.ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#resourcerequirements-v1-core) | No           | `{"limits": {"cpu": "500m","memory": "64Mi"},"requests": {"cpu": "250m","memory": "32Mi"}}` | Limitador deployment resource requirements |

## Example with resource limits 
The resource requests and limits for the deployment can be set like the following:

```yaml
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
  resourceRequirements:
    limits:
      cpu: 200m
      memory: 400Mi
    requests:
      cpu: 101m  
      memory: 201Mi    
```

To specify the deployment without resource requests or limits, set an empty struct `{}` to the field:
```yaml
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
    - conditions: [ "get_toy == 'yes'" ]
      max_value: 2
      namespace: toystore-app
      seconds: 30
      variables: []
  resourceRequirements: {}
```
