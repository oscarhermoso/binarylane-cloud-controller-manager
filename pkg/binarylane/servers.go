package binarylane

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Server represents a BinaryLane server (instance)
type Server struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Region    Region   `json:"region"`
	Networks  Networks `json:"networks"`
	CreatedAt string   `json:"created_at"`
	Memory    int      `json:"memory"`
	Vcpus     int      `json:"vcpus"`
	Disk      int      `json:"disk"`
}

// Region represents a BinaryLane region
type Region struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Networks represents network configuration
type Networks struct {
	V4 []Network `json:"v4"`
	V6 []Network `json:"v6"`
}

// Network represents a network interface
type Network struct {
	IPAddress   string      `json:"ip_address"`
	Netmask     interface{} `json:"netmask"` // Can be string or int
	Gateway     string      `json:"gateway"`
	Type        string      `json:"type"` // "public" or "private"
	ReverseName string      `json:"reverse_name,omitempty"`
}

// ServersResponse represents the response from listing servers
type ServersResponse struct {
	Servers []Server `json:"servers"`
	Meta    Meta     `json:"meta"`
}

// ServerResponse represents the response from getting a single server
type ServerResponse struct {
	Server Server `json:"server"`
}

// Meta represents pagination metadata
type Meta struct {
	Total int `json:"total"`
}

// GetServer retrieves a server by ID
func (c *Client) GetServer(ctx context.Context, serverID int64) (*Server, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/servers/%d", serverID), nil)
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

	var serverResp ServerResponse
	if err := json.Unmarshal(body, &serverResp); err != nil {
		return nil, err
	}

	return &serverResp.Server, nil
}

// ListServers retrieves all servers
func (c *Client) ListServers(ctx context.Context) ([]Server, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/servers", nil)
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

	var serversResp ServersResponse
	if err := json.Unmarshal(body, &serversResp); err != nil {
		return nil, err
	}

	return serversResp.Servers, nil
}

// GetServerByName retrieves a server by name
func (c *Client) GetServerByName(ctx context.Context, name string) (*Server, error) {
	servers, err := c.ListServers(ctx)
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		if server.Name == name {
			return &server, nil
		}
	}

	return nil, fmt.Errorf("server %q not found", name)
}
