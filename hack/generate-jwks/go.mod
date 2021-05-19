module github.com/Azure/aad-pod-managed-identity/hack/generate-jwks

go 1.16

require (
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1
	k8s.io/client-go v0.21.1
	k8s.io/klog/v2 v2.8.0
)

// fixes CVE-2020-29652
replace golang.org/x/crypto => golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f
