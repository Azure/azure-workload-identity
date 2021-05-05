REGISTRY?=aramase
IMAGE_VERSION?=v0.0.1
PROXY_IMAGE_NAME := pod-identity-proxy
INIT_IMAGE_NAME := proxy-init

PROXY_IMAGE_TAG := $(REGISTRY)/$(PROXY_IMAGE_NAME):$(IMAGE_VERSION)
INIT_IMAGE_TAG := $(REGISTRY)/$(INIT_IMAGE_NAME):$(IMAGE_VERSION)


# Directories
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BIN_DIR := $(abspath $(ROOT_DIR)/bin)
TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(abspath $(TOOLS_DIR)/bin)

# Binaries
E2E_TEST_BIN := e2e.test
E2E_TEST := $(BIN_DIR)/$(E2E_TEST_BIN)

GINKGO_VER := v1.16.2
GINKGO_BIN := ginkgo
GINKGO := $(TOOLS_BIN_DIR)/$(GINKGO_BIN)-$(GINKGO_VER)

# Scripts:
GO_INSTALL = ./hack/go_install.sh

# Ginkgo configurations
GINKGO_FOCUS ?=
GINKGO_SKIP ?=
GINKGO_NODES ?= 3
GINKGO_NO_COLOR ?= false
GINKGO_ARGS ?=

# E2E configurations
E2E_ARGS ?=

build-proxy:
	CGO_ENABLED=0 GOOS=linux go build -a -o _output/proxy ./cmd/proxy

container-proxy:
	docker buildx build --no-cache -t $(PROXY_IMAGE_TAG) -f docker/proxy.Dockerfile --platform="linux/amd64" --push .

container-init:
	docker buildx build --no-cache -t $(INIT_IMAGE_TAG) -f docker/init.Dockerfile --platform="linux/amd64" --push .

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager cmd/webhook/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run .cmd/webhook/main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..."

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG} -f docker/webhook.Dockerfile

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

# Install cert manager in the cluster
install-cert-manager:
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.2.0/cert-manager.yaml

$(E2E_TEST):
	go test -c ./test/e2e -o $(E2E_TEST)

$(GINKGO):
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/onsi/ginkgo/ginkgo $(GINKGO_BIN) $(GINKGO_VER)

.PHONY: test-e2e-run
test-e2e-run: $(E2E_TEST) $(GINKGO)
	$(GINKGO) -v -trace \
		-focus="$(GINKGO_FOCUS)" \
		-skip="$(GINKGO_SKIP)" \
		-nodes=$(GINKGO_NODES) \
		-noColor=$(GINKGO_NO_COLOR) \
		$(E2E_TEST) -- $(E2E_ARGS)

# TODO(chewong): include cluster creation and component installation
.PHONY: test-e2e
test-e2e:
	@echo "no op"
	$(MAKE) test-e2e-run

.PHONY: clean
clean:
	@rm -rf bin/
