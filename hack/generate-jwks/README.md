# Generate JWKS

Use the go file to generate the JSON Web Key Sets (JWKS).

```bash
cd hack/generate-jwks
go install .
generate-jwks --public-keys <comma separated list of public keys>
```

Sample output:

```bash
➜ generate-jwks --public-keys /tmp/apiserver.crt
{
  "keys": [
    {
      "use": "sig",
      "kty": "RSA",
      "kid": "P6_QkE57hno3Yprg8gXNhFCXn6zZDbLrFC02SUWALfI",
      "alg": "RS256",
      "n": "qZdFZCsDEkvjEaut9mqhySyni3A7vtWqIfltuX1gz4i9pqKfLmWfT884OnrBnrOVgypNFXt0ZD-KRzWGlJAnpxk_t095eQRulK_sDg6JJTU1YSHfQhy3sSPaC5bzlQOiupViBrq7bRAkAKgHeUyujoSsV7llCqOq_o4xBe-F9gHOAGYBB9jpiJ6Nfxm57-x-j0HPyn31icP5VjGXNlHRC5RZrSaDq-PCooC_keqltANrG6yJ-jWOUIYIvbqH9Dqltq-NUEQmG1f4YSQYXmVwiAsMLynaYLfar9Pz4OvgvuFRcXIsS3RLeAVbJMuetukA8TCANmk8Ci1XWrkLVNoWmFmp2Akxud2uA2KnJFdP0ZuZcZjViAStFc_iYohiYNvnsjTV-oTRiGMTwzE8JlTujXsiOpdgTowV3d3TiS1RVD_uuI6fl0Qshwa0FoG5nCANouHcoSEdjKUSQxaeNA39DvoPaThGLFusuNfL6QkESCj9q0UtrXPly_DF7gw_jX0pFetrX2NFf0R9vOKsXWvzX0IlDzpsrLgkPgtMY7LyGrz6K1g61R4Mc5VWIof7w8fqWdkgzqJE3r1mdUVLkcMYpUs0AvmpsCN2DR_z4TGdfQRDGM2MU7iNU7gtsB2gFE6REYnfHsDJ3uD5DyP9m-nZ5wiPZl9SrZmb8JDC9nkAxy8",
      "e": "AQAB"
    }
  ]
}
```
