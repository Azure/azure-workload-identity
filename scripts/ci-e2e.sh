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
      echo "Granting AcrPull permission to the cluster's managed identity"
      NODE_RESOURCE_GROUP="$(az aks show --resource-group "${CLUSTER_NAME}" --name "${CLUSTER_NAME}" --query nodeResourceGroup -otsv)"
      ASSIGNEE_OBJECT_ID="$(az identity show --resource-group "${NODE_RESOURCE_GROUP}" --name "${CLUSTER_NAME}-agentpool" --query principalId -otsv)"
      REGISTRY_SCOPE="$(az acr show --name "${REGISTRY}" --query id -otsv)"
      az role assignment create --assignee-object-id "${ASSIGNEE_OBJECT_ID}" --role AcrPull --scope "${REGISTRY_SCOPE}" > /dev/null
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
  if [[ -n "${ASSIGNEE_OBJECT_ID:-}" ]] && [[ -n "${REGISTRY_SCOPE:-}" ]]; then
      az role assignment delete --assignee "${ASSIGNEE_OBJECT_ID}" --scope "${REGISTRY_SCOPE}" || true
  fi
  az group delete --name "${CLUSTER_NAME}" --yes --no-wait || true
}
trap cleanup EXIT

main() {
  az login -i > /dev/null && echo "Using machine identity for az commands" || echo "Using pre-existing credential for az commands"

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
}

test_helm_chart() {
  readonly HELM="${REPO_ROOT}/hack/tools/bin/helm"

  # test helm upgrade from the latest released chart to manifest_staging/chart
  # TODO(chewong) switch to https://azure.github.io/azure-workload-identity/charts once it is available
  git checkout origin/gh-pages -- "${REPO_ROOT}/charts/"
  # shellcheck disable=SC2086
  LATEST_CHART_TARBALL="$(find ${REPO_ROOT}/charts/workload-identity-webhook-*.tgz | sort | tail -n 1)"
  ${HELM} install workload-identity-webhook "${LATEST_CHART_TARBALL}" \
    --set azureTenantID="${AZURE_TENANT_ID}" \
    --namespace azure-workload-identity-system \
    --create-namespace \
    --wait
  poll_webhook_readiness
  # TODO(aramase) remove the service account token expiration after v0.7.0 release
  E2E_EXTRA_ARGS=-e2e.service-account-token-expiration=24h make test-e2e-run

  ${HELM} upgrade --install workload-identity-webhook "${REPO_ROOT}/manifest_staging/charts/workload-identity-webhook" \
    --set image.repository="${REGISTRY:-mcr.microsoft.com/oss/azure/workload-identity/webhook}" \
    --set image.release="${IMAGE_VERSION}" \
    --set azureTenantID="${AZURE_TENANT_ID}" \
    --namespace azure-workload-identity-system \
    --reuse-values \
    --wait
  poll_webhook_readiness
  make test-e2e-run
}

poll_webhook_readiness() {
  TEST_RESOURCE=$(cat <<-EOF
apiVersion: v1
kind: Namespace
metadata:
  name: azure-workload-identity-system-test
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-service-account
  namespace: azure-workload-identity-system-test
  labels:
    azure.workload.identity/use: "true"
  annotations:
    azure.workload.identity/service-account-token-expiration: "100"
---
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
  namespace: azure-workload-identity-system-test
spec:
  serviceAccountName: test-service-account
  containers:
  - name: nginx
    image: nginx:1.15.8
EOF
)
  for _ in {1..30}; do
    # webhook is considered ready when it starts denying requests
    # with invalid service account token expiration
    if echo "${TEST_RESOURCE}" | ${KUBECTL} apply -f -; then
      echo "${TEST_RESOURCE}" | ${KUBECTL} delete --grace-period=1 --ignore-not-found -f -
      sleep 4
    else
      break
    fi
  done
  echo "${TEST_RESOURCE}" | ${KUBECTL} delete --ignore-not-found -f -
}

main
