# BinaryLane Cloud Controller Manager

A Kubernetes Cloud Controller Manager implementation for BinaryLane cloud infrastructure.

## Features

This cloud controller manager implements the following Kubernetes cloud provider interfaces:

- **Node Controller**: Manages node lifecycle and updates node metadata with cloud-specific information
- **Zone Controller**: Provides availability zone information for nodes

> **Note**: Load balancer support has been removed as BinaryLane's load balancer API is not fully compatible with the Kubernetes Load Balancer specification.

## Installation

### Prerequisites

- A Kubernetes cluster running on BinaryLane servers
- BinaryLane API token with appropriate permissions
- Kubernetes 1.24+

### Quick Start with Helm (Recommended)

```bash
# Add the Helm repository
helm repo add binarylane https://oscarhermoso.github.io/binarylane-cloud-controller-manager
helm repo update

# Install the chart
helm install binarylane-ccm binarylane/binarylane-cloud-controller-manager \
  --namespace kube-system \
  --set cloudControllerManager.apiToken="YOUR_API_TOKEN" \
  --set cloudControllerManager.region="per"
```

For more Helm configuration options, see the [Helm chart documentation](charts/binarylane-cloud-controller-manager/README.md).

### Manual Deployment with Kubernetes Manifests

1. **Create a secret with your BinaryLane API token:**

```bash
kubectl create secret generic binarylane-cloud-controller-manager \
  --from-literal=access-token=YOUR_BINARYLANE_API_TOKEN \
  -n kube-system
```

2. **Deploy the RBAC configuration:**

```bash
kubectl apply -f deploy/kubernetes/rbac.yaml
```

3. **Deploy the cloud controller manager:**

```bash
kubectl apply -f deploy/kubernetes/deployment.yaml
```

Note: Update the `BINARYLANE_REGION` environment variable in `deployment.yaml` to match your region.

### Building from Source

**API Client:**

The BinaryLane API client is automatically generated from the OpenAPI specification and committed to the repository. The generated files are:
- `pkg/binarylane/client_gen.go` - HTTP client implementation
- `pkg/binarylane/types_gen.go` - API type definitions
- `openapi.json` - BinaryLane OpenAPI specification

To regenerate the client (only needed when updating to a new API version):

```bash
go generate ./...
```

This will:
1. Fetch the latest OpenAPI spec from BinaryLane's API
2. Generate type-safe client code using `oapi-codegen`

**Build the Project:**

```bash
# Build the binary
make build

# Run tests
make test

# Build Docker image
make docker-build
```

## Configuration

### Environment Variables

- `BINARYLANE_ACCESS_TOKEN` (required): Your BinaryLane API token
- `BINARYLANE_REGION` (optional): The default region for resources


Nodes will be automatically configured with:
- Provider ID in the format `binarylane://<server-id>`
- Node addresses (internal and external IPs)
- Zone/region information
- Instance type metadata

## Development

### Project Structure

```
├── cmd/
│   └── binarylane-cloud-controller-manager/  # Main application
├── pkg/
│   ├── binarylane/                            # BinaryLane API client
│   └── cloud/                                 # Cloud provider implementation
├── deploy/
│   └── kubernetes/                            # Kubernetes manifests
├── Dockerfile
├── Makefile
└── README.md
```

### Running Tests

**Unit Tests:**
```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run linters
make lint
```

**End-to-End Tests:**

E2E tests deploy a real Kubernetes cluster on BinaryLane and verify CCM functionality:

```bash
# Via GitHub Actions (recommended)
# Go to: Actions → End-to-End Tests → Run workflow

# Via local script (deploys complete K8s cluster with CCM)
export BINARYLANE_API_TOKEN="your-token"
./scripts/deploy-k8s-cluster.sh

# Clean up after testing
./scripts/delete-cluster.sh
```

The deployment script is idempotent and can be safely re-run. It will:
- Create BinaryLane servers (or reuse existing ones)
- Install Kubernetes with kubeadm
- Deploy the BinaryLane Cloud Controller Manager
- Validate cluster health

See [E2E Testing Guide](docs/E2E_TESTING.md) for detailed information.

### API Client

The BinaryLane API client is located in `pkg/binarylane/` and provides methods for:
- Server management
- Load balancer management
- Network configuration

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.

## Support

For issues related to:
- **This cloud controller manager**: Open an issue in this repository
- **BinaryLane API**: Contact BinaryLane support at support@binarylane.com.au
- **Kubernetes**: Refer to the [Kubernetes documentation](https://kubernetes.io/docs/)

