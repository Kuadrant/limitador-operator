# Development Guide

## Technology stack required for development

* [operator-sdk] version 1.32.0
* [kind] version v0.22.0
* [git][git_tool]
* [go] version 1.21+
* [kubernetes] version v1.25+
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

## Deploy the operator using OLM

You can deploy the operator using OLM just running a few commands.
No need to build any image. Kuadrant engineering team provides `latest` and
released version tagged images. They are available in
the [Quay.io/Kuadrant](https://quay.io/organization/kuadrant) image repository.

Create kind cluster

```sh
make kind-create-cluster
```

Deploy OLM system

```sh
make install-olm
```

Deploy the operator using OLM. The `make deploy-catalog` target accepts the following variables:

| **Makefile Variable** | **Description**   | **Default value**                                    |
|-----------------------|-------------------|------------------------------------------------------|
| `CATALOG_IMG`         | Catalog image URL | `quay.io/kuadrant/limitador-operator-catalog:latest` |

```sh
make deploy-catalog [CATALOG_IMG=quay.io/kuadrant/limitador-operator-catalog:latest]
```

## Build custom OLM catalog

If you want to deploy (using OLM) a custom limitador operator, you need to build your own catalog.

### Build operator bundle image

The `make bundle` target accepts the following variables:

| **Makefile Variable**     | **Description**      | **Default value**                            | **Notes**                                                                |
|---------------------------|----------------------|----------------------------------------------|--------------------------------------------------------------------------|
| `IMG`                     | Operator image URL   | `quay.io/kuadrant/limitador-operator:latest` |                                                                          |
| `VERSION`                 | Bundle version       | `0.0.0`                                      |                                                                          |
| `RELATED_IMAGE_LIMITADOR` | Limitador bundle URL | `quay.io/kuadrant/limitador:latest`          | `LIMITADOR_VERSION` var could be use to build this URL providing the tag |
| `CHANNELS`                | Bundle channels used in the bundle, comma separated                 | `alpha`                                                                  |
| `DEFAULT_CHANNEL`         | The default channel used in the bundle                              | `alpha`                                                                  |

* Build the bundle manifests

```bash
make bundle [IMG=quay.io/kuadrant/limitador-operator:latest] \
            [VERSION=0.0.0] \
            [RELATED_IMAGE_LIMITADOR=quay.io/kuadrant/limitador:latest] \
            [CHANNELS=alpha] \
            [DEFAULT_CHANNEL=alpha]
```

* Build the bundle image from the manifests

| **Makefile Variable** | **Description**           | **Default value**                                   |
|-----------------------|---------------------------|-----------------------------------------------------|
| `BUNDLE_IMG`          | Operator bundle image URL | `quay.io/kuadrant/limitador-operator-bundle:latest` |

```sh
make bundle-build [BUNDLE_IMG=quay.io/kuadrant/limitador-operator-bundle:latest]
```

* Push the bundle image to a registry

| **Makefile Variable** | **Description**           | **Default value**                                   |
|-----------------------|---------------------------|-----------------------------------------------------|
| `BUNDLE_IMG`          | Operator bundle image URL | `quay.io/kuadrant/limitador-operator-bundle:latest` |

```sh
make bundle-push [BUNDLE_IMG=quay.io/kuadrant/limitador-operator-bundle:latest]
```

### Build custom catalog

The catalog format will be [File-based Catalog](https://olm.operatorframework.io/docs/reference/file-based-catalogs/).

Make sure all the required bundles are pushed to the registry. It is required by the `opm` tool.

The `make catalog` target accepts the following variables:

| **Makefile Variable** | **Description**           | **Default value**                                   |
|-----------------------|---------------------------|-----------------------------------------------------|
| `BUNDLE_IMG`          | Operator bundle image URL | `quay.io/kuadrant/limitador-operator-bundle:latest` |
| `DEFAULT_CHANNEL`     | Catalog default channel   | `alpha`                                             |

```sh
make catalog [BUNDLE_IMG=quay.io/kuadrant/limitador-operator-bundle:latest] [DEFAULT_CHANNEL=alpha]
```

* Build the catalog image from the manifests

| **Makefile Variable** | **Description**            | **Default value**                                    |
|-----------------------|----------------------------|------------------------------------------------------|
| `CATALOG_IMG`         | Operator catalog image URL | `quay.io/kuadrant/limitador-operator-catalog:latest` |

```sh
make catalog-build [CATALOG_IMG=quay.io/kuadrant/limitador-operator-catalog:latest]
```

* Push the catalog image to a registry

```sh
make catalog-push [CATALOG_IMG=quay.io/kuadrant/limitador-operator-bundle:latest]
```

You can try out your custom catalog image following the steps of the
[Deploy the operator using OLM](#deploy-the-operator-using-olm) section.

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

[git_tool]:https://git-scm.com/downloads
[operator-sdk]:https://github.com/operator-framework/operator-sdk
[go]:https://golang.org/
[kind]:https://kind.sigs.k8s.io/
[kubernetes]:https://kubernetes.io/
[kubectl]:https://kubernetes.io/docs/tasks/tools/#kubectl
