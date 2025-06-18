#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

: "${AZURE_TENANT_ID:?Environment variable empty or not defined.}"

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}" || exit 1

readonly CLUSTER_NAME="${CLUSTER_NAME:-azwi-e2e-$(openssl rand -hex 2)}"
readonly KUBECTL="${REPO_ROOT}/hack/tools/bin/kubectl"

IMAGE_VERSION="$(git describe --tags --always)"
export IMAGE_VERSION

create_cluster() {
  if [[ "${LOCAL_ONLY:-}" == "true" ]]; then
    # create a kind cluster, then build and load the webhook manager image to the kind cluster
    make kind-create
    # only build amd64 images for now
    OUTPUT_TYPE="type=docker" ALL_LINUX_ARCH="amd64" make docker-build docker-build-e2e-msal-go
    make kind-load-images
  else
    : "${REGISTRY:?Environment variable empty or not defined.}"

    "${REPO_ROOT}/scripts/create-aks-cluster.sh"

    # assume BYO cluster if KUBECONFIG is defined
    if [[ -z "${KUBECONFIG:-}" ]]; then
      az aks get-credentials --resource-group "${CLUSTER_NAME}" --name "${CLUSTER_NAME}" --overwrite-existing
    fi

    # assume one windows node for now
    WINDOWS_NODE_NAME="$(${KUBECTL} get node --selector=kubernetes.io/os=windows -ojson | jq -r '.items[0].metadata.name')"
    if [[ "${WINDOWS_NODE_NAME}" == "null" ]]; then
      unset WINDOWS_NODE_NAME
    fi

    if [[ "${REGISTRY}" =~ \.azurecr\.io ]]; then
      az acr login --name "${REGISTRY}"
    fi

    # build webhook manager and msal-go-e2e images
    ALL_LINUX_ARCH="amd64" make docker-build docker-push-manifest docker-build-e2e-msal-go
  fi
  ${KUBECTL} get nodes -owide
}

cleanup() {
  if [[ "${SOAK_CLUSTER:-}" == "true" ]] || [[ "${SKIP_CLEANUP:-}" == "true" ]]; then
    return
  fi
  if [[ "${LOCAL_ONLY:-}" == "true" ]]; then
    make kind-delete
    return
  fi
  az group delete --name "${CLUSTER_NAME}" --yes --no-wait || true
}
trap cleanup EXIT

main() {
  az login -i > /dev/null && echo "Using machine identity for az commands" || echo "Using pre-existing credential for az commands"
  az account set --subscription "${AZURE_SUBSCRIPTION_ID}" > /dev/null

  create_cluster
  make deploy
  poll_webhook_readiness

  if [[ -n "${WINDOWS_NODE_NAME:-}" ]]; then
    E2E_ARGS="--node-os-distro=windows ${E2E_ARGS:-}"
    export E2E_ARGS
  fi

  make test-e2e-run

  if [[ "${TEST_HELM_CHART:-}" == "true" ]]; then
    make uninstall-deploy
    test_helm_chart
  fi

  test_version_flag "webhook"
  test_version_flag "proxy"
}

test_helm_chart() {
  readonly HELM="${REPO_ROOT}/hack/tools/bin/helm"

  ${HELM} repo add azure-workload-identity https://azure.github.io/azure-workload-identity/charts
  ${HELM} repo update
  ${HELM} install workload-identity-webhook azure-workload-identity/workload-identity-webhook \
    --set azureTenantID="${AZURE_TENANT_ID}" \
    --namespace azure-workload-identity-system \
    --create-namespace \
    --wait \
    --debug \
    -v=5 \
    --devel
  poll_webhook_readiness
  GINKGO_SKIP=Proxy make test-e2e-run

  ${HELM} upgrade --install workload-identity-webhook "${REPO_ROOT}/manifest_staging/charts/workload-identity-webhook" \
    --set image.repository="${REGISTRY:-mcr.microsoft.com/oss/azure/workload-identity/webhook}" \
    --set image.release="${IMAGE_VERSION}" \
    --set azureTenantID="${AZURE_TENANT_ID}" \
    --namespace azure-workload-identity-system \
    --reuse-values \
    -f "${REPO_ROOT}/manifest_staging/charts/workload-identity-webhook/values.yaml" \
    --wait \
    --debug \
    -v=5
  poll_webhook_readiness
  make test-e2e-run
}

poll_webhook_readiness() {
  ${KUBECTL} wait --for=condition=available --timeout=5m deployment/azure-wi-webhook-controller-manager -n azure-workload-identity-system
}

test_version_flag() {
  echo "Testing version flag for $1 image..."
  local image_name="$1"
  message=$(docker run --platform linux/amd64 --quiet "${REGISTRY:-mcr.microsoft.com/oss/azure/workload-identity}/${image_name}:${IMAGE_VERSION}" --version 2>&1)

  # Validate that it's valid JSON
  if ! echo "$message" | jq empty > /dev/null 2>&1; then
    echo "Invalid JSON output: $message"
    return 1
  fi

  # Extract and validate fields
  build_version=$(echo "$message" | jq -r .buildVersion)
  git_commit=$(echo "$message" | jq -r .gitCommit)
  build_date=$(echo "$message" | jq -r .buildDate)
  go_version=$(echo "$message" | jq -r .goVersion)
  platform=$(echo "$message" | jq -r .platform)

  if [[ -z "$build_version" || "$build_version" == "null" ]]; then
    echo "Missing BuildVersion"
    return 1
  fi

  if [[ "$build_version" != "$IMAGE_VERSION" ]]; then
    echo "BuildVersion does not match IMAGE_VERSION: $build_version != $IMAGE_VERSION"
    return 1
  fi

  if [[ -z "$git_commit" || "$git_commit" == "null" ]]; then
    echo "Missing GitCommit"
    return 1
  fi

  if [[ -z "$build_date" || "$build_date" == "null" ]]; then
    echo "Missing BuildDate"
    return 1
  fi

  if [[ -z "$go_version" || "$go_version" == "null" ]]; then
    echo "Missing GoVersion"
    return 1
  fi

  if [[ -z "$platform" || "$platform" == "null" ]]; then
    echo "Missing Platform"
    return 1
  fi

  echo "Version info looks good:"
  echo "$message"
}

main
