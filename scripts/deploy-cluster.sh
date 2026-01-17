#!/bin/bash
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-binarylane-ccm}"
REGION="${REGION:-per}"
SERVER_SIZE="${SERVER_SIZE:-std-min}"
CONTROL_PLANE_COUNT="${CONTROL_PLANE_COUNT:-1}"
WORKER_COUNT="${WORKER_COUNT:-2}"
K8S_VERSION="${K8S_VERSION:-1.29.15}"
POD_NETWORK_CIDR="${POD_NETWORK_CIDR:-10.244.0.0/16}"
SSH_KEY_PATH="${SSH_KEY_PATH:-.ssh/binarylane-k8s}"
SSH_KEY_NAME="${SSH_KEY_NAME:-binarylane-k8s-cluster}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

#=============================================================================
# Helper Functions
#=============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

validate_environment() {
    log_info "Validating environment..."

    # Check for required environment variables
    if [ -f .env ]; then
        source .env
    fi

    if [ -z "${BINARYLANE_API_TOKEN:-}" ]; then
        log_error "BINARYLANE_API_TOKEN not set. Please set it in .env or as an environment variable"
        exit 1
    fi

    # Check for required tools
    local required_tools="curl jq ssh ssh-keygen docker kubectl helm"
    for tool in $required_tools; do
        if ! command -v $tool &> /dev/null; then
            log_error "Required tool not found: $tool"
            exit 1
        fi
    done

    log_success "Environment validated"
}

generate_and_upload_ssh_key() {
    log_info "Setting up SSH key for cluster..."

    # Generate SSH key if it doesn't exist
    if [ ! -f "$SSH_KEY_PATH" ]; then
        log_info "Generating new SSH key pair: $SSH_KEY_PATH"
        mkdir -p "$(dirname "$SSH_KEY_PATH")"
        ssh-keygen -t ed25519 -f "$SSH_KEY_PATH" -N "" -C "$SSH_KEY_NAME"
        log_success "SSH key pair generated"
    else
        log_info "Using existing SSH key: $SSH_KEY_PATH"
    fi

    # Check if key is already uploaded to BinaryLane
    local public_key=$(cat "${SSH_KEY_PATH}.pub")
    local existing_key=$(api_call GET "/account/keys" | jq -r ".ssh_keys[] | select(.name == \"$SSH_KEY_NAME\")")

    if [ -n "$existing_key" ]; then
        SSH_KEY_ID=$(echo "$existing_key" | jq -r '.id')
        log_info "SSH key already uploaded (ID: $SSH_KEY_ID)"
    else
        log_info "Uploading SSH key to BinaryLane..."
        local response=$(api_call POST "/account/keys" "{\"name\": \"$SSH_KEY_NAME\", \"public_key\": \"$public_key\"}")
        SSH_KEY_ID=$(echo "$response" | jq -r '.ssh_key.id')

        if [ -z "$SSH_KEY_ID" ] || [ "$SSH_KEY_ID" == "null" ]; then
            log_error "Failed to upload SSH key"
            exit 1
        fi

        log_success "SSH key uploaded (ID: $SSH_KEY_ID)"
    fi
}

api_call() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"

    local url="https://api.binarylane.com.au/v2${endpoint}"

    if [ -n "$data" ]; then
        curl -s --max-time 30 --connect-timeout 10 -X "$method" "$url" \
            -H "Authorization: Bearer $BINARYLANE_API_TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data"
    else
        curl -s --max-time 30 --connect-timeout 10 -X "$method" "$url" \
            -H "Authorization: Bearer $BINARYLANE_API_TOKEN"
    fi
}

get_server_by_name() {
    local name="$1"
    api_call GET "/servers?per_page=200" | jq -r ".servers[] | select(.name == \"$name\")"
}

