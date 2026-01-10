# End-to-End Testing Guide

This guide explains how to run end-to-end tests for the BinaryLane Cloud Controller Manager.

## Overview

The E2E tests deploy a real Kubernetes cluster on BinaryLane infrastructure and verify that the cloud controller manager correctly:
- Assigns provider IDs to nodes
- Populates node metadata (addresses, zones, regions)
- Enables topology-aware scheduling
- Maintains node lifecycle

## Prerequisites

### For GitHub Actions

1. Set up a BinaryLane API token as a repository secret:
   ```
   Repository Settings → Secrets and variables → Actions → New repository secret
   Name: BINARYLANE_API_TOKEN
   Value: <your-binarylane-api-token>
   ```

2. Ensure your account has permissions to:
   - Create and delete servers
   - Access the BinaryLane API

### For Local Testing

Install required tools:
```bash
# On Ubuntu/Debian
sudo apt-get install curl jq kubectl helm docker.io ssh sshpass

# On macOS
brew install curl jq kubectl helm docker ssh
# For sshpass on macOS:
brew install hudochenkov/sshpass/sshpass
```

## Running E2E Tests

### Via GitHub Actions (Recommended)

The E2E tests can be triggered manually or run on a schedule:

**Manual Trigger:**
1. Go to: Actions → End-to-End Tests → Run workflow
2. Configure options:
   - `cluster_name`: Name prefix for test servers (default: `ccm-e2e-test`)
   - `region`: BinaryLane region (default: `syd`)
3. Click "Run workflow"

**Scheduled Run:**
- Automatically runs daily at 2 AM UTC
- Results available in the Actions tab

### Via Local Script

```bash
export BINARYLANE_API_TOKEN="your-token-here"

# Optional: Configure test parameters
export CLUSTER_NAME="my-test-cluster"
export REGION="per"
export WORKER_COUNT="2"

./scripts/deploy-cluster.sh

./scripts/delete-cluster.sh
```

## What the Tests Do

### 1. Infrastructure Provisioning
- Creates BinaryLane servers (1 control plane + N workers)
- Uses std-2vcpu size (required for Kubernetes minimum specs)
- Waits for servers to be in `active` state
- Verifies SSH connectivity

### 2. Kubernetes Installation
- Installs container runtime (containerd with systemd cgroup)
- Installs Kubernetes 1.29.15 (kubeadm, kubelet, kubectl)
- Configures kubelet with `--cloud-provider=external` flag
- Initializes cluster with kubeadm
- Installs Flannel CNI plugin
- Joins worker nodes to cluster

### 3. Cloud Controller Manager Deployment
- Builds CCM Docker image from source
- Imports image to control plane node
- Deploys CCM using Helm chart with:
  - BinaryLane API token
  - Region configuration
  - Local image (no registry pull)
- Waits for CCM pods to be ready
- Sets provider IDs for nodes

### 4. Verification & Validation
- ✅ All nodes are Ready
- ✅ Provider IDs assigned (format: `binarylane://<server-id>`)
- ✅ Node addresses populated correctly
- ✅ CCM pod is Running
- ✅ Test workloads can be scheduled and run
- ✅ Full cluster health validation

### 5. Cleanup
- Deletes all BinaryLane servers
- Removes local kubeconfig

## Test Configuration

### Environment Variables

| Variable               | Description                   | Default          |
| ---------------------- | ----------------------------- | ---------------- |
| `BINARYLANE_API_TOKEN` | API token for BinaryLane      | (required)       |
| `CLUSTER_NAME`         | Prefix for cluster resources  | `k8s-binarylane` |
| `REGION`               | BinaryLane region             | `per`            |
| `SERVER_SIZE`          | Server size for all nodes     | `std-2vcpu`      |
| `CONTROL_PLANE_COUNT`  | Number of control plane nodes | `1`              |
| `WORKER_COUNT`         | Number of worker nodes        | `2`              |
| `K8S_VERSION`          | Kubernetes version            | `1.29.15`        |
| `POD_NETWORK_CIDR`     | Pod network CIDR              | `10.244.0.0/16`  |

### Available Regions

- `syd` - Sydney, Australia
- `mel` - Melbourne, Australia
- `per` - Perth, Australia
- (check BinaryLane API for full list)

## Cost Considerations

⚠️ **Important**: Running E2E tests will incur costs on your BinaryLane account.

