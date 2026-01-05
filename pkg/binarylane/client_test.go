package binarylane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-token")
	if client == nil {
		t.Fatal("expected client to be created")
	}
	if client.token != "test-token" {
		t.Errorf("expected token to be 'test-token', got %s", client.token)
	}
	if client.baseURL != defaultBaseURL {
		t.Errorf("expected baseURL to be %s, got %s", defaultBaseURL, client.baseURL)
	}
}

func TestGetServer(t *testing.T) {
	server := &Server{
		ID:     123,
		Name:   "test-server",
		Status: "active",
		Region: Region{
			Name: "Sydney",
			Slug: "syd",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/servers/123" {
			t.Errorf("expected path /servers/123, got %s", r.URL.Path)
		}
		
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %s", auth)
		}

		resp := ServerResponse{Server: *server}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClient("test-token")
	client.baseURL = ts.URL

	result, err := client.GetServer(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != server.ID {
		t.Errorf("expected ID %d, got %d", server.ID, result.ID)
	}
	if result.Name != server.Name {
		t.Errorf("expected name %s, got %s", server.Name, result.Name)
	}
}

func TestListServers(t *testing.T) {
	servers := []Server{
		{ID: 1, Name: "server1", Status: "active"},
		{ID: 2, Name: "server2", Status: "active"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/servers" {
			t.Errorf("expected path /servers, got %s", r.URL.Path)
		}

		resp := ServersResponse{
			Servers: servers,
			Meta:    Meta{Total: len(servers)},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClient("test-token")
	client.baseURL = ts.URL

	result, err := client.ListServers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(servers) {
		t.Errorf("expected %d servers, got %d", len(servers), len(result))
	}
}

func TestGetServerByName(t *testing.T) {
	servers := []Server{
		{ID: 1, Name: "server1", Status: "active"},
		{ID: 2, Name: "server2", Status: "active"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ServersResponse{
			Servers: servers,
			Meta:    Meta{Total: len(servers)},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClient("test-token")
	client.baseURL = ts.URL

	result, err := client.GetServerByName(context.Background(), "server2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "server2" {
		t.Errorf("expected name server2, got %s", result.Name)
	}
	if result.ID != 2 {
		t.Errorf("expected ID 2, got %d", result.ID)
	}
}

func TestGetServerByName_NotFound(t *testing.T) {
	servers := []Server{
		{ID: 1, Name: "server1", Status: "active"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ServersResponse{
			Servers: servers,
			Meta:    Meta{Total: len(servers)},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClient("test-token")
	client.baseURL = ts.URL

	_, err := client.GetServerByName(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "success status",
			statusCode: 200,
			body:       "",
			wantErr:    false,
		},
		{
			name:       "error with json",
			statusCode: 404,
			body:       `{"message": "not found"}`,
			wantErr:    true,
		},
		{
			name:       "error without json",
			statusCode: 500,
			body:       "internal server error",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer ts.Close()

			resp, err := http.Get(ts.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			err = parseError(resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
