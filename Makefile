REGISTRY?=aramase
IMAGE_VERSION?=v0.0.1
PROXY_IMAGE_NAME := pod-identity-proxy
INIT_IMAGE_NAME := proxy-init

PROXY_IMAGE_TAG := $(REGISTRY)/$(PROXY_IMAGE_NAME):$(IMAGE_VERSION)
INIT_IMAGE_TAG := $(REGISTRY)/$(INIT_IMAGE_NAME):$(IMAGE_VERSION)

build-proxy:
	CGO_ENABLED=0 GOOS=linux go build -a -o _output/proxy ./cmd/proxy

container-proxy:
	docker buildx build --no-cache -t $(PROXY_IMAGE_TAG) -f docker/proxy.Dockerfile --platform="linux/amd64" --push .

container-init:
	docker buildx build --no-cache -t $(INIT_IMAGE_TAG) -f docker/init.Dockerfile --platform="linux/amd64" --push .
