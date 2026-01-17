package cloud

import (
	"context"
	"testing"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/internal/binarylane"
	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
)

func TestListRoutes(t *testing.T) {
	vpcID := int64(100)
	tests := []struct {
		name          string
		servers       map[int64]*binarylane.Server
		vpcs          map[int64]*binarylane.Vpc
		cidr          string
		wantRoutes    int
		wantErr       bool
		wantErrPrefix string
	}{
		{
			name: "VPC with route entries",
			servers: map[int64]*binarylane.Server{
				1: {
					Id:    1,
					Name:  "test-cluster-node-1",
					VpcId: &vpcID,
					Networks: binarylane.Networks{
						V4: []binarylane.Network{
							{Type: "private", IpAddress: "10.240.0.10"},
						},
					},
				},
			},
			vpcs: map[int64]*binarylane.Vpc{
				100: {
					Id:   100,
					Name: "test-vpc",
					RouteEntries: []binarylane.RouteEntry{
						{
							Router:      "10.240.0.10",
							Destination: "10.244.1.0/24",
							Description: func() *string { s := "route for node-1"; return &s }(),
						},
					},
				},
			},
			cidr:       "10.244.0.0/16",
			wantRoutes: 1,
			wantErr:    false,
		},
		{
			name: "servers without VPC",
			servers: map[int64]*binarylane.Server{
				1: {
					Id:    1,
					Name:  "test-cluster-node-1",
					VpcId: nil,
				},
			},
			vpcs:       map[int64]*binarylane.Vpc{},
			cidr:       "10.244.0.0/16",
			wantRoutes: 0,
			wantErr:    false,
		},
		{
			name: "no CIDR configured",
			servers: map[int64]*binarylane.Server{
				1: {
					Id:    1,
					Name:  "test-cluster-node-1",
					VpcId: &vpcID,
				},
			},
			vpcs:          map[int64]*binarylane.Vpc{},
			cidr:          "",
			wantErr:       true,
			wantErrPrefix: "cluster CIDR not configured",
		},
		{
			name: "filters servers by cluster name",
			servers: map[int64]*binarylane.Server{
				1: {
					Id:    1,
					Name:  "test-cluster-node-1",
					VpcId: &vpcID,
					Networks: binarylane.Networks{
						V4: []binarylane.Network{
							{Type: "private", IpAddress: "10.240.0.10"},
						},
					},
				},
				2: {
					Id:    2,
					Name:  "other-cluster-node-1",
					VpcId: &vpcID,
					Networks: binarylane.Networks{
						V4: []binarylane.Network{
							{Type: "private", IpAddress: "10.240.0.20"},
						},
					},
				},
			},
			vpcs: map[int64]*binarylane.Vpc{
				100: {
					Id:   100,
					Name: "test-vpc",
					RouteEntries: []binarylane.RouteEntry{
						{
							Router:      "10.240.0.10",
							Destination: "10.244.1.0/24",
						},
						{
							Router:      "10.240.0.20",
							Destination: "10.244.2.0/24",
						},
					},
				},
			},
			cidr:       "10.244.0.0/16",
			wantRoutes: 2,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClient{
				servers: tt.servers,
				vpcs:    tt.vpcs,
			}
			r := &routes{
				client: mock,
				cidr:   tt.cidr,
			}

			routeList, err := r.ListRoutes(context.Background(), "test-cluster")

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListRoutes() expected error, got nil")
					return
				}
				if tt.wantErrPrefix != "" && len(err.Error()) < len(tt.wantErrPrefix) {
					t.Errorf("ListRoutes() error = %v, want prefix %v", err, tt.wantErrPrefix)
				}
				return
			}

			if err != nil {
				t.Errorf("ListRoutes() unexpected error = %v", err)
				return
			}

			if len(routeList) != tt.wantRoutes {
				t.Errorf("ListRoutes() returned %d routes, want %d", len(routeList), tt.wantRoutes)
			}
		})
	}
}

