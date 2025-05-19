#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

: "${SERVICE_ACCOUNT_ISSUER:?Environment variable empty or not defined.}"

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}" || exit 1

readonly KIND="${REPO_ROOT}/hack/tools/bin/kind"
readonly KUBECTL="${REPO_ROOT}/hack/tools/bin/kubectl"
SERVICE_ACCOUNT_SIGNING_KEY_FILE="$(pwd)/sa.key"
SERVICE_ACCOUNT_KEY_FILE="$(pwd)/sa.pub"

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-azure-workload-identity}"

preflight() {
  if [[ ! -f "${SERVICE_ACCOUNT_SIGNING_KEY_FILE}" ]]; then
    echo "${SERVICE_ACCOUNT_SIGNING_KEY_FILE} is missing. You can generate it by running 'openssl genrsa -out ${REPO_ROOT}/sa.key 2048'"
    exit 1
  fi
  if [[ ! -f "${SERVICE_ACCOUNT_KEY_FILE}" ]]; then
    echo "${SERVICE_ACCOUNT_KEY_FILE} is missing. You can generate it by running 'openssl rsa -in sa.key -pubout -out ${REPO_ROOT}/sa.pub'"
    exit 1
  fi
  if ! curl -sSf "${SERVICE_ACCOUNT_ISSUER}.well-known/openid-configuration" > /dev/null 2>&1; then
    cat <<EOF
${SERVICE_ACCOUNT_ISSUER}.well-known/openid-configuration is missing. You can upload the following JSON to the storage account:
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
  exit 1
  fi
  if ! curl -sSf "${SERVICE_ACCOUNT_ISSUER}openid/v1/jwks" > /dev/null 2>&1; then
    make azwi
    JWKS="$(./bin/azwi jwks --public-keys "${SERVICE_ACCOUNT_KEY_FILE}")"
    cat <<EOF
${SERVICE_ACCOUNT_ISSUER}openid/v1/jwks is missing. You can upload the following JSON to the storage account:
${JWKS}
EOF
  exit 1
  fi
}

create_kind_cluster() {
  # create a kind cluster
  cat <<EOF | ${KIND} create cluster --name "${KIND_CLUSTER_NAME}" --image "mcr.microsoft.com/mirror/kindest/node:${KIND_NODE_VERSION:-v1.32.3}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
    - hostPath: ${SERVICE_ACCOUNT_KEY_FILE}
      containerPath: /etc/kubernetes/pki/sa.pub
    - hostPath: ${SERVICE_ACCOUNT_SIGNING_KEY_FILE}
      containerPath: /etc/kubernetes/pki/sa.key
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
    taints:
    - key: "kubeadmNode"
      value: "master"
      effect: "NoSchedule"
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        service-account-issuer: ${SERVICE_ACCOUNT_ISSUER}
        service-account-key-file: /etc/kubernetes/pki/sa.pub
        service-account-signing-key-file: /etc/kubernetes/pki/sa.key
    controllerManager:
      extraArgs:
        service-account-private-key-file: /etc/kubernetes/pki/sa.key
- role: worker
- role: worker
- role: worker
EOF

  ${KUBECTL} wait node "${KIND_CLUSTER_NAME}-control-plane" --for=condition=Ready --timeout=90s
}

download_service_account_keys() {
  if [[ -z "${SERVICE_ACCOUNT_KEYVAULT_NAME:-}" ]] || [[ "${SKIP_PREFLIGHT:-}" == "true" ]]; then
    return
  fi
  az keyvault secret show --vault-name "${SERVICE_ACCOUNT_KEYVAULT_NAME}" --name sa-pub | jq -r .value | base64 -d > "${SERVICE_ACCOUNT_KEY_FILE}"
  az keyvault secret show --vault-name "${SERVICE_ACCOUNT_KEYVAULT_NAME}" --name sa-key | jq -r .value | base64 -d > "${SERVICE_ACCOUNT_SIGNING_KEY_FILE}"
}

download_service_account_keys
if [[ "${SKIP_PREFLIGHT:-}" != "true" ]]; then
  preflight
fi
create_kind_cluster
