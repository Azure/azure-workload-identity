#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}" || exit 1

readonly KUBECTL="${REPO_ROOT}/hack/tools/bin/kubectl"

IMAGE_VERSION="$(git rev-parse --short HEAD)"
export IMAGE_VERSION

get_random_region() {
    local REGIONS=("eastus" "eastus2" "westus2" "westeurope" "uksouth" "northeurope" "francecentral")
    echo "${REGIONS[${RANDOM} % ${#REGIONS[@]}]}"
}

should_create_aks_cluster() {
  if [[ "${SOAK_CLUSTER:-}" == "true" ]] || [[ -n "${KUBECONFIG:-}" ]]; then
    echo "false" && return
  fi
  if az aks show --resource-group "${CLUSTER_NAME}" --name "${CLUSTER_NAME}" > /dev/null; then
    echo "false" && return
  fi
  echo "true" && return
}

create_cluster() {
  if [[ "${LOCAL_ONLY:-}" == "true" ]]; then
    # create a kind cluster, then build and load the webhook manager image to the cluster
    make kind-create
  else
    : "${REGISTRY:?Environment variable empty or not defined.}"

    az login -i > /dev/null && echo "Using machine identity for az commands" || echo "Using pre-existing credential for az commands"

    CLUSTER_NAME="${CLUSTER_NAME:-pod-managed-identity-e2e-$(openssl rand -hex 2)}"
    if [[ "$(should_create_aks_cluster)" == "true" ]]; then
      echo "Creating an AKS cluster '${CLUSTER_NAME}'"
      az group create --name "${CLUSTER_NAME}" --location "$(get_random_region)" > /dev/null
      # TODO(chewong): ability to create an arc-enabled cluster
      az aks create \
        --resource-group "${CLUSTER_NAME}" \
        --name "${CLUSTER_NAME}" \
        --node-vm-size Standard_DS3_v2 \
        --enable-managed-identity \
        --network-plugin azure \
        --node-count 1 \
        --generate-ssh-keys > /dev/null
      if [[ "${WINDOWS_CLUSTER:-}" == "true" ]]; then
        az aks nodepool add \
          --resource-group "${CLUSTER_NAME}" \
          --cluster-name "${CLUSTER_NAME}" \
          --os-type Windows \
          --name npwin \
          --node-count 1 > /dev/null
      fi
    fi

    # assume BYO cluster if KUBECONFIG is defined
    if [[ -z "${KUBECONFIG:-}" ]]; then
      az aks get-credentials --resource-group "${CLUSTER_NAME}" --name "${CLUSTER_NAME}"
    fi

    # assume one windows node for now
    WINDOWS_NODE_NAME="$(${KUBECTL} get node --selector=kubernetes.io/os=windows -ojson | jq -r '.items[0].metadata.name')"
    if [[ "${WINDOWS_NODE_NAME}" == "null" ]]; then
      unset WINDOWS_NODE_NAME
    else
      # taint the windows node to prevent cert-manager pods from scheduling to it
      ${KUBECTL} taint nodes "${WINDOWS_NODE_NAME}" kubernetes.io/os=windows:NoSchedule
    fi

    if [[ "${REGISTRY}" =~ \.azurecr\.io ]]; then
      az acr login --name "${REGISTRY}"
      echo "Granting AcrPull permission to the cluster's managed identity"
      NODE_RESOURCE_GROUP="$(az aks show --resource-group "${CLUSTER_NAME}" --name "${CLUSTER_NAME}" --query nodeResourceGroup -otsv)"
      ASSIGNEE_OBJECT_ID="$(az identity show --resource-group "${NODE_RESOURCE_GROUP}" -n "${CLUSTER_NAME}-agentpool" --query principalId -otsv)"
      ROLE_ASSIGNMENT_ID="$(az role assignment create --assignee-object-id "${ASSIGNEE_OBJECT_ID}" --role AcrPull --scope "$(az acr show --name "${REGISTRY}" --query id -otsv)" --query id -otsv)"
    fi

    echo "Building controller and deploying webhook to the cluster"
    make docker-build-webhook
  fi
}

cleanup() {
  if [[ "${SOAK_CLUSTER:-}" == "true" ]] || [[ "${SKIP_CLEANUP:-}" == "true" ]]; then
    return
  fi
  if [[ "${LOCAL_ONLY:-}" == "true" ]]; then
    make kind-delete
    return
  fi
  if [[ -n "${ROLE_ASSIGNMENT_ID:-}" ]]; then
    az role assignment delete --ids "${ROLE_ASSIGNMENT_ID}"
  fi
  az group delete --name "${CLUSTER_NAME}" --yes --no-wait || true
}
trap cleanup EXIT

create_cluster
${KUBECTL} get nodes -owide

make clean deploy

if [[ -n "${WINDOWS_NODE_NAME:-}" ]]; then
  # remove the taint from the windows node we introduced above
  ${KUBECTL} taint nodes "${WINDOWS_NODE_NAME}" kubernetes.io/os=windows:NoSchedule-
  E2E_ARGS="--node-os-distro=windows ${E2E_ARGS:-}"
  export E2E_ARGS
fi

make test-e2e-run
