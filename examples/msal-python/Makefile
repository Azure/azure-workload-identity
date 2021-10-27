REGISTRY ?= ghcr.io/azure/azure-workload-identity
IMAGE_NAME := msal-python
IMAGE_VERSION ?= latest

DEMO_IMAGE := $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)

## --------------------------------------
## Images
## --------------------------------------

# Output type of docker buildx build
OUTPUT_TYPE ?= type=registry

ALL_OS = linux
ALL_ARCH.linux = amd64 arm64
ALL_OS_ARCH.linux = $(foreach arch, ${ALL_ARCH.linux}, linux-$(arch))
ALL_OS_ARCH = $(foreach os, $(ALL_OS), ${ALL_OS_ARCH.${os}})

# The architecture of the image
ARCH ?= amd64

.PHONY: container-linux
container-linux:
	docker buildx build \
		--output=$(OUTPUT_TYPE) \
		--platform="linux/$(ARCH)" \
		--tag=$(DEMO_IMAGE)-linux-$(ARCH) .

.PHONY: container-all
container-all:
	for arch in $(ALL_ARCH.linux); do \
		ARCH=$${arch} $(MAKE) container-linux; \
	done

.PHONY: push-manifest
push-manifest:
	docker manifest create --amend $(DEMO_IMAGE) $(foreach osarch, $(ALL_OS_ARCH), $(DEMO_IMAGE)-${osarch})
	for arch in $(ALL_ARCH.linux); do docker manifest annotate --os linux --arch $${arch} $(DEMO_IMAGE) $(DEMO_IMAGE)-linux-$${arch}; done; \
	docker manifest push --purge $(DEMO_IMAGE)
