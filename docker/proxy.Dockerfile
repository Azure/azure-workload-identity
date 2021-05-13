# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/proxy/main.go main.go

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o proxy main.go

FROM alpine:latest
WORKDIR /
COPY --from=builder /workspace/proxy .
USER nonroot:nonroot

ENTRYPOINT [ "/proxy" ]
EXPOSE 8000