func TestCreateRoute(t *testing.T) {
	vpcID := int64(100)
	tests := []struct {
		name    string
		servers map[int64]*binarylane.Server
		vpcs    map[int64]*binarylane.Vpc
		route   *cloudprovider.Route
		wantErr bool
	}{
		{
			name: "create new route",
			servers: map[int64]*binarylane.Server{
				1: {
					Id:    1,
					Name:  "node-1",
					VpcId: &vpcID,
					Networks: binarylane.Networks{
						V4: []binarylane.Network{
							{Type: "private", IpAddress: "10.240.0.10"},
						},
					},
				},
			},
			vpcs: map[int64]*binarylane.Vpc{
				100: {
					Id:           100,
					Name:         "test-vpc",
					RouteEntries: []binarylane.RouteEntry{},
				},
			},
			route: &cloudprovider.Route{
				TargetNode:      types.NodeName("node-1"),
				DestinationCIDR: "10.244.1.0/24",
			},
			wantErr: false,
		},
		{
			name: "route already exists",
			servers: map[int64]*binarylane.Server{
				1: {
					Id:    1,
					Name:  "node-1",
					VpcId: &vpcID,
					Networks: binarylane.Networks{
						V4: []binarylane.Network{
							{Type: "private", IpAddress: "10.240.0.10"},
						},
					},
				},
			},
			vpcs: map[int64]*binarylane.Vpc{
				100: {
					Id:   100,
					Name: "test-vpc",
					RouteEntries: []binarylane.RouteEntry{
						{
							Router:      "10.240.0.10",
							Destination: "10.244.1.0/24",
						},
					},
				},
			},
			route: &cloudprovider.Route{
				TargetNode:      types.NodeName("node-1"),
				DestinationCIDR: "10.244.1.0/24",
			},
			wantErr: false,
		},
		{
			name:    "server not found",
			servers: map[int64]*binarylane.Server{},
			vpcs:    map[int64]*binarylane.Vpc{},
			route: &cloudprovider.Route{
				TargetNode:      types.NodeName("node-1"),
				DestinationCIDR: "10.244.1.0/24",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClient{
				servers: tt.servers,
				vpcs:    tt.vpcs,
			}
			r := &routes{
				client: mock,
				cidr:   "10.244.0.0/16",
			}

			err := r.CreateRoute(context.Background(), "test-cluster", "hint", tt.route)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteRoute(t *testing.T) {
	vpcID := int64(100)
	tests := []struct {
		name    string
		servers map[int64]*binarylane.Server
		vpcs    map[int64]*binarylane.Vpc
		route   *cloudprovider.Route
		wantErr bool
	}{
		{
			name: "delete existing route",
			servers: map[int64]*binarylane.Server{
				1: {
					Id:    1,
					Name:  "node-1",
					VpcId: &vpcID,
					Networks: binarylane.Networks{
						V4: []binarylane.Network{
							{Type: "private", IpAddress: "10.240.0.10"},
						},
					},
				},
			},
			vpcs: map[int64]*binarylane.Vpc{
				100: {
					Id:   100,
					Name: "test-vpc",
					RouteEntries: []binarylane.RouteEntry{
						{
							Router:      "10.240.0.10",
							Destination: "10.244.1.0/24",
						},
					},
				},
			},
			route: &cloudprovider.Route{
				TargetNode:      types.NodeName("node-1"),
				DestinationCIDR: "10.244.1.0/24",
			},
			wantErr: false,
		},
		{
			name: "route does not exist",
			servers: map[int64]*binarylane.Server{
				1: {
					Id:    1,
					Name:  "node-1",
					VpcId: &vpcID,
					Networks: binarylane.Networks{
						V4: []binarylane.Network{
							{Type: "private", IpAddress: "10.240.0.10"},
						},
					},
				},
			},
			vpcs: map[int64]*binarylane.Vpc{
				100: {
					Id:           100,
					Name:         "test-vpc",
					RouteEntries: []binarylane.RouteEntry{},
				},
			},
			route: &cloudprovider.Route{
				TargetNode:      types.NodeName("node-1"),
				DestinationCIDR: "10.244.1.0/24",
			},
			wantErr: false,
		},
		{
			name:    "server not found",
			servers: map[int64]*binarylane.Server{},
			vpcs:    map[int64]*binarylane.Vpc{},
			route: &cloudprovider.Route{
				TargetNode:      types.NodeName("node-1"),
				DestinationCIDR: "10.244.1.0/24",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClient{
				servers: tt.servers,
				vpcs:    tt.vpcs,
			}
			r := &routes{
				client: mock,
				cidr:   "10.244.0.0/16",
			}

			err := r.DeleteRoute(context.Background(), "test-cluster", tt.route)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteRoute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
