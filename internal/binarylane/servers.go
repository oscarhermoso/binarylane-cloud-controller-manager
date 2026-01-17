package binarylane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var ErrServerNotFound = errors.New("server not found")

func (c *BinaryLaneClient) ListServers(ctx context.Context) ([]Server, error) {
	var allServers []Server
	page := int32(1)

	for {
		resp, err := c.GetServers(ctx, &GetServersParams{Page: &page})
		if err != nil {
			return nil, fmt.Errorf("failed to list servers: %w", err)
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		var serversResp ServersResponse
		if err := json.Unmarshal(body, &serversResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		allServers = append(allServers, serversResp.Servers...)

		if serversResp.Links == nil || serversResp.Links.Pages.Next == nil {
			break
		}
		page++
	}

	return allServers, nil
}

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

	if resp.StatusCode == 404 {
		return nil, ErrServerNotFound
	}
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
		return nil, fmt.Errorf("%w: %s", ErrServerNotFound, name)
	}

	return &serversResp.Servers[0], nil
}
