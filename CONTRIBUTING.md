# Contributing to BinaryLane Cloud Controller Manager

Thanks for your interest in contributing! Here's what you need to know.

## Development Setup

### Project Structure

```
├── cmd/
│   └── binarylane-cloud-controller-manager/  # Main application
├── pkg/
│   ├── binarylane/                            # Generated BinaryLane API client
│   └── cloud/                                 # Cloud provider implementation
├── charts/
│   └── binarylane-cloud-controller-manager/  # Helm chart
├── deploy/
│   └── kubernetes/                            # Kubernetes manifests
├── scripts/                                   # Deployment and testing scripts
├── Dockerfile
├── Makefile
└── go.mod
```

### Prerequisites

- Go 1.25+
- Docker (for building container images)
- kubectl (for E2E testing)
- BinaryLane API token (for E2E testing)

## Building

```bash
make build       # Build the binary
make docker-build # Build the Docker image
```

## Testing

### Unit Tests

```bash
make test        # Run unit tests
make coverage    # Run tests with coverage
make lint        # Run linting
```

The test workflow checks:
- Code formatting with `gofmt`
- Code analysis with `go vet`
- Generated files are up to date
- Tests pass with race detector enabled
- Test coverage is tracked

### Regenerating the API Client

The BinaryLane API client is auto-generated from the OpenAPI spec. If you need to update it:

```bash
go generate ./...
```

This fetches the latest OpenAPI spec and regenerates the type-safe client code.

### End-to-End Tests

E2E tests deploy a real Kubernetes cluster on BinaryLane and verify the CCM works:

```bash
export BINARYLANE_API_TOKEN="your-token"
./scripts/deploy-cluster.sh
```

When you're done testing:

```bash
./scripts/delete-cluster.sh
```

The deploy script handles:
- Creating BinaryLane servers (or reusing existing ones)
- Installing Kubernetes with kubeadm
- Deploying the cloud controller manager
- Validating cluster health

## Making Changes

1. Fork the repo and create a feature branch
2. Make your changes
3. Run tests locally: `make test lint`
4. Commit and push
5. Open a pull request with a clear description

## Code Style

- Follow standard Go conventions
- Run `gofmt` before committing
- Generated files should not be manually edited
