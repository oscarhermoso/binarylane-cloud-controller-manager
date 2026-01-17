package cloud

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/internal/binarylane"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
)

var _ cloudprovider.InstancesV2 = &instancesV2{}

type cloudClientInterface interface {
	GetServer(ctx context.Context, serverID int64) (*binarylane.Server, error)
	GetServerByName(ctx context.Context, name string) (*binarylane.Server, error)
	ListServers(ctx context.Context) ([]binarylane.Server, error)
	GetVpc(ctx context.Context, vpcID int64) (*binarylane.Vpc, error)
	UpdateVpc(ctx context.Context, vpcID int64, req binarylane.UpdateVpcRequest) (*binarylane.Vpc, error)
}

type instancesV2 struct {
	client cloudClientInterface
}

func (i *instancesV2) InstanceExists(ctx context.Context, node *v1.Node) (bool, error) {
	server, err := i.getServerForNode(ctx, node)
	if err != nil {
		if errors.Is(err, binarylane.ErrServerNotFound) {
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

	if server.VpcId != nil {
		for _, net := range server.Networks.V4 {
			if net.Type == "private" {
				addresses = append(addresses, v1.NodeAddress{
					Type:    v1.NodeInternalIP,
					Address: net.IpAddress,
				})
			}
		}
	}

	for _, net := range server.Networks.V4 {
		if net.Type == "public" {
			addresses = append(addresses, v1.NodeAddress{
				Type:    v1.NodeExternalIP,
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

	labels := make(map[string]string)

	if server.Host.DisplayName != "" {
		labels["binarylane.com/host"] = server.Host.DisplayName
	}

	return &cloudprovider.InstanceMetadata{
		ProviderID:       providerID,
		NodeAddresses:    addresses,
		InstanceType:     server.Size.Slug,
		Zone:             server.Region.Slug,
		Region:           server.Region.Slug,
		AdditionalLabels: labels,
	}, nil
}

func (i *instancesV2) getServerForNode(ctx context.Context, node *v1.Node) (*binarylane.Server, error) {
	if node.Spec.ProviderID != "" {
		id, err := parseProviderID(node.Spec.ProviderID)
		if err == nil {
			return i.client.GetServer(ctx, id)
		}
	}

	return i.client.GetServerByName(ctx, node.Name)
}

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
