##@ Helm Charts

.PHONY: helm-build
helm-build: ## Build the helm chart from kustomize manifests
	# Generate kustomize manifests out of code notations
	$(OPERATOR_SDK) generate kustomize manifests -q
	# Set desired operator image and related limitador image
	V="$(RELATED_IMAGE_LIMITADOR)" $(YQ) eval '(select(.kind == "Deployment").spec.template.spec.containers[].env[] | select(.name == "RELATED_IMAGE_LIMITADOR").value) = strenv(V)' -i config/manager/manager.yaml
	# Replace the controller image
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	# Build the helm chart templates from kustomize manifests
	$(KUSTOMIZE) build config/helm > charts/limitador-operator/templates/manifests.yaml

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
