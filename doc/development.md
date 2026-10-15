# Development Guide

## Technology stack required for development

* [kind] version v0.22.0
* [git][git_tool]
* [go] version 1.25+
* [kubernetes] version v1.28+
* [kubectl] version v1.25+

## Build

```sh
make
```

## Run locally

You need an active session open to a kubernetes cluster.

Optionally, run kind with `local-env-setup`.

```sh
make local-env-setup
```

Then, run the operator locally

```sh
make run
```

## Deploy the operator in a deployment object

```sh
make local-setup
```

## Deploy the operator using the Helm chart

The Helm chart is the packaging mechanism for the operator. The released charts are published to the
[Kuadrant Helm charts repository](https://kuadrant.io/helm-charts/), and the operator images are available in
the [Quay.io/Kuadrant](https://quay.io/organization/kuadrant) image repository.

Install the released chart:

```sh
helm repo add kuadrant https://kuadrant.io/helm-charts/ --force-update
helm install \
 limitador-operator kuadrant/limitador-operator \
 --create-namespace \
 --namespace limitador-system
```

### Build and install the chart from source

The `make helm-build` target regenerates `charts/limitador-operator/templates/manifests.yaml` from the
kustomize manifests in `config/helm`. It accepts the following variables:

| **Makefile Variable**     | **Description**    | **Default value**                            | **Notes**                                                                |
|---------------------------|--------------------|----------------------------------------------|--------------------------------------------------------------------------|
| `IMG`                     | Operator image URL | `quay.io/kuadrant/limitador-operator:latest` |                                                                          |
| `VERSION`                 | Chart version      | `0.0.0`                                      |                                                                          |
| `RELATED_IMAGE_LIMITADOR` | Limitador image URL | `quay.io/kuadrant/limitador:latest`         | `LIMITADOR_VERSION` var could be use to build this URL providing the tag |

```sh
make helm-build [IMG=quay.io/kuadrant/limitador-operator:latest] \
                [VERSION=0.0.0] \
                [RELATED_IMAGE_LIMITADOR=quay.io/kuadrant/limitador:latest]
```

Install, upgrade and uninstall the local chart:

```sh
make helm-install
make helm-upgrade
make helm-uninstall
```

Package the chart as a `.tgz` archive:

```sh
make helm-package
```

## Cleaning up

```sh
make local-cleanup
```

## Run tests

### Unittests

```sh
make test-unit
```

Optionally, add `TEST_NAME` makefile variable to run specific test

```sh
make test-unit TEST_NAME=TestConstants
```

or even subtest

```sh
make test-unit TEST_NAME=TestLimitIndexEquals/empty_indexes_are_equal
```

### Integration tests

You need an active session open to a kubernetes cluster.

Optionally, run local cluster with kind

```sh
make local-env-setup
```

Run integration tests

```sh
make test-integration
```

### All tests

You need an active session open to a kubernetes cluster.

Optionally, run local cluster with kind

```sh
make local-env-setup
```

Run all tests

```sh
make test
```

### Lint tests

```sh
make run-lint
```

## (Un)Install Limitador CRD

You need an active session open to a kubernetes cluster.

Remove CRDs

```sh
make uninstall
```

## Image builds

| Docker image | Tag | Expiration | When |
| --- | --- | --- | --- |
| latest image | `latest` | never | Push to `main` |
| branch images | `${{ github.ref_name }}` | 1w | Push to branch |
| nightly images | `${{ nightly-$(date +'%d-%m-%Y') }}` | 1w | Scheduled daily `cron: '0 0 * * *'` |
| release images | release version | never | release created |

[git_tool]:https://git-scm.com/downloads
[go]:https://golang.org/
[kind]:https://kind.sigs.k8s.io/
[kubernetes]:https://kubernetes.io/
[kubectl]:https://kubernetes.io/docs/tasks/tools/#kubectl