wait_for_server_ready() {
    local server_id="$1"
    local max_attempts=60
    local attempt=0

    log_info "Waiting for server $server_id to be ready..."

    while [ $attempt -lt $max_attempts ]; do
        local response=$(api_call GET "/servers/$server_id" 2>&1)
        local status=$(echo "$response" | jq -r '.server.status' 2>/dev/null)

        # Debug: Show status every 10 attempts
        if [ $((attempt % 10)) -eq 0 ] && [ $attempt -gt 0 ]; then
            log_info "Status after $attempt attempts: '$status'" >&2
        fi

        if [ "$status" == "active" ]; then
            echo "" >&2  # New line after dots
            log_success "Server $server_id is ready"
            return 0
        elif [ -z "$status" ] || [ "$status" == "null" ]; then
            log_error "Failed to get server status (attempt $attempt/$max_attempts)"
            log_error "API Response: $response"
            if [ $attempt -ge 5 ]; then
                return 1
            fi
        fi

        echo -n "." >&2
        sleep 5
        attempt=$((attempt + 1))
    done

    echo "" >&2  # New line after dots
    log_error "Server $server_id did not become ready in time (last status: '$status')"
    return 1
}

create_server() {
    local name="$1"
    local role="$2"

    # Check if server already exists
    local existing_server=$(get_server_by_name "$name")
    if [ -n "$existing_server" ]; then
        local server_id=$(echo "$existing_server" | jq -r '.id')
        local server_ip=$(echo "$existing_server" | jq -r '.networks.v4[] | select(.type == "public") | .ip_address')
        log_info "Server $name already exists (ID: $server_id, IP: $server_ip)"
        echo "$server_id:$server_ip"
        return 0
    fi

    log_info "Creating server: $name"

    # Get Ubuntu 24.04 image
    local image_id=$(api_call GET "/images?per_page=200" | jq -r '.images[] | select(.slug == "ubuntu-24.04") | .id' | head -1)

    if [ -z "$image_id" ] || [ "$image_id" == "null" ]; then
        log_error "Failed to find Ubuntu 24.04 image"
        log_error "Available Ubuntu images:"
        api_call GET "/images?per_page=200" | jq -r '.images[] | select(.distribution == "Ubuntu") | "\(.slug) - \(.name)"'
        return 1
    fi

    # Get SSH key ID
    if [ -z "${SSH_KEY_ID:-}" ]; then
        log_error "SSH_KEY_ID not set. This should have been set by generate_and_upload_ssh_key"
        return 1
    fi

    # Generate random password to avoid email notifications
    local random_password=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)

    local data=$(cat <<EOF
{
  "name": "$name",
  "region": "$REGION",
  "size": "$SERVER_SIZE",
  "image": $image_id,
  "ssh_keys": [$SSH_KEY_ID],
  "password": "$random_password",
  "backups": false
}
EOF
)

    log_info "Creating with: region=$REGION, size=$SERVER_SIZE, image=$image_id, ssh_key=$SSH_KEY_ID"

    local response=$(api_call POST "/servers" "$data")
    local server_id=$(echo "$response" | jq -r '.server.id')

    if [ -z "$server_id" ] || [ "$server_id" == "null" ]; then
        log_error "Failed to create server: $name"
        log_error "API Response:"
        echo "$response" | jq '.'
        return 1
    fi

    wait_for_server_ready "$server_id"

    # Get IP address - wait for it to be assigned
    log_info "Waiting for IP address assignment..."
    local server_ip=""
    local ip_attempts=0
    while [ -z "$server_ip" ] && [ $ip_attempts -lt 30 ]; do
        server_ip=$(api_call GET "/servers/$server_id" | jq -r '.server.networks.v4[]? | select(.type == "public")? | .ip_address?' | head -1)
        if [ -z "$server_ip" ] || [ "$server_ip" == "null" ]; then
            echo -n "." >&2
            sleep 2
            ip_attempts=$((ip_attempts + 1))
            server_ip=""
        fi
    done

    if [ -z "$server_ip" ]; then
        log_error "Failed to get IP address for server $name (ID: $server_id)"
        return 1
    fi

    log_success "Created server: $name (ID: $server_id, IP: $server_ip)"
    echo "$server_id:$server_ip"
}

