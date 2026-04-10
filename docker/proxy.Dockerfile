FROM mcr.microsoft.com/oss/go/microsoft/golang:1.25.9-bookworm@sha256:345c514cca33c2fe021ca905f44fdf7e5eff9a3f0585bcf32cd57b5782881cc3 as builder

ARG LDFLAGS

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/proxy/main.go main.go
COPY pkg/ pkg/

# Build
ARG TARGETARCH
RUN MS_GO_NOSYSTEMCRYPTO=1 CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GO111MODULE=on go build -a -ldflags "${LDFLAGS:--X github.com/Azure/azure-workload-identity/pkg/version.BuildVersion=latest}" -o proxy main.go

FROM --platform=${TARGETPLATFORM:-linux/amd64} mcr.microsoft.com/cbl-mariner/distroless/minimal:2.0-nonroot
WORKDIR /
COPY --from=builder /workspace/proxy .
# Kubernetes runAsNonRoot requires USER to be numeric
USER 1501:1501

ENTRYPOINT [ "/proxy" ]
