package cloud

import (
	"fmt"
	"io"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/internal/binarylane"
	cloudprovider "k8s.io/cloud-provider"
)

const (
	// ProviderName is the name of the BinaryLane cloud provider
	ProviderName = "binarylane"
)

var _ cloudprovider.Interface = &Cloud{}

// Cloud is the BinaryLane implementation of the cloud provider interface
type Cloud struct {
	client *binarylane.BinaryLaneClient
	region string
}

// NewCloud creates a new BinaryLane cloud provider
func NewCloud(token, region string) (cloudprovider.Interface, error) {
	if token == "" {
		return nil, fmt.Errorf("BinaryLane API token is required")
	}

	client, err := binarylane.NewBinaryLaneClient(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create BinaryLane client: %w", err)
	}

	return &Cloud{
		client: client,
		region: region,
	}, nil
}

// Initialize provides the cloud with a kubernetes client builder and may spawn goroutines
// to perform housekeeping or run custom controllers specific to the cloud provider.
func (c *Cloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
}

// LoadBalancer returns a load balancer interface. Also returns true if the interface is supported, false otherwise.
func (c *Cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return nil, false
}

// Instances returns an instances interface. Also returns true if the interface is supported, false otherwise.
func (c *Cloud) Instances() (cloudprovider.Instances, bool) {
	return nil, false
}

// InstancesV2 is an implementation for instances and should only be implemented by external cloud providers.
// Implementing InstancesV2 is behaviorally identical to Instances but is optimized to significantly reduce
// API calls to the cloud provider when registering and syncing nodes.
func (c *Cloud) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return &instancesV2{
		client: c.client,
		region: c.region,
	}, true
}

// Zones returns a zones interface. Also returns true if the interface is supported, false otherwise.
func (c *Cloud) Zones() (cloudprovider.Zones, bool) {
	return &zones{
		region: c.region,
	}, true
}

// Clusters returns a clusters interface. Also returns true if the interface is supported, false otherwise.
func (c *Cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// Routes returns a routes interface along with whether the interface is supported.
func (c *Cloud) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

// ProviderName returns the cloud provider ID.
func (c *Cloud) ProviderName() string {
	return ProviderName
}

// HasClusterID returns true if a ClusterID is required and set
func (c *Cloud) HasClusterID() bool {
	return false
}

func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		// This function is called by the cloud provider framework
		// In practice, we'll use NewCloud directly with environment variables
		return nil, fmt.Errorf("use NewCloud function to create BinaryLane cloud provider")
	})
}
