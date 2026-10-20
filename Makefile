# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))

# Load release defaults early so version-dependent conditionals (e.g. LIMITADOR_VERSION)
# evaluate correctly. The full include at the bottom of this file is kept for other .mk files.
-include $(PROJECT_PATH)/make/release.mk

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

VERSION ?= 0.0.0

# Quay image default expiry
QUAY_IMAGE_EXPIRY ?= never

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
#
# For example, running 'make docker-build docker-push' will build and push
# quay.io/kuadrant/limitador-operator:$IMAGE_TAG.
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


# IMAGE_TAG defines the tag used for the operator image.
# If the version is not semantic, will use the default one
version_is_semantic := $(call is_semantic_version,$(VERSION))
ifeq (0.0.0,$(VERSION))
IMAGE_TAG = latest
else ifeq ($(version_is_semantic),true)
IMAGE_TAG ?= v$(VERSION)
else
IMAGE_TAG ?= $(DEFAULT_IMAGE_TAG)
endif

# Image URL to use all building/pushing image targets
DEFAULT_IMG ?= $(IMAGE_TAG_BASE):$(IMAGE_TAG)
IMG ?= $(DEFAULT_IMG)

UNIT_DIRS := ./pkg/... ./api/...
INTEGRATION_TEST_SUITE_PATHS := ./controllers/...
INTEGRATION_COVER_PKGS := ./pkg/...,./controllers/...,./api/...
INTEGRATION_TEST_NUM_CORES ?= 4
INTEGRATION_TEST_NUM_PROCESSES ?= 10
INTEGRATION_TESTS_EXTRA_ARGS ?=

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

RELEASE_FILE = $(PROJECT_PATH)/make/release.mk

all: build

##@ Tools

# go-install-tool will 'go install' a package $2 with version $3 to $1 creating a versioned binary.
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(shell basename $(1))-$(3) $(1)
endef

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
KUSTOMIZE ?= $(LOCALBIN)/kustomize
YQ ?= $(LOCALBIN)/yq
HELM ?= $(LOCALBIN)/helm
KIND ?= $(LOCALBIN)/kind
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GINKGO ?= $(LOCALBIN)/ginkgo
ACT ?= $(LOCALBIN)/act
RATCHET ?= $(LOCALBIN)/ratchet

## Tool Versions
CONTROLLER_GEN_VERSION ?= v0.19.0
KUSTOMIZE_VERSION ?= v4.5.5
YQ_VERSION ?= v4.34.2
HELM_VERSION ?= v3.15.0
KIND_VERSION ?= v0.23.0
GOLANGCI_LINT_VERSION ?= v2.12.2
ACT_VERSION ?= latest
RATCHET_VERSION ?= v0.11.4

## Versioned Binaries (the actual files that 'make' will check for)
CONTROLLER_GEN_V_BINARY := $(LOCALBIN)/controller-gen-$(CONTROLLER_GEN_VERSION)
KUSTOMIZE_V_BINARY := $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)
YQ_V_BINARY := $(LOCALBIN)/yq-$(YQ_VERSION)
HELM_V_BINARY := $(LOCALBIN)/helm-$(HELM_VERSION)
KIND_V_BINARY := $(LOCALBIN)/kind-$(KIND_VERSION)
GOLANGCI_LINT_V_BINARY := $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
ACT_V_BINARY := $(LOCALBIN)/act-$(ACT_VERSION)
RATCHET_V_BINARY := $(LOCALBIN)/ratchet-$(RATCHET_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN_V_BINARY) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN_V_BINARY): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_GEN_VERSION))

.PHONY: kustomize
kustomize: $(KUSTOMIZE_V_BINARY) ## Download kustomize locally if necessary.
$(KUSTOMIZE_V_BINARY): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4,$(KUSTOMIZE_VERSION))

.PHONY: yq
yq: $(YQ_V_BINARY) ## Download yq locally if necessary.
$(YQ_V_BINARY): $(LOCALBIN)
	$(call go-install-tool,$(YQ),github.com/mikefarah/yq/v4,$(YQ_VERSION))

.PHONY: helm
helm: $(HELM_V_BINARY) ## Download helm locally if necessary.
$(HELM_V_BINARY): $(LOCALBIN)
	$(call go-install-tool,$(HELM),helm.sh/helm/v3/cmd/helm,$(HELM_VERSION))

.PHONY: kind
kind: $(KIND_V_BINARY) ## Download kind locally if necessary.
$(KIND_V_BINARY): $(LOCALBIN)
	$(call go-install-tool,$(KIND),sigs.k8s.io/kind,$(KIND_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT_V_BINARY) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT_V_BINARY): $(LOCALBIN)
	@[ -f "$(GOLANGCI_LINT)-$(GOLANGCI_LINT_VERSION)" ] || { \
	set -e ;\
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/$(GOLANGCI_LINT_VERSION)/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION) ;\
	mv $(GOLANGCI_LINT) $(GOLANGCI_LINT)-$(GOLANGCI_LINT_VERSION) ;\
	} ;\
	ln -sf $(shell basename $(GOLANGCI_LINT))-$(GOLANGCI_LINT_VERSION) $(GOLANGCI_LINT)

