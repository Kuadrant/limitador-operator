# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))

VERSION ?= 0.0.0

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Address of the container registry
REGISTRY = quay.io

# Organization in container resgistry
ORG ?= kuadrant
REPO_NAME ?= limitador-operator

# kubebuilder-tools still doesn't support darwin/arm64. This is a workaround (https://github.com/kubernetes-sigs/controller-runtime/issues/1657)
ARCH_PARAM =
ifeq ($(shell uname -sm),Darwin arm64)
	ARCH_PARAM = --arch=amd64
endif

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# quay.io/kuadrant/limitador-operator-bundle:$VERSION and quay.io/kuadrant/limitador-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= $(REGISTRY)/$(ORG)/limitador-operator

# Semantic versioning (i.e. Major.Minor.Patch)
is_semantic_version = $(shell [[ $(1) =~ ^[0-9]+\.[0-9]+\.[0-9]+(-.+)?$$ ]] && echo "true")

# Limitador version
LIMITADOR_VERSION ?= latest

limitador_version_is_semantic := $(call is_semantic_version,$(LIMITADOR_VERSION))

ifeq (true,$(limitador_version_is_semantic))
RELATED_IMAGE_LIMITADOR ?= quay.io/kuadrant/limitador:v$(LIMITADOR_VERSION)
else
RELATED_IMAGE_LIMITADOR ?= quay.io/kuadrant/limitador:$(LIMITADOR_VERSION)
endif


# BUNDLE_VERSION defines the version for the limitador-operator bundle.
# If the version is not semantic, will use the default one
bundle_is_semantic := $(call is_semantic_version,$(VERSION))
ifeq (0.0.0,$(VERSION))
BUNDLE_VERSION = $(VERSION)
IMAGE_TAG = latest
else ifeq ($(bundle_is_semantic),true)
BUNDLE_VERSION = $(VERSION)
IMAGE_TAG = v$(VERSION)
else
BUNDLE_VERSION = 0.0.0
IMAGE_TAG ?= $(DEFAULT_IMAGE_TAG)
endif

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:$(IMAGE_TAG)

# Image URL to use all building/pushing image targets
DEFAULT_IMG ?= $(IMAGE_TAG_BASE):$(IMAGE_TAG)
IMG ?= $(DEFAULT_IMG)

UNIT_DIRS := ./pkg/... ./api/...
INTEGRATION_TEST_SUITE_PATHS := ./controllers/...
INTEGRATION_COVER_PKGS := ./pkg/...,./controllers/...,./api/...
INTEGRATION_TEST_NUM_CORES ?= 4
INTEGRATION_TEST_NUM_PROCESSES ?= 10

# Limitador Operator replaced version
DEFAULT_REPLACES_VERSION = 0.0.0-alpha
REPLACES_VERSION ?= $(DEFAULT_REPLACES_VERSION)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


all: build

##@ Tools

OPERATOR_SDK = $(PROJECT_PATH)/bin/operator-sdk
OPERATOR_SDK_VERSION = v1.32.0
$(OPERATOR_SDK):
	./utils/install-operator-sdk.sh $(OPERATOR_SDK) $(OPERATOR_SDK_VERSION)

.PHONY: operator-sdk
operator-sdk: $(OPERATOR_SDK) ## Download operator-sdk locally if necessary.

CONTROLLER_GEN = $(PROJECT_PATH)/bin/controller-gen
$(CONTROLLER_GEN):
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.15.0)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN)  ## Download controller-gen locally if necessary.

KUSTOMIZE = $(PROJECT_PATH)/bin/kustomize
$(KUSTOMIZE):
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.5)

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.

YQ=$(PROJECT_PATH)/bin/yq
YQ_VERSION := v4.34.2
$(YQ):
	$(call go-install-tool,$(YQ),github.com/mikefarah/yq/v4@$(YQ_VERSION))

.PHONY: yq
yq: $(YQ) ## Download yq locally if necessary.

OPM = $(PROJECT_PATH)/bin/opm
OPM_VERSION = v1.26.2
$(OPM):
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/$(OPM_VERSION)/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}

.PHONY: opm
opm: $(OPM) ## Download opm locally if necessary.

KIND = $(PROJECT_PATH)/bin/kind
$(KIND):
	$(call go-install-tool,$(KIND),sigs.k8s.io/kind@v0.22.0)

.PHONY: kind
kind: $(KIND) ## Download kind locally if necessary.

ACT = $(PROJECT_PATH)/bin/act
$(ACT):
	$(call go-install-tool,$(ACT),github.com/nektos/act@latest)

.PHONY: act
act: $(ACT) ## Download act locally if necessary.

