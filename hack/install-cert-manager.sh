#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

: "${CERT_MANAGER_VERSION:?Environment variable empty or not defined.}"

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}" || exit 1

readonly KUBECTL="${REPO_ROOT}/hack/tools/bin/kubectl"

TEST_RESOURCE=$(cat <<-END
apiVersion: v1
kind: Namespace
metadata:
  name: cert-manager-test
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: test-selfsigned
  namespace: cert-manager-test
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: selfsigned-cert
  namespace: cert-manager-test
spec:
  dnsNames:
    - example.com
  secretName: selfsigned-cert-tls
  issuerRef:
    name: test-selfsigned
END
)

## Install cert manager and wait for availability
${KUBECTL} apply -f "https://github.com/jetstack/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml"
${KUBECTL} wait --for=condition=Available --timeout=5m -n cert-manager deployment/cert-manager
${KUBECTL} wait --for=condition=Available --timeout=5m -n cert-manager deployment/cert-manager-cainjector
${KUBECTL} wait --for=condition=Available --timeout=5m -n cert-manager deployment/cert-manager-webhook

for _ in {1..6}; do
  if echo "${TEST_RESOURCE}" | kubectl apply -f -; then
    break
  fi
  sleep 15
done

${KUBECTL} wait --for=condition=Ready --timeout=5m -n cert-manager-test certificate/selfsigned-cert

trap 'echo "${TEST_RESOURCE}" | kubectl delete -f -' EXIT
