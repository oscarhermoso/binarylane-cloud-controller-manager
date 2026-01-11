# BinaryLane Cloud Controller Manager Helm Chart

This Helm chart deploys the BinaryLane Cloud Controller Manager to your Kubernetes cluster.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.0+
- A BinaryLane API token

## Installation

### Add the Helm Repository

```bash
helm repo add binarylane https://oscarhermoso.github.io/binarylane-cloud-controller-manager
helm repo update
```

### Install the Chart

```bash
# Create secret with API token
kubectl create secret generic binarylane-api-token \
  --from-literal=api-token="YOUR_API_TOKEN" \
  -n kube-system

# Install the chart
helm install binarylane-ccm binarylane/binarylane-cloud-controller-manager \
  --namespace kube-system \
  --set cloudControllerManager.secret.name="binarylane-api-token" \
  --set cloudControllerManager.region="syd"
```

## Configuration

The following table lists the configurable parameters of the chart and their default values.

| Parameter                            | Description                             | Default                                                    |
| ------------------------------------ | --------------------------------------- | ---------------------------------------------------------- |
| `replicaCount`                       | Number of replicas                      | `1`                                                        |
| `image.repository`                   | Image repository                        | `ghcr.io/oscarhermoso/binarylane-cloud-controller-manager` |
| `image.pullPolicy`                   | Image pull policy                       | `IfNotPresent`                                             |
| `image.tag`                          | Image tag                               | Chart appVersion                                           |
| `cloudControllerManager.secret.name` | Name of secret containing API token     | `""`                                                       |
| `cloudControllerManager.secret.key`  | Key in secret for API token             | `api-token`                                                |
| `cloudControllerManager.region`      | BinaryLane region (e.g., syd, bne, per) | `""`                                                       |
| `cloudControllerManager.apiUrl`      | BinaryLane API URL                      | `https://api.binarylane.com.au`                            |
| `serviceAccount.create`              | Create service account                  | `true`                                                     |
| `serviceAccount.name`                | Service account name                    | Generated from template                                    |
| `resources.limits.cpu`               | CPU limit                               | `200m`                                                     |
| `resources.limits.memory`            | Memory limit                            | `128Mi`                                                    |
| `resources.requests.cpu`             | CPU request                             | `100m`                                                     |
| `resources.requests.memory`          | Memory request                          | `64Mi`                                                     |
| `nodeSelector`                       | Node selector                           | `node-role.kubernetes.io/control-plane: ""`                |
| `tolerations`                        | Tolerations                             | Control plane tolerations                                  |
| `hostNetwork`                        | Use host network                        | `true`                                                     |
| `priorityClassName`                  | Priority class                          | `system-cluster-critical`                                  |
| `verbosity`                          | Logging verbosity (0-10)                | `2`                                                        |
| `extraArgs`                          | Additional arguments                    | `[]`                                                       |
| `extraEnv`                           | Additional environment variables        | `[]`                                                       |

## Examples

### Basic Configuration

```yaml
# values.yaml
cloudControllerManager:
  secret:
    name: "binarylane-api-token"
  region: "syd"
```

### Advanced Configuration

```yaml
# values-advanced.yaml
cloudControllerManager:
  secret:
    name: "binarylane-api-token"
  region: "syd"
  apiUrl: "https://api.binarylane.com.au"

replicaCount: 2

resources:
  limits:
    cpu: 500m
    memory: 256Mi
  requests:
    cpu: 200m
    memory: 128Mi

extraArgs:
  - "--cluster-cidr=10.244.0.0/16"
  - "--configure-cloud-routes=false"

extraEnv:
  - name: HTTP_PROXY
    value: "http://proxy.example.com:8080"

tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
  - key: node-role.kubernetes.io/master
    operator: Exists
    effect: NoSchedule
  - key: my-custom-taint
    operator: Equal
    value: "true"
    effect: NoSchedule
```

### Using with Custom Node Labels

```yaml
# values-custom-nodes.yaml
cloudControllerManager:
  apiToken: "your-api-token-here"
  region: "syd"

nodeSelector:
  node-role.kubernetes.io/control-plane: ""
  custom-label: "ccm"

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchExpressions:
              - key: app.kubernetes.io/name
                operator: In
                values:
                  - binarylane-cloud-controller-manager
          topologyKey: kubernetes.io/hostname
```

## Upgrading

### To 0.2.0

No breaking changes.

## Uninstalling

```bash
helm uninstall binarylane-ccm -n kube-system
```

## Testing

Run Helm tests after installation:

```bash
helm test binarylane-ccm -n kube-system
```

## Troubleshooting

### Pods not starting

Check if the API token is correct:

```bash
kubectl get secret -n kube-system
kubectl describe pod -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager
```

### Permission errors

Verify RBAC resources are created:

```bash
kubectl get clusterrole,clusterrolebinding -l app.kubernetes.io/name=binarylane-cloud-controller-manager
```

### Node not getting metadata

Check controller logs:

```bash
kubectl logs -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager
```

## Support

For issues related to:
- **This Helm chart**: Open an issue in the GitHub repository
- **BinaryLane Cloud Controller Manager**: Check the main project documentation
- **BinaryLane API**: Contact support@binarylane.com.au
