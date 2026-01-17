package cloud

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/internal/binarylane"
	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
)

var _ cloudprovider.Routes = &routes{}

type routes struct {
	client cloudClientInterface
	cidr   string
}

func (r *routes) ListRoutes(ctx context.Context, clusterName string) ([]*cloudprovider.Route, error) {
	servers, err := r.client.ListServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	ipToName := make(map[string]string)
	clusterVpcs := make(map[int64]bool)
	for _, server := range servers {
		if !isClusterServer(server.Name, clusterName) {
			continue
		}
		if server.VpcId != nil {
			clusterVpcs[*server.VpcId] = true
			for _, net := range server.Networks.V4 {
				if net.Type == "private" {
					ipToName[net.IpAddress] = server.Name
					break
				}
			}
		}
	}

	if len(clusterVpcs) == 0 {
		return []*cloudprovider.Route{}, nil
	}

	vpcRoutes := make(map[int64][]*cloudprovider.Route)

	for vpcID := range clusterVpcs {
		vpc, err := r.client.GetVpc(ctx, vpcID)
		if err != nil {
			if errors.Is(err, binarylane.ErrVpcNotFound) {
				continue
			}
			return nil, fmt.Errorf("failed to get VPC %d: %w", vpcID, err)
		}

		for _, routeEntry := range vpc.RouteEntries {
			nodeName := ipToName[routeEntry.Router]
			if nodeName == "" {
				nodeName = routeEntry.Router
			}
			vpcRoutes[vpcID] = append(vpcRoutes[vpcID], &cloudprovider.Route{
				Name:            fmt.Sprintf("%s-%s", nodeName, routeEntry.Destination),
				TargetNode:      types.NodeName(nodeName),
				DestinationCIDR: routeEntry.Destination,
			})
		}
	}

	var allRoutes []*cloudprovider.Route
	for _, routes := range vpcRoutes {
		allRoutes = append(allRoutes, routes...)
	}

	return allRoutes, nil
}

func (r *routes) CreateRoute(ctx context.Context, clusterName string, nameHint string, route *cloudprovider.Route) error {
	targetNode := string(route.TargetNode)
	server, err := r.client.GetServerByName(ctx, targetNode)
	if err != nil {
		if errors.Is(err, binarylane.ErrServerNotFound) {
			return fmt.Errorf("target node %s not found", targetNode)
		}
		return fmt.Errorf("failed to get server: %w", err)
	}

	if server.VpcId == nil {
		return fmt.Errorf("server %s is not in a VPC", targetNode)
	}

	var privateIP string
	for _, net := range server.Networks.V4 {
		if net.Type == "private" {
			privateIP = net.IpAddress
			break
		}
	}
	if privateIP == "" {
		return fmt.Errorf("server %s has no private IP", targetNode)
	}

	vpc, err := r.client.GetVpc(ctx, *server.VpcId)
	if err != nil {
		return fmt.Errorf("failed to get VPC: %w", err)
	}

	for _, existingRoute := range vpc.RouteEntries {
		if existingRoute.Destination == route.DestinationCIDR && existingRoute.Router == privateIP {
			return nil
		}
	}

	newRouteEntries := make([]binarylane.RouteEntryRequest, len(vpc.RouteEntries))
	for i, re := range vpc.RouteEntries {
		newRouteEntries[i] = binarylane.RouteEntryRequest(re)
	}

	newRouteEntries = append(newRouteEntries, binarylane.RouteEntryRequest{
		Router:      privateIP,
		Destination: route.DestinationCIDR,
		Description: func() *string { s := fmt.Sprintf("Kubernetes route for node %s", targetNode); return &s }(),
	})

	_, err = r.client.UpdateVpc(ctx, *server.VpcId, binarylane.UpdateVpcRequest{
		Name:         vpc.Name,
		RouteEntries: &newRouteEntries,
	})
	if err != nil {
		return fmt.Errorf("failed to update VPC routes: %w", err)
	}

	return nil
}

func (r *routes) DeleteRoute(ctx context.Context, clusterName string, route *cloudprovider.Route) error {
	targetNode := string(route.TargetNode)
	server, err := r.client.GetServerByName(ctx, targetNode)
	if err != nil {
		if errors.Is(err, binarylane.ErrServerNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get server: %w", err)
	}

	if server.VpcId == nil {
		return nil
	}

	var privateIP string
	for _, net := range server.Networks.V4 {
		if net.Type == "private" {
			privateIP = net.IpAddress
			break
		}
	}
	if privateIP == "" {
		return nil
	}

	vpc, err := r.client.GetVpc(ctx, *server.VpcId)
	if err != nil {
		if errors.Is(err, binarylane.ErrVpcNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get VPC: %w", err)
	}

	var newRouteEntries []binarylane.RouteEntryRequest
	found := false
	for _, re := range vpc.RouteEntries {
		if re.Destination == route.DestinationCIDR && re.Router == privateIP {
			found = true
			continue
		}
		newRouteEntries = append(newRouteEntries, binarylane.RouteEntryRequest(re))
	}

	if !found {
		return nil
	}

	_, err = r.client.UpdateVpc(ctx, *server.VpcId, binarylane.UpdateVpcRequest{
		Name:         vpc.Name,
		RouteEntries: &newRouteEntries,
	})
	if err != nil {
		return fmt.Errorf("failed to update VPC routes: %w", err)
	}

	return nil
}

func isClusterServer(serverName, clusterName string) bool {
	if clusterName == "" {
		return true
	}
	return strings.HasPrefix(serverName, clusterName)
}
