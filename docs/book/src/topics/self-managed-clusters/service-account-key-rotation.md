# Service Account Key Rotation

<!-- toc -->

A security best practice is to routinely rotate your key pair used to sign the service account tokens. This page explains the best practices, guidelines, as well as how to generate and rotate it in the case of self-managed Kubernetes clusters where you have access to the control plane.

> This technique requires that the Kubernetes control plane is running in a high-availability (HA) setup with multiple API servers. Clusters that use a single API server will become unavailable while the API server is restarted.

## Best Practices

### Key rotation

Key pair should be rotated on a regular basis. For references, AKS clusters rotate their service account signing key pairs **every three months**.

### Key retirement

Key pair should be retired when they are no longer needed. In most cases, this means permanently removing them to guarantee that it poses no more risk and to minimize the number of active key pairs that are being handled.

## Steps to manually generate and rotate keys

### 1. Generate a new key pair

> Skip this step if you are planning to bring your own keys.

```bash
openssl genrsa -out sa-new.key 2048
openssl rsa -in sa-new.key -pubout -out sa-new.pub
```

### 2. Backup the old key pair and distribute the new key pair

Schedule a jump pod to each control plane node, which mounts the `/etc/kubernetes/pki` folder:

> `/etc/kubernetes/pki/sa.pub` and `/etc/kubernetes/pki/sa.key` are the paths of the service account key pair for a kind cluster. The paths can vary depending on your provider.

```bash
cat << EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: jump
  labels:
    k8s-app: jump
spec:
  selector:
    matchLabels:
      name: jump
  template:
    metadata:
      labels:
        name: jump
    spec:
      tolerations:
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      containers:
        - name: busybox
          image: busybox
          command:
            - sleep
            - "3600"
          volumeMounts:
              - mountPath: /etc/kubernetes/pki
                name: etc-kubernetes-pki
      volumes:
        - name: etc-kubernetes-pki
          hostPath:
            path: /etc/kubernetes/pki
EOF
```

Backup the old service account key pair to your local machine:

```bash
POD_NAME="$(kubectl get po -l name=jump -ojson | jq -r '.items[0].metadata.name')"
kubectl cp default/${POD_NAME}:/etc/kubernetes/pki/sa.pub sa-old.pub
kubectl cp default/${POD_NAME}:/etc/kubernetes/pki/sa.key sa-old.key
```

Distribute the new key pair to the certificate directory of each control plane node:

```bash
for POD_NAME in "$(kubectl get po -l name=jump -ojson | jq -r '.items[].metadata.name')"; do
  kubectl cp sa-new.pub default/${POD_NAME}:/etc/kubernetes/pki/sa-new.pub
  kubectl cp sa-new.key default/${POD_NAME}:/etc/kubernetes/pki/sa-new.key
done
```

### 3. Update the JWKS

In the case of service account tokens generated before you initiated the key rotation, you would need a time period where the old and new public keys exist in the JWKS. The relying party can then validate service account tokens signed by both the old and new private key.

Download `azwi` from our [latest GitHub releases][4], which is a CLI tool that helps generate the JWKS document in JSON.

Generate and upload the JWKS:

> Assuming you followed our [Quick Start][2] and store your OIDC discovery document and JWKS in an Azure storage account.

```bash
azwi jwks --public-keys sa-old.pub --public-keys sa-new.pub --output-file jwks.json
export AZURE_STORAGE_ACCOUNT=<AzureStorageAccount>
az storage blob upload \
  --container-name "${AZURE_STORAGE_CONTAINER}" \
  --file jwks.json \
  --name openid/v1/jwks
```

### 4. Key Rotation

With the new key pair distributed, you can utilize [kubectl-node-shell][1] to update the following core components arguments by spawning a root shell to each control plane node:

```bash
kubectl node-shell <NodeName>

# Run in the root shell
# download yq (jq for yaml)
curl -L https://github.com/mikefarah/yq/releases/download/v4.12.1/yq_linux_amd64 --output /usr/bin/yq
chmod +x /usr/bin/yq

# append the new public key as an kube-apiserver argument
yq eval -i '.spec.containers[0].command |= . + ["--service-account-key-file=/etc/kubernetes/pki/sa-new.pub"]' /etc/kubernetes/manifests/kube-apiserver.yaml

# replace the old private key with the new private key for kube-apiserver and kube-controller-manager
sed -i 's|--service-account-signing-key-file=.*|--service-account-signing-key-file=/etc/kubernetes/pki/sa-new.key|' /etc/kubernetes/manifests/kube-apiserver.yaml
sed -i 's|--service-account-private-key-file=.*|--service-account-private-key-file=/etc/kubernetes/pki/sa-new.key|' /etc/kubernetes/manifests/kube-controller-manager.yaml
```

The commands above should trigger a restart for kube-apiserver and kube-controller-manager pod.

### 5. Verification

Create a dummy pod that uses an annotated service account.

```bash
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    azure.workload.identity/client-id: dummy
  name: workload-identity-sa
---
apiVersion: v1
kind: Pod
metadata:
  name: dummy-pod
  labels:
    azure.workload.identity/use: "true"
spec:
  serviceAccountName: workload-identity-sa
  containers:
    - name: busybox
      image: busybox
      command:
        - sleep
        - "3600"
EOF
```

Output the projected service account token:

```bash
kubectl exec dummy-pod -- cat /var/run/secrets/azure/tokens/azure-identity-token
```

Decode your token using [jwt.ms][3]. The `kid` field in the token header should be the same as the `kid` of `azwi jwks --public-keys sa-new.pub | jq -r '.keys[0].kid'`. This means that the service account token is signed by the new private key.

### 6. Cleanup

Delete the dangling resources created above:

```bash
kubectl delete ds jump
kubectl delete pod dummy-pod
kubectl delete sa workload-identity-sa
```

### 7. Remove old JWK after maximum token expiration

After the maximum token expiration (the default expiration is 24 hours) has passed, projected service account tokens signed by the old private key will be rotated by kubelet and signed with the new signing key. The kubelet proactively rotates the token if it is older than 80% of its total TTL, or if the token is older than 24 hours. You should update the JWKS accordingly to only include the new public key:

```bash
azwi jwks --public-keys sa-new.pub --output-file jwks.json
az storage blob upload \
  --container-name "${AZURE_STORAGE_CONTAINER}" \
  --file jwks.json \
  --name openid/v1/jwks
```

Remove the old public key from kube-apiserver's arguments:

```bash
# get the index of the old public key from the kube-apiserver argument array
INDEX="$(yq e '.spec.containers[0].command' /etc/kubernetes/manifests/kube-apiserver.yaml | grep -Fn 'service-account-key-file' | head -n 1 | cut -d':' -f1)"

# convert to zero-index
INDEX="$(expr ${INDEX} - 1)"

# remove the old public key argument using yq
yq eval -i "del(.spec.containers[0].command[${INDEX}])" /etc/kubernetes/manifests/kube-apiserver.yaml

# remove the old key pair from disk
rm sa.*
```

[1]: https://github.com/kvaps/kubectl-node-shell

[2]: ../../quick-start.md

[3]: https://jwt.ms

[4]: https://github.com/Azure/azure-workload-identity/releases