get_or_create_servers() {
    log_info "Setting up cluster servers..."

    # Create all servers in parallel (control plane + workers)
    log_info "Creating control plane and $WORKER_COUNT worker nodes in parallel..."

    WORKER_IPS=()
    WORKER_IDS=()

    declare -a server_pids
    declare -a server_temp_files

    # Create control plane in background
    local control_temp_file="/tmp/control-${CLUSTER_NAME}-$$"
    server_temp_files+=( "$control_temp_file" )
    (
        result=$(create_server "${CLUSTER_NAME}-control-1" "control")
        echo "$result" > "$control_temp_file"
    ) &
    server_pids+=( $! )

    # Create workers in background
    for i in $(seq 1 $WORKER_COUNT); do
        local temp_file="/tmp/worker-${CLUSTER_NAME}-$i-$$"
        server_temp_files+=( "$temp_file" )
        (
            result=$(create_server "${CLUSTER_NAME}-worker-$i" "worker")
            echo "$result" > "$temp_file"
        ) &
        server_pids+=( $! )
    done

    # Wait for all server creations to complete
    for pid in "${server_pids[@]}"; do
        wait $pid
    done

    # Collect control plane result
    local control_result=$(cat "$control_temp_file" 2>/dev/null || echo "")
    rm -f "$control_temp_file"

    if [ -z "$control_result" ]; then
        log_error "Failed to create or retrieve control plane server"
        return 1
    fi
    CONTROL_PLANE_ID=$(echo "$control_result" | cut -d: -f1)
    CONTROL_PLANE_IP=$(echo "$control_result" | cut -d: -f2)

    if [ -z "$CONTROL_PLANE_ID" ] || [ -z "$CONTROL_PLANE_IP" ]; then
        log_error "Invalid control plane server info: $control_result"
        return 1
    fi

    log_info "Control plane: ID=$CONTROL_PLANE_ID, IP=$CONTROL_PLANE_IP"

    # Collect worker results
    for i in $(seq 1 $WORKER_COUNT); do
        local temp_file="/tmp/worker-${CLUSTER_NAME}-$i-$$"
        local worker_result=$(cat "$temp_file" 2>/dev/null || echo "")
        rm -f "$temp_file"

        if [ -z "$worker_result" ]; then
            log_error "Failed to create or retrieve worker node $i"
            return 1
        fi
        local worker_id=$(echo "$worker_result" | cut -d: -f1)
        local worker_ip=$(echo "$worker_result" | cut -d: -f2)

        if [ -z "$worker_id" ] || [ -z "$worker_ip" ]; then
            log_error "Invalid worker $i server info: $worker_result"
            return 1
        fi

        WORKER_IDS+=("$worker_id")
        WORKER_IPS+=("$worker_ip")
        log_info "Worker $i: ID=$worker_id, IP=$worker_ip"
    done

    log_success "All servers ready"
}

wait_for_ssh() {
    local ip="$1"
    local hostname="${2:-server}"
    local max_attempts=60
    local attempt=0

    log_info "Waiting for SSH on $hostname ($ip)..."

    while [ $attempt -lt $max_attempts ]; do
        if ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=3 -o BatchMode=yes root@$ip "echo 'SSH ready'" 2>/dev/null; then
            echo "" >&2  # New line after dots
            log_success "SSH ready on $hostname ($ip)"
            return 0
        fi

        # Show progress every 12 attempts
        if [ $((attempt % 12)) -eq 0 ] && [ $attempt -gt 0 ]; then
            log_info "Still waiting for SSH (attempt $attempt/$max_attempts)..." >&2
        fi

        echo -n "." >&2
        sleep 2
        attempt=$((attempt + 1))
    done

    echo "" >&2  # New line after dots
    log_error "SSH did not become ready on $hostname ($ip) after $((max_attempts * 2)) seconds"
    log_error "Please verify:"
    log_error "  1. SSH key is added to your BinaryLane account"
    log_error "  2. Server can be accessed at: ssh root@$ip"
    log_error "  3. Security groups allow SSH access"
    return 1
}

