package binarylane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListServersPagination(t *testing.T) {
	page1Servers := []Server{
		{Id: 1, Name: "server-1"},
		{Id: 2, Name: "server-2"},
	}
	page2Servers := []Server{
		{Id: 3, Name: "server-3"},
		{Id: 4, Name: "server-4"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers" {
			t.Errorf("expected path /servers, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		page := r.URL.Query().Get("page")
		var resp ServersResponse

		switch page {
		case "", "1":
			nextURL := "http://example.com/v2/servers?page=2"
			resp = ServersResponse{
				Servers: page1Servers,
				Links: &Links{
					Pages: Pages{
						Next: &nextURL,
					},
				},
				Meta: Meta{Total: 4},
			}
		case "2":
			resp = ServersResponse{
				Servers: page2Servers,
				Links: &Links{
					Pages: Pages{
						Next: nil,
					},
				},
				Meta: Meta{Total: 4},
			}
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewBinaryLaneClient("test-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	client.Server = server.URL

	servers, err := client.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	expectedCount := len(page1Servers) + len(page2Servers)
	if len(servers) != expectedCount {
		t.Errorf("ListServers() returned %d servers, want %d", len(servers), expectedCount)
	}

	for i, expected := range append(page1Servers, page2Servers...) {
		if servers[i].Id != expected.Id {
			t.Errorf("server[%d].Id = %d, want %d", i, servers[i].Id, expected.Id)
		}
		if servers[i].Name != expected.Name {
			t.Errorf("server[%d].Name = %s, want %s", i, servers[i].Name, expected.Name)
		}
	}
}

func TestListServersNoPagination(t *testing.T) {
	allServers := []Server{
		{Id: 1, Name: "server-1"},
		{Id: 2, Name: "server-2"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ServersResponse{
			Servers: allServers,
			Links:   nil,
			Meta:    Meta{Total: 2},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewBinaryLaneClient("test-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	client.Server = server.URL

	servers, err := client.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	if len(servers) != len(allServers) {
		t.Errorf("ListServers() returned %d servers, want %d", len(servers), len(allServers))
	}
}
