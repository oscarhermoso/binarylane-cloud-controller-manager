# Kubernetes Cluster Deployment Guide

This guide explains how to deploy a Kubernetes cluster to BinaryLane using your API token.

## Prerequisites

Install required tools:
```bash
sudo apt-get update
sudo apt-get install -y curl jq ssh kubectl helm
```

## Configuration

Your API token is already configured in `.env`:
```bash
BINARYLANE_API_TOKEN=<your-token>
```

You can customize the deployment by setting these environment variables in `.env`:
```bash
CLUSTER_NAME=k8s-binarylane          # Name prefix for your cluster
REGION=syd                            # BinaryLane region (syd, per, bne)
CONTROL_PLANE_SIZE=std-min            # Control plane server size
WORKER_SIZE=std-min                   # Worker server size
WORKER_COUNT=1                        # Number of worker nodes
K8S_VERSION=1.29                      # Kubernetes version
```

## Deploy the Cluster

Run the deployment script:
```bash
./scripts/deploy-cluster.sh
```

This script will:
1. ✅ Create SSH keys if needed
2. ✅ Create 1 control plane server on BinaryLane
3. ✅ Create worker node(s) on BinaryLane
4. ✅ Install Kubernetes on all nodes
5. ✅ Initialize the cluster with kubeadm
6. ✅ Install Flannel CNI
7. ✅ Deploy the BinaryLane Cloud Controller Manager
8. ✅ Verify everything is working

**Deployment takes approximately 10-15 minutes.**

## Access Your Cluster

After deployment completes:

```bash
# Set kubeconfig
export KUBECONFIG=~/.kube/config-binarylane

# View nodes
kubectl get nodes

# View pods
kubectl get pods -A

# View cloud controller manager
kubectl get pods -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager
```

## SSH Access

Access the control plane:
```bash
ssh root@<control-plane-ip>
```

## Cost Estimate

Minimal cluster (1 control + 1 worker, std-min size):
- Approximately $10-20 AUD per month per server
- **Remember to delete when done testing!**

## Delete the Cluster

**IMPORTANT:** Delete your cluster when finished to avoid ongoing charges:

```bash
./scripts/delete-cluster.sh
```

This will:
- List all servers matching your cluster name
- Confirm before deletion
- Delete all cluster servers
- Clean up local kubeconfig

## Troubleshooting

### Check server creation
```bash
# List your servers
curl -X GET "https://api.binarylane.com.au/v2/servers" \
  -H "Authorization: Bearer $BINARYLANE_API_TOKEN" | jq '.servers[] | {id, name, status}'
```

### Check available sizes
```bash
curl -X GET "https://api.binarylane.com.au/v2/sizes" \
  -H "Authorization: Bearer $BINARYLANE_API_TOKEN" | jq '.sizes[] | {slug, memory, vcpus, price_monthly}'
```

### Check available regions
```bash
curl -X GET "https://api.binarylane.com.au/v2/regions" \
  -H "Authorization: Bearer $BINARYLANE_API_TOKEN" | jq '.regions[] | {slug, name, available}'
```

### SSH issues
If you can't SSH to servers:
1. Wait 2-3 minutes after server creation
2. Check server is "active" status
3. Verify SSH key was added correctly

### Cluster not forming
If nodes don't join:
1. Check control plane is fully initialized
2. Verify network connectivity between nodes
3. Check kubelet logs: `ssh root@<node-ip> "journalctl -u kubelet -n 50"`

## Manual Steps (if needed)

If the automated deployment fails, you can manually:

1. **Create servers** via BinaryLane web interface
2. **Install Kubernetes**:
   ```bash
   ssh root@<server-ip>
   # Follow Kubernetes install guide
   ```
3. **Deploy CCM**:
   ```bash
   helm install binarylane-ccm charts/binarylane-cloud-controller-manager \
     --namespace kube-system \
     --set cloudControllerManager.apiToken=$BINARYLANE_API_TOKEN \
     --set cloudControllerManager.region=syd
   ```

## Security Notes

- The deployment uses root SSH access (standard for BinaryLane)
- Consider setting up proper RBAC and user accounts for production
- Store your API token securely
- Use firewall rules to restrict access
- Consider setting up a VPC for private networking

## Next Steps

Once your cluster is running:
1. Deploy applications: `kubectl apply -f your-app.yaml`
2. Test the cloud controller manager features
3. Verify provider IDs are assigned to nodes
4. Check node metadata and zone information

## Support

For issues with:
- **This script**: Check the repository issues
- **BinaryLane API**: Contact support@binarylane.com.au
- **Kubernetes**: See kubernetes.io documentation