install_kubernetes_prerequisites() {
    local ip="$1"
    local hostname="${2:-server}"

    log_info "Installing Kubernetes prerequisites on $hostname ($ip)..."

    # Check if already installed
    if ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes root@$ip "which kubeadm" &>/dev/null; then
        log_info "Kubernetes already installed on $hostname"
        return 0
    fi

    ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$ip bash <<'EOF'
set -euo pipefail

# Update system

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.33/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.33/deb/ /" | tee /etc/apt/sources.list.d/kubernetes.list
KUBE_VERSION="1.33.3-1.1"

apt-get update

# Install required packages
apt-get install -y --no-install-recommends \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    containerd.io \
    kubelet=$KUBE_VERSION kubeadm=$KUBE_VERSION kubectl=$KUBE_VERSION

apt-mark hold kubelet kubeadm kubectl

# Configure containerd
mkdir -p /etc/containerd
containerd config default | tee /etc/containerd/config.toml > /dev/null
systemctl restart containerd

# Disable swap
swapoff -a
sed -i '/ swap / s/^/#/' /etc/fstab

# Load kernel modules
cat > /etc/modules-load.d/k8s.conf <<'EOL'
overlay
br_netfilter
EOL
modprobe overlay
modprobe br_netfilter

# Set kernel parameters
cat > /etc/sysctl.d/k8s.conf <<'EOL'
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward = 1
EOL
sysctl --system

# Enable kubelet service
systemctl enable kubelet

# Configure kubelet to use external cloud provider
mkdir -p /etc/systemd/system/kubelet.service.d
cat > /etc/systemd/system/kubelet.service.d/20-cloud-provider.conf <<'EOL'
[Service]
Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
EOL
systemctl daemon-reload

echo "Kubernetes installation complete"
EOF

    log_success "Kubernetes installed on $hostname"
}

initialize_control_plane() {
    log_info "Initializing Kubernetes control plane..."

    # Check if cluster is already initialized
    if ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP "test -f /etc/kubernetes/admin.conf" 2>/dev/null; then
        log_info "Control plane already initialized"
        return 0
    fi

    ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP bash <<EOF
set -euo pipefail

# Create kubeadm config with cloud provider settings
cat > /tmp/kubeadm-config.yaml <<'EOL'
apiVersion: kubeadm.k8s.io/v1beta4
kind: InitConfiguration
nodeRegistration:
  kubeletExtraArgs:
  - name: cloud-provider
    value: external
---
apiVersion: kubeadm.k8s.io/v1beta4
kind: ClusterConfiguration
networking:
  podSubnet: $POD_NETWORK_CIDR
controlPlaneEndpoint: "$CONTROL_PLANE_IP"
apiServer:
  certSANs:
  - $CONTROL_PLANE_IP
EOL

kubeadm init --config /tmp/kubeadm-config.yaml --ignore-preflight-errors=NumCPU,Mem

mkdir -p /root/.kube
cp /etc/kubernetes/admin.conf /root/.kube/config

# Apply Flannel CNI (no wait here, workers will wait when joining)
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
EOF

    log_success "Control plane initialized"
}

join_worker_nodes() {
    log_info "Joining worker nodes to cluster..."

    # Get join command
    local join_command=$(ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP \
        "kubeadm token create --print-join-command")

    # Join all worker nodes in parallel
    log_info "Joining ${#WORKER_IPS[@]} worker nodes in parallel..."
    declare -a join_pids

    for i in "${!WORKER_IPS[@]}"; do
        (
            local worker_ip="${WORKER_IPS[$i]}"
            local worker_name="${CLUSTER_NAME}-worker-$((i+1))"

            # Check if node is already joined
            local node_exists=$(ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP \
                "kubectl get nodes --no-headers | grep -c '$worker_name' || true")

            if [ "$node_exists" != "0" ]; then
                log_info "Worker $worker_name already joined"
                exit 0
            fi

            log_info "Joining worker: $worker_name ($worker_ip)"

            # Get CA cert hash and token from control plane
            local ca_hash=$(ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP \
                "openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outform der 2>/dev/null | openssl dgst -sha256 -hex | sed 's/^.* //'")
            local token=$(ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP \
                "kubeadm token create")

            ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$worker_ip bash <<EOF
set -euo pipefail

# Create kubeadm join config with cloud provider settings
cat > /tmp/kubeadm-join-config.yaml <<'EOL'
apiVersion: kubeadm.k8s.io/v1beta4
kind: JoinConfiguration
discovery:
  bootstrapToken:
    apiServerEndpoint: "$CONTROL_PLANE_IP:6443"
    token: "$token"
    caCertHashes:
    - "sha256:$ca_hash"
nodeRegistration:
  kubeletExtraArgs:
  - name: cloud-provider
    value: external
EOL

kubeadm join --config /tmp/kubeadm-join-config.yaml --ignore-preflight-errors=NumCPU,Mem
EOF

            log_success "Worker $worker_name joined"
        ) &
        join_pids+=( $! )
    done

    # Wait for all worker join operations to complete
    for pid in "${join_pids[@]}"; do
        wait $pid || { log_error "Failed to join worker node"; exit 1; }
    done

    # Wait for all nodes to be ready
    log_info "Waiting for all nodes to be ready..."
    ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP \
        "kubectl wait --for=condition=Ready node --all --timeout=120s"

    log_success "All worker nodes joined and ready"
}

