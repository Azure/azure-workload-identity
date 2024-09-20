#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

: "${CLUSTER_NAME:?Environment variable empty or not defined.}"

get_random_region() {
    local REGIONS=("eastus" "eastus2" "southcentralus" "westeurope" "uksouth" "northeurope" "francecentral")
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

main() {
  if [[ "$(should_create_aks_cluster)" == "true" ]]; then
    echo "Creating an AKS cluster '${CLUSTER_NAME}'"
    LOCATION="$(get_random_region)"
    az group create --name "${CLUSTER_NAME}" --location "${LOCATION}" > /dev/null
    # TODO(chewong): ability to create an arc-enabled cluster
    az aks create \
      --resource-group "${CLUSTER_NAME}" \
      --name "${CLUSTER_NAME}" \
      --node-vm-size Standard_DS3_v2 \
      --enable-managed-identity \
      --network-plugin azure \
      --node-count 3 \
      --generate-ssh-keys \
      --enable-oidc-issuer > /dev/null
    if [[ "${WINDOWS_CLUSTER:-}" == "true" ]]; then
      # shellcheck disable=SC2086
      az aks nodepool add --resource-group "${CLUSTER_NAME}" --cluster-name "${CLUSTER_NAME}" --os-type Windows --name npwin --node-count 3 ${EXTRA_ARGS:-} > /dev/null
    fi
  fi
}

main
