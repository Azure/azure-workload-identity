ARG SERVERCORE_CACHE=gcr.io/k8s-staging-e2e-test-images/windows-servercore-cache:1.0-linux-amd64-${OS_VERSION:-1809}
ARG BASEIMAGE=mcr.microsoft.com/windows/nanoserver:${OS_VERSION:-1809}

FROM --platform=linux/amd64 mcr.microsoft.com/oss/go/microsoft/golang:1.24.6-bookworm@sha256:b55cd0118651789c1c3623f8d8ba7e9f9e131817ec20af3505ca5410603e47a2 as builder

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
RUN CGO_ENABLED=0 GOOS=windows GO111MODULE=on go build -a -o msalgo.exe .

FROM --platform=linux/amd64 ${SERVERCORE_CACHE} as core

FROM --platform=${TARGETPLATFORM:-windows/amd64} ${BASEIMAGE}
WORKDIR /
COPY --from=builder /workspace/msalgo.exe .
COPY --from=core /Windows/System32/netapi32.dll /Windows/System32/netapi32.dll
USER ContainerAdministrator

ENTRYPOINT [ "/msalgo.exe" ]