deploy_cloud_controller_manager() {
    log_info "Deploying BinaryLane Cloud Controller Manager..."

    # Copy kubeconfig locally
    mkdir -p ~/.kube
    scp -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP:/etc/kubernetes/admin.conf ~/.kube/config-binarylane
    export KUBECONFIG=~/.kube/config-binarylane

    # Check if CCM is already deployed
    if kubectl get deployment -n kube-system binarylane-ccm-binarylane-cloud-controller-manager &>/dev/null; then
        log_info "CCM already deployed"
    else
        # Build CCM image (always rebuild locally, skip in CI where it's pre-built)
        if [ -z "${CI:-}" ]; then
            log_info "Building CCM Docker image..."
            docker build -t binarylane-cloud-controller-manager:local .
        else
            log_info "Running in CI - using pre-built Docker image"
        fi

        # Import image to control plane
        log_info "Importing CCM image to control plane..."
        docker save binarylane-cloud-controller-manager:local | \
            ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP "ctr -n k8s.io images import -"

        # Create secret with API token
        log_info "Creating secret with API token..."
        kubectl create secret generic binarylane-api-token \
            --from-literal=api-token=$BINARYLANE_API_TOKEN \
            -n kube-system

        # Deploy with Helm
        log_info "Installing CCM with Helm..."
        helm install binarylane-ccm charts/binarylane-cloud-controller-manager \
            --namespace kube-system \
            --set cloudControllerManager.secret.name=binarylane-api-token \
            --set image.repository=docker.io/library/binarylane-cloud-controller-manager \
            --set image.tag=local \
            --set image.pullPolicy=Never

        log_info "Waiting for CCM deployment to be created..."
        local wait_attempts=0
        while [ $wait_attempts -lt 30 ]; do
            if kubectl get deployment -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager &>/dev/null; then
                break
            fi
            sleep 1
            wait_attempts=$((wait_attempts + 1))
        done

        # Get pod name for debugging
        local pod_name=$(kubectl get pods -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

        if [ -n "$pod_name" ]; then
            log_info "CCM pod name: $pod_name"
        fi

        # Wait for CCM to be ready
        log_info "Waiting for CCM pod to be ready (timeout: 60s)..."
        if ! kubectl wait --for=condition=Ready pod -n kube-system \
            -l app.kubernetes.io/name=binarylane-cloud-controller-manager \
            --timeout=60s; then

            log_error "CCM pod did not become ready"

            # Detailed debugging information
            log_info "=== DEBUGGING INFORMATION ==="

            log_info "All pods in kube-system:"
            kubectl get pods -n kube-system -o wide || true

            if [ -n "$pod_name" ]; then
                log_info "Pod description:"
                kubectl describe pod -n kube-system "$pod_name" || true

                log_info "Pod logs:"
                kubectl logs -n kube-system "$pod_name" --tail=100 || true

                log_info "Previous pod logs (if restarted):"
                kubectl logs -n kube-system "$pod_name" --previous --tail=100 2>/dev/null || echo "No previous logs"
            fi

            log_info "Deployment status:"
            kubectl get deployment -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager -o yaml || true

            log_info "ReplicaSet status:"
            kubectl get replicaset -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager -o yaml || true

            log_info "Images in containerd:"
            ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP "ctr -n k8s.io images ls | grep binarylane" || true

            return 1
        fi

        log_success "CCM deployed"
    fi

    # CCM will automatically set provider IDs and node metadata
    log_success "CCM deployed and will initialize nodes automatically"
}

validate_cluster() {
    log_info "Validating cluster health..."

    export KUBECONFIG=~/.kube/config-binarylane

    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                    CLUSTER VALIDATION                          ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""

    echo "=== NODES ==="
    kubectl get nodes -o wide
    echo ""

    echo "=== PROVIDER IDs ==="
    kubectl get nodes -o json | jq -r '.items[] | "\(.metadata.name): \(.spec.providerID // "NOT SET")"'
    echo ""

    echo "=== NODE ADDRESSES ==="
    kubectl get nodes -o json | jq -r '.items[] | "\(.metadata.name): " + ([.status.addresses[]? | "\(.type)=\(.address)"] | join(", "))'
    echo ""

    echo "=== CCM STATUS ==="
    kubectl get pods -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager
    echo ""

    echo "=== SYSTEM PODS ==="
    kubectl get pods -n kube-system
    echo ""

    # Cluster info
    echo "=== CLUSTER INFO ==="
    echo "Kubeconfig: ~/.kube/config-binarylane"
    echo "Control Plane: $CONTROL_PLANE_IP"
    echo "Region: $REGION"
    echo ""

    # Validate all nodes are ready
    local total_nodes=$((CONTROL_PLANE_COUNT + WORKER_COUNT))
    local ready_nodes=$(kubectl get nodes --no-headers | grep -c " Ready " || true)

    if [ "$ready_nodes" -eq "$total_nodes" ]; then
        log_success "Cluster is healthy! All $total_nodes nodes are Ready"
    else
        log_warning "Expected $total_nodes nodes, but only $ready_nodes are Ready"
    fi

    # Check CCM
    local ccm_ready=$(kubectl get pods -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager --no-headers | grep -c "Running" || true)
    if [ "$ccm_ready" -gt 0 ]; then
        log_success "Cloud Controller Manager is running"
    else
        log_warning "Cloud Controller Manager is not running"
    fi
}

#=============================================================================
# Main Deployment Flow
#=============================================================================

main() {
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║   BinaryLane Kubernetes Cluster Deployment                     ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    echo "Cluster: $CLUSTER_NAME"
    echo "Region: $REGION"
    echo "Control Planes: $CONTROL_PLANE_COUNT"
    echo "Workers: $WORKER_COUNT"
    echo "Kubernetes Version: $K8S_VERSION"
    echo ""

    validate_environment
    generate_and_upload_ssh_key

    get_or_create_servers

    # Wait for SSH on all servers in parallel
    log_info "Waiting for SSH on all nodes in parallel..."
    declare -a ssh_pids

    wait_for_ssh $CONTROL_PLANE_IP "${CLUSTER_NAME}-control-1" &
    ssh_pids+=( $! )

    for i in "${!WORKER_IPS[@]}"; do
        wait_for_ssh "${WORKER_IPS[$i]}" "${CLUSTER_NAME}-worker-$((i+1))" &
        ssh_pids+=( $! )
    done

    # Wait for all SSH connections
    for pid in "${ssh_pids[@]}"; do
        wait $pid || { log_error "Failed to connect to a node via SSH"; exit 1; }
    done
    log_success "All nodes are accessible via SSH"

    # Install Kubernetes prerequisites on all nodes in parallel
    log_info "Installing Kubernetes prerequisites on all nodes in parallel..."
    install_kubernetes_prerequisites $CONTROL_PLANE_IP "${CLUSTER_NAME}-control-1" &
    local control_k8s_pid=$!

    declare -a k8s_pids
    for i in "${!WORKER_IPS[@]}"; do
        install_kubernetes_prerequisites "${WORKER_IPS[$i]}" "${CLUSTER_NAME}-worker-$((i+1))" &
        k8s_pids+=( $! )
    done

    # Wait for all Kubernetes installations to complete
    wait $control_k8s_pid || { log_error "Failed to install Kubernetes on control plane"; exit 1; }
    for pid in "${k8s_pids[@]}"; do
        wait $pid || { log_error "Failed to install Kubernetes on worker node"; exit 1; }
    done
    log_success "Kubernetes installed on all nodes"

    initialize_control_plane

    join_worker_nodes

    deploy_cloud_controller_manager

    validate_cluster

    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                    DEPLOYMENT COMPLETE                         ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    echo "To use the cluster:"
    echo "  export KUBECONFIG=~/.kube/config-binarylane"
    echo "  kubectl get nodes"
    echo ""
    echo "To delete the cluster:"
    echo "  ./scripts/delete-cluster.sh"
    echo ""
}

main "$@"
