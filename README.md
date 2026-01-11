# BinaryLane Cloud Controller Manager

A Kubernetes Cloud Controller Manager implementation for BinaryLane cloud provider.

## Features

This [cloud controller manager](https://kubernetes.io/docs/concepts/architecture/cloud-controller/) implements the following Kubernetes cloud provider interfaces:

- **Instances Controller**: Manages node lifecycle and updates node metadata with cloud-specific information
- **Zones Controller**: Provides availability zone information for nodes


The cloud controller manager automatically applies the following labels to nodes:

| Label                              | Description                                                                  |
| ---------------------------------- | ---------------------------------------------------------------------------- |
| `binarylane.com/host`              | Physical host machine name (if running on shared infrastructure)             |
| `node.kubernetes.io/instance-type` | Server size (e.g., `std-2vcpu`)                                              |
| `topology.kubernetes.io/region`    | Server region (e.g., `syd`, `per`)                                           |
| `topology.kubernetes.io/zone`      | Same as region, may update to distinguish between data centres in the future |


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

## Contributing

Want to help? Check out [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and code guidelines.

