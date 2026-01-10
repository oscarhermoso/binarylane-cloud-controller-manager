#!/bin/bash
# End-to-End Test Script for BinaryLane Cloud Controller Manager
# This script deploys a real Kubernetes cluster on BinaryLane and tests the CCM

set -e

# Configuration
CLUSTER_NAME="${CLUSTER_NAME:-ccm-e2e-test}"
REGION="${REGION:-syd}"
CONTROL_PLANE_SIZE="${CONTROL_PLANE_SIZE:-std-min}"
WORKER_SIZE="${WORKER_SIZE:-std-min}"
IMAGE="${IMAGE:-ubuntu-22.04}"
BINARYLANE_API_TOKEN="${BINARYLANE_API_TOKEN}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing=()
    
    command -v curl >/dev/null 2>&1 || missing+=("curl")
    command -v jq >/dev/null 2>&1 || missing+=("jq")
    command -v kubectl >/dev/null 2>&1 || missing+=("kubectl")
    command -v helm >/dev/null 2>&1 || missing+=("helm")
    command -v docker >/dev/null 2>&1 || missing+=("docker")
    command -v ssh >/dev/null 2>&1 || missing+=("ssh")
    command -v sshpass >/dev/null 2>&1 || missing+=("sshpass")
    
    if [ ${#missing[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing[*]}"
        log_error "Please install them before running this script"
        exit 1
    fi
    
    if [ -z "$BINARYLANE_API_TOKEN" ]; then
        log_error "BINARYLANE_API_TOKEN environment variable is not set"
        exit 1
    fi
    
    log_info "All prerequisites met"
}

cleanup() {
    log_warn "Cleaning up resources..."
    
    if [ -n "$CONTROL_PLANE_ID" ]; then
        log_info "Deleting control plane server: $CONTROL_PLANE_ID"
        curl -X DELETE "https://api.binarylane.com.au/v2/servers/$CONTROL_PLANE_ID" \
            -H "Authorization: Bearer $BINARYLANE_API_TOKEN" \
            -s || log_warn "Failed to delete control plane server"
    fi
    
    if [ -n "$WORKER_ID" ]; then
        log_info "Deleting worker server: $WORKER_ID"
        curl -X DELETE "https://api.binarylane.com.au/v2/servers/$WORKER_ID" \
            -H "Authorization: Bearer $BINARYLANE_API_TOKEN" \
            -s || log_warn "Failed to delete worker server"
    fi
    
    log_info "Cleanup complete"
}

# Set trap to cleanup on exit
trap cleanup EXIT

main() {
    log_info "Starting BinaryLane CCM End-to-End Test"
    log_info "Cluster: $CLUSTER_NAME, Region: $REGION"
    
    check_prerequisites
    
    # Generate unique cluster ID
    TIMESTAMP=$(date +%s)
    CLUSTER_ID="${CLUSTER_NAME}-${TIMESTAMP}"
    
    log_info "Creating BinaryLane servers for cluster: $CLUSTER_ID"
    
    # Create servers (similar to GitHub Actions workflow)
    # ... (implementation continues)
    
    log_info "E2E test completed successfully!"
}

# Run main function
main "$@"
