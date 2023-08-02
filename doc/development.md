# Development Guide

<!--ts-->
   * [Technology stack required for development](#technology-stack-required-for-development)
   * [Build](#build)
   * [Run locally](#run-locally)
   * [Deploy the operator in a deployment object](#deploy-the-operator-in-a-deployment-object)
   * [Deploy the operator using OLM](#deploy-the-operator-using-olm)
   * [Build custom OLM catalog](#build-custom-olm-catalog)
      * [Build operator bundle image](#build-operator-bundle-image)
      * [Build custom catalog](#build-custom-catalog)
   * [Cleaning up](#cleaning-up)
   * [Run tests](#run-tests)
      * [Lint tests](#lint-tests)
   * [(Un)Install Limitador CRD](#uninstall-limitador-crd)

<!-- Created by https://github.com/ekalinin/github-markdown-toc -->

<!--te-->

## Technology stack required for development

* [operator-sdk] version v1.22.0
* [kind] version v0.20.0
* [git][git_tool]
* [go] version 1.19+
* [kubernetes] version v1.19+
* [kubectl] version v1.19+

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

* Build the bundle manifests

```bash
make bundle [IMG=quay.io/kuadrant/limitador-operator:latest] \
            [VERSION=0.0.0] \
            [RELATED_IMAGE_LIMITADOR=quay.io/kuadrant/limitador:latest]
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

```sh
make catalog [BUNDLE_IMG=quay.io/kuadrant/limitador-operator-bundle:latest]
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
