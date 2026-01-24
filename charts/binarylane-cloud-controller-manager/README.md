# BinaryLane Cloud Controller Manager Helm Chart

This Helm chart deploys the BinaryLane Cloud Controller Manager to your Kubernetes cluster.

## Installation

See the main [README.md](../../README.md#installation) for installation instructions.

## Configuration

The following table lists the configurable parameters of the chart and their default values.

| Parameter                            | Description                         | Default                                                    |
| ------------------------------------ | ----------------------------------- | ---------------------------------------------------------- |
| `replicaCount`                       | Number of replicas                  | `1`                                                        |
| `image.repository`                   | Image repository                    | `ghcr.io/oscarhermoso/binarylane-cloud-controller-manager` |
| `image.pullPolicy`                   | Image pull policy                   | `IfNotPresent`                                             |
| `image.tag`                          | Image tag                           | Chart appVersion                                           |
| `cloudControllerManager.secret.name` | Name of secret containing API token | `""`                                                       |
| `cloudControllerManager.secret.key`  | Key in secret for API token         | `api-token`                                                |
| `serviceAccount.create`              | Create service account              | `true`                                                     |
| `serviceAccount.name`                | Service account name                | Generated from template                                    |
| `resources.limits.cpu`               | CPU limit                           | `200m`                                                     |
| `resources.limits.memory`            | Memory limit                        | `128Mi`                                                    |
| `resources.requests.cpu`             | CPU request                         | `100m`                                                     |
| `resources.requests.memory`          | Memory request                      | `64Mi`                                                     |
| `nodeSelector`                       | Node selector                       | `node-role.kubernetes.io/control-plane: ""`                |
| `tolerations`                        | Tolerations                         | Control plane tolerations                                  |
| `hostNetwork`                        | Use host network                    | `true`                                                     |
| `priorityClassName`                  | Priority class                      | `system-cluster-critical`                                  |
| `verbosity`                          | Logging verbosity (0-10)            | `2`                                                        |
| `extraArgs`                          | Additional arguments                | `[]`                                                       |
| `extraEnv`                           | Additional environment variables    | `[]`                                                       |

## Examples

### Basic Configuration

```yaml
# values.yaml
cloudControllerManager:
  secret:
    name: "binarylane-api-token"
```

### Advanced Configuration

```yaml
# values-advanced.yaml
cloudControllerManager:
  secret:
    name: "binarylane-api-token"

replicaCount: 2

resources:
  limits:
    cpu: 500m
    memory: 256Mi
  requests:
    cpu: 200m
    memory: 128Mi

extraArgs:
  - "--kube-api-qps=50"
  - "--kube-api-burst=100"

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
