#Release default values
LIMITADOR_VERSION?=2.4.2
IMAGE_TAG?=v0.18.3
IMG?=quay.io/kuadrant/limitador-operator:$(IMAGE_TAG)
BUNDLE_IMG?=quay.io/kuadrant/limitador-operator-bundle:$(IMAGE_TAG)
CATALOG_IMG?=quay.io/kuadrant/limitador-operator-catalog:$(IMAGE_TAG)
CHANNELS?=stable
BUNDLE_CHANNELS?=--channels=stable
VERSION?=0.18.3
