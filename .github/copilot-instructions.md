# Azure Workload Identity

Azure Workload Identity is a Kubernetes-focused Go project that enables workloads to access Azure resources securely using Azure AD based on annotated service accounts. The project includes a mutating admission webhook, proxy sidecar containers, and the `azwi` CLI tool.

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Prerequisites and Setup
- Install Go 1.24.6 or later: `go version` should show 1.24.6+
- Install jq: `jq --version` should work
- Install make: `make --version` should work  
- Docker and Kind are available for local development
- Clone repository: `git clone https://github.com/Azure/azure-workload-identity.git`

### Bootstrap and Build
- Bootstrap the project (first time setup): 
  - `make generate` -- NEVER CANCEL. Takes ~60 seconds first time, downloads controller-gen and other tools. Set timeout to 120+ seconds.
  - `make manifests` -- NEVER CANCEL. Takes ~4 seconds, generates Kubernetes manifests. Set timeout to 30+ seconds.
- Build all components:
  - `make manager` -- NEVER CANCEL. Takes ~90 seconds, builds webhook manager. Set timeout to 180+ seconds.
  - `make proxy` -- NEVER CANCEL. Takes ~60 seconds, builds proxy sidecar. Set timeout to 120+ seconds.
  - `make bin/azwi` -- Takes ~6 seconds, builds azwi CLI tool. Set timeout to 30+ seconds.

### Testing
- Unit tests: `make test` -- NEVER CANCEL. Takes ~4 minutes, includes generate and manifests. Set timeout to 10+ minutes.
- Linting: `make lint` -- NEVER CANCEL. Takes ~2 minutes first time, downloads golangci-lint. Set timeout to 5+ minutes.
- Shell script linting: `make shellcheck` -- Takes <1 second. Set timeout to 30+ seconds.
- Code formatting: `make fmt` -- Takes <1 second. Set timeout to 30+ seconds.

### Local Development with Kind
For local testing with Kind (requires Azure setup):
- Create service account keys: `openssl genrsa -out sa.key 2048 && openssl rsa -in sa.key -pubout -out sa.pub`
- Set up OIDC issuer: requires Azure storage account and SERVICE_ACCOUNT_ISSUER environment variable
- Create Kind cluster: `make kind-create` (requires proper Azure OIDC setup)
- Load images: `make kind-load-images`

## Validation

### Manual Testing Scenarios
- ALWAYS validate that binaries work after building:
  - `./bin/azwi version` should show version and git commit
  - `./bin/azwi --help` should show help text
  - `./bin/manager --help` should show webhook manager options
  - `./bin/proxy --help` should show proxy sidecar options
- ALWAYS run the complete test cycle when making changes:
  - `make generate && make manifests && make test` 
- ALWAYS run linting before committing: `make lint && make shellcheck && make fmt`

### Build Validation Commands
The project builds successfully and all tests pass. You can validate this by running:
```bash
# Verify prerequisites
go version  # Should be 1.24.6+
jq --version
make --version

# Full build and test cycle (run in order)
make generate    # ~60s first time
make manifests   # ~4s  
make test        # ~4m
make lint        # ~2m first time
make shellcheck  # <1s
make fmt         # <1s

# Build all binaries
make manager     # ~90s
make proxy       # ~60s  
make bin/azwi    # ~6s
```

### E2E Testing Requirements
- E2E tests require Azure credentials and OIDC issuer setup
- Use `make test-e2e` for full Azure-integrated testing (requires AZURE_TENANT_ID)
- Use `make test-e2e-run` for direct Ginkgo execution
- Local E2E testing requires SERVICE_ACCOUNT_ISSUER environment variable

## Common Tasks

### Repository Structure
```
/cmd/                    # Main applications (azwi, webhook, proxy)
  /azwi/                 # azwi CLI tool source
  /webhook/              # Webhook manager source  
  /proxy/                # Proxy sidecar source
/pkg/                    # Shared packages and libraries
/test/e2e/              # E2E test suite with Ginkgo
/config/                # Kubernetes manifests and kustomize configs
/scripts/               # Setup scripts (CI, Kind, AKS)
/docs/book/             # Documentation in mdbook format
/docker/                # Dockerfile definitions
/examples/              # Usage examples
/hack/                  # Build tools and utilities
```

### Key Files and Directories
- `Makefile` - Main build system with all targets
- `go.mod`, `go.sum` - Go module dependencies
- `.golangci.yml` - Linting configuration
- `/hack/tools/bin/` - Downloaded build tools (controller-gen, golangci-lint, etc.)
- `/bin/` - Built binaries
- `manifest_staging/` - Generated Kubernetes manifests

### Build Artifacts
After successful builds you will have:
- `/bin/azwi-linux-amd64` - azwi CLI tool
- `/bin/manager` - Webhook manager binary
- `/bin/proxy` - Proxy sidecar binary
- `/bin/e2e.test` - E2E test binary
- `manifest_staging/deploy/azure-wi-webhook.yaml` - Deployment manifest
- `manifest_staging/charts/workload-identity-webhook/` - Helm chart

### Development Workflow
1. Make code changes
2. Run `make generate` if modifying CRDs or generating code
3. Run `make manifests` if modifying Kubernetes resources
4. Run `make test` to validate unit tests  
5. Run `make lint` and `make shellcheck` for code quality
6. Build relevant binaries with `make manager`, `make proxy`, or `make bin/azwi`
7. Test manually with built binaries

### CI/CD Integration
- Always run `make lint` and `make shellcheck` before committing - CI will fail otherwise
- Unit tests with `make test` are required for PR validation
- E2E tests run in CI with Azure credentials
- Docker images are built with multi-arch support via `make docker-build`

## Performance Notes
- First-time builds download tools and take longer
- Subsequent builds are much faster due to Go build cache
- Unit test suite has 78.7% coverage and runs reliably
- Linting is comprehensive but may take 2+ minutes first time
- NEVER CANCEL long-running operations - they will complete successfully