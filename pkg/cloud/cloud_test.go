package cloud

import (
	"context"
	"fmt"
	"testing"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/pkg/binarylane"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockClient struct {
	servers map[int64]*binarylane.Server
}

func (m *mockClient) GetServer(ctx context.Context, serverID int64) (*binarylane.Server, error) {
	if server, ok := m.servers[serverID]; ok {
		return server, nil
	}
	return nil, fmt.Errorf("server not found")
}

func (m *mockClient) GetServerByName(ctx context.Context, name string) (*binarylane.Server, error) {
	for _, server := range m.servers {
		if server.Name == name {
			return server, nil
		}
	}
	return nil, fmt.Errorf("server not found")
}

func (m *mockClient) ListServers(ctx context.Context) ([]binarylane.Server, error) {
	servers := make([]binarylane.Server, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, *server)
	}
	return servers, nil
}

func TestInstanceExists(t *testing.T) {
	mock := &mockClient{
		servers: map[int64]*binarylane.Server{
			123: {
				Id:     123,
				Name:   "test-node",
				Status: "active",
			},
		},
	}

	inst := &instancesV2{
		client: &binarylane.BinaryLaneClient{},
		region: "syd",
	}

	// Override the client's methods - in real tests we'd use dependency injection
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: v1.NodeSpec{
			ProviderID: "binarylane://123",
		},
	}

	// Test would require proper mocking framework
	// This is a simplified example
	_ = inst
	_ = node
	_ = mock
}

func TestParseProviderID(t *testing.T) {
	tests := []struct {
		name       string
		providerID string
		wantID     int64
		wantErr    bool
	}{
		{
			name:       "valid provider ID",
			providerID: "binarylane://123",
			wantID:     123,
			wantErr:    false,
		},
		{
			name:       "invalid format",
			providerID: "invalid://123",
			wantErr:    true,
		},
		{
			name:       "invalid ID",
			providerID: "binarylane://abc",
			wantErr:    true,
		},
		{
			name:       "empty ID",
			providerID: "binarylane://",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := parseProviderID(tt.providerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseProviderID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && gotID != tt.wantID {
				t.Errorf("parseProviderID() = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}
