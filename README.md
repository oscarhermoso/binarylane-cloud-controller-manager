# BinaryLane Cloud Controller Manager

A Kubernetes Cloud Controller Manager implementation for BinaryLane cloud infrastructure.

## Features

This cloud controller manager implements the following Kubernetes cloud provider interfaces:

- **Node Controller**: Manages node lifecycle and updates node metadata with cloud-specific information
- **Service Controller**: Provisions and manages load balancers for LoadBalancer-type services
- **Zone Controller**: Provides availability zone information for nodes

## Installation

### Prerequisites

- A Kubernetes cluster running on BinaryLane servers
- BinaryLane API token with appropriate permissions
- Kubernetes 1.24+

### Deployment

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

**Generate API Client:**

The BinaryLane API client is automatically generated from the OpenAPI specification:

```bash
# Generate the API client
go generate ./...
```

This will:
1. Fetch the latest OpenAPI spec from BinaryLane's API
2. Generate type-safe client code using `oapi-codegen`

The generated files (`client_gen.go` and `types_gen.go`) are gitignored and should be regenerated during build.

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

### Service Annotations

When creating a LoadBalancer service, you can use the following annotations:

- `service.beta.kubernetes.io/binarylane-loadbalancer-protocol`: Protocol for the load balancer (http or https)
- `service.beta.kubernetes.io/binarylane-loadbalancer-healthcheck-protocol`: Health check protocol
- `service.beta.kubernetes.io/binarylane-loadbalancer-healthcheck-path`: Health check path (default: "/")

Example service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    service.beta.kubernetes.io/binarylane-loadbalancer-protocol: "https"
    service.beta.kubernetes.io/binarylane-loadbalancer-healthcheck-path: "/health"
spec:
  type: LoadBalancer
  ports:
    - port: 443
      targetPort: 8080
  selector:
    app: my-app
```

## Node Configuration

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

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run linters
make lint
```

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

