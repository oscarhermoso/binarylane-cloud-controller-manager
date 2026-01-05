package cloud

import (
	"context"
	"testing"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/pkg/binarylane"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockClient struct {
	servers       map[int64]*binarylane.Server
	loadBalancers map[int64]*binarylane.LoadBalancer
}

func (m *mockClient) GetServer(ctx context.Context, serverID int64) (*binarylane.Server, error) {
	if server, ok := m.servers[serverID]; ok {
		return server, nil
	}
	return nil, &binarylane.ErrorResponse{Message: "server not found"}
}

func (m *mockClient) GetServerByName(ctx context.Context, name string) (*binarylane.Server, error) {
	for _, server := range m.servers {
		if server.Name == name {
			return server, nil
		}
	}
	return nil, &binarylane.ErrorResponse{Message: "server not found"}
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
				ID:     123,
				Name:   "test-node",
				Status: "active",
			},
		},
	}

	inst := &instancesV2{
		client: &binarylane.Client{},
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

func TestGetLoadBalancerName(t *testing.T) {
	lb := &loadBalancers{
		region: "syd",
	}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
	}

	name := lb.getLoadBalancerName("test-cluster", service)
	expected := "k8s-test-cluster-default-test-service"
	if name != expected {
		t.Errorf("getLoadBalancerName() = %v, want %v", name, expected)
	}
}

func TestDiffServerIDs(t *testing.T) {
	tests := []struct {
		name        string
		current     []int64
		desired     []int64
		wantToAdd   []int64
		wantToRemove []int64
	}{
		{
			name:        "no changes",
			current:     []int64{1, 2, 3},
			desired:     []int64{1, 2, 3},
			wantToAdd:   nil,
			wantToRemove: nil,
		},
		{
			name:        "add servers",
			current:     []int64{1, 2},
			desired:     []int64{1, 2, 3, 4},
			wantToAdd:   []int64{3, 4},
			wantToRemove: nil,
		},
		{
			name:        "remove servers",
			current:     []int64{1, 2, 3, 4},
			desired:     []int64{1, 2},
			wantToAdd:   nil,
			wantToRemove: []int64{3, 4},
		},
		{
			name:        "add and remove",
			current:     []int64{1, 2, 3},
			desired:     []int64{2, 3, 4},
			wantToAdd:   []int64{4},
			wantToRemove: []int64{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToAdd, gotToRemove := diffServerIDs(tt.current, tt.desired)
			
			if !equalInt64Slices(gotToAdd, tt.wantToAdd) {
				t.Errorf("diffServerIDs() toAdd = %v, want %v", gotToAdd, tt.wantToAdd)
			}
			if !equalInt64Slices(gotToRemove, tt.wantToRemove) {
				t.Errorf("diffServerIDs() toRemove = %v, want %v", gotToRemove, tt.wantToRemove)
			}
		})
	}
}

func equalInt64Slices(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	
	aMap := make(map[int64]bool)
	for _, v := range a {
		aMap[v] = true
	}
	
	for _, v := range b {
		if !aMap[v] {
			return false
		}
	}
	
	return true
}
