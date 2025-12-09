# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Limitador Operator is a Kubernetes operator that manages [Limitador](https://github.com/Kuadrant/limitador) deployments.
Limitador is a rate-limiting service. The operator reconciles `Limitador` custom resources (CRD) to create and manage
deployments, services, ConfigMaps, PVCs, and PodDisruptionBudgets in Kubernetes.

**Built with:** Kubebuilder v3, operator-sdk v1.32.0, Go 1.25+

## Common Commands

### Building and Running

```bash
# Build the operator binary
make build

# Run the operator locally (requires active k8s cluster)
make run

# Build docker image
make docker-build [IMG=<image-url>]

# Push docker image
make docker-push [IMG=<image-url>]
```

### Testing

```bash
# Run all tests (unit + integration)
make test

# Run unit tests only
make test-unit

# Run specific unit test by name
make test-unit TEST_NAME=TestConstants

# Run specific subtest
make test-unit TEST_NAME=TestLimitIndexEquals/empty_indexes_are_equal

# Run integration tests (requires k8s cluster)
make test-integration

# Run with verbose output
make test-unit VERBOSE=1
make test-integration VERBOSE=1

# Lint tests
make run-lint
```

### Code Generation

```bash
# Generate manifests (CRDs, RBAC, etc.)
make manifests

# Generate DeepCopy methods
make generate

# Format code
make fmt

# Run go vet
make vet
```

### Verification (CI/CD)

```bash
# Verify manifests are up to date
make verify-manifests

# Verify bundle is up to date
make verify-bundle

# Verify code is formatted
make verify-fmt

# Verify go.mod is tidy
make verify-go-mod

# Verify helm charts are up to date
make verify-helm-charts
```

### Local Development with Kind

```bash
# Setup kind cluster and install CRDs
make local-env-setup

# Full local setup (creates cluster, builds image, deploys operator)
make local-setup

# Re-deploy operator to existing cluster
make local-redeploy

# Cleanup local environment
make local-cleanup

# Create kind cluster manually
make kind-create-cluster

# Delete kind cluster manually
make kind-delete-cluster
```

### CRD Management

```bash
# Install CRDs into cluster
make install

# Uninstall CRDs from cluster
make uninstall

# Deploy operator to cluster
make deploy [IMG=<image-url>]

# Deploy in development mode (debug logging)
make deploy-develmode [IMG=<image-url>]

# Undeploy operator
make undeploy
```

### OLM (Operator Lifecycle Manager)

```bash
# Install OLM
make install-olm

# Uninstall OLM
make uninstall-olm

# Build bundle manifests
make bundle [IMG=<operator-image>] [VERSION=0.0.0] [LIMITADOR_VERSION=latest]

# Build bundle image
make bundle-build [BUNDLE_IMG=<bundle-image>]

# Push bundle image
make bundle-push [BUNDLE_IMG=<bundle-image>]

# Generate catalog
make catalog [BUNDLE_IMG=<bundle-image>] [DEFAULT_CHANNEL=alpha]

# Build catalog image
make catalog-build [CATALOG_IMG=<catalog-image>]

# Push catalog image
make catalog-push [CATALOG_IMG=<catalog-image>]

# Deploy via OLM catalog
make deploy-catalog [CATALOG_IMG=<catalog-image>]

# Undeploy via OLM catalog
make undeploy-catalog
```

### Helm Charts

```bash
# Build helm charts from kustomize
make helm-build [VERSION=0.0.0]

# Install helm chart
make helm-install

# Uninstall helm chart
make helm-uninstall

# Upgrade helm chart
make helm-upgrade

# Package helm chart
make helm-package
```

### GitHub Actions Testing (with act)

```bash
# Test all pull request jobs locally
make act-pull-request-jobs

# Test unit tests job
make act-test-unit-tests

# Test integration tests job
make act-test-integration-tests

# Test verify manifests job
make act-test-verify-manifests
```

## Architecture

### Controller Architecture

The operator follows standard Kubebuilder patterns:

- **Main Controller**: `controllers/limitador_controller.go`
    - Reconciles `Limitador` CRs
    - Uses `reconcilers.BaseReconciler` pattern for common reconciliation logic
    - Splits reconciliation into `reconcileSpec()` and `reconcileStatus()`
    - Manages child resources: Deployment, Service, ConfigMap, PVC, PodDisruptionBudget

- **Reconciliation Order** (in `reconcileSpec()`):
    1. Service - Creates headless service for Limitador pods
    2. PVC - Creates PersistentVolumeClaim if disk storage is configured
    3. Deployment - Main Limitador deployment
    4. Limits ConfigMap - Stores rate limit definitions
    5. PodDisruptionBudget - Controls voluntary disruptions
    6. Pod Annotation Update - Triggers rollout when limits change

- **Reconcilers Pattern** (`pkg/reconcilers/`):
    - `BaseReconciler`: Provides common client, scheme, logger, recorder access
    - `ReconcileResource()`: Generic mutate-or-create pattern for K8s resources
    - Specialized reconcilers for Deployment, Service, PodDisruptionBudget

- **API Types** (`api/v1alpha1/limitador_types.go`):
    - `Limitador`: Main CR defining desired Limitador deployment
    - `LimitadorSpec`: Configuration including storage, limits, replicas, resource requirements
    - `LimitadorStatus`: Observed state with conditions and service info
    - Key validation: Disk storage disallows multiple replicas (CEL validation)

### Storage Options

Limitador supports multiple storage backends (`pkg/limitador/*_storage_options.go`):

- **In-Memory**: Default, no external dependencies
- **Redis**: Persistent storage with Redis backend
- **Redis-Cached**: Redis with local caching for performance (configurable flush-period, max-cached, response-timeout,
  batch-size)
