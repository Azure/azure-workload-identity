# Discovery Document

<!-- toc -->

OpenID Connect describes a [metadata document][1] that contains the metadata of the issuer. This includes information such as the URLs to use and the location of the service's public signing keys. The following section will walk you through how to generate and upload a minimal discovery document to an Azure Blob storage account.

## Walkthrough

### 1. Create an Azure Blob storage account

```bash
export RESOURCE_GROUP="azwi"
export LOCATION="westus2"
az group create --name "${RESOURCE_GROUP}" --location "${LOCATION}"

export AZURE_STORAGE_ACCOUNT="azwi$(openssl rand -hex 4)"
export AZURE_STORAGE_CONTAINER="oidc-test"
az storage account create --resource-group "${RESOURCE_GROUP}" --name "${AZURE_STORAGE_ACCOUNT}"
az storage container create --name "${AZURE_STORAGE_CONTAINER}" --public-access container
```

### 2. Generate the discovery document

```bash
cat <<EOF > openid-configuration.json
{
  "issuer": "https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/",
  "jwks_uri": "https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/openid/v1/jwks",
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
```

### 3. Upload the discovery document

```bash
az storage blob upload \
  --container-name "${AZURE_STORAGE_CONTAINER}" \
  --file openid-configuration.json \
  --name .well-known/openid-configuration
```

### 4. Verify that the discovery document is publicly accessible

```bash
curl -s "https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/.well-known/openid-configuration"
```

<details>
<summary>Output</summary>

```json
{
  "issuer": "https://<REDACTED>.blob.core.windows.net/oidc-test/",
  "jwks_uri": "https://<REDACTED>.blob.core.windows.net/oidc-test/openid/v1/jwks",
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
```

[1]: https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderConfig
