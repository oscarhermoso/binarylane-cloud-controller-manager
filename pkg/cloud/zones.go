package cloud

import (
	"context"

	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/apimachinery/pkg/types"
)

type zones struct {
	region string
}

// GetZone returns the Zone containing the current failure zone and locality region that the program is running in
func (z *zones) GetZone(ctx context.Context) (cloudprovider.Zone, error) {
	return cloudprovider.Zone{
		FailureDomain: z.region,
		Region:        z.region,
	}, nil
}

// GetZoneByProviderID returns the Zone containing the current zone and locality region of the node specified by providerId
func (z *zones) GetZoneByProviderID(ctx context.Context, providerID string) (cloudprovider.Zone, error) {
	// For BinaryLane, we use the region as both failure domain and region
	// In a more sophisticated implementation, you might parse this from metadata
	return cloudprovider.Zone{
		FailureDomain: z.region,
		Region:        z.region,
	}, nil
}

// GetZoneByNodeName returns the Zone containing the current zone and locality region of the node specified by node name
func (z *zones) GetZoneByNodeName(ctx context.Context, nodeName types.NodeName) (cloudprovider.Zone, error) {
	// For BinaryLane, we use the region as both failure domain and region
	return cloudprovider.Zone{
		FailureDomain: z.region,
		Region:        z.region,
	}, nil
}
