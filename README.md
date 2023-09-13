# Limitador Operator

[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0) 
[![codecov](https://codecov.io/gh/Kuadrant/limitador-operator/branch/main/graph/badge.svg?token=181Q05ZJBJ)](https://codecov.io/gh/Kuadrant/limitador-operator)

## Overview

The Operator to manage [Limitador](https://github.com/Kuadrant/limitador) deployments.

## CustomResourceDefinitions

* [Limitador](#limitador-crd), which defines a desired Limitador deployment.

### Limitador CRD

[Limitador v1alpha1 API reference](./api/v1alpha1/limitador_types.go)

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
    - conditions: ["get_toy == 'yes'"]
      max_value: 2
      namespace: toystore-app
      seconds: 30
      variables: []
```

## Features

* [Storage Options](./doc/storage.md)
* [Rate Limit Headers](./doc/rate-limit-headers.md)
* [Logging](./doc/logging.md)

## Contributing

The [Development guide](./doc/development.md) describes how to build the operator and
how to test your changes before submitting a patch or opening a PR.

Join us on the [#kuadrant](https://kubernetes.slack.com/archives/C05J0D0V525) channel in the Kubernetes Slack workspace, 
for live discussions about the roadmap and more.

## Licensing

This software is licensed under the [Apache 2.0 license](https://www.apache.org/licenses/LICENSE-2.0).

See the LICENSE and NOTICE files that should have been provided along with this software for details.
