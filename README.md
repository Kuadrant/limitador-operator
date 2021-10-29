# Limitador Operator

[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)

## Overview

The Operator to manage [Limitador](https://github.com/Kuadrant/limitador) deployments.

## CustomResourceDefinitions

* [Limitador](#limitador), which defines a desired Limitador deployment.
* [RateLimit](#ratelimit), which declaratively specifies rate limit configurations.

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
  replicas: 1
  version: "0.4.0"
```

### RateLimit

[RateLimit v1alpha1 API reference](api/v1alpha1/ratelimit_types.go)

Example:

```yaml
---
apiVersion: limitador.kuadrant.io/v1alpha1
kind: RateLimit
metadata:
  name: ratelimit-sample
spec:
  namespace: test_namespace
  max_value: 10
  seconds: 60
  conditions:
    - "req.method == GET"
  variables:
    - user_id
```

## Licensing

This software is licensed under the [Apache 2.0 license](https://www.apache.org/licenses/LICENSE-2.0).

See the LICENSE and NOTICE files that should have been provided along with this software for details.
