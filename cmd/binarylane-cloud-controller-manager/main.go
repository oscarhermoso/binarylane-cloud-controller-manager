package main

import (
	"os"

	"k8s.io/apimachinery/pkg/util/wait"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/cloud-provider/app"
	"k8s.io/cloud-provider/app/config"
	"k8s.io/cloud-provider/names"
	"k8s.io/cloud-provider/options"
	"k8s.io/component-base/cli"
	cliflag "k8s.io/component-base/cli/flag"
	_ "k8s.io/component-base/metrics/prometheus/clientgo"
	_ "k8s.io/component-base/metrics/prometheus/version"
	"k8s.io/klog/v2"

	_ "github.com/oscarhermoso/binarylane-cloud-controller-manager/internal/cloud"
)

func main() {
	opts, err := options.NewCloudControllerManagerOptions()
	if err != nil {
		klog.Fatalf("unable to initialize command options: %v", err)
	}

	controllerInitializers := app.DefaultInitFuncConstructors
	controllerAliases := names.CCMControllerAliases()
	fss := cliflag.NamedFlagSets{}

	command := app.NewCloudControllerManagerCommand(
		opts,
		cloudInitializer,
		controllerInitializers,
		controllerAliases,
		fss,
		wait.NeverStop,
	)
	command.Use = "binarylane-cloud-controller-manager"

	code := cli.Run(command)
	os.Exit(code)
}

func cloudInitializer(config *config.CompletedConfig) cloudprovider.Interface {
	cloudConfig := config.ComponentConfig.KubeCloudShared.CloudProvider
	cloud, err := cloudprovider.InitCloudProvider(cloudConfig.Name, cloudConfig.CloudConfigFile)
	if err != nil {
		klog.Fatalf("Cloud provider could not be initialized: %v", err)
	}
	if cloud == nil {
		klog.Fatalf("Cloud provider is nil")
	}

	// TODO: Uncomment after cluster IDs are implemented
	// if !cloud.HasClusterID() {
	// 	if config.ComponentConfig.KubeCloudShared.AllowUntaggedCloud {
	// 		klog.Warning("detected a cluster without a ClusterID.  A ClusterID will be required in the future.  Please tag your cluster to avoid any future issues")
	// 	} else {
	// 		klog.Fatalf("no ClusterID found.  A ClusterID is required for the cloud provider to function properly.  This check can be bypassed by setting the allow-untagged-cloud option")
	// 	}
	// }

	// TODO: There's a lot of potentially valuable configuration in config.ComponentConfig.KubeCloudShared..., consider passing it to the cloud provider here

	return cloud
}
