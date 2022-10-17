# Managed Clusters

<!-- toc -->

For managed clusters, the service account signing keys will be set up and managed by the cloud provider.

Before deploying Azure AD Workload Identity, you will need to enable any **OIDC-specific** feature flags and obtain the **OIDC issuer URL** when setting up the federated identity credentials.

## Azure Kubernetes Service (AKS)

To create a new AKS cluster with OIDC Issuer URL enabled or update an existing cluster, follow the instructions in the [Azure Kubernetes Service (AKS) documentation][4].

To get your cluster's OIDC issuer URL run:

```bash
# Output the OIDC issuer URL
az aks show --resource-group <resource_group> --name <cluster_name> --query "oidcIssuerProfile.issuerUrl" -otsv
```

Ensure your cluster is running a mutating admission webhook. If your cluster is not running a webhook, follow the instructions at [Mutating Admission Webhook](./mutating-admission-webhook.md).
```bash
kubectl get mutatingwebhookconfigurations.admissionregistration.k8s.io | grep azure-wi-webhook-mutating-webhook-configuration
# You should see a webhook running
azure-wi-webhook-mutating-webhook-configuration   1          28m
```

## Amazon Elastic Kubernetes Service (EKS)

EKS cluster has an OIDC issuer URL associated with it by default. To get your cluster's OIDC issuer URL run:

```bash
# Output the OIDC issuer URL
aws eks describe-cluster --name <cluster_name> --query "cluster.identity.oidc.issuer" --output text
```

Refer to the [Amazon EKS documentation][1] for more information on the OIDC issuer URL for the EKS cluster.

## Google Kubernetes Engine (GKE)

GKE cluster has an OIDC issuer URL associated with it by default. Follow the [steps](#steps-to-get-the-oidc-issuer-url-from-a-generic-managed-cluster) to get the OIDC issuer URL.

## Steps to get the OIDC issuer URL from a generic managed cluster

In this section, we will cover how to get the OIDC issuer URL from a generic managed cluster using a jump pod.

### 1. Create a service account for the jump pod

Run the following commands to set up a service account for the jump pod:

```bash
export NAMESPACE="default"
export SERVICE_ACCOUNT_NAME="jump-pod-sa"

kubectl create serviceaccount ${SERVICE_ACCOUNT_NAME} -n ${NAMESPACE}
```

<details>
<summary>Output</summary>

```bash
serviceaccount/jump-pod-sa created
```

</details>

### 2. Deploy a jump pod referencing the service account

Deploy a jump pod with [projected service account token][2] to your cluster. The jump pod uses the [step-cli][3] docker image that is used for inspecting the service account token to retrieve the OIDC issuer URL.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: jump
  namespace: ${NAMESPACE}
spec:
  containers:
  - image: smallstep/step-cli
    name: step-cli
    command:
    - /bin/sh
    - -c
    - cat /var/run/secrets/tokens/test-token | step crypto jwt inspect --insecure
    volumeMounts:
    - mountPath: /var/run/secrets/tokens
      name: test-token
  serviceAccountName: ${SERVICE_ACCOUNT_NAME}
  volumes:
  - name: test-token
    projected:
      sources:
      - serviceAccountToken:
          path: test-token
          expirationSeconds: 3600
          audience: test
EOF
```

<details>
<summary>Output</summary>

```bash
pod/jump created
```

</details>

### 3. Get the OIDC issuer URL from the jump pod

The jump pod logs will contain the decoded JWT. Run the following command to get the logs and extract the OIDC issuer URL:

```bash
kubectl logs jump -n ${NAMESPACE}
```

<details>
<summary>Output</summary>

```json
{
  "header": {
    "alg": "RS256",
    "kid": "[REDACTED]"
  },
  "payload": {
    "aud": [
      "test"
    ],
    "exp": 1634671190,
    "iat": 1634667590,
    "iss": "https://container.googleapis.com/v1/projects/[REDACTED]/locations/us-central1-c/clusters/[REDACTED]",
    "kubernetes.io": {
      "namespace": "default",
      "pod": {
        "name": "jump",
        "uid": "c4e09c90-3007-4255-ab74-f5f97d944db2"
      },
      "serviceaccount": {
        "name": "jump-pod-sa",
        "uid": "6af8dfb1-8a28-48f8-a7fe-e2abd99cd35e"
      }
    },
    "nbf": 1634667590,
    "sub": "system:serviceaccount:default:jump-pod-sa"
  },
  "signature": "[REDACTED]"
}
```

</details>

The OIDC issuer URL is the value of the `iss` claim in the JWT.

To just get the issuer from the JWT, run the following command:

```bash
kubectl logs jump -n ${NAMESPACE} | jq -r '.payload.iss'
```

<details>
<summary>Output</summary>

```log
https://container.googleapis.com/v1/projects/[REDACTED]/locations/us-central1-c/clusters/[REDACTED]
```

</details>

### 4. Cleanup

```bash
kubectl delete pod jump -n ${NAMESPACE}
kubectl delete serviceaccount ${SERVICE_ACCOUNT_NAME} -n ${NAMESPACE}
```

[1]: https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html

[2]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection

[3]: https://smallstep.com/cli/

[4]: https://docs.microsoft.com/en-us/azure/aks/cluster-configuration#oidc-issuer-preview
