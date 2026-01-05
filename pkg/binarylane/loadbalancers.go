package binarylane

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// LoadBalancer represents a BinaryLane load balancer
type LoadBalancer struct {
	ID              int64            `json:"id"`
	Name            string           `json:"name"`
	IP              string           `json:"ip"`
	Status          string           `json:"status"`
	Region          *Region          `json:"region"`
	ForwardingRules []ForwardingRule `json:"forwarding_rules"`
	HealthCheck     HealthCheck      `json:"health_check"`
	ServerIDs       []int64          `json:"server_ids"`
	CreatedAt       string           `json:"created_at"`
}

// ForwardingRule represents a load balancer forwarding rule
type ForwardingRule struct {
	EntryProtocol string `json:"entry_protocol"`
}

// HealthCheck represents a load balancer health check configuration
type HealthCheck struct {
	Protocol string `json:"protocol"`
	Path     string `json:"path"`
}

// LoadBalancerRequest represents a request to create or update a load balancer
type LoadBalancerRequest struct {
	Name            string            `json:"name"`
	Region          string            `json:"region,omitempty"`
	ForwardingRules []ForwardingRule  `json:"forwarding_rules,omitempty"`
	HealthCheck     *HealthCheck      `json:"health_check,omitempty"`
	ServerIDs       []int64           `json:"server_ids,omitempty"`
}

// LoadBalancersResponse represents the response from listing load balancers
type LoadBalancersResponse struct {
	LoadBalancers []LoadBalancer `json:"load_balancers"`
	Meta          Meta           `json:"meta"`
}

// LoadBalancerResponse represents the response from getting a single load balancer
type LoadBalancerResponse struct {
	LoadBalancer LoadBalancer `json:"load_balancer"`
}

// GetLoadBalancer retrieves a load balancer by ID
func (c *Client) GetLoadBalancer(ctx context.Context, lbID int64) (*LoadBalancer, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/load_balancers/%d", lbID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := parseError(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lbResp LoadBalancerResponse
	if err := json.Unmarshal(body, &lbResp); err != nil {
		return nil, err
	}

	return &lbResp.LoadBalancer, nil
}

// ListLoadBalancers retrieves all load balancers
func (c *Client) ListLoadBalancers(ctx context.Context) ([]LoadBalancer, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/load_balancers", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := parseError(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lbsResp LoadBalancersResponse
	if err := json.Unmarshal(body, &lbsResp); err != nil {
		return nil, err
	}

	return lbsResp.LoadBalancers, nil
}

// CreateLoadBalancer creates a new load balancer
func (c *Client) CreateLoadBalancer(ctx context.Context, req *LoadBalancerRequest) (*LoadBalancer, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/load_balancers", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := parseError(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lbResp LoadBalancerResponse
	if err := json.Unmarshal(body, &lbResp); err != nil {
		return nil, err
	}

	return &lbResp.LoadBalancer, nil
}

// UpdateLoadBalancer updates an existing load balancer
func (c *Client) UpdateLoadBalancer(ctx context.Context, lbID int64, req *LoadBalancerRequest) (*LoadBalancer, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/load_balancers/%d", lbID), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := parseError(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lbResp LoadBalancerResponse
	if err := json.Unmarshal(body, &lbResp); err != nil {
		return nil, err
	}

	return &lbResp.LoadBalancer, nil
}

// DeleteLoadBalancer deletes a load balancer
func (c *Client) DeleteLoadBalancer(ctx context.Context, lbID int64) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/load_balancers/%d", lbID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return parseError(resp)
}

// AddServersToLoadBalancer adds servers to a load balancer
func (c *Client) AddServersToLoadBalancer(ctx context.Context, lbID int64, serverIDs []int64) error {
	req := map[string]interface{}{
		"server_ids": serverIDs,
	}
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/load_balancers/%d/servers", lbID), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return parseError(resp)
}

// RemoveServersFromLoadBalancer removes servers from a load balancer
func (c *Client) RemoveServersFromLoadBalancer(ctx context.Context, lbID int64, serverIDs []int64) error {
	req := map[string]interface{}{
		"server_ids": serverIDs,
	}
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/load_balancers/%d/servers", lbID), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return parseError(resp)
}
