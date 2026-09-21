# Release

This repository uses a two-phase release workflow as defined in the [Two-Phase Release Workflow RFC](https://github.com/Kuadrant/architecture/blob/main/rfcs/0020-two-phase-release-workflow.md).

## Overview

Every release is split into two GitHub Actions workflows with a human review gate between them:

1. **Pre-release** (`pre-release.yaml`) — Makes version-related code changes and opens a pull request to the release branch
2. **Release** (`release.yaml`) — Runs smoke tests, tags, builds all artifacts, and creates the GitHub Release as its final step

A `release.yaml` file at the repository root is the machine-readable source of truth for the component version.
A **version gate** CI check validates this file on pull requests to release branches, ensuring:

- Version is not `0.0.0` on release branches  
- All declared dependencies have corresponding published GitHub Releases in the Kuadrant organization

## New Minor Release (e.g. 0.17.0)

1. Trigger the [Pre-release workflow](https://github.com/Kuadrant/limitador-operator/actions/workflows/pre-release.yaml) with:
   - **version**: `0.17.0`
   - **limitador-version**: `1.2.0` (the Limitador version this release depends on)
   - **source-branch**: `main` (default)

2. The workflow will:
   - Create the release branch `release-0.17` from main (if it doesn't exist)
   - Update `release.yaml` with version `0.17.0` and the `limitador` dependency `1.2.0`
   - Run `make prepare-release VERSION=0.17.0 CHANNELS=stable DEFAULT_CHANNEL=stable`
   - Open a PR from `pre-release-v0.17.0` to `release-0.17`

3. Review the PR:
   - CI runs tests and the version gate check
   - Verify version numbers, bundle manifests, and helm chart
   - Approve and merge

4. Trigger the [Release workflow](https://github.com/Kuadrant/limitador-operator/actions/workflows/release.yaml) with:
   - **release-branch**: `release-0.17`

5. The workflow will (in order):
   - Read the version from `release.yaml`
   - Run smoke tests (verify-manifests, verify-bundle, verify-helm-charts, unit tests)
   - Create and push tag `v0.17.0`
   - Build and push container images (operator, bundle, catalog)
   - Package and sign the Helm chart
   - Create the GitHub Release with all artifacts attached

6. Verify the release:
   - Check that all images are available on quay.io
   - Verify OLM deployment (see [Verify OLM Deployment](#verify-olm-deployment))

## New Patch Release (e.g. 0.17.1)

1. Trigger the [Pre-release workflow](https://github.com/Kuadrant/limitador-operator/actions/workflows/pre-release.yaml) with:
   - **version**: `0.17.1`
   - **limitador-version**: `1.2.0` (the Limitador version this release depends on)
   - **source-branch**: `release-0.17` (or a branch with cherry-picked fixes)

2. Review and merge the resulting PR to `release-0.17`

3. Trigger the [Release workflow](https://github.com/Kuadrant/limitador-operator/actions/workflows/release.yaml) with:
   - **release-branch**: `release-0.17`

4. Verify the release as above

## Release Artifacts

The release workflow produces the following artifacts:

| Artifact | Registry/Location |
|---|---|
| Operator image | `quay.io/kuadrant/limitador-operator:vX.Y.Z` |
| Bundle image | `quay.io/kuadrant/limitador-operator-bundle:vX.Y.Z` |
| Catalog image | `quay.io/kuadrant/limitador-operator-catalog:vX.Y.Z` |
| Helm chart | GitHub Release asset (GPG signed) |

All container images are built for platforms: `linux/amd64`, `linux/arm64`, `linux/s390x`, `linux/ppc64le`.

## release.yaml

The `release.yaml` file at the repository root declares the component version:

```yaml
limitador-operator:
  version: "0.0.0"    # 0.0.0 on main, concrete version on release branches

# Optional dependencies section:
dependencies:
  limitador: "1.2.0"  # Requires GitHub Release v1.2.0 in Kuadrant/limitador
```

On the `main` branch, the version is always `0.0.0` (sentinel for "under active development").
On release branches, the version is updated to the target version after the pre-release PR is merged.

The version gate validates that any declared dependencies have corresponding published GitHub Releases in the Kuadrant organization.

## Generated Files

During the pre-release process (`make prepare-release`), the following files are generated or modified:

### Modified files
- `release.yaml`
- `bundle.Dockerfile`
- `bundle/manifests/limitador-operator.clusterserviceversion.yaml`
- `bundle/metadata/annotations.yaml`
- `charts/limitador-operator/Chart.yaml`
- `charts/limitador-operator/templates/manifests.yaml`
- `config/deploy/olm/catalogsource.yaml`
- `config/deploy/olm/subscription.yaml`
- `config/manager/kustomization.yaml`
- `config/manifests/bases/limitador-operator.clusterserviceversion.yaml`

### Generated files
- `make/release.mk`

The `make/release.mk` contains release-specific variable overrides:
```sh
#Release default values
LIMITADOR_VERSION?=latest
IMAGE_TAG?=v0.17.0
IMG?=quay.io/kuadrant/limitador-operator:$(IMAGE_TAG)
BUNDLE_IMG?=quay.io/kuadrant/limitador-operator-bundle:$(IMAGE_TAG)
CATALOG_IMG?=quay.io/kuadrant/limitador-operator-catalog:$(IMAGE_TAG)
CHANNELS?=stable
BUNDLE_CHANNELS?=--channels=stable
DEFAULT_CHANNEL?=stable
BUNDLE_DEFAULT_CHANNEL?=--default-channel=stable
VERSION?=0.17.0
```

Note: The `VERSION` number is **not** prefixed with `v` (e.g. `0.17.0`), but image tags **are** prefixed with `v` (e.g. `v0.17.0`).

## Required GitHub Secrets

The release workflows require the following secrets to be configured in the repository settings:

| Secret | Used by | Description |
|---|---|---|
| `GITHUB_TOKEN` | Pre-release, Release, Version Gate | Built-in GitHub token. Used for creating branches, PRs, tags, and GitHub Releases. No manual configuration needed. |
| `IMG_REGISTRY_USERNAME` | Release | Quay.io username for pushing container images (operator, bundle, catalog). |
| `IMG_REGISTRY_TOKEN` | Release | Quay.io API token or password for pushing container images. |
| `HELM_CHARTS_SIGNING_KEY` | Release | Base64-encoded GPG private key used to sign the Helm chart package. |
| `HELM_WORKFLOWS_TOKEN` | Release | GitHub token for triggering workflows in the helm-charts repository. |

## Verify OLM Deployment

1. Deploy the OLM catalog image:
```sh
make local-setup install-olm deploy-catalog CATALOG_IMG=quay.io/kuadrant/limitador-operator-catalog:v0.17.0 DEFAULT_CHANNEL=stable
```

2. Wait for deployment:
```sh
kubectl -n limitador-system wait --timeout=60s --for=condition=Available deployments --all
```

3. Check the logs:
```sh
kubectl -n limitador-system logs -f deployment/limitador-operator-controller-manager
```

4. Verify the deployed version:
```sh
kubectl -n limitador-system get deployment limitador-operator-controller-manager -o jsonpath='{.spec.template.spec.containers[0].image}' | grep -q "v0.17.0" && echo "Version OK" || echo "Wrong version"
```