GOLANGCI-LINT = $(PROJECT_PATH)/bin/golangci-lint
$(GOLANGCI-LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(PROJECT_PATH)/bin v1.54.2

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI-LINT) ## Download golangci-lint locally if necessary.

GINKGO = $(PROJECT_PATH)/bin/ginkgo
$(GINKGO):
	# In order to make sure the version of the ginkgo cli installed
	# is the same as the version of go.mod,
	# instead of calling go-install-tool,
	# running go install from the current module will pick version from current go.mod file.
	GOBIN=$(PROJECT_PATH)/bin go install github.com/onsi/ginkgo/v2/ginkgo

.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download ginkgo locally if necessary.

HELM = $(PROJECT_PATH)/bin/helm
HELM_VERSION = v3.15.0
$(HELM):
	@{ \
	set -e ;\
	mkdir -p $(dir $(HELM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	wget -O helm.tar.gz https://get.helm.sh/helm-$(HELM_VERSION)-$${OS}-$${ARCH}.tar.gz ;\
	tar -zxvf helm.tar.gz ;\
	mv $${OS}-$${ARCH}/helm $(HELM) ;\
	chmod +x $(HELM) ;\
	rm -rf $${OS}-$${ARCH} helm.tar.gz ;\
	}

.PHONY: helm
helm: $(HELM) ## Download helm locally if necessary.

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

.PHONY: clean-cov
clean-cov: ## Remove coverage reports
	rm -rf $(PROJECT_PATH)/coverage

.PHONY: test
test: test-unit test-integration ## Run all tests

test-integration: clean-cov generate fmt vet ginkgo ## Run Integration tests.
	mkdir -p $(PROJECT_PATH)/coverage/integration
#	Check `ginkgo help run` for command line options. For example to filtering tests.
	$(GINKGO) \
		--coverpkg $(INTEGRATION_COVER_PKGS) \
		--output-dir $(PROJECT_PATH)/coverage/integration \
		--coverprofile cover.out \
		-v \
		--compilers=$(INTEGRATION_TEST_NUM_CORES) \
		--procs=$(INTEGRATION_TEST_NUM_PROCESSES) \
		--randomize-all \
		--randomize-suites \
		--fail-on-pending \
		--keep-going \
		--race \
		--trace \
		$(INTEGRATION_TEST_SUITE_PATHS)

ifdef TEST_NAME
test-unit: TEST_PATTERN := --run $(TEST_NAME)
endif
test-unit: clean-cov generate fmt vet ## Run Unit tests.
	mkdir -p $(PROJECT_PATH)/coverage/unit
	go test $(UNIT_DIRS) -coverprofile $(PROJECT_PATH)/coverage/unit/cover.out -v -timeout 0 $(TEST_PATTERN)

##@ Build
build: GIT_SHA=$(shell git rev-parse HEAD || echo "unknown")
build: DIRTY=$(shell $(PROJECT_PATH)/utils/check-git-dirty.sh || echo "unknown")
build: generate fmt vet ## Build manager binary.
	   go build -ldflags "-X main.gitSHA=${GIT_SHA} -X main.dirty=${DIRTY}" -o bin/manager main.go

run: export LOG_LEVEL = debug
run: export LOG_MODE = development
run: GIT_SHA=$(shell git rev-parse HEAD || echo "unknown")
run: DIRTY=$(shell $(PROJECT_PATH)/utils/check-git-dirty.sh || echo "unknown")
run: manifests generate fmt vet ## Run a controller from your host.)
	go run -ldflags "-X main.gitSHA=${GIT_SHA} -X main.dirty=${DIRTY}" ./main.go

docker-build: GIT_SHA=$(shell git rev-parse HEAD || echo "unknown")
docker-build: DIRTY=$(shell $(PROJECT_PATH)/utils/check-git-dirty.sh || echo "unknown")
docker-build: ## Build docker image with the manager.
	docker build --build-arg GIT_SHA=$(GIT_SHA) --build-arg DIRTY=$(DIRTY) --build-arg QUAY_IMAGE_EXPIRY=$(QUAY_IMAGE_EXPIRY) -t $(IMG) .

docker-push: ## Push docker image with the manager.
	docker push $(IMG)

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -
	cd config/manager && $(KUSTOMIZE) edit set image controller=${DEFAULT_IMG}

deploy-develmode: manifests kustomize ## Deploy controller in debug mode to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/deploy-develmode | kubectl apply -f -
	cd config/manager && $(KUSTOMIZE) edit set image controller=${DEFAULT_IMG}

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

.PHONY: install-olm
install-olm: $(OPERATOR_SDK)
	$(OPERATOR_SDK) olm install

.PHONY: uninstall-olm
uninstall-olm:
	$(OPERATOR_SDK) olm uninstall

# go-install-tool will 'go install' any package $2 and install it to $1.
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_PATH)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: bundle
bundle: $(KUSTOMIZE) $(OPERATOR_SDK) $(YQ) manifests ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	# Set desired operator image and related limitador image
	V="$(RELATED_IMAGE_LIMITADOR)" $(YQ) eval '(select(.kind == "Deployment").spec.template.spec.containers[].env[] | select(.name == "RELATED_IMAGE_LIMITADOR").value) = strenv(V)' -i config/manager/manager.yaml
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	# Update CSV
	V="limitador-operator.v$(BUNDLE_VERSION)" $(YQ) eval '.metadata.name = strenv(V)' -i config/manifests/bases/limitador-operator.clusterserviceversion.yaml
	V="$(BUNDLE_VERSION)" $(YQ) eval '.spec.version = strenv(V)' -i config/manifests/bases/limitador-operator.clusterserviceversion.yaml
	V="$(IMG)" $(YQ) eval '.metadata.annotations.containerImage = strenv(V)' -i config/manifests/bases/limitador-operator.clusterserviceversion.yaml
	V="limitador-operator.v$(REPLACES_VERSION)" $(YQ) eval '.spec.replaces = strenv(V)' -i config/manifests/bases/limitador-operator.clusterserviceversion.yaml
	# Generate bundle
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(BUNDLE_VERSION) $(BUNDLE_METADATA_OPTS)
	# Validate bundle manifests
	$(OPERATOR_SDK) bundle validate ./bundle
	$(MAKE) bundle-ignore-createdAt
	echo "$$QUAY_EXPIRY_TIME_LABEL" >> bundle.Dockerfile

.PHONY: bundle-ignore-createdAt
bundle-ignore-createdAt:
	# Since operator-sdk 1.26.0, `make bundle` changes the `createdAt` field from the bundle
	# even if it is patched:
	#   https://github.com/operator-framework/operator-sdk/pull/6136
	# This code checks if only the createdAt field. If is the only change, it is ignored.
	# Else, it will do nothing.
	# https://github.com/operator-framework/operator-sdk/issues/6285#issuecomment-1415350333
	# https://github.com/operator-framework/operator-sdk/issues/6285#issuecomment-1532150678
	git diff --quiet -I'^    createdAt: ' ./bundle && git checkout ./bundle || true

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build --build-arg QUAY_IMAGE_EXPIRY=$(QUAY_IMAGE_EXPIRY) -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: prepare-release
prepare-release: ## Prepare the manifests for OLM and Helm Chart for a release.
	$(MAKE) bundle VERSION=$(VERSION) \
		LIMITADOR_VERSION=$(LIMITADOR_VERSION) \
		REPLACES_VERSION=$(REPLACES_VERSION)
	$(MAKE) helm-build VERSION=$(VERSION) \
		LIMITADOR_VERSION=$(LIMITADOR_VERSION)

##@ Misc

.PHONY: local-env-setup
local-env-setup: ## Prepare environment to run the operator with "make run"
	$(MAKE) kind-delete-cluster
	$(MAKE) kind-create-cluster
	$(MAKE) install

## Miscellaneous Custom targets
.PHONY: local-setup
local-setup: export IMG := localhost/limitador-operator:dev
local-setup: ## Deploy operator in local kind cluster
	$(MAKE) local-env-setup
	$(MAKE) docker-build
	@echo "Deploying Limitador control plane"
	$(KIND) load docker-image ${IMG} --name ${KIND_CLUSTER_NAME}
	$(MAKE) deploy-develmode
	@echo "Wait for all deployments to be up"
	kubectl -n limitador-operator-system wait --timeout=300s --for=condition=Available deployments --all

.PHONY: local-cleanup
local-cleanup: ## Clean up local kind cluster
	$(MAKE) kind-delete-cluster

.PHONY: local-redeploy
local-redeploy: export IMG := limitador-operator:dev
local-redeploy: ## re-deploy operator in local kind cluster
	$(MAKE) docker-build
	@echo "Deploying Limitador control plane"
	$(KIND) load docker-image ${IMG} --name ${KIND_CLUSTER_NAME}
	$(MAKE) deploy-develmode
	kubectl rollout restart deployment -n limitador-operator-system limitador-operator-controller-manager
	@echo "Wait for all deployments to be up"
	kubectl -n limitador-operator-system wait --timeout=300s --for=condition=Available deployments --all

##@ Code Style

.PHONY: run-lint
run-lint: $(GOLANGCI-LINT) ## Run lint tests
	$(GOLANGCI-LINT) run

# Include last to avoid changing MAKEFILE_LIST used above
include ./make/*.mk
