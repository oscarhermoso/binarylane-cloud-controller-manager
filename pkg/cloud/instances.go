package cloud

import (
	"context"
	"fmt"
	"strconv"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/pkg/binarylane"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
)

type instancesV2 struct {
	client *binarylane.Client
	region string
}

// InstanceExists returns true if the instance for the given node exists according to the cloud provider.
// Use the node.name or node.spec.providerID field to find the node in the cloud provider.
func (i *instancesV2) InstanceExists(ctx context.Context, node *v1.Node) (bool, error) {
	server, err := i.getServerForNode(ctx, node)
	if err != nil {
		if err.Error() == fmt.Sprintf("server %q not found", node.Name) {
			return false, nil
		}
		return false, err
	}
	return server != nil, nil
}

// InstanceShutdown returns true if the instance is shutdown according to the cloud provider.
func (i *instancesV2) InstanceShutdown(ctx context.Context, node *v1.Node) (bool, error) {
	server, err := i.getServerForNode(ctx, node)
	if err != nil {
		return false, err
	}

	// Common status values: "new", "active", "off", "archive"
	return server.Status == "off" || server.Status == "archive", nil
}

// InstanceMetadata returns the instance's metadata.
func (i *instancesV2) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	server, err := i.getServerForNode(ctx, node)
	if err != nil {
		return nil, err
	}

	// Build the provider ID
	providerID := fmt.Sprintf("binarylane://%d", server.ID)

	// Extract addresses
	addresses := []v1.NodeAddress{
		{
			Type:    v1.NodeHostName,
			Address: server.Name,
		},
	}

	// Add IPv4 addresses
	for _, net := range server.Networks.V4 {
		if net.Type == "public" {
			addresses = append(addresses, v1.NodeAddress{
				Type:    v1.NodeExternalIP,
				Address: net.IPAddress,
			})
		} else {
			addresses = append(addresses, v1.NodeAddress{
				Type:    v1.NodeInternalIP,
				Address: net.IPAddress,
			})
		}
	}

	// Add IPv6 addresses
	for _, net := range server.Networks.V6 {
		if net.Type == "public" {
			addresses = append(addresses, v1.NodeAddress{
				Type:    v1.NodeExternalIP,
				Address: net.IPAddress,
			})
		}
	}

	return &cloudprovider.InstanceMetadata{
		ProviderID:    providerID,
		NodeAddresses: addresses,
		InstanceType:  "", // Could be populated if BinaryLane provides instance type info
		Zone:          server.Region.Slug,
		Region:        server.Region.Slug,
	}, nil
}

// getServerForNode retrieves the server for a given node
func (i *instancesV2) getServerForNode(ctx context.Context, node *v1.Node) (*binarylane.Server, error) {
	// Try to get server by provider ID first
	if node.Spec.ProviderID != "" {
		id, err := parseProviderID(node.Spec.ProviderID)
		if err == nil {
			return i.client.GetServer(ctx, id)
		}
	}

	// Fall back to name lookup
	return i.client.GetServerByName(ctx, node.Name)
}

// parseProviderID extracts the server ID from a provider ID
// Provider ID format: binarylane://123456
func parseProviderID(providerID string) (int64, error) {
	prefix := "binarylane://"
	if len(providerID) <= len(prefix) {
		return 0, fmt.Errorf("invalid provider ID format")
	}

	idStr := providerID[len(prefix):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid provider ID: %w", err)
	}

	return id, nil
}
