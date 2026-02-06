# Build the manager binary
FROM mcr.microsoft.com/oss/go/microsoft/golang:1.25.6-bookworm@sha256:7ee2dbc1bae8e4c4d0256aed778a1a9e40fdd8a0c04b87b40cb5f4670774f5b4 as builder

ARG LDFLAGS

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/webhook/main.go main.go
COPY pkg/ pkg/

# Build
ARG TARGETARCH
RUN MS_GO_NOSYSTEMCRYPTO=1 CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GO111MODULE=on go build -a -ldflags "${LDFLAGS:--X github.com/Azure/azure-workload-identity/pkg/version.BuildVersion=latest}" -o manager main.go

FROM --platform=${TARGETPLATFORM:-linux/amd64} mcr.microsoft.com/cbl-mariner/distroless/minimal:2.0-nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532

ENTRYPOINT ["/manager"]
