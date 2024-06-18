#!/usr/bin/env bash

# This script requires the following tools:
# - azure-cli : This is used for interacting with Azure services.
# - kind      : This is required if you need a kind cluster.
# - kubectl   : This is required and the context should be configured to the cluster if SKIP_CLUSTER=true.
# - openssl   : This is used to generate a random string.
# - jq        : This is used to process JSON data.
# 
# Note: A kind cluster with the same name will be deleted if it already exists.
# Please ensure you have these tools installed and configured correctly before running this script.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_PATH="$(dirname "${BASH_SOURCE[0]}")"
KIND_CLUSTER_NAME="azure-workload-identity"
KIND_IMAGE_VERSION="${KIND_IMAGE_VERSION:-v1.29.0}"

help() {
    echo "Usage: $0 [LOCATION] [RESOURCE_GROUP]"
    echo
    echo "Arguments:"
    echo "  LOCATION        The location for the Azure resources."
    echo "  RESOURCE_GROUP  The resource group for the Azure resources."
    echo
    echo "Environment variables:"
    echo "  SKIP_CLUSTER            If set to 'true', the script will skip the kind cluster creation. Default: false"
    echo "  KIND_CLUSTER_NAME       The name of the kind cluster. Default: ${KIND_CLUSTER_NAME}"
    echo "  KIND_IMAGE_VERSION      The version of the kind image. Default: ${KIND_IMAGE_VERSION}"
    echo
    echo "This script requires the following tools:"
    echo "  - azure-cli : This is used for interacting with Azure services."
    echo "  - kind      : This is required if you need a kind cluster."
    echo "  - kubectl   : This is required and the context should be configured to the cluster if SKIP_CLUSTER=true."
    echo "  - openssl   : This is used to generate a random string."
    echo "  - jq        : This is used to process JSON data."
    echo 
    echo "Note: A kind cluster with the same name will be deleted if it already exists."
    echo "Please ensure you have these tools installed and configured correctly before running this script."
}

if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    help
    exit 0
fi

LOCATION="${1}"
RESOURCE_GROUP="${2}"
AZURE_STORAGE_ACCOUNT="oidcissuer$(openssl rand -hex 4)"
# This $web container is a special container that serves static web content without requiring public access enablement.
# See https://learn.microsoft.com/en-us/azure/storage/blobs/storage-blob-static-website
AZURE_STORAGE_CONTAINER="\$web"

validate() {
    # check if user is logged into azure cli
    if ! az account show > /dev/null 2>&1; then
        echo "Please login to Azure CLI using 'az login'"
        exit 1
    fi

    # check if RESOURCE_GROUP and LOCATION are provided
    if [ -z "${RESOURCE_GROUP:-}" ] || [ -z "${LOCATION:-}" ]; then
        echo "RESOURCE_GROUP and LOCATION are required."
        exit 1
    fi
}

create_azure_blob_storage_account() {
    if [ "$(az group exists --name "${RESOURCE_GROUP}" --output tsv)" == 'false' ]; then
        echo "Creating resource group '${RESOURCE_GROUP}' in '${LOCATION}'"
        az group create --name "${RESOURCE_GROUP}" --location "${LOCATION}" --output none --only-show-errors
    fi

    if ! az storage account show --name "${AZURE_STORAGE_ACCOUNT}" --resource-group "${RESOURCE_GROUP}" > /dev/null 2>&1; then
        echo "Creating storage account '${AZURE_STORAGE_ACCOUNT}' in '${RESOURCE_GROUP}'"
        az storage account create --resource-group "${RESOURCE_GROUP}" --name "${AZURE_STORAGE_ACCOUNT}" --output none --only-show-errors
    fi

    # Enable static website serving
    az storage blob service-properties update --account-name "${AZURE_STORAGE_ACCOUNT}" --static-website --output none --only-show-errors

    if ! az storage container show --name "${AZURE_STORAGE_CONTAINER}" --account-name "${AZURE_STORAGE_ACCOUNT}" > /dev/null 2>&1; then
        echo "Creating storage container '${AZURE_STORAGE_CONTAINER}' in '${AZURE_STORAGE_ACCOUNT}'"
        az storage container create --name "${AZURE_STORAGE_CONTAINER}" --account-name "${AZURE_STORAGE_ACCOUNT}" --output none --only-show-errors
    fi
}

upload_openid_docs(){    
    cat <<EOF > "${SCRIPT_PATH}/openid-configuration.json"
{
  "issuer": "${SERVICE_ACCOUNT_ISSUER}",
  "jwks_uri": "${SERVICE_ACCOUNT_ISSUER}openid/v1/jwks",
  "response_types_supported": [
    "id_token"
  ],
  "subject_types_supported": [
    "public"
  ],
  "id_token_signing_alg_values_supported": [
    "RS256"
  ]
}
EOF

    echo "Uploading openid-configuration document to '${AZURE_STORAGE_ACCOUNT}' storage account"
    upload_to_blob "${AZURE_STORAGE_CONTAINER}" "${SCRIPT_PATH}/openid-configuration.json" ".well-known/openid-configuration"

    echo "Getting public signing key from the cluster"
    kubectl get --raw /openid/v1/jwks | jq > "${SCRIPT_PATH}/jwks.json"
    echo "Uploading jwks document to '${AZURE_STORAGE_ACCOUNT}' storage account"
    upload_to_blob "${AZURE_STORAGE_CONTAINER}" "${SCRIPT_PATH}/jwks.json" "openid/v1/jwks"
}

upload_to_blob() {
    local container_name=$1
    local file_path=$2
    local blob_name=$3

    echo "Uploading ${file_path} to '${AZURE_STORAGE_ACCOUNT}' storage account"
    az storage blob upload \
        --container-name "${container_name}" \
        --file "${file_path}" \
        --name "${blob_name}" \
        --account-name "${AZURE_STORAGE_ACCOUNT}" \
        --output none --only-show-errors
}

create_kind_cluster() {
    SERVICE_ACCOUNT_ISSUER=$(az storage account show --name "${AZURE_STORAGE_ACCOUNT}" -o json | jq -r .primaryEndpoints.web)

    if [ "${SKIP_CLUSTER:-}" = "true" ]; then
        echo "Skipping cluster creation"
        return
    fi

    echo "Creating kind cluster"    
    kind delete cluster --name "${KIND_CLUSTER_NAME}"
    cat <<EOF | kind create cluster --name ${KIND_CLUSTER_NAME} --image kindest/node:"${KIND_IMAGE_VERSION}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        service-account-issuer: ${SERVICE_ACCOUNT_ISSUER}
EOF
}

validate "$@"
create_kind_cluster
create_azure_blob_storage_account "$@"
upload_openid_docs
