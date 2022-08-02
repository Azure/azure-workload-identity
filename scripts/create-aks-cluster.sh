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

register_feature() {
  # pinning to 0.5.87 because of https://github.com/Azure/azure-cli/issues/23267
  az extension add --name aks-preview --version 0.5.87
  # register enable oidc preview feature
  az feature register --namespace Microsoft.ContainerService --name EnableOIDCIssuerPreview > /dev/null
  # https://docs.microsoft.com/en-us/azure/aks/windows-container-cli#add-a-windows-server-node-pool-with-containerd-preview
  az feature register --namespace Microsoft.ContainerService --name UseCustomizedWindowsContainerRuntime > /dev/null
  while [[ "$(az feature list --query "[?contains(name, 'Microsoft.ContainerService/EnableOIDCIssuerPreview')].{Name:name,State:properties.state}" | jq -r '.[].State')" != "Registered" ]] &&
    [[ "$(az feature list --query "[?contains(name, 'Microsoft.ContainerService/UseCustomizedWindowsContainerRuntime')].{Name:name,State:properties.state}" | jq -r '.[].State')" != "Registered" ]]; do
      sleep 20
  done
  az provider register --namespace Microsoft.ContainerService
}

main() {
  if [[ "$(should_create_aks_cluster)" == "true" ]]; then
    export -f register_feature
    # might take around 20 minutes to register
    timeout --foreground 1200 bash -c register_feature
    echo "Creating an AKS cluster '${CLUSTER_NAME}'"
    LOCATION="$(get_random_region)"
    # get the latest patch version of 1.21
    KUBERNETES_VERSION="$(az aks get-versions --location "${LOCATION}" --query 'orchestrators[*].orchestratorVersion' -otsv | grep '1.21' | tail -1)"
    az group create --name "${CLUSTER_NAME}" --location "${LOCATION}" > /dev/null
    # TODO(chewong): ability to create an arc-enabled cluster
    az aks create \
      --resource-group "${CLUSTER_NAME}" \
      --name "${CLUSTER_NAME}" \
      --node-vm-size Standard_DS3_v2 \
      --enable-managed-identity \
      --network-plugin azure \
      --kubernetes-version "${KUBERNETES_VERSION}" \
      --node-count 3 \
      --generate-ssh-keys \
      --enable-oidc-issuer > /dev/null
    if [[ "${WINDOWS_CLUSTER:-}" == "true" ]]; then
      if [[ "${WINDOWS_CONTAINERD:-}" == "true" ]]; then
        EXTRA_ARGS="--aks-custom-headers WindowsContainerRuntime=containerd"
      fi
      # shellcheck disable=SC2086
      az aks nodepool add --resource-group "${CLUSTER_NAME}" --cluster-name "${CLUSTER_NAME}" --os-type Windows --name npwin --kubernetes-version "${KUBERNETES_VERSION}" --node-count 3 ${EXTRA_ARGS:-} > /dev/null
    fi
  fi
}

main