- **Disk**: Persistent local disk storage (requires PVC, incompatible with replicas > 1)

Storage configuration is translated into Limitador deployment command-line arguments.

### Key Packages

- `api/v1alpha1/`: API definitions (CRD types)
- `controllers/`: Main reconciliation logic
- `pkg/reconcilers/`: Reusable reconciliation patterns and K8s object builders
- `pkg/limitador/`: Limitador-specific logic
    - `deployment_options.go`: Builds command-line args and env vars for Limitador
    - `k8s_objects.go`: Creates Service, Deployment, ConfigMap, PVC, PodDisruptionBudget
    - `*_storage_options.go`: Storage backend configuration builders
    - `image.go`: Image URL handling (deprecated `version` vs new `image` field)
- `pkg/helpers/`: Kubernetes utilities (conditions, labels, etc.)
- `pkg/log/`: Logging configuration

### Limits ConfigMap and Pod Sync Pattern

The operator implements a mechanism to accelerate ConfigMap sync to pods when rate limits change:

1. Limits are stored in a ConfigMap (name: `<limitador-name>-config`)
2. The ConfigMap is mounted as a volume in Limitador pods
3. When ConfigMap changes, its `ResourceVersion` changes
4. Controller updates all running pods with annotation `limits-cm-resource-version` = ConfigMap's ResourceVersion
5. This annotation update triggers Kubernetes to re-sync the mounted ConfigMap volume more quickly than waiting for the eventual consistency propagation
6. This happens in `reconcilePodLimitsHashAnnotation()` - runs after all other resources are reconciled

**Important**: This annotation update does NOT restart pods - it's a workaround to force faster ConfigMap sync to the mounted volume, ensuring pods get the latest limits without waiting for Kubernetes' eventual consistency.

### Status Conditions

The operator uses standard metav1.Condition for status reporting:

- Condition type: `"Ready"` (constant: `StatusConditionReady`)
- Status comparison uses `helpers.ConditionMarshal()` for deterministic equality checks
- Status reconciliation is separate from spec reconciliation

## Configuration System

### Kustomize Overlays

The project uses kustomize with multiple overlays in `config/`:

- `config/default/`: Standard deployment with auth proxy
- `config/deploy-develmode/`: Development mode (debug logging, etc.)
- `config/helm/`: Helm chart generation base
- `config/manifests/`: OLM bundle generation base
- `config/deploy/olm/`: OLM catalog deployment manifests

### Environment Variables

Operator configuration:

- `LOG_LEVEL`: info, debug, etc. (default: info)
- `LOG_MODE`: production or development (default: production)
- `RELATED_IMAGE_LIMITADOR`: Limitador image URL injected at build time

When running locally with `make run`, defaults to:

- `LOG_LEVEL=debug`
- `LOG_MODE=development`

## Testing Notes

- **Integration tests** use Ginkgo/Gomega framework
- **Test suite setup**: `controllers/suite_test.go` and `pkg/reconcilers/suite_test.go`
- Integration tests run with:
    - `INTEGRATION_TEST_NUM_CORES=4` (parallel compile)
    - `INTEGRATION_TEST_NUM_PROCESSES=10` (parallel execution)
    - Race detector enabled
- Unit test coverage: `coverage/unit/cover.out`
- Integration test coverage: `coverage/integration/cover.out`
- **GitHub Actions can be tested locally** using `act` tool (see act-* make targets)

## Build and Release

- **Version handling**: Set via `VERSION` make variable (semantic versioning)
- **Image versioning**:
    - Semantic versions get `v` prefix (e.g., `v1.0.0`)
    - `VERSION=0.0.0` maps to `IMAGE_TAG=latest`
- **Related images**: `RELATED_IMAGE_LIMITADOR` environment variable injects Limitador image reference
- **Build info**: Git SHA and dirty state are embedded via ldflags at build time
- **Helm charts**: Generated from kustomize manifests in `config/helm`
- **OLM Catalogs**: Built using File-based Catalog format via `opm` tool
- **Quay image expiry**: Controlled via `QUAY_IMAGE_EXPIRY` variable (default: never)

## Important Makefile Variables

- `VERSION`: Operator version (default: 0.0.0)
- `LIMITADOR_VERSION`: Limitador image version (default: latest)
- `IMG`: Operator image URL
- `BUNDLE_IMG`: Bundle image URL for OLM
- `CATALOG_IMG`: Catalog image URL for OLM
- `KIND_CLUSTER_NAME`: Kind cluster name (default: limitador-local)
- `CHANNELS`: Bundle channels (default: alpha)
- `DEFAULT_CHANNEL`: Default bundle channel (default: alpha)
- `QUAY_IMAGE_EXPIRY`: Quay image expiration (default: never)

## Important Constants

From `api/v1alpha1/limitador_types.go`:

- `DefaultServiceHTTPPort`: 8080
- `DefaultServiceGRPCPort`: 8081
- `DefaultReplicas`: 1
- `PodAnnotationConfigMapResourceVersion`: Annotation key for tracking ConfigMap changes
- `StatusConditionReady`: Status condition type

From `pkg/limitador/k8s_objects.go`:

- `StatusEndpoint`: "/status" - Limitador health check endpoint

## Development Workflow

Typical development workflow:

1. Make code changes
2. Run `make generate` and `make manifests` to update generated code
3. Run `make bundle` to update bundle manifests
4. Run `make helm-build` to update helm chart
5. Run `make fmt` to format code
6. Run `make test-unit` for quick feedback
7. Run `make local-env-setup` to create test cluster
8. Run `make run` to test locally OR `make local-setup` to deploy in cluster
9. Run `make test-integration` for full integration tests
10. Run verify commands before committing: `make verify-manifests verify-bundle verify-fmt`
