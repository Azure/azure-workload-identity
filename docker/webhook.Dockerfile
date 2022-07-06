ARG BUILDER=mcr.microsoft.com/oss/go/microsoft/golang:1.18-bullseye
ARG BASEIMAGE=gcr.io/distroless/static:nonroot

# Build the manager binary
FROM ${BUILDER} as builder

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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GO111MODULE=on go build -a -ldflags "${LDFLAGS:--X github.com/Azure/azure-workload-identity/pkg/version.BuildVersion=latest}" -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM --platform=${TARGETPLATFORM:-linux/amd64} ${BASEIMAGE}
WORKDIR /
COPY --from=builder /workspace/manager .
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532

ENTRYPOINT ["/manager"]
