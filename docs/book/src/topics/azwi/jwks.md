# `azwi jwks`

Create JSON Web Key Sets for the service account issuer keys.

## Synopsis

This command provides the ability to generate the JSON Web Key Sets (JWKS) for the service account issuer keys

    azwi jwks [flags]

## Options

      -h, --help                  help for jwks
          --output-file string    The name of the file to write the JWKS to. If not provided, the default output is stdout
          --public-keys strings   List of public keys to include in the JWKS

## Example

```bash
azwi jwks --public-keys sa.key
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
