#!/bin/bash
set -e

CLUSTER_NAME="${CLUSTER_NAME:-ccm-e2e-test}"
REGION="${REGION:-syd}"
SERVER_SIZE="${SERVER_SIZE:-std-min}"
WORKER_COUNT="${WORKER_COUNT:-2}"
BINARYLANE_API_TOKEN="${BINARYLANE_API_TOKEN}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TEST_RESULTS=()

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

test_passed() {
    local test_name="$1"
    TEST_RESULTS+=("✓ $test_name")
    log_success "$test_name"
}

test_failed() {
    local test_name="$1"
    local error="$2"
    TEST_RESULTS+=("✗ $test_name: $error")
    log_error "$test_name failed: $error"
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    local missing=()

    command -v kubectl >/dev/null 2>&1 || missing+=("kubectl")
    command -v jq >/dev/null 2>&1 || missing+=("jq")

    if [ ${#missing[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing[*]}"
        exit 1
    fi

    if [ -z "$BINARYLANE_API_TOKEN" ]; then
        log_error "BINARYLANE_API_TOKEN environment variable is not set"
        exit 1
    fi

    if [ ! -f "$(dirname "$0")/deploy-cluster.sh" ]; then
        log_error "deploy-cluster.sh not found"
        exit 1
    fi

    log_success "All prerequisites met"
}

cleanup() {
    log_info "Cleaning up test cluster..."

    if [ -f "$(dirname "$0")/delete-cluster.sh" ]; then
        export CLUSTER_NAME="$CLUSTER_NAME"
        echo "yes" | "$(dirname "$0")/delete-cluster.sh" || log_warn "Cleanup may be incomplete"
    fi
}

run_tests() {
    log_info "Running E2E tests..."

    export KUBECONFIG=~/.kube/config-binarylane

    if [ ! -f "$KUBECONFIG" ]; then
        test_failed "Kubeconfig Check" "kubeconfig not found at $KUBECONFIG"
        return 1
    fi

    log_info "Test 1: Verify all nodes are Ready"
    local total_nodes=$((1 + WORKER_COUNT))
    local ready_nodes=$(kubectl get nodes --no-headers 2>/dev/null | grep -c " Ready " || true)
    if [ "$ready_nodes" -eq "$total_nodes" ]; then
        test_passed "All $total_nodes nodes are Ready"
    else
        test_failed "Node Readiness" "Expected $total_nodes ready nodes, got $ready_nodes"
    fi

    log_info "Test 2: Verify provider IDs are set"
    local nodes_with_provider_id=$(kubectl get nodes -o json 2>/dev/null | jq -r '.items[] | select(.spec.providerID != null and .spec.providerID != "") | .metadata.name' | wc -l)
    if [ "$nodes_with_provider_id" -eq "$total_nodes" ]; then
        test_passed "All nodes have provider IDs set"
    else
        test_failed "Provider IDs" "Only $nodes_with_provider_id/$total_nodes nodes have provider IDs"
    fi

    log_info "Test 3: Verify external IPs are assigned"
    local nodes_with_external_ip=$(kubectl get nodes -o json 2>/dev/null | jq -r '.items[] | select(.status.addresses[] | select(.type == "ExternalIP")) | .metadata.name' | wc -l)
    if [ "$nodes_with_external_ip" -eq "$total_nodes" ]; then
        test_passed "All nodes have external IPs assigned"
    else
        test_failed "External IPs" "Only $nodes_with_external_ip/$total_nodes nodes have external IPs"
    fi

    log_info "Test 4: Verify CCM pod is running"
    local ccm_running=$(kubectl get pods -n kube-system -l app.kubernetes.io/name=binarylane-cloud-controller-manager --no-headers 2>/dev/null | grep -c "Running" || true)
    if [ "$ccm_running" -ge 1 ]; then
        test_passed "CCM pod is running"
    else
        test_failed "CCM Status" "CCM pod is not running"
    fi

    log_info "Test 5: Verify node addresses are populated"
    local nodes_with_addresses=$(kubectl get nodes -o json 2>/dev/null | jq -r '.items[] | select(.status.addresses | length > 0) | .metadata.name' | wc -l)
    if [ "$nodes_with_addresses" -eq "$total_nodes" ]; then
        test_passed "All nodes have addresses populated"
    else
        test_failed "Node Addresses" "Only $nodes_with_addresses/$total_nodes nodes have addresses"
    fi

    log_info "Test 6: Verify zone information"
    local nodes_with_zone=$(kubectl get nodes -o json 2>/dev/null | jq -r '.items[] | select(.metadata.labels["topology.kubernetes.io/region"]) | .metadata.name' | wc -l)
    if [ "$nodes_with_zone" -eq "$total_nodes" ]; then
        test_passed "All nodes have zone/region labels"
    else
        test_failed "Zone Labels" "Only $nodes_with_zone/$total_nodes nodes have zone labels"
    fi

    log_info "Test 7: Verify VPC internal IPs match vpc_ipv4_address"
    local vpc_mismatch=0
    local nodes=$(kubectl get nodes -o json 2>/dev/null | jq -r '.items[] | .metadata.name')

    for node_name in $nodes; do
        # Get Kubernetes internal IP
        local k8s_internal_ip=$(kubectl get node "$node_name" -o json 2>/dev/null | jq -r '.status.addresses[] | select(.type == "InternalIP") | .address')

        if [ -z "$k8s_internal_ip" ]; then
            log_warn "Node $node_name has no internal IP in Kubernetes"
            ((vpc_mismatch++))
            continue
        fi

        # Get server info from BinaryLane API
        local server_info=$(curl -s -X GET "https://api.binarylane.com.au/v2/servers?per_page=200" \
            -H "Authorization: Bearer $BINARYLANE_API_TOKEN" 2>/dev/null | \
            jq -r ".servers[] | select(.name == \"$node_name\")")

        if [ -z "$server_info" ]; then
            log_warn "Node $node_name not found in BinaryLane API"
            ((vpc_mismatch++))
            continue
        fi

        # Check if server is in a VPC
        local vpc_id=$(echo "$server_info" | jq -r '.vpc_id')

        if [ -z "$vpc_id" ] || [ "$vpc_id" == "null" ]; then
            log_info "Node $node_name is not in a VPC (expected for non-VPC deployments)"
        else
            # Get VPC private IP from server networks
            local vpc_private_ip=$(echo "$server_info" | jq -r '.networks.v4[] | select(.type == "private") | .ip_address')

            if [ -z "$vpc_private_ip" ] || [ "$vpc_private_ip" == "null" ]; then
                log_error "Node $node_name is in VPC but has no private IP"
                ((vpc_mismatch++))
            elif [ "$k8s_internal_ip" != "$vpc_private_ip" ]; then
                log_error "Node $node_name: K8s internal IP ($k8s_internal_ip) != VPC private IP ($vpc_private_ip)"
                ((vpc_mismatch++))
            else
                log_info "✓ Node $node_name: VPC internal IP matches ($vpc_private_ip)"
            fi
        fi
    done

    if [ "$vpc_mismatch" -eq 0 ]; then
        test_passed "All VPC internal IPs match"
    else
        test_failed "VPC Internal IPs" "$vpc_mismatch mismatches found"
    fi

}

print_results() {
    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                    E2E TEST RESULTS                            ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""

    local passed=0
    local failed=0

    for result in "${TEST_RESULTS[@]}"; do
        if [[ "$result" == ✓* ]]; then
            echo -e "${GREEN}$result${NC}"
            ((passed++))
        else
            echo -e "${RED}$result${NC}"
            ((failed++))
        fi
    done

    echo ""
    echo "Total: $((passed + failed)) tests, $passed passed, $failed failed"
    echo ""

    if [ "$failed" -gt 0 ]; then
        return 1
    fi
    return 0
}

main() {
    log_info "╔════════════════════════════════════════════════════════════════╗"
    log_info "║   BinaryLane CCM End-to-End Test Suite                        ║"
    log_info "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    log_info "Cluster: $CLUSTER_NAME"
    log_info "Region: $REGION"
    log_info "Workers: $WORKER_COUNT"
    echo ""

    check_prerequisites

    log_info "Deploying test cluster..."
    export CLUSTER_NAME="$CLUSTER_NAME"
    export REGION="$REGION"
    export SERVER_SIZE="$SERVER_SIZE"
    export WORKER_COUNT="$WORKER_COUNT"

    if ! "$(dirname "$0")/deploy-cluster.sh"; then
        log_error "Cluster deployment failed"
        exit 1
    fi

    log_success "Cluster deployed successfully"
    echo ""

    log_info "Running E2E tests..."
    export KUBECONFIG=~/.kube/config-binarylane

    # Give CCM a moment to complete initialization if needed
    sleep 5

    run_tests

    if print_results; then
        log_success "All E2E tests passed!"
        cleanup
        exit 0
    else
        log_error "Some E2E tests failed"
        log_warn "Cluster left running for debugging. Clean up with: ./scripts/delete-cluster.sh"
        exit 1
    fi
}

main "$@"
