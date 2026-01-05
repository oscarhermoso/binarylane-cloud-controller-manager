package cloud

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/pkg/binarylane"
	v1 "k8s.io/api/core/v1"
)

const (
	// annBinaryLaneLoadBalancerID is the annotation used to store the load balancer ID
	annBinaryLaneLoadBalancerID = "service.beta.kubernetes.io/binarylane-loadbalancer-id"
	
	// annBinaryLaneProtocol is the annotation for specifying the protocol
	annBinaryLaneProtocol = "service.beta.kubernetes.io/binarylane-loadbalancer-protocol"
	
	// annBinaryLaneHealthCheckPath is the annotation for health check path
	annBinaryLaneHealthCheckPath = "service.beta.kubernetes.io/binarylane-loadbalancer-healthcheck-path"
	
	// annBinaryLaneHealthCheckProtocol is the annotation for health check protocol
	annBinaryLaneHealthCheckProtocol = "service.beta.kubernetes.io/binarylane-loadbalancer-healthcheck-protocol"
)

type loadBalancers struct {
	client *binarylane.Client
	region string
}

// GetLoadBalancer returns whether the specified load balancer exists, and if so, what its status is.
func (l *loadBalancers) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (*v1.LoadBalancerStatus, bool, error) {
	lbID, err := l.loadBalancerID(service)
	if err != nil {
		return nil, false, err
	}

	if lbID == 0 {
		return nil, false, nil
	}

	lb, err := l.client.GetLoadBalancer(ctx, lbID)
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{
			{
				IP: lb.IP,
			},
		},
	}, true, nil
}

// GetLoadBalancerName returns the name of the load balancer.
func (l *loadBalancers) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	return l.getLoadBalancerName(clusterName, service)
}

// EnsureLoadBalancer creates a new load balancer or updates the existing one.
func (l *loadBalancers) EnsureLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	lbID, err := l.loadBalancerID(service)
	if err != nil {
		return nil, err
	}

	// Get server IDs for nodes
	serverIDs, err := l.getServerIDsForNodes(ctx, nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get server IDs for nodes: %w", err)
	}

	// Build forwarding rules from service ports
	forwardingRules := l.buildForwardingRules(service)

	// Build health check configuration
	healthCheck := l.buildHealthCheck(service)

	lbReq := &binarylane.LoadBalancerRequest{
		Name:            l.getLoadBalancerName(clusterName, service),
		Region:          l.region,
		ForwardingRules: forwardingRules,
		HealthCheck:     healthCheck,
		ServerIDs:       serverIDs,
	}

	var lb *binarylane.LoadBalancer
	if lbID == 0 {
		// Create new load balancer
		lb, err = l.client.CreateLoadBalancer(ctx, lbReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create load balancer: %w", err)
		}

		// The load balancer ID is stored in the service status.LoadBalancer.Ingress
		// by the Kubernetes controller manager, so we don't need to update annotations here
	} else {
		// Update existing load balancer
		lb, err = l.client.UpdateLoadBalancer(ctx, lbID, lbReq)
		if err != nil {
			return nil, fmt.Errorf("failed to update load balancer: %w", err)
		}
	}

	return &v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{
			{
				IP: lb.IP,
			},
		},
	}, nil
}

// UpdateLoadBalancer updates hosts under the specified load balancer.
func (l *loadBalancers) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {
	lbID, err := l.loadBalancerID(service)
	if err != nil {
		return err
	}

	if lbID == 0 {
		return fmt.Errorf("load balancer ID not found in service annotations")
	}

	// Get current load balancer
	lb, err := l.client.GetLoadBalancer(ctx, lbID)
	if err != nil {
		return fmt.Errorf("failed to get load balancer: %w", err)
	}

	// Get server IDs for nodes
	serverIDs, err := l.getServerIDsForNodes(ctx, nodes)
	if err != nil {
		return fmt.Errorf("failed to get server IDs for nodes: %w", err)
	}

	// Determine which servers to add and remove
	toAdd, toRemove := diffServerIDs(lb.ServerIDs, serverIDs)

	// Add servers
	if len(toAdd) > 0 {
		if err := l.client.AddServersToLoadBalancer(ctx, lbID, toAdd); err != nil {
			return fmt.Errorf("failed to add servers to load balancer: %w", err)
		}
	}

	// Remove servers
	if len(toRemove) > 0 {
		if err := l.client.RemoveServersFromLoadBalancer(ctx, lbID, toRemove); err != nil {
			return fmt.Errorf("failed to remove servers from load balancer: %w", err)
		}
	}

	return nil
}

