package cloud

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/internal/binarylane"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
)

type instancesV2 struct {
	client *binarylane.BinaryLaneClient
	region string
}

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

func (i *instancesV2) InstanceShutdown(ctx context.Context, node *v1.Node) (bool, error) {
	server, err := i.getServerForNode(ctx, node)
	if err != nil {
		return false, err
	}

	return server.Status == "off" || server.Status == "archive", nil
}

func (i *instancesV2) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	server, err := i.getServerForNode(ctx, node)
	if err != nil {
		return nil, err
	}

	providerID := fmt.Sprintf("binarylane://%d", server.Id)

	addresses := []v1.NodeAddress{
		{
			Type:    v1.NodeHostName,
			Address: server.Name,
		},
	}

	for _, net := range server.Networks.V4 {
		if net.Type == "public" {
			addresses = append(addresses, v1.NodeAddress{
				Type:    v1.NodeExternalIP,
				Address: net.IpAddress,
			})
		} else {
			addresses = append(addresses, v1.NodeAddress{
				Type:    v1.NodeInternalIP,
				Address: net.IpAddress,
			})
		}
	}

	for _, net := range server.Networks.V6 {
		if net.Type == "public" {
			addresses = append(addresses, v1.NodeAddress{
				Type:    v1.NodeExternalIP,
				Address: net.IpAddress,
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

func (i *instancesV2) getServerForNode(ctx context.Context, node *v1.Node) (*binarylane.Server, error) {
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
	if !strings.HasPrefix(providerID, prefix) {
		return 0, fmt.Errorf("invalid provider ID format: %s (expected prefix: %s)", providerID, prefix)
	}

	idStr := providerID[len(prefix):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid provider ID: %w", err)
	}

	return id, nil
}
