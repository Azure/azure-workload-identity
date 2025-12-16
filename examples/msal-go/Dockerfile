FROM mcr.microsoft.com/oss/go/microsoft/golang:1.24.11-bookworm@sha256:3b7a6e64a0b4804551d553dae580d1fc46dd56fe68be5db3b373c0a38baa514e as builder

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
