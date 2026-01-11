# BinaryLane Cloud Controller Manager

A Kubernetes Cloud Controller Manager implementation for BinaryLane cloud provider.

## Features

This [cloud controller manager](https://kubernetes.io/docs/concepts/architecture/cloud-controller/) implements the following Kubernetes cloud provider interfaces:

- **Instances Controller**: Manages node lifecycle and updates node metadata with cloud-specific information
- **Zones Controller**: Provides availability zone information for nodes

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
  --set cloudControllerManager.secret.name="binarylane-api-token"
```

For more Helm configuration options, see the [Helm chart documentation](charts/binarylane-cloud-controller-manager/README.md).

### Manual Deployment with Kubernetes Manifests

1. **Create a secret with your BinaryLane API token:**

```bash
kubectl create secret generic binarylane-api-token \
  --from-literal=api-token=YOUR_BINARYLANE_API_TOKEN \
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

## Configuration

### Environment Variables

- `BINARYLANE_ACCESS_TOKEN` (required): Your BinaryLane API token

Nodes are automatically detected and configured with:
- Provider ID in the format `binarylane://<server-id>`
- Node addresses (internal and external IPs)
- Zone/region information (queried per-node from BinaryLane API)

## Contributing

Want to help? Check out [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and code guidelines.

