#!/bin/bash
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-binarylane-ccm}"
REGION="${REGION:-per}"
SERVER_SIZE="${SERVER_SIZE:-std-2vcpu}"
CONTROL_PLANE_COUNT="${CONTROL_PLANE_COUNT:-1}"
WORKER_COUNT="${WORKER_COUNT:-2}"
K8S_VERSION="${K8S_VERSION:-1.29.15}"
POD_NETWORK_CIDR="${POD_NETWORK_CIDR:-10.244.0.0/16}"
SSH_KEY_PATH="${SSH_KEY_PATH:-.ssh/binarylane-k8s}"
SSH_KEY_NAME="binarylane-k8s-cluster"

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
        curl -s -X "$method" "$url" \
            -H "Authorization: Bearer $BINARYLANE_API_TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data"
    else
        curl -s -X "$method" "$url" \
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
        local status=$(api_call GET "/servers/$server_id" | jq -r '.server.status')

        if [ "$status" == "active" ]; then
            log_success "Server $server_id is ready"
            return 0
        fi

        echo -n "." >&2
        sleep 5
        attempt=$((attempt + 1))
    done

    log_error "Server $server_id did not become ready in time"
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

    local data=$(cat <<EOF
{
  "name": "$name",
  "region": "$REGION",
  "size": "$SERVER_SIZE",
  "image": $image_id,
  "ssh_keys": [$SSH_KEY_ID],
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

    # Create control plane
    local control_result=$(create_server "${CLUSTER_NAME}-control-1" "control")
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

    # Create workers
    WORKER_IPS=()
    WORKER_IDS=()

    for i in $(seq 1 $WORKER_COUNT); do
        local worker_result=$(create_server "${CLUSTER_NAME}-worker-$i" "worker")
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
        if ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes root@$ip "echo 'SSH ready'"; then
            log_success "SSH ready on $hostname ($ip)"
            return 0
        fi

        echo -n "." >&2
        sleep 5
        attempt=$((attempt + 1))
    done

    log_error "SSH did not become ready on $hostname ($ip) after $((max_attempts * 5)) seconds"
    log_error "Please verify:"
    log_error "  1. SSH key is added to your BinaryLane account"
    log_error "  2. Server can be accessed at: ssh root@$ip"
    log_error "  3. Security groups allow SSH access"
    return 1
}

install_kubernetes_prerequisites() {
    local ip="$1"
    local hostname="$2"

    log_info "Installing Kubernetes prerequisites on $hostname ($ip)..."

    ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$ip bash <<'EOF'
set -euo pipefail

# Set hostname
hostname $(cat /etc/hostname)

# Disable swap
swapoff -a
sed -i '/ swap / s/^/#/' /etc/fstab

# Load kernel modules
cat <<MODULES > /etc/modules-load.d/k8s.conf
overlay
br_netfilter
MODULES

modprobe overlay
modprobe br_netfilter

# Set sysctl parameters
cat <<SYSCTL > /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
SYSCTL

sysctl --system

# Install containerd
apt-get update
apt-get install -y containerd
mkdir -p /etc/containerd
containerd config default > /etc/containerd/config.toml
sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
systemctl restart containerd
systemctl enable containerd

# Install kubeadm, kubelet, kubectl
apt-get install -y apt-transport-https ca-certificates curl gpg
mkdir -p /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.29/deb/Release.key | gpg --batch --yes --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.29/deb/ /' > /etc/apt/sources.list.d/kubernetes.list
apt-get update
apt-get install -y kubelet=1.29.15-1.1 kubeadm=1.29.15-1.1 kubectl=1.29.15-1.1
apt-mark hold kubelet kubeadm kubectl

# Configure kubelet for external cloud provider
mkdir -p /etc/default
echo 'KUBELET_EXTRA_ARGS="--cloud-provider=external"' > /etc/default/kubelet
systemctl daemon-reload
systemctl restart kubelet
EOF

    log_success "Kubernetes prerequisites installed on $hostname"
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

kubeadm init \
    --pod-network-cidr=$POD_NETWORK_CIDR \
    --apiserver-cert-extra-sans=$CONTROL_PLANE_IP \
    --control-plane-endpoint=$CONTROL_PLANE_IP \
    --ignore-preflight-errors=NumCPU

mkdir -p /root/.kube
cp /etc/kubernetes/admin.conf /root/.kube/config

# Apply Flannel CNI
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# Wait for control plane to be ready
kubectl wait --for=condition=Ready node --all --timeout=300s
EOF

    log_success "Control plane initialized"
}

join_worker_nodes() {
    log_info "Joining worker nodes to cluster..."

    # Get join command
    local join_command=$(ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP \
        "kubeadm token create --print-join-command")

    for i in "${!WORKER_IPS[@]}"; do
        local worker_ip="${WORKER_IPS[$i]}"
        local worker_name="${CLUSTER_NAME}-worker-$((i+1))"

        # Check if node is already joined
        local node_exists=$(ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP \
            "kubectl get nodes --no-headers | grep -c '$worker_name' || true")

        if [ "$node_exists" != "0" ]; then
            log_info "Worker $worker_name already joined"
            continue
        fi

        log_info "Joining worker: $worker_name ($worker_ip)"

        ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$worker_ip bash <<EOF
set -euo pipefail
$join_command
EOF

        log_success "Worker $worker_name joined"
    done

    # Wait for all nodes to be ready
    ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP \
        "kubectl wait --for=condition=Ready node --all --timeout=300s"

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
        # Build CCM image
        log_info "Building CCM Docker image..."
        docker build -t binarylane-cloud-controller-manager:local .

        # Import image to control plane
        log_info "Importing CCM image to control plane..."
        docker save binarylane-cloud-controller-manager:local | \
            ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@$CONTROL_PLANE_IP "ctr -n k8s.io images import -"

        # Deploy with Helm
        log_info "Installing CCM with Helm..."
        helm install binarylane-ccm charts/binarylane-cloud-controller-manager \
            --namespace kube-system \
            --set cloudControllerManager.apiToken=$BINARYLANE_API_TOKEN \
            --set cloudControllerManager.region=$REGION \
            --set image.repository=docker.io/library/binarylane-cloud-controller-manager \
            --set image.tag=local \
            --set image.pullPolicy=Never

        # Wait for CCM to be ready
        kubectl wait --for=condition=Ready pod -n kube-system \
            -l app.kubernetes.io/name=binarylane-cloud-controller-manager \
            --timeout=120s

        log_success "CCM deployed"
    fi

    # Set provider IDs for existing nodes
    log_info "Setting provider IDs for nodes..."

    kubectl patch node ${CLUSTER_NAME}-control-1 -p "{\"spec\":{\"providerID\":\"binarylane://$CONTROL_PLANE_ID\"}}" 2>/dev/null || true

    for i in "${!WORKER_IDS[@]}"; do
        local worker_id="${WORKER_IDS[$i]}"
        local worker_name="${CLUSTER_NAME}-worker-$((i+1))"
        kubectl patch node $worker_name -p "{\"spec\":{\"providerID\":\"binarylane://$worker_id\"}}" 2>/dev/null || true
    done

    log_success "Provider IDs set"
}

validate_cluster() {
    log_info "Validating cluster health..."

    export KUBECONFIG=~/.kube/config-binarylane

    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                    CLUSTER VALIDATION                          ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""

    # Node status
    echo "=== NODES ==="
    kubectl get nodes -o wide
    echo ""

    # Provider IDs
    echo "=== PROVIDER IDs ==="
    kubectl get nodes -o json | jq -r '.items[] | "\(.metadata.name): \(.spec.providerID // "NOT SET")"'
    echo ""

    # Node addresses
    echo "=== NODE ADDRESSES ==="
    kubectl get nodes -o json | jq -r '.items[] | "\(.metadata.name): " + ([.status.addresses[]? | "\(.type)=\(.address)"] | join(", "))'
    echo ""

    # CCM status
    echo "=== CCM STATUS ==="
    kubectl get pods -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager
    echo ""

    # System pods
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

    # Wait for SSH on all servers
    wait_for_ssh $CONTROL_PLANE_IP "${CLUSTER_NAME}-control-1"
    for i in "${!WORKER_IPS[@]}"; do
        wait_for_ssh "${WORKER_IPS[$i]}" "${CLUSTER_NAME}-worker-$((i+1))"
    done

    # Install prerequisites on all nodes
    install_kubernetes_prerequisites $CONTROL_PLANE_IP "${CLUSTER_NAME}-control-1"
    for i in "${!WORKER_IPS[@]}"; do
        install_kubernetes_prerequisites "${WORKER_IPS[$i]}" "${CLUSTER_NAME}-worker-$((i+1))"
    done

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
