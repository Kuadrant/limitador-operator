# Limitador Operator

[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)

## Overview

The Operator to manage [Limitador](https://github.com/Kuadrant/limitador) deployments.

## CustomResourceDefinitions

* [Limitador](#limitador), which defines a desired Limitador deployment.

### Limitador CRD

[Limitador v1alpha1 API reference](api/v1alpha1/limitador_types.go)

Example:

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
    - conditions: ["get-toy == yes"]
      max_value: 2
      namespace: toystore-app
      seconds: 30
      variables: []
```

## Licensing

This software is licensed under the [Apache 2.0 license](https://www.apache.org/licenses/LICENSE-2.0).

See the LICENSE and NOTICE files that should have been provided along with this software for details.
