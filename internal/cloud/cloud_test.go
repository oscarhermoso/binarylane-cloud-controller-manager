package cloud

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/internal/binarylane"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: Replace this with the built-in new() function after upgrading to Go 1.26
func toPtr[T any](v T) *T {
	return &v
}

type ClientInterface interface {
	GetServer(ctx context.Context, serverID int64) (*binarylane.Server, error)
	GetServerByName(ctx context.Context, name string) (*binarylane.Server, error)
	ListServers(ctx context.Context) ([]binarylane.Server, error)
	GetVpc(ctx context.Context, vpcID int64) (*binarylane.Vpc, error)
	UpdateVpc(ctx context.Context, vpcID int64, req binarylane.UpdateVpcRequest) (*binarylane.Vpc, error)
}

type mockClient struct {
	servers map[int64]*binarylane.Server
	vpcs    map[int64]*binarylane.Vpc
}

func (m *mockClient) GetServer(ctx context.Context, serverID int64) (*binarylane.Server, error) {
	if server, ok := m.servers[serverID]; ok {
		return server, nil
	}
	return nil, binarylane.ErrServerNotFound
}

func (m *mockClient) GetServerByName(ctx context.Context, name string) (*binarylane.Server, error) {
	for _, server := range m.servers {
		if server.Name == name {
			return server, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", binarylane.ErrServerNotFound, name)
}

func (m *mockClient) ListServers(ctx context.Context) ([]binarylane.Server, error) {
	servers := make([]binarylane.Server, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, *server)
	}
	return servers, nil
}

func (m *mockClient) GetVpc(ctx context.Context, vpcID int64) (*binarylane.Vpc, error) {
	if vpc, ok := m.vpcs[vpcID]; ok {
		return vpc, nil
	}
	return nil, binarylane.ErrVpcNotFound
}

func (m *mockClient) UpdateVpc(ctx context.Context, vpcID int64, req binarylane.UpdateVpcRequest) (*binarylane.Vpc, error) {
	vpc, ok := m.vpcs[vpcID]
	if !ok {
		return nil, binarylane.ErrVpcNotFound
	}

	vpc.Name = req.Name
	if req.RouteEntries != nil {
		vpc.RouteEntries = make([]binarylane.RouteEntry, len(*req.RouteEntries))
		for i, r := range *req.RouteEntries {
			vpc.RouteEntries[i] = binarylane.RouteEntry(r)
		}
	}

	return vpc, nil
}

func TestInstanceMetadata(t *testing.T) {
	tests := []struct {
		name              string
		server            *binarylane.Server
		wantInternalIPs   []string
		wantExternalIPs   []string
		wantProviderID    string
		wantInstanceType  string
		wantHostLabel     string
		wantServerIDLabel string
	}{
		{
			name: "server with both private and public IPs",
			server: &binarylane.Server{
				Id:     123,
				Name:   "test-node",
				Size:   binarylane.Size{Slug: "std-2vcpu"},
				Region: binarylane.Region{Slug: "syd"},
				Host:   binarylane.Host{DisplayName: "physical-host-01"},
				VpcId:  toPtr(int64(1)),
				Networks: binarylane.Networks{
					V4: []binarylane.Network{
						{
							IpAddress: "43.229.63.57",
							Type:      "public",
						},
						{
							IpAddress: "172.24.41.233",
							Type:      "private",
						},
					},
				},
			},
			wantInternalIPs:   []string{"172.24.41.233"},
			wantExternalIPs:   []string{"43.229.63.57"},
			wantProviderID:    "binarylane://123",
			wantInstanceType:  "std-2vcpu",
			wantHostLabel:     "physical-host-01",
			wantServerIDLabel: "123",
		},
		{
			name: "server with only public IP",
			server: &binarylane.Server{
				Id:     456,
				Name:   "worker-node",
				Size:   binarylane.Size{Slug: "std-1vcpu"},
				Region: binarylane.Region{Slug: "per"},
				Host:   binarylane.Host{DisplayName: ""},
				Networks: binarylane.Networks{
					V4: []binarylane.Network{
						{
							IpAddress: "103.1.186.136",
							Type:      "public",
						},
					},
				},
			},
			wantInternalIPs:   []string{},
			wantExternalIPs:   []string{"103.1.186.136"},
			wantProviderID:    "binarylane://456",
			wantInstanceType:  "std-1vcpu",
			wantHostLabel:     "",
			wantServerIDLabel: "456",
		},
		{
			name: "server with multiple private and public IPs",
			server: &binarylane.Server{
				Id:     789,
				Name:   "multi-nic-node",
				Size:   binarylane.Size{Slug: "std-4vcpu"},
				Region: binarylane.Region{Slug: "mel-1"},
				Host:   binarylane.Host{DisplayName: "physical-host-02"},
				VpcId:  toPtr(int64(2)),
				Networks: binarylane.Networks{
					V4: []binarylane.Network{
						{
							IpAddress: "203.29.241.229",
							Type:      "public",
						},
						{
							IpAddress: "172.24.50.100",
							Type:      "private",
						},
						{
							IpAddress: "10.0.0.50",
							Type:      "private",
						},
					},
				},
			},
			wantInternalIPs:   []string{"172.24.50.100", "10.0.0.50"},
			wantExternalIPs:   []string{"203.29.241.229"},
			wantProviderID:    "binarylane://789",
			wantInstanceType:  "std-4vcpu",
			wantHostLabel:     "physical-host-02",
			wantServerIDLabel: "789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClient{
				servers: map[int64]*binarylane.Server{
					tt.server.Id: tt.server,
				},
			}

			inst := &instancesV2{client: mock}
			node := &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.server.Name,
				},
				Spec: v1.NodeSpec{
					ProviderID: fmt.Sprintf("binarylane://%d", tt.server.Id),
				},
			}

			metadata, err := inst.InstanceMetadata(context.Background(), node)
			if err != nil {
				t.Fatalf("InstanceMetadata() error = %v", err)
			}

			// Check provider ID
			if metadata.ProviderID != tt.wantProviderID {
				t.Errorf("ProviderID = %v, want %v", metadata.ProviderID, tt.wantProviderID)
			}

			// Check instance type
			if metadata.InstanceType != tt.wantInstanceType {
				t.Errorf("InstanceType = %v, want %v", metadata.InstanceType, tt.wantInstanceType)
			}

			// Check internal IPs
			var internalIPs []string
			for _, addr := range metadata.NodeAddresses {
				if addr.Type == v1.NodeInternalIP {
					internalIPs = append(internalIPs, addr.Address)
				}
			}
			if len(internalIPs) != len(tt.wantInternalIPs) {
				t.Errorf("Number of internal IPs = %v, want %v", internalIPs, tt.wantInternalIPs)
			}
			for i, ip := range internalIPs {
				if i >= len(tt.wantInternalIPs) || ip != tt.wantInternalIPs[i] {
					t.Errorf("Internal IP[%d] = %v, want %v", i, ip, tt.wantInternalIPs)
				}
			}

			// Check external IPs
			var externalIPs []string
			for _, addr := range metadata.NodeAddresses {
				if addr.Type == v1.NodeExternalIP {
					externalIPs = append(externalIPs, addr.Address)
				}
			}
			if len(externalIPs) != len(tt.wantExternalIPs) {
				t.Errorf("Number of external IPs = %v, want %v", externalIPs, tt.wantExternalIPs)
			}
			for i, ip := range externalIPs {
				if i >= len(tt.wantExternalIPs) || ip != tt.wantExternalIPs[i] {
					t.Errorf("External IP[%d] = %v, want %v", i, ip, tt.wantExternalIPs)
				}
			}

			// Check labels
			if tt.wantHostLabel != "" {
				if metadata.AdditionalLabels["binarylane.com/host"] != tt.wantHostLabel {
					t.Errorf("Host label = %v, want %v", metadata.AdditionalLabels["binarylane.com/host"], tt.wantHostLabel)
				}
			}
		})
	}
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

