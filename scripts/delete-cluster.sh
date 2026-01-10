#!/bin/bash
set -e

if [ -f "$(dirname "$0")/../.env" ]; then
    source "$(dirname "$0")/../.env"
fi

CLUSTER_NAME="${CLUSTER_NAME:-binarylane-ccm}"


RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

if [ -z "$BINARYLANE_API_TOKEN" ]; then
    log_error "BINARYLANE_API_TOKEN environment variable is not set"
    exit 1
fi

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║  Delete BinaryLane Kubernetes Cluster                         ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

log_info "Finding servers for cluster: $CLUSTER_NAME"

SERVERS=$(curl -s -X GET "https://api.binarylane.com.au/v2/servers" \
    -H "Authorization: Bearer $BINARYLANE_API_TOKEN")

MATCHING_SERVERS=$(echo "$SERVERS" | jq -r --arg name "$CLUSTER_NAME" \
    '.servers[] | select(.name | startswith($name)) | "\(.id) \(.name) \(.networks.v4[0].ip_address)"')

if [ -z "$MATCHING_SERVERS" ]; then
    log_warn "No servers found matching: $CLUSTER_NAME"
    exit 0
fi

echo "$MATCHING_SERVERS" | while read id name ip; do
    echo "  - $name (ID: $id, IP: $ip)"
done

echo ""
read -p "Delete these servers? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    log_warn "Deletion cancelled"
    exit 0
fi

echo "$MATCHING_SERVERS" | while read id name ip; do
    log_info "Deleting: $name (ID: $id)"

    HTTP_CODE=$(curl -s -w "%{http_code}" -o /tmp/delete_response_$id.json -X DELETE "https://api.binarylane.com.au/v2/servers/$id" \
        -H "Authorization: Bearer $BINARYLANE_API_TOKEN")

    DELETE_RESPONSE=$(cat /tmp/delete_response_$id.json)
    rm -f /tmp/delete_response_$id.json

    if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
        log_info "✓ Deleted: $name"
    elif [ "$HTTP_CODE" = "404" ]; then
        log_warn "Server $name (ID: $id) not found (already deleted)"
    else
        log_error "Failed to delete $name (HTTP $HTTP_CODE): $(echo "$DELETE_RESPONSE" | jq -r '.message // .error // "Unknown error"')"
    fi
done

log_info "Cluster deletion complete"

# Clean up local kubeconfig
if [ -f ~/.kube/config-binarylane ]; then
    log_info "Removing local kubeconfig"
    rm ~/.kube/config-binarylane
fi