.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download ginkgo locally if necessary.
$(GINKGO): $(LOCALBIN) go.mod
	# In order to make sure the version of the ginkgo cli installed
	# is the same as the version of go.mod,
	# instead of calling go-install-tool,
	# running go install from the current module will pick version from current go.mod file.
	@GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo

.PHONY: act
act: $(ACT_V_BINARY) ## Download act locally if necessary.
$(ACT_V_BINARY): $(LOCALBIN)
	$(call go-install-tool,$(ACT),github.com/nektos/act,$(ACT_VERSION))

.PHONY: ratchet
ratchet: $(RATCHET_V_BINARY) ## Download ratchet locally if necessary.
$(RATCHET_V_BINARY): $(LOCALBIN)
	$(call go-install-tool,$(RATCHET),github.com/sethvargo/ratchet,$(RATCHET_VERSION))

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

ifdef VERBOSE
test-integration: VERBOSE_FLAG = -v
endif
test-integration: clean-cov generate fmt vet ginkgo ## Run Integration tests.
	mkdir -p $(PROJECT_PATH)/coverage/integration
#	Check `ginkgo help run` for command line options. For example to filtering tests.
	$(GINKGO) $(VERBOSE_FLAG) \
		--coverpkg $(INTEGRATION_COVER_PKGS) \
		--output-dir $(PROJECT_PATH)/coverage/integration \
		--coverprofile cover.out \
		--compilers=$(INTEGRATION_TEST_NUM_CORES) \
		--procs=$(INTEGRATION_TEST_NUM_PROCESSES) \
		--randomize-all \
		--randomize-suites \
		--fail-on-pending \
		--keep-going \
		--race \
		--trace \
		$(INTEGRATION_TESTS_EXTRA_ARGS) $(INTEGRATION_TEST_SUITE_PATHS)

ifdef TEST_NAME
test-unit: TEST_PATTERN := --run $(TEST_NAME)
endif
ifdef VERBOSE
test-unit: VERBOSE_FLAG = -v
endif
test-unit: clean-cov generate fmt vet ## Run Unit tests.
	mkdir -p $(PROJECT_PATH)/coverage/unit
	go test $(UNIT_DIRS) -coverprofile $(PROJECT_PATH)/coverage/unit/cover.out $(VERBOSE_FLAG) -timeout 0 $(TEST_PATTERN) -race

##@ Build
build: GIT_SHA=$(shell git rev-parse HEAD || echo "unknown")
build: DIRTY=$(shell $(PROJECT_PATH)/utils/check-git-dirty.sh || echo "unknown")
build: generate fmt vet ## Build manager binary.
	   go build -ldflags "-X main.version=v$(VERSION) -X main.gitSHA=${GIT_SHA} -X main.dirty=${DIRTY}" -o bin/manager main.go

run: export LOG_LEVEL = debug
run: export LOG_MODE = development
run: GIT_SHA=$(shell git rev-parse HEAD || echo "unknown")
run: DIRTY=$(shell $(PROJECT_PATH)/utils/check-git-dirty.sh || echo "unknown")
run: manifests generate fmt vet ## Run a controller from your host.)
	go run -ldflags "-X main.version=v$(VERSION) -X main.gitSHA=${GIT_SHA} -X main.dirty=${DIRTY}" ./main.go $(MAIN_EXTRA_ARGS)

docker-build: GIT_SHA=$(shell git rev-parse HEAD || echo "unknown")
docker-build: DIRTY=$(shell $(PROJECT_PATH)/utils/check-git-dirty.sh || echo "unknown")
docker-build: ## Build docker image with the manager.
	docker build --build-arg VERSION=v$(VERSION) --build-arg GIT_SHA=$(GIT_SHA) --build-arg DIRTY=$(DIRTY) --build-arg QUAY_IMAGE_EXPIRY=$(QUAY_IMAGE_EXPIRY) -t $(IMG) .

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

.PHONY: prepare-release
prepare-release: IMG_TAG=v$(VERSION)
prepare-release: ## Prepare the manifests for the Helm Chart for a release.
	printf '%s\n' \
		'#Release default values' \
		'LIMITADOR_VERSION?=$(LIMITADOR_VERSION)' \
		'IMAGE_TAG?=$(IMG_TAG)' \
		'IMG?=$(IMAGE_TAG_BASE):$$(IMAGE_TAG)' \
		'VERSION?=$(VERSION)' \
		> $(RELEASE_FILE)
	$(MAKE) helm-build VERSION=$(VERSION) LIMITADOR_VERSION=$(LIMITADOR_VERSION)

.PHONY: read-release-version
read-release-version: ## Reads release version
	@echo "v$(VERSION)"

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
run-lint: golangci-lint ## Run lint tests
	$(GOLANGCI_LINT) run

.PHONY: ratchet-pin
ratchet-pin: ratchet ## Pin GitHub Actions to commit SHAs.
	$(RATCHET) pin $$(find .github/workflows -name '*.yaml' -o -name '*.yml')

.PHONY: ratchet-update-all
ratchet-update-all: ratchet ## Update all pinned GitHub Actions to latest SHAs.
	$(RATCHET) update $$(find .github/workflows -name '*.yaml' -o -name '*.yml')

# Include last to avoid changing MAKEFILE_LIST used above
include ./make/*.mk