- std-2vcpu server: ~$0.021 per hour (~$15/month)
- Default configuration: 3 servers (1 control plane + 2 workers)
- Total hourly cost: ~$0.063/hour
- **Estimated cost for 1-hour test**: ~$0.06
- **Monthly cost if left running**: ~$45

**Important**: Always run the cleanup script when done:
```bash
./scripts/delete-cluster.sh
```

The cleanup script deletes all servers to prevent ongoing charges.

## Troubleshooting

### Test Failures

**Servers fail to become active:**
- Check BinaryLane account has sufficient quota
- Verify region has available capacity
- Check API token has correct permissions

**SSH connection failures:**
- Wait longer for servers to fully initialize
- Check that SSH keys are properly configured in BinaryLane account
- Verify network connectivity to BinaryLane servers

**Kubernetes installation fails:**
- Ensure servers meet minimum requirements (2 vCPU, 4GB RAM)
- Check that containerd is running properly
- Verify kubeadm can access required container registries

**CCM deployment fails:**
- Verify BINARYLANE_API_TOKEN is set correctly
- Check that Docker image built successfully
- Ensure Helm chart is valid (run `helm lint charts/binarylane-cloud-controller-manager`)

**Provider IDs not set:**
- This is expected on first deployment (nodes existed before CCM)
- The script automatically sets provider IDs after CCM deployment
- Run the script again to verify they persist

### Manual Cleanup

If the cleanup script fails, manually delete servers:

```bash
# List cluster servers
source .env
curl -s -H "Authorization: Bearer $BINARYLANE_API_TOKEN" \
  "https://api.binarylane.com.au/v2/servers?per_page=200" | \
  jq '.servers[] | select(.name | contains("k8s-binarylane"))'

# Delete each server
curl -X DELETE -H "Authorization: Bearer $BINARYLANE_API_TOKEN" \
  "https://api.binarylane.com.au/v2/servers/<server-id>"
```

### Idempotency

The deployment script is idempotent and can be safely re-run:
- Existing servers are reused (not recreated)
- Kubernetes will not be reinitialized if already running
- CCM will not be reinstalled if already deployed
- Provider IDs are set/updated as needed

This makes it safe to:
- Resume interrupted deployments
- Update CCM configuration
- Verify cluster state
- Check network connectivity to BinaryLane servers

**Kubernetes installation fails:**
- Review server logs via BinaryLane console
- Check if servers have sufficient resources
- Verify internet connectivity for package downloads

**CCM deployment fails:**
- Check API token is valid
- Review CCM pod logs: `kubectl logs -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager`
- Verify cloud config is correct

**Node provider IDs not set:**
- Ensure nodes were initialized with `--cloud-provider=external`
- Check CCM is running and not crashing
- Review CCM logs for errors

### Viewing Test Logs

**GitHub Actions:**
- Navigate to Actions → End-to-End Tests → Select run
- View logs for each step
- Download artifacts (kubeconfig, logs)

**Local Testing:**
- Logs are printed to stdout
- Check `~/.kube/config` for cluster access
- Use `kubectl logs` to view CCM logs

## CI/CD Integration

The E2E tests are integrated into the development workflow:

```yaml
# Triggered on:
- Manual dispatch (workflow_dispatch)
- Daily schedule (cron: 0 2 * * *)

# Can be extended to:
- Run on release tags
- Run on pull requests (with approval)
- Parallel tests in multiple regions
```

## Security Notes

1. **API Token Security**:
   - Never commit API tokens to repository
   - Use GitHub Secrets for CI/CD
   - Rotate tokens regularly

2. **SSH Access**:
   - Tests use password-based authentication for convenience
   - SSH host key checking is disabled for test environments
   - **Note**: These practices are acceptable for temporary test infrastructure but should NOT be used in production
   - For production deployments, use SSH key-based authentication via BinaryLane's SSH key management

3. **Resource Cleanup**:
   - Cleanup runs even if tests fail
   - Manual verification recommended
   - Set up billing alerts

## Future Enhancements

Potential improvements for E2E testing:

- [ ] Multi-region testing
- [ ] HA control plane testing
- [ ] Load balancer integration tests (if supported)
- [ ] Upgrade path testing
- [ ] Performance benchmarks
- [ ] Network policy testing
- [ ] Storage provisioning tests

## Support

For issues related to:
- **BinaryLane API**: https://support.binarylane.com.au
- **Cloud Controller Manager**: Open an issue in this repository
- **Kubernetes**: https://kubernetes.io/docs/home/
