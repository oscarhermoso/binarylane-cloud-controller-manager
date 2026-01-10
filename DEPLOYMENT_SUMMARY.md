# Deployment Script Consolidation Summary

## Changes Made

### Scripts Consolidated
All deployment scripts have been consolidated into a single, comprehensive script:

**New Script:**
- `scripts/deploy-k8s-cluster.sh` - Complete, idempotent deployment script

**Removed Scripts:**
- `scripts/continue-deployment.sh` - Functionality merged into main script
- `scripts/deploy-cluster.sh` - Replaced by deploy-k8s-cluster.sh
- `scripts/deploy-simple.py` - No longer needed
- `scripts/install-k8s-ccm.sh` - Integrated into main script
- `scripts/install-k8s-now.sh` - Integrated into main script

**Kept Scripts:**
- `scripts/delete-cluster.sh` - Cleanup script (unchanged)
- `scripts/e2e-test.sh` - E2E test runner (unchanged)
- `scripts/fetch-openapi.sh` - API client generator (unchanged)

## New deploy-k8s-cluster.sh Features

### Idempotency
The script can be safely re-run multiple times:
- Reuses existing servers (doesn't recreate)
- Skips Kubernetes initialization if already done
- Skips CCM deployment if already installed
- Updates provider IDs as needed

### Comprehensive Deployment
Single script handles everything:
1. **Environment Validation** - Checks for required tools and credentials
2. **Server Management** - Creates or reuses BinaryLane servers
3. **Kubernetes Installation** - Full K8s setup with cloud-provider=external
4. **CCM Deployment** - Builds image, deploys with Helm, sets provider IDs
5. **Health Validation** - Comprehensive cluster health checks

### Configuration
All settings configurable via environment variables:
```bash
export CLUSTER_NAME="k8s-binarylane"     # Cluster name prefix
export REGION="per"                       # BinaryLane region
export SERVER_SIZE="std-2vcpu"            # Server size
export CONTROL_PLANE_COUNT="1"            # Number of control planes
export WORKER_COUNT="2"                   # Number of workers
export K8S_VERSION="1.29.15"              # Kubernetes version
export POD_NETWORK_CIDR="10.244.0.0/16"   # Pod network CIDR
```

### Validation Output
After deployment, the script shows:
- Node status (all nodes Ready)
- Provider IDs (binarylane://server-id format)
- Node addresses (public IPs)
- CCM pod status
- System pods status
- Cluster connection info

## Usage

### Deploy Cluster
```bash
# Set API token (or use .env file)
export BINARYLANE_API_TOKEN="your-token"

# Optional: Configure deployment
export REGION="per"        # Perth
export WORKER_COUNT="2"    # 2 workers

# Deploy
./scripts/deploy-k8s-cluster.sh
```

### Use Cluster
```bash
# Set kubeconfig
export KUBECONFIG=~/.kube/config-binarylane

# Use kubectl
kubectl get nodes
kubectl get pods --all-namespaces
```

### Delete Cluster
```bash
./scripts/delete-cluster.sh
```

## Documentation Updates

Updated documentation to reflect new simplified deployment:
- `README.md` - Updated E2E testing section
- `docs/E2E_TESTING.md` - Comprehensive update with new script usage

## Benefits

1. **Simplified**: One command deploys everything
2. **Safe**: Idempotent design prevents errors from re-runs
3. **Reliable**: Comprehensive error handling and validation
4. **Maintainable**: Single script is easier to update and debug
5. **Clear**: Color-coded output and progress indicators
6. **Complete**: Includes full validation and health checks

## Cost Information

Running the default configuration (3 nodes):
- std-2vcpu: ~$0.021/hour per server (~$15/month)
- Total: ~$0.063/hour (~$45/month for 3 servers)

**Always clean up when done:**
```bash
./scripts/delete-cluster.sh
```
