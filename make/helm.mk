##@ Helm Charts

.PHONY: helm-build
helm-build: $(KUSTOMIZE) $(OPERATOR_SDK) $(YQ) manifests ## Build the helm chart from kustomize manifests
	# Set desired operator image and related limitador image
	V="$(RELATED_IMAGE_LIMITADOR)" $(YQ) eval '(select(.kind == "Deployment").spec.template.spec.containers[].env[] | select(.name == "RELATED_IMAGE_LIMITADOR").value) = strenv(V)' -i config/manager/manager.yaml
	# Replace the controller image
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	# Build the helm chart templates from kustomize manifests
	$(KUSTOMIZE) build config/helm > charts/limitador-operator/templates/manifests.yaml
	V="$(VERSION)" $(YQ) eval '.version = strenv(V)' -i charts/limitador-operator/Chart.yaml

.PHONY: helm-install
helm-install: $(HELM) ## Install the helm chart
	# Install the helm chart in the cluster
	$(HELM) install limitador-operator charts/limitador-operator

.PHONY: helm-uninstall
helm-uninstall: $(HELM) ## Uninstall the helm chart
	# Uninstall the helm chart from the cluster
	$(HELM) uninstall limitador-operator

.PHONY: helm-upgrade
helm-upgrade: $(HELM) ## Upgrade the helm chart
	# Upgrade the helm chart in the cluster
	$(HELM) upgrade limitador-operator charts/limitador-operator

.PHONY: helm-package
helm-package: $(HELM) ## Package the helm chart
	# Package the helm chart
	$(HELM) package charts/limitador-operator

# GitHub Token with permissions to upload to the release assets
HELM_WORKFLOWS_TOKEN ?= <YOUR-TOKEN>
# GitHub Release Asset Browser Download URL, it can be find in the output of the uploaded asset
BROWSER_DOWNLOAD_URL ?= <BROWSER-DOWNLOAD-URL>
# Github repo name for the helm charts repository
HELM_REPO_NAME ?= helm-charts
ifeq (0.0.0,$(VERSION))
CHART_VERSION = $(VERSION)-dev
else
CHART_VERSION = $(VERSION)
endif

.PHONY: helm-sync-package
helm-sync-package: $(HELM) ## Sync the helm chart package to the helm-charts repo
	curl -L \
	  -X POST \
	  -H "Accept: application/vnd.github+json" \
	  -H "Authorization: Bearer $(HELM_WORKFLOWS_TOKEN)" \
	  -H "X-GitHub-Api-Version: 2022-11-28" \
	  https://api.github.com/repos/$(ORG)/$(HELM_REPO_NAME)/dispatches \
	  -d '{"event_type":"sync-chart","client_payload":{"chart":"$(REPO_NAME)","version":"$(CHART_VERSION)", "browser_download_url": "$(BROWSER_DOWNLOAD_URL)"}}'
