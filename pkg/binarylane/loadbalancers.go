package binarylane

import (
"context"
"encoding/json"
"fmt"
"io"
)

// GetLoadBalancer retrieves a load balancer by ID
func (c *BinaryLaneClient) GetLoadBalancer(ctx context.Context, lbID int64) (*LoadBalancer, error) {
resp, err := c.GetLoadBalancersLoadBalancerId(ctx, lbID)
if err != nil {
return nil, fmt.Errorf("failed to get load balancer: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 200 {
body, _ := io.ReadAll(resp.Body)
return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, fmt.Errorf("failed to read response: %w", err)
}

var lbResp LoadBalancerResponse
if err := json.Unmarshal(body, &lbResp); err != nil {
return nil, fmt.Errorf("failed to unmarshal response: %w", err)
}

return &lbResp.LoadBalancer, nil
}

// ListLoadBalancers retrieves all load balancers
func (c *BinaryLaneClient) ListLoadBalancers(ctx context.Context) ([]LoadBalancer, error) {
resp, err := c.GetLoadBalancers(ctx, &GetLoadBalancersParams{})
if err != nil {
return nil, fmt.Errorf("failed to list load balancers: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 200 {
body, _ := io.ReadAll(resp.Body)
return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, fmt.Errorf("failed to read response: %w", err)
}

var lbsResp LoadBalancersResponse
if err := json.Unmarshal(body, &lbsResp); err != nil {
return nil, fmt.Errorf("failed to unmarshal response: %w", err)
}

return lbsResp.LoadBalancers, nil
}

// CreateLoadBalancer creates a new load balancer
func (c *BinaryLaneClient) CreateLoadBalancer(ctx context.Context, req *CreateLoadBalancerRequest) (*LoadBalancer, error) {
resp, err := c.PostLoadBalancers(ctx, PostLoadBalancersJSONRequestBody(*req))
if err != nil {
return nil, fmt.Errorf("failed to create load balancer: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 200 {
body, _ := io.ReadAll(resp.Body)
return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, fmt.Errorf("failed to read response: %w", err)
}

var lbResp CreateLoadBalancerResponse
if err := json.Unmarshal(body, &lbResp); err != nil {
return nil, fmt.Errorf("failed to unmarshal response: %w", err)
}

return &lbResp.LoadBalancer, nil
}

// UpdateLoadBalancer updates an existing load balancer
func (c *BinaryLaneClient) UpdateLoadBalancer(ctx context.Context, lbID int64, req *UpdateLoadBalancerRequest) (*LoadBalancer, error) {
resp, err := c.PutLoadBalancersLoadBalancerId(ctx, lbID, PutLoadBalancersLoadBalancerIdJSONRequestBody(*req))
if err != nil {
return nil, fmt.Errorf("failed to update load balancer: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 200 {
body, _ := io.ReadAll(resp.Body)
return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, fmt.Errorf("failed to read response: %w", err)
}

var lbResp UpdateLoadBalancerResponse
if err := json.Unmarshal(body, &lbResp); err != nil {
return nil, fmt.Errorf("failed to unmarshal response: %w", err)
}

return &lbResp.LoadBalancer, nil
}

// DeleteLoadBalancer deletes a load balancer
func (c *BinaryLaneClient) DeleteLoadBalancer(ctx context.Context, lbID int64) error {
resp, err := c.DeleteLoadBalancersLoadBalancerId(ctx, lbID)
if err != nil {
return fmt.Errorf("failed to delete load balancer: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 204 {
body, _ := io.ReadAll(resp.Body)
return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
}

return nil
}

// AddServersToLoadBalancer adds servers to a load balancer
func (c *BinaryLaneClient) AddServersToLoadBalancer(ctx context.Context, lbID int64, serverIDs []int64) error {
req := ServerIdsRequest{
ServerIds: serverIDs,
}
resp, err := c.PostLoadBalancersLoadBalancerIdServers(ctx, lbID, PostLoadBalancersLoadBalancerIdServersJSONRequestBody(req))
if err != nil {
return fmt.Errorf("failed to add servers to load balancer: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 204 {
body, _ := io.ReadAll(resp.Body)
return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
}

return nil
}

// RemoveServersFromLoadBalancer removes servers from a load balancer
func (c *BinaryLaneClient) RemoveServersFromLoadBalancer(ctx context.Context, lbID int64, serverIDs []int64) error {
req := ServerIdsRequest{
ServerIds: serverIDs,
}
resp, err := c.DeleteLoadBalancersLoadBalancerIdServers(ctx, lbID, DeleteLoadBalancersLoadBalancerIdServersJSONRequestBody(req))
if err != nil {
return fmt.Errorf("failed to remove servers from load balancer: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 204 {
body, _ := io.ReadAll(resp.Body)
return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
}

return nil
}
