#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}" || exit 1

: "${AZURE_TENANT_ID:?Environment variable empty or not defined.}"

readonly KIND="${REPO_ROOT}/hack/tools/bin/kind"
readonly KUBECTL="${REPO_ROOT}/hack/tools/bin/kubectl"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-aad-pod-managed-identity}"

# create a kind cluster
cat <<EOF | ${KIND} create cluster --name "${KIND_CLUSTER_NAME}" --image "kindest/node:${KIND_NODE_VERSION:-v1.20.2}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        service-account-issuer: https://kubernetes.default.svc.cluster.local
        service-account-key-file: /etc/kubernetes/pki/sa.pub
        service-account-signing-key-file: /etc/kubernetes/pki/sa.key
EOF

${KUBECTL} wait node "${KIND_CLUSTER_NAME}-control-plane" --for=condition=Ready --timeout=90s
