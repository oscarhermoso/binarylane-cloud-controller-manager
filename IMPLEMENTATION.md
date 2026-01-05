# BinaryLane Cloud Controller Manager Implementation

This document provides details about the implementation of the BinaryLane Cloud Controller Manager for Kubernetes.

## Overview

The BinaryLane Cloud Controller Manager is a Kubernetes controller that enables integration between Kubernetes clusters and BinaryLane cloud infrastructure. It implements the standard Kubernetes Cloud Controller Manager interfaces to provide:

1. **Node Management**: Automatic node registration and metadata management
2. **Load Balancer Provisioning**: Automatic creation and management of BinaryLane load balancers
3. **Zone Awareness**: Topology information for scheduling and high availability

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────┐
│         Kubernetes Cloud Controller Manager             │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Instances  │  │LoadBalancers │  │    Zones     │  │
│  │  Controller  │  │  Controller  │  │  Controller  │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│         │                  │                  │          │
│         └──────────────────┴──────────────────┘          │
│                            │                             │
│                   ┌────────▼────────┐                    │
│                   │ BinaryLane API  │                    │
│                   │     Client      │                    │
│                   └─────────────────┘                    │
│                            │                             │
└────────────────────────────┼─────────────────────────────┘
                             │
                    ┌────────▼────────┐
                    │  BinaryLane API │
                    │ https://api     │
                    │ .binarylane     │
                    │ .com.au/v2      │
                    └─────────────────┘
```

### Package Structure

- **`cmd/binarylane-cloud-controller-manager/`**: Main application entry point
- **`pkg/binarylane/`**: BinaryLane API client library
- **`pkg/cloud/`**: Kubernetes cloud provider interface implementation
- **`deploy/kubernetes/`**: Kubernetes deployment manifests

## Implementation Details

### BinaryLane API Client (`pkg/binarylane/`)

The API client provides a Go interface to the BinaryLane API v2:

**Features:**
- HTTP client with Bearer token authentication
- Automatic JSON marshaling/unmarshaling
- Comprehensive error handling
- Support for pagination

**Key Methods:**
```go
// Server Management
GetServer(ctx, serverID) (*Server, error)
ListServers(ctx) ([]Server, error)
GetServerByName(ctx, name) (*Server, error)

// Load Balancer Management
CreateLoadBalancer(ctx, req) (*LoadBalancer, error)
GetLoadBalancer(ctx, lbID) (*LoadBalancer, error)
UpdateLoadBalancer(ctx, lbID, req) (*LoadBalancer, error)
DeleteLoadBalancer(ctx, lbID) error
AddServersToLoadBalancer(ctx, lbID, serverIDs) error
RemoveServersFromLoadBalancer(ctx, lbID, serverIDs) error
```

### Instances Controller (`pkg/cloud/instances.go`)

Implements the `cloudprovider.InstancesV2` interface for node management.

**Responsibilities:**
- Fetch instance metadata from BinaryLane API
- Populate node addresses (internal/external IPs)
- Set provider ID (format: `binarylane://<server-id>`)
- Provide instance type information
- Check instance existence

**Key Operations:**
1. **InstanceExists**: Checks if a server exists for a node
2. **InstanceShutdown**: Checks if a server is powered off
3. **InstanceMetadata**: Returns server metadata including:
   - Provider ID
   - Node addresses (internal/external)
   - Instance type
   - Region/zone information

### Load Balancer Controller (`pkg/cloud/loadbalancers.go`)

Implements the `cloudprovider.LoadBalancer` interface for service load balancer provisioning.

**Responsibilities:**
- Create BinaryLane load balancers for LoadBalancer services
- Configure forwarding rules based on service ports
- Set up health checks
- Manage backend server pools
- Update load balancer configuration on service changes
- Clean up load balancers on service deletion

**Key Operations:**
1. **GetLoadBalancer**: Retrieves existing load balancer by service name
2. **GetLoadBalancerName**: Generates unique load balancer name from service
3. **EnsureLoadBalancer**: Creates or updates load balancer to match service spec
4. **UpdateLoadBalancer**: Updates load balancer when nodes or service changes
5. **EnsureLoadBalancerDeleted**: Removes load balancer when service is deleted

**Service Annotations:**
- `service.beta.kubernetes.io/binarylane-loadbalancer-protocol`: Protocol (http/https)
- `service.beta.kubernetes.io/binarylane-loadbalancer-health-check-protocol`: Health check protocol
- `service.beta.kubernetes.io/binarylane-loadbalancer-health-check-path`: Health check path

### Zones Controller (`pkg/cloud/zones.go`)

Implements the `cloudprovider.Zones` interface for topology awareness.

**Responsibilities:**
- Provide region/zone information for nodes
- Enable topology-aware scheduling

**Key Operations:**
1. **GetZone**: Returns zone information for a node (maps to BinaryLane region)
2. **GetZoneByProviderID**: Returns zone information by provider ID
3. **GetZoneByNodeName**: Returns zone information by node name

## Data Flow

### Node Registration

```
1. Node starts with --cloud-provider=external flag
2. Kubelet creates Node object with providerID unset
3. Cloud Controller Manager detects new node
4. Instances controller calls InstanceMetadata()
5. InstanceMetadata() queries BinaryLane API for server info
6. Controller updates Node with:
   - providerID: binarylane://<server-id>
   - addresses: internal and external IPs
   - labels: region, zone, instance-type
7. Node becomes Ready
```