func TestRoutesWithoutCIDR(t *testing.T) {
	cloud := &Cloud{
		client: nil,
		cidr:   "",
	}

	routes, enabled := cloud.Routes()
	if enabled {
		t.Error("Routes() should be disabled when CIDR is not configured")
	}
	if routes != nil {
		t.Error("Routes() should return nil when disabled")
	}
}

func TestRoutesWithCIDR(t *testing.T) {
	cloud := &Cloud{
		client: nil,
		cidr:   "10.244.0.0/16",
	}

	routes, enabled := cloud.Routes()
	if !enabled {
		t.Error("Routes() should be enabled when CIDR is configured")
	}
	if routes == nil {
		t.Error("Routes() should return non-nil when enabled")
	}
}

func TestNewCloud_RequiresToken(t *testing.T) {
	t.Setenv("BINARYLANE_API_TOKEN", "")

	cloud, err := newCloud(strings.NewReader(""))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if cloud != nil {
		t.Fatalf("expected nil cloud, got %#v", cloud)
	}
}

func TestNewCloud_ReturnsCloudWithToken(t *testing.T) {
	t.Setenv("BINARYLANE_API_TOKEN", "test-token")

	cloud, err := newCloud(strings.NewReader(""))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cloud == nil {
		t.Fatalf("expected non-nil cloud")
	}
}
