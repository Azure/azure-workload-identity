# Using AAD Managed Pod Identity with MSAL .NET to access Keyvault

## Standard Walkthrough

Run the following commands to set Azure-related environment variables and login to Azure via az login:

```bash
export SUBSCRIPTION_ID="<SubscriptionID>"

# login as a user and set the appropriate subscription ID
az login
az account set -s "${SUBSCRIPTION_ID}"

export RESOURCE_GROUP="<AKV resource group or existing resource group>"
export LOCATION="<location>"
export KEYVAULT_NAME="<key vault name>"
export KEYVAULT_SECRET_NAME="<key vault secret name>"
# this is only required if the keyvault instance is not in the same tenant as the cluster
export TENANT_ID="<tenant id for the key vault instance>"
```

**Prerequisites**

- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest)
- [.NET 5.0](https://dotnet.microsoft.com/download)

### 1. Create the Keyvault and secret

If you're using an existing keyvault instance and secret, skip this step.

```bash
# create keyvault
az keyvault create -l ${LOCATION} --name ${KEYVAULT_NAME} -g ${RESOURCE_GROUP}

# create secret
az keyvault secret set --vault-name ${KEYVAULT_NAME} --name ${KEYVAULT_SECRET_NAME} --value Hello!
```

### 2. Create a service principal and grant it permissions to access the secret

```bash
# create service principal
export SERVICE_PRINCIPAL_CLIENT_ID=$(az ad sp create-for-rbac --skip-assignment --name https://test-sp --query appId -o tsv)

# set policy to access keyvault secrets
az keyvault set-policy -n ${KEYVAULT_NAME} --secret-permissions get --spn ${SERVICE_PRINCIPAL_CLIENT_ID}
```

### 3. Setup trust between service principal and cluster OIDC issuer

*TODO*

### 4. Deploy the AAD Pod Managed Identity webhook

Refer to [doc](../../../README.md#install-webhook)

### 5. Create the Kubernetes service account

Create the Kubernetes service account with the required labels and annotations.

```yml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    azure.pod.identity/tenant-id: ${TENANT_ID}
    azure.pod.identity/client-id: ${SERVICE_PRINCIPAL_CLIENT_ID}
  labels:
    azure.pod.identity/use: "true"
  name: pod-identity-sa
EOF
```

### 6. Deployment and Validation

Deploy a pod referencing the service account created in the last step. The mutating webhook will inject the required environment variables based on the annotation in service account.

```yml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: demo
spec:
  serviceAccountName: pod-identity-sa
  containers:
    - image: aramase/dotnet:v0.4
      imagePullPolicy: IfNotPresent
      name: oidc
      env:
      - name: KEYVAULT_NAME
        value: ${KEYVAULT_NAME}
      - name: SECRET_NAME
        value: ${KEYVAULT_SECRET_NAME}
  nodeSelector:
    kubernetes.io/os: linux
EOF
```

To verify that pod is able to get a token and access secret in keyvault:

```bash
kubectl logs demo
```

If successful, the log output would be similar to the following output:

```bash
START 06/03/2021 19:54:03 (dotnet-v0)
Your secret is Hello!
```

Once you are done with the demo, clean up your resources:

```bash
# delete the pod and service account
kubectl delete pod demo
kubectl delete sa pod-identity-sa

# delete the keyvault and sp
az keyvault delete -n ${KEYVAULT_NAME} -g ${RESOURCE_GROUP}
az ad sp delete --id ${SERVICE_PRINCIPAL_CLIENT_ID}
```
