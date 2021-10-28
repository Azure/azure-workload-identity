# JSON Web Key Sets (JWKS)

<!-- toc -->

The JSON Web Key Sets (JWKS) document contains the public signing key(s) that allows AAD to verify the authenticity of the service account token.

## Walkthrough

> Assuming you have access to your service account signing key pair and followed [the guide][1] on how to create and upload the discovery document to an Azure blob storage account. See [this section][2] on how to generate a minimal signing key pair.

### 1. Install `azwi`

[Installation guide][3]

### 2. Generate the JWKS document

```bash
azwi jwks --public-keys <PublicKeyPath> --output-file jwks.json
```

> If you have multiple public signing keys, you can append additional `--public-keys` flag to the command.

### 3. Upload the JWKS document

```bash
az storage blob upload \
  --container-name "${AZURE_STORAGE_CONTAINER}" \
  --file jwks.json \
  --name openid/v1/jwks
```

### 4. Verify that the JWKS document is publicly accessible

```bash
curl -s "https://${AZURE_STORAGE_ACCOUNT}.blob.core.windows.net/${AZURE_STORAGE_CONTAINER}/openid/v1/jwks"
```

<details>
<summary>Output</summary>

```json
{
  "keys": [
    {
      "use": "sig",
      "kty": "RSA",
      "kid": "Me5VC6i4_4mymFj7T5rcUftFjYX70YoCfSnZB6-nBY4",
      "alg": "RS256",
      "n": "ywg7HeKIFX3vleVKZHeYoNpuLHIDisnczYXrUdIGCNilCJFA1ymjG2UAADnt_FpYUsCVyKYJTqcxNbK4boNg_P3uK39OAqXabwYrilEZvsVJQKhzn8dXLeqAnM98L8eBpySU208KTsfMkS3Q6lqwurUP7c_a3g_1XRJukz_EmQxg9jLD_fQd5VwPTEo8HJQIFqIxFWzjTkkK5hbcL9Cclkf6RpeRyjh7Vem57Fu-jAlxDUiYiqyieM4OBNm4CQjiqDE8_xOC8viNpHNw542MYVDKSRnYui31lCOj32wBDphczR8BbnrZgbqN3K_zzB3gIjcGbWbbGA5xKJYqSu5uRwN89_CWrT3vGw5RN3XQPSbhGC4smgZkOCw3N9i1b-x-rrd-mRse6F95ONaoslCJUbJvxvDdb5X0P4_CVZRwJvUyP3OJ44ZvwzshA-zilG-QC9E1j2R9DTSMqOJzUuOxS0JIvoboteI1FAByV9KyU948zQRM7r7MMZYBKWIsu6h7",
      "e": "AQAB"
    }
  ]
}
```

</details>

[1]: ./discovery-document.md

[2]: ../service-account-key-generation.md

[3]: ../../azwi.md
