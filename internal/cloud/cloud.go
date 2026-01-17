package cloud

import (
	"fmt"
	"io"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/internal/binarylane"
	cloudprovider "k8s.io/cloud-provider"
)

const (
	ProviderName = "binarylane"
)

var _ cloudprovider.Interface = &Cloud{}

type Cloud struct {
	client *binarylane.BinaryLaneClient
	cidr   string
}

func NewCloud(token string, cidr string) (cloudprovider.Interface, error) {
	if token == "" {
		return nil, fmt.Errorf("BinaryLane API token is required")
	}

	client, err := binarylane.NewBinaryLaneClient(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create BinaryLane client: %w", err)
	}

	return &Cloud{
		client: client,
		cidr:   cidr,
	}, nil
}

func (c *Cloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
}

func (c *Cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return nil, false
}

func (c *Cloud) Instances() (cloudprovider.Instances, bool) {
	// Replaced by InstancesV2
	return nil, false
}

func (c *Cloud) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return &instancesV2{
		client: c.client,
	}, true
}

func (c *Cloud) Zones() (cloudprovider.Zones, bool) {
	// Replaced by InstancesV2
	return nil, false
}

func (c *Cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

func (c *Cloud) Routes() (cloudprovider.Routes, bool) {
	return &routes{
		client: c.client,
		cidr:   c.cidr,
	}, true
}

func (c *Cloud) ProviderName() string {
	return ProviderName
}

func (c *Cloud) HasClusterID() bool {
	return false
}

func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		return nil, fmt.Errorf("use NewCloud function to create BinaryLane cloud provider")
	})
}
