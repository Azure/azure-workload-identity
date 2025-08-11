FROM mcr.microsoft.com/oss/go/microsoft/golang:1.24.6-bookworm@sha256:b55cd0118651789c1c3623f8d8ba7e9f9e131817ec20af3505ca5410603e47a2 as builder

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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GO111MODULE=on go build -a -ldflags "${LDFLAGS:--X github.com/Azure/azure-workload-identity/pkg/version.BuildVersion=latest}" -o proxy main.go

FROM --platform=${TARGETPLATFORM:-linux/amd64} mcr.microsoft.com/cbl-mariner/distroless/minimal:2.0-nonroot
WORKDIR /
COPY --from=builder /workspace/proxy .
# Kubernetes runAsNonRoot requires USER to be numeric
USER 1501:1501

ENTRYPOINT [ "/proxy" ]