// EnsureLoadBalancerDeleted deletes the specified load balancer if it exists.
func (l *loadBalancers) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	lbID, err := l.loadBalancerID(service)
	if err != nil {
		return err
	}

	if lbID == 0 {
		return nil
	}

	if err := l.client.DeleteLoadBalancer(ctx, lbID); err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return nil
		}
		return fmt.Errorf("failed to delete load balancer: %w", err)
	}

	return nil
}

// getLoadBalancerName returns a name for the load balancer
func (l *loadBalancers) getLoadBalancerName(clusterName string, service *v1.Service) string {
	return fmt.Sprintf("k8s-%s-%s-%s", clusterName, service.Namespace, service.Name)
}

// loadBalancerID extracts the load balancer ID from service annotations
func (l *loadBalancers) loadBalancerID(service *v1.Service) (int64, error) {
	if service.Annotations == nil {
		return 0, nil
	}

	idStr, ok := service.Annotations[annBinaryLaneLoadBalancerID]
	if !ok || idStr == "" {
		return 0, nil
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid load balancer ID in annotation: %w", err)
	}

	return id, nil
}

// getServerIDsForNodes extracts server IDs for the given nodes
func (l *loadBalancers) getServerIDsForNodes(ctx context.Context, nodes []*v1.Node) ([]int64, error) {
	serverIDs := make([]int64, 0, len(nodes))

	for _, node := range nodes {
		// Try provider ID first
		if node.Spec.ProviderID != "" {
			id, err := parseProviderID(node.Spec.ProviderID)
			if err == nil {
				serverIDs = append(serverIDs, id)
				continue
			}
		}

		// Fall back to name lookup
		server, err := l.client.GetServerByName(ctx, node.Name)
		if err != nil {
			// Skip nodes that can't be found
			continue
		}
		serverIDs = append(serverIDs, server.ID)
	}

	return serverIDs, nil
}

// buildForwardingRules creates forwarding rules from service ports
func (l *loadBalancers) buildForwardingRules(service *v1.Service) []binarylane.ForwardingRule {
	protocol := l.getProtocol(service)
	if protocol == "" {
		protocol = "http"
	}
	
	return []binarylane.ForwardingRule{
		{
			EntryProtocol: strings.ToLower(protocol),
		},
	}
}

// buildHealthCheck creates health check configuration from service annotations
func (l *loadBalancers) buildHealthCheck(service *v1.Service) *binarylane.HealthCheck {
	protocol := "http"
	if p, ok := service.Annotations[annBinaryLaneHealthCheckProtocol]; ok && p != "" {
		protocol = p
	}

	path := "/"
	if p, ok := service.Annotations[annBinaryLaneHealthCheckPath]; ok && p != "" {
		path = p
	}

	return &binarylane.HealthCheck{
		Protocol: protocol,
		Path:     path,
	}
}

// getProtocol returns the protocol from service annotations
func (l *loadBalancers) getProtocol(service *v1.Service) string {
	if protocol, ok := service.Annotations[annBinaryLaneProtocol]; ok {
		return protocol
	}
	return ""
}

// diffServerIDs returns servers to add and remove
func diffServerIDs(current, desired []int64) (toAdd, toRemove []int64) {
	currentMap := make(map[int64]bool)
	for _, id := range current {
		currentMap[id] = true
	}

	desiredMap := make(map[int64]bool)
	for _, id := range desired {
		desiredMap[id] = true
		if !currentMap[id] {
			toAdd = append(toAdd, id)
		}
	}

	for _, id := range current {
		if !desiredMap[id] {
			toRemove = append(toRemove, id)
		}
	}

	return toAdd, toRemove
}
