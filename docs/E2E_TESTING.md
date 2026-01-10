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
# Export your BinaryLane API token
export BINARYLANE_API_TOKEN="your-token-here"

# Optional: Configure test parameters
export CLUSTER_NAME="my-test-cluster"
export REGION="syd"

# Run the E2E test
./scripts/e2e-test.sh
```

## What the Tests Do

### 1. Infrastructure Provisioning
- Creates 2 BinaryLane servers:
  - Control plane node (size: std-min)
  - Worker node (size: std-min)
- Waits for servers to be in `active` state
- Configures SSH access

### 2. Kubernetes Installation
- Installs container runtime (containerd)
- Installs Kubernetes components (kubeadm, kubelet, kubectl)
- Initializes cluster with `--cloud-provider=external` flag
- Installs CNI plugin (Flannel)
- Joins worker node to cluster

### 3. Cloud Controller Manager Deployment
- Builds CCM from source
- Creates BinaryLane API credentials secret
- Creates cloud configuration ConfigMap
- Deploys CCM using Helm chart
- Waits for CCM to be ready

### 4. Verification Tests
- ✅ CCM pods are running
- ✅ All nodes have provider IDs (format: `binarylane://<server-id>`)
- ✅ Node metadata is populated:
  - IP addresses (public, private)
  - Topology labels (zone, region)
- ✅ Test workload can be scheduled
- ✅ Pods distribute across zones

### 5. Cleanup
- Deletes all created BinaryLane servers
- Removes test artifacts

## Test Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BINARYLANE_API_TOKEN` | API token for BinaryLane | (required) |
| `CLUSTER_NAME` | Prefix for cluster resources | `ccm-e2e-test` |
| `REGION` | BinaryLane region | `syd` |
| `CONTROL_PLANE_SIZE` | Control plane server size | `std-min` |
| `WORKER_SIZE` | Worker node server size | `std-min` |
| `IMAGE` | OS image for servers | `ubuntu-22.04` |

### Available Regions

- `syd` - Sydney, Australia
- `mel` - Melbourne, Australia
- `per` - Perth, Australia
- (check BinaryLane API for full list)

## Cost Considerations

⚠️ **Important**: Running E2E tests will incur costs on your BinaryLane account.

- Control plane server: ~$0.01-0.02 per hour (std-min)
- Worker server: ~$0.01-0.02 per hour (std-min)
- Total test duration: ~30-45 minutes
- **Estimated cost per test run**: ~$0.02-0.03

The cleanup step ensures servers are deleted after tests complete or fail.

## Troubleshooting

### Test Failures

**Servers fail to become active:**
- Check BinaryLane account has sufficient quota
- Verify region has available capacity
- Check API token has correct permissions

**SSH connection failures:**
- Ensure security groups allow SSH access
- Verify server passwords are retrieved correctly
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
   - Tests use temporary SSH keys
   - Keys are not stored permanently
   - Consider using BinaryLane's SSH key management

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
