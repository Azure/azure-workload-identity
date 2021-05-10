REGISTRY ?= aramase
IMAGE_VERSION ?= v0.0.1
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
CONTROLLER_GEN_VER := v0.5.0
CONTROLLER_GEN_BIN := controller-gen
CONTROLLER_GEN := $(TOOLS_BIN_DIR)/$(CONTROLLER_GEN_BIN)-$(CONTROLLER_GEN_VER)

E2E_TEST_BIN := e2e.test
E2E_TEST := $(BIN_DIR)/$(E2E_TEST_BIN)

GINKGO_VER := v1.16.2
GINKGO_BIN := ginkgo
GINKGO := $(TOOLS_BIN_DIR)/$(GINKGO_BIN)-$(GINKGO_VER)

KUBECTL_VER := v1.20.2
KUBECTL_BIN := kubectl
KUBECTL := $(TOOLS_BIN_DIR)/$(KUBECTL_BIN)-$(KUBECTL_VER)

KUSTOMIZE_VER := v4.1.2
KUSTOMIZE_BIN := kustomize
KUSTOMIZE := $(TOOLS_BIN_DIR)/$(KUSTOMIZE_BIN)-$(KUSTOMIZE_VER)

GOLANGCI_LINT_VER := v1.38.0
GOLANGCI_LINT_BIN := golangci-lint
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/$(GOLANGCI_LINT_BIN)-$(GOLANGCI_LINT_VER)

# Scripts
GO_INSTALL := ./hack/go-install.sh

KIND_CLUSTER_NAME ?= aad-pod-managed-identity

.PHONY: build-proxy
build-proxy:
	CGO_ENABLED=0 GOOS=linux go build -a -o bin/proxy ./cmd/proxy

## --------------------------------------
## Containers
## --------------------------------------

OUTPUT_TYPE ?= type=registry

.PHONY: container-proxy
container-proxy:
	docker buildx build --no-cache -t $(PROXY_IMAGE_TAG) -f docker/proxy.Dockerfile --platform="linux/amd64" --output=$(OUTPUT_TYPE) .

.PHONY: container-init
container-init:
	docker buildx build --no-cache -t $(INIT_IMAGE_TAG) -f docker/init.Dockerfile --platform="linux/amd64" --output=$(OUTPUT_TYPE) .

.PHONY: container-manager
container-manager:
	docker buildx build --no-cache -t $(IMG) -f docker/webhook.Dockerfile --platform="linux/amd64" --output=$(OUTPUT_TYPE) .

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

.PHONY: all
all: manager

# Build manager binary
.PHONY: manager
manager: generate fmt vet
	go build -o bin/manager cmd/webhook/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: generate fmt vet manifests
	go run .cmd/webhook/main.go

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
.PHONY: deploy
deploy: $(KUBECTL) $(KUSTOMIZE)
	$(MAKE) manifests install-cert-manager
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

## --------------------------------------
## Code Generation
## --------------------------------------

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..."

# Generate code
.PHONY: generate
generate: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

## --------------------------------------
## Tooling Binaries and Manifests
## --------------------------------------

$(CONTROLLER_GEN):
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) sigs.k8s.io/controller-tools/cmd/controller-gen $(CONTROLLER_GEN_BIN) $(CONTROLLER_GEN_VER)

$(GINKGO):
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/onsi/ginkgo/ginkgo $(GINKGO_BIN) $(GINKGO_VER)

$(KUSTOMIZE):
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) sigs.k8s.io/kustomize/kustomize/$(shell echo $(KUSTOMIZE_VER) | cut -d'.' -f1) $(KUSTOMIZE_BIN) $(KUSTOMIZE_VER)

$(KUBECTL):
	mkdir -p $(TOOLS_BIN_DIR)
	rm -f "$(KUBECTL)*"
	curl -sfL https://storage.googleapis.com/kubernetes-release/release/$(KUBECTL_VER)/bin/$(shell go env GOOS)/$(shell go env GOARCH)/kubectl -o $(KUBECTL)
	ln -sf "$(KUBECTL)" "$(TOOLS_BIN_DIR)/$(KUBECTL_BIN)"
	chmod +x "$(TOOLS_BIN_DIR)/$(KUBECTL_BIN)" "$(KUBECTL)"

$(GOLANGCI_LINT):
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/golangci/golangci-lint/cmd/golangci-lint $(GOLANGCI_LINT_BIN) $(GOLANGCI_LINT_VER)

CERT_MANAGER_VERSION ?= v1.2.0

# Install cert manager in the cluster
.PHONY: install-cert-manager
install-cert-manager: $(KUBECTL)
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/$(CERT_MANAGER_VERSION)/cert-manager.yaml
	$(KUBECTL) wait --for=condition=Available --timeout=5m -n cert-manager deployment/cert-manager
	$(KUBECTL) wait --for=condition=Available --timeout=5m -n cert-manager deployment/cert-manager-cainjector
	$(KUBECTL) wait --for=condition=Available --timeout=5m -n cert-manager deployment/cert-manager-webhook

## --------------------------------------
## Testing
## --------------------------------------

# Run go fmt against code
.PHONY: fmt
fmt:
	go fmt ./...

# Run go vet against code
.PHONY: vet
vet:
	go vet ./...

# Run tests
.PHONY: test
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

$(E2E_TEST):
	go test -tags=e2e -c ./test/e2e -o $(E2E_TEST)

# Ginkgo configurations
GINKGO_FOCUS ?=
GINKGO_SKIP ?=
GINKGO_NODES ?= 1
GINKGO_NO_COLOR ?= false
GINKGO_ARGS ?=

# E2E configurations
E2E_ARGS ?=
KUBECONFIG ?= $(HOME)/.kube/config

.PHONY: test-e2e-run
test-e2e-run: $(E2E_TEST) $(GINKGO)
	$(GINKGO) -v -trace \
		-focus="$(GINKGO_FOCUS)" \
		-skip="$(GINKGO_SKIP)" \
		-nodes=$(GINKGO_NODES) \
		-noColor=$(GINKGO_NO_COLOR) \
		$(E2E_TEST) -- -kubeconfig=$(KUBECONFIG) $(E2E_ARGS)

.PHONY: test-e2e
test-e2e: $(KUBECTL)
	./scripts/ci-e2e.sh

## --------------------------------------
## Kind
## --------------------------------------

.PHONY: kind-create
kind-create: $(KUBECTL)
	kind create cluster --name $(KIND_CLUSTER_NAME) --image kindest/node:v1.20.2
	$(KUBECTL) wait node "$(KIND_CLUSTER_NAME)-control-plane" --for=condition=Ready --timeout=90s

.PHONY: kind-delete
kind-delete:
	kind delete cluster --name=$(KIND_CLUSTER_NAME) || true

## --------------------------------------
## Cleanup
## --------------------------------------

.PHONY: clean
clean:
	@rm -rf $(BIN_DIR)

## --------------------------------------
## Linting
## --------------------------------------

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run -v

lint-full: $(GOLANGCI_LINT) ## Run slower linters to detect possible issues
	$(GOLANGCI_LINT) run -v --fast=false