### Load Balancer Creation

```
1. User creates Service with type: LoadBalancer
2. Cloud Controller Manager detects new LoadBalancer service
3. LoadBalancer controller calls EnsureLoadBalancer()
4. Controller builds load balancer configuration:
   - Name from service namespace/name
   - Forwarding rules from service ports
   - Health checks from annotations or defaults
   - Backend servers from node list
5. Controller calls BinaryLane API to create load balancer
6. Controller updates Service status with load balancer IP
7. Traffic flows: Internet → Load Balancer → Service → Pods
```

### Load Balancer Updates

```
1. Service or nodes change
2. Cloud Controller Manager detects change
3. LoadBalancer controller calls UpdateLoadBalancer()
4. Controller fetches current load balancer config
5. Controller calculates desired config from current state
6. Controller updates load balancer via API:
   - Update backend server list
   - Update forwarding rules if ports changed
7. Load balancer reconfigured with new settings
```

## API Mapping

### BinaryLane API → Kubernetes Concepts

| BinaryLane Concept | Kubernetes Concept | Mapping |
|-------------------|-------------------|---------|
| Server | Node | Server ID → Provider ID |
| Server Region | Zone | Region slug → Zone label |
| Server IPv4 | Node Address | Primary IP → External address |
| Server VPC IPv4 | Node Address | VPC IP → Internal address |
| Load Balancer | Service (LoadBalancer) | LB name ← Service namespace/name |
| Forwarding Rule | Service Port | Port mapping |
| Health Check | Service annotation | Health check config |
| Load Balancer Server | Node | Backend pool membership |

## Configuration

### Environment Variables

- `BINARYLANE_ACCESS_TOKEN`: API authentication token (required)
- `BINARYLANE_REGION`: Default region for resources (optional)
- `BINARYLANE_VPC_ID`: Default VPC for load balancers (optional)

### Command-Line Flags

- `--cloud-provider=binarylane`: Cloud provider name
- `--cloud-config`: Path to cloud config file (optional)
- `--leader-elect`: Enable leader election for HA (default: true)

## Testing Strategy

### Unit Tests

1. **API Client Tests** (`pkg/binarylane/client_test.go`)
   - Mock HTTP responses
   - Test all API methods
   - Test error handling
   - Test JSON parsing

2. **Cloud Provider Tests** (`pkg/cloud/cloud_test.go`)
   - Mock API client
   - Test interface implementations
   - Test error conditions
   - Test configuration parsing

### Integration Testing

For integration testing with a real BinaryLane environment:

1. Set up test environment with API token
2. Create test servers
3. Run controller against test cluster
4. Verify node metadata population
5. Create LoadBalancer service
6. Verify load balancer creation
7. Clean up resources

## Known Limitations

1. **Sticky Sessions**: Not currently supported (BinaryLane API doesn't expose this)
2. **SSL Certificates**: Load balancer SSL certificate management not implemented
3. **Multiple VPCs**: Controller assumes single VPC or public network
4. **Regions**: Controller works with any region, but load balancers may be region-specific

## Future Enhancements

Potential areas for future development:

1. **Volume Management**: Implement volume/storage interface
2. **SSL Certificate Management**: Automate certificate provisioning for HTTPS load balancers
3. **Advanced Load Balancer Features**: Support for sticky sessions, custom timeouts
4. **Multi-VPC Support**: Better handling of multiple VPCs
5. **Metrics and Monitoring**: Prometheus metrics for controller operations
6. **Webhooks**: Admission webhooks for validation

## Security Considerations

1. **API Token Storage**: Token stored in Kubernetes Secret, mounted as environment variable
2. **RBAC**: Minimal required permissions defined in rbac.yaml
3. **TLS**: All API communication uses HTTPS
4. **Network Policies**: Consider network policies to restrict controller egress

## Performance Considerations

1. **API Rate Limiting**: Client respects BinaryLane API rate limits
2. **Caching**: Consider implementing caching for server/load balancer data
3. **Reconciliation**: Controller uses standard Kubernetes controller reconciliation patterns
4. **Resource Limits**: Container has appropriate CPU/memory limits set

## Troubleshooting

### Common Issues

1. **Nodes not getting metadata**
   - Verify nodes started with `--cloud-provider=external`
   - Check controller logs for API errors
   - Verify API token has correct permissions

2. **Load balancer not creating**
   - Check service annotations are correct
   - Verify API token can create load balancers
   - Check controller logs for errors
   - Verify region availability

3. **Controller not starting**
   - Verify secret exists with valid token
   - Check RBAC permissions
   - Review controller logs

### Debug Mode

Enable debug logging:
```bash
kubectl edit deployment binarylane-cloud-controller-manager -n kube-system
# Add --v=4 flag for verbose logging
```

## References

- [BinaryLane API Documentation](https://api.binarylane.com.au/reference/)
- [Kubernetes Cloud Controller Manager](https://kubernetes.io/docs/concepts/architecture/cloud-controller/)
- [Cloud Provider Interface](https://github.com/kubernetes/cloud-provider)
