package main

import (
	"fmt"
	"os"

	"github.com/oscarhermoso/binarylane-cloud-controller-manager/internal/cloud"
	"k8s.io/apimachinery/pkg/util/wait"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/cloud-provider/app"
	"k8s.io/cloud-provider/app/config"
	"k8s.io/cloud-provider/options"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	_ "k8s.io/component-base/metrics/prometheus/clientgo"
	_ "k8s.io/component-base/metrics/prometheus/version"
	"k8s.io/klog/v2"
)

const (
	binaryLaneAccessTokenEnv = "BINARYLANE_ACCESS_TOKEN"
	binaryLaneRegionEnv      = "BINARYLANE_REGION"
)

func main() {
	opts, err := options.NewCloudControllerManagerOptions()
	if err != nil {
		klog.Fatalf("unable to initialize command options: %v", err)
	}

	controllerInitializers := app.DefaultInitFuncConstructors
	fss := cliflag.NamedFlagSets{}
	featureGates := make(map[string]string)

	command := app.NewCloudControllerManagerCommand(
		opts,
		cloudInitializer,
		controllerInitializers,
		featureGates,
		fss,
		wait.NeverStop,
	)

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cloudInitializer(config *config.CompletedConfig) cloudprovider.Interface {
	cloudConfig := config.ComponentConfig.KubeCloudShared.CloudProvider

	token := os.Getenv(binaryLaneAccessTokenEnv)
	if token == "" {
		klog.Fatalf("%s environment variable is required", binaryLaneAccessTokenEnv)
	}

	region := os.Getenv(binaryLaneRegionEnv)
	if region == "" {
		klog.Warning("BINARYLANE_REGION environment variable not set, using default")
		region = "default"
	}

	cloudProvider, err := cloud.NewCloud(token, region)
	if err != nil {
		klog.Fatalf("failed to initialize BinaryLane cloud provider: %v", err)
	}

	cloudProvider.Initialize(config.ClientBuilder, wait.NeverStop)

	klog.Infof("BinaryLane cloud controller manager initialized (provider: %s, region: %s)",
		cloudConfig.Name, region)

	return cloudProvider
}
