##@ Operator Catalog

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:$(IMAGE_TAG)

CATALOG_FILE = $(PROJECT_PATH)/catalog/limitador-operator-catalog/operator.yaml
CATALOG_DOCKERFILE = $(PROJECT_PATH)/catalog/limitador-operator-catalog.Dockerfile

# Quay image default expiry
QUAY_IMAGE_EXPIRY ?= never

# A LABEL that can be appended to a generated Dockerfile to set the Quay image expiration through Docker arguments.
define QUAY_EXPIRY_TIME_LABEL

# Quay image expiry
ARG QUAY_IMAGE_EXPIRY=never
LABEL quay.expires-after=$${QUAY_IMAGE_EXPIRY}
endef
export QUAY_EXPIRY_TIME_LABEL

$(CATALOG_DOCKERFILE): $(OPM)
	-mkdir -p $(PROJECT_PATH)/catalog/limitador-operator-catalog
	cd $(PROJECT_PATH)/catalog && $(OPM) generate dockerfile limitador-operator-catalog -l quay.expires-after=$(QUAY_IMAGE_EXPIRY)
catalog-dockerfile: $(CATALOG_DOCKERFILE) ## Generate catalog dockerfile.

$(CATALOG_FILE): $(OPM) $(YQ)
	@echo "************************************************************"
	@echo Build limitador operator catalog
	@echo
	@echo BUNDLE_IMG                   = $(BUNDLE_IMG)
	@echo CHANNEL                      = $(DEFAULT_CHANNEL)
	@echo "************************************************************"
	@echo
	@echo Please check this matches your expectations and override variables if needed.
	@echo
	$(PROJECT_PATH)/utils/generate-catalog.sh $(OPM) $(YQ) $(BUNDLE_IMG) $@ $(DEFAULT_CHANNEL)

.PHONY: catalog
catalog: $(OPM) ## Generate catalog content and validate.
	# Initializing the Catalog
	-rm -rf $(PROJECT_PATH)/catalog/limitador-operator-catalog
	-rm -rf $(PROJECT_PATH)/catalog/limitador-operator-catalog.Dockerfile
	$(MAKE) $(CATALOG_DOCKERFILE) DEFAULT_CHANNEL=$(DEFAULT_CHANNEL)
	$(MAKE) $(CATALOG_FILE) BUNDLE_IMG=$(BUNDLE_IMG)
	cd $(PROJECT_PATH)/catalog && $(OPM) validate limitador-operator-catalog

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# Ref https://olm.operatorframework.io/docs/tasks/creating-a-catalog/#catalog-creation-with-raw-file-based-catalogs
.PHONY: catalog-build
catalog-build: ## Build a catalog image.
	# Build the Catalog
	docker build $(PROJECT_PATH)/catalog -f $(PROJECT_PATH)/catalog/limitador-operator-catalog.Dockerfile -t $(CATALOG_IMG)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

deploy-catalog: $(KUSTOMIZE) $(YQ) ## Deploy operator to the K8s cluster specified in ~/.kube/config using OLM catalog image.
	V="$(CATALOG_IMG)" $(YQ) eval '.spec.image = strenv(V)' -i config/deploy/olm/catalogsource.yaml
	V="$(DEFAULT_CHANNEL)" $(YQ) eval '.spec.channel = strenv(V)' -i config/deploy/olm/subscription.yaml
	$(KUSTOMIZE) build config/deploy/olm | kubectl apply -f -

undeploy-catalog: $(KUSTOMIZE) ## Undeploy controller from the K8s cluster specified in ~/.kube/config using OLM catalog image.
	$(KUSTOMIZE) build config/deploy/olm | kubectl delete -f -
