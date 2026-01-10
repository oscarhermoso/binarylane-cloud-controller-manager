package binarylane

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// ListServers lists all servers
func (c *BinaryLaneClient) ListServers(ctx context.Context) ([]Server, error) {
	resp, err := c.GetServers(ctx, &GetServersParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var serversResp ServersResponse
	if err := json.Unmarshal(body, &serversResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return serversResp.Servers, nil
}

// GetServer gets a server by ID
func (c *BinaryLaneClient) GetServer(ctx context.Context, serverID int64) (*Server, error) {
	resp, err := c.GetServersServerId(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var serverResp ServerResponse
	if err := json.Unmarshal(body, &serverResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &serverResp.Server, nil
}

// GetServerByName gets a server by hostname
func (c *BinaryLaneClient) GetServerByName(ctx context.Context, name string) (*Server, error) {
	hostname := name
	resp, err := c.GetServers(ctx, &GetServersParams{
		Hostname: &hostname,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get server by name: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var serversResp ServersResponse
	if err := json.Unmarshal(body, &serversResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(serversResp.Servers) == 0 {
		return nil, fmt.Errorf("server not found: %s", name)
	}

	return &serversResp.Servers[0], nil
}
