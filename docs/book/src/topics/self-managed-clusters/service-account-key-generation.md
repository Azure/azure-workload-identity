# Service Account Key Generation

<!-- toc -->

There are two keys in an RSA key pair: a private key and a public key. The RSA private key is used to generate the digital signature and the RSA public key is used to verify them. In the case of service account tokens, they are signed by your private key/signing key before being projected to your workload's volume. Azure Active Directory will then use your public key to verify the signature and ensure that the service account tokens are not malicious.

This section will you through how to generate an RSA key pair using `openssl`.

> Feel free to skip this section if you are planning to bring your own keys.

## Walkthrough

### 1. Generate an RSA private key using `openssl`

```bash
openssl genrsa -out sa.key 2048
```

<details>
<summary>Output</summary>

```bash
Generating RSA private key, 2048 bit long modulus
.............................................+++
.......+++
e is 65537 (0x10001)
```

</details>

### 2. Generate an RSA public key from a private key using `openssl`

```bash
openssl rsa -in sa.key -pubout -out sa.pub
```
