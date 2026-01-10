# BinaryLane Cloud Controller Manager

A Kubernetes Cloud Controller Manager implementation for BinaryLane cloud infrastructure.

## Features

This cloud controller manager implements the following Kubernetes cloud provider interfaces:

- **Instances Controller**: Manages node lifecycle and updates node metadata with cloud-specific information
- **Zones Controller**: Provides availability zone information for nodes

> **Note**: Load balancer support is not currently implemented.

## Installation

### Prerequisites

- A Kubernetes cluster running on BinaryLane servers
- BinaryLane API token with appropriate permissions
- Kubernetes 1.24+

### Quick Start with Helm

```bash
# Create a secret with your API token
kubectl create secret generic binarylane-api-token \
  --from-literal=api-token="YOUR_API_TOKEN" \
  -n kube-system

# Add the Helm repository
helm repo add binarylane https://oscarhermoso.github.io/binarylane-cloud-controller-manager
helm repo update

# Install using the existing secret
helm install binarylane-ccm binarylane/binarylane-cloud-controller-manager \
  --namespace kube-system \
  --set cloudControllerManager.existingSecret="binarylane-api-token" \
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
- `internal/binarylane/client_gen.go` - HTTP client implementation
- `internal/binarylane/types_gen.go` - API type definitions
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
make build
make docker-build
```

## Configuration

### Environment Variables

- `BINARYLANE_ACCESS_TOKEN` (required): Your BinaryLane API token
- `BINARYLANE_REGION` (required): The BinaryLane region where the cloud controller manager runs (typically the control plane region)

### Multi-Region Clusters

The cloud controller manager supports clusters with nodes in multiple BinaryLane regions. Each node will be automatically labeled with its actual region from the BinaryLane API:

- `topology.kubernetes.io/zone` - The node's BinaryLane region (e.g., "syd", "per", "bne")
- `topology.kubernetes.io/region` - The node's BinaryLane region (same as zone)

The `BINARYLANE_REGION` environment variable is used for the controller manager's own zone awareness and does not restrict which regions your nodes can be in.

### Node Configuration

Nodes will be automatically configured with:
- Provider ID in the format `binarylane://<server-id>`
- Node addresses (internal and external IPs)
- Zone/region labels based on each VM's actual BinaryLane region

## Development

### Project Structure

```
├── cmd/
│   └── binarylane-cloud-controller-manager/  # Main application
├── internal/
│   ├── binarylane/                            # Generated BinaryLane API client
│   └── cloud/                                 # Cloud provider implementation
├── charts/
│   └── binarylane-cloud-controller-manager/  # Helm chart
├── deploy/
│   └── kubernetes/                            # Kubernetes manifests
├── scripts/                                   # Deployment and testing scripts
├── Dockerfile
├── Makefile
└── README.md
```

### Running Tests

**Unit Tests:**
```bash
make test
make coverage
make lint
```

**End-to-End Tests:**

E2E tests deploy a real Kubernetes cluster on BinaryLane and verify CCM functionality:

```bash
export BINARYLANE_API_TOKEN="your-token"
./scripts/deploy-cluster.sh

./scripts/delete-cluster.sh
```

The deployment script is idempotent and can be safely re-run. It will:
- Create BinaryLane servers (or reuse existing ones)
- Install Kubernetes with kubeadm
- Deploy the BinaryLane Cloud Controller Manager
- Validate cluster health

See [E2E Testing Guide](docs/E2E_TESTING.md) for detailed information.

### API Client

The BinaryLane API client is located in `internal/binarylane/` and is automatically generated from the BinaryLane OpenAPI specification. It provides type-safe methods for:
- Server management and queries
- Network information retrieval

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
