FROM mcr.microsoft.com/oss/go/microsoft/golang:1.23.9-bookworm@sha256:c705d51664f5057656a05df694f72990e19ff8089f7e184d37a8a1893e313bd7 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY token_credential.go token_credential.go

# Build
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GO111MODULE=on go build -a -o msalgo .

FROM --platform=${TARGETPLATFORM:-linux/amd64} mcr.microsoft.com/cbl-mariner/distroless/minimal:2.0-nonroot
WORKDIR /
COPY --from=builder /workspace/msalgo .
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532

ENTRYPOINT ["/msalgo"]
