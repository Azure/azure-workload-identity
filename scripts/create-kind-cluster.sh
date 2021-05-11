#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}" || exit 1

: "${AZURE_TENANT_ID:?Environment variable empty or not defined.}"
: "${AZURE_SUBSCRIPTION_ID:?Environment variable empty or not defined.}"

readonly KIND="${REPO_ROOT}/hack/tools/bin/kind"
readonly KUBECTL="${REPO_ROOT}/hack/tools/bin/kubectl"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-aad-pod-managed-identity}"

# create a fake azure.json based on environment variables
cat <<EOF > "${REPO_ROOT}/azure.json"
{
  "cloud": "${AZURE_ENVIRONMENT:-AzurePublicCloud}",
  "tenantId": "${AZURE_TENANT_ID}",
  "subscriptionId": "${AZURE_SUBSCRIPTION_ID}"
}
EOF

# create a kind cluster with azure.json mounted
cat <<EOF | ${KIND} create cluster --name "${KIND_CLUSTER_NAME}" --image "kindest/node:${KIND_NODE_VERSION:-v1.20.2}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
  - hostPath: ${REPO_ROOT}/azure.json
    containerPath: /etc/kubernetes/azure.json
EOF

${KUBECTL} wait node "${KIND_CLUSTER_NAME}-control-plane" --for=condition=Ready --timeout=90s
OUTPUT_TYPE="type=docker" make container-manager
${KIND} load docker-image controller:latest --name "${KIND_CLUSTER_NAME}"
