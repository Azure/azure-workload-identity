REGISTRY ?= ghcr.io/azure/azure-workload-identity
IMAGE_NAME := msal-go
IMAGE_VERSION ?= latest

DEMO_IMAGE := $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)

## --------------------------------------
## Images
## --------------------------------------

# Output type of docker buildx build
OUTPUT_TYPE ?= type=registry

ALL_OS = linux windows
ALL_ARCH.linux = amd64 arm64
ALL_ARCH.windows = amd64
ALL_OSVERSIONS.windows := 1809 ltsc2022
ALL_OS_ARCH.linux = $(foreach arch, ${ALL_ARCH.linux}, linux-$(arch))
ALL_OS_ARCH.windows = $(foreach osver, ${ALL_OSVERSIONS.windows}, windows-$(osver)-$(foreach arch, ${ALL_ARCH.windows},$(arch)))
ALL_OS_ARCH = $(foreach os, $(ALL_OS), ${ALL_OS_ARCH.${os}})

# The architecture of the image
ARCH ?= amd64
# OS Version for the Windows images: 1809, ltsc2022
OSVERSION ?= 1809

.PHONY: container-linux
container-linux:
	docker buildx build \
		--output=$(OUTPUT_TYPE) \
		--platform="linux/$(ARCH)" \
		--tag=$(DEMO_IMAGE)-linux-$(ARCH) .

.PHONY: container-windows
container-windows:
	docker buildx build \
		--build-arg OS_VERSION=$(OSVERSION) \
		--output=$(OUTPUT_TYPE) \
		--platform="windows/$(ARCH)" \
		--file=windows.Dockerfile \
		--tag=$(DEMO_IMAGE)-windows-$(OSVERSION)-$(ARCH) .

.PHONY: container-all
container-all:
	for arch in $(ALL_ARCH.linux); do \
		ARCH=$${arch} $(MAKE) container-linux; \
	done
	for osversion in $(ALL_OSVERSIONS.windows); do \
  		OSVERSION=$${osversion} $(MAKE) container-windows; \
  	done

.PHONY: push-manifest
push-manifest:
	docker manifest create --amend $(DEMO_IMAGE) $(foreach osarch, $(ALL_OS_ARCH), $(DEMO_IMAGE)-${osarch})
	for arch in $(ALL_ARCH.linux); do docker manifest annotate --os linux --arch $${arch} $(DEMO_IMAGE) $(DEMO_IMAGE)-linux-$${arch}; done; \
	set -x; \
	for arch in $(ALL_ARCH.windows); do \
		for osversion in $(ALL_OSVERSIONS.windows); do \
			BASEIMAGE=mcr.microsoft.com/windows/nanoserver:$${osversion}; \
			full_version=`docker manifest inspect $${BASEIMAGE} | jq -r '.manifests[0].platform["os.version"]'`; \
			docker manifest annotate --os windows --arch $${arch} --os-version $${full_version} $(DEMO_IMAGE) $(DEMO_IMAGE)-windows-$${osversion}-$${arch}; \
		done; \
	done; \
	docker manifest push --purge $(DEMO_IMAGE)
