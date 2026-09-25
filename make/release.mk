#Release default values
LIMITADOR_VERSION?=2.5.0
IMAGE_TAG?=v0.19.0
IMG?=quay.io/kuadrant/limitador-operator:$(IMAGE_TAG)
BUNDLE_IMG?=quay.io/kuadrant/limitador-operator-bundle:$(IMAGE_TAG)
CATALOG_IMG?=quay.io/kuadrant/limitador-operator-catalog:$(IMAGE_TAG)
CHANNELS?=stable
BUNDLE_CHANNELS?=--channels=stable
DEFAULT_CHANNEL?=stable
BUNDLE_DEFAULT_CHANNEL?=--default-channel=stable
VERSION?=0.19.0
