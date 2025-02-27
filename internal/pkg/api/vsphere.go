package api

import (
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/cluster"
)

type VSphereConfig struct {
	datacenterConfig *anywherev1.VSphereDatacenterConfig
	machineConfigs   map[string]*anywherev1.VSphereMachineConfig
}

type VSphereFiller func(config VSphereConfig)

// VSphereToConfigFiller transforms a set of VSphereFiller's in a single ClusterConfigFiller.
func VSphereToConfigFiller(fillers ...VSphereFiller) ClusterConfigFiller {
	return func(c *cluster.Config) {
		updateVSphere(c, fillers...)
	}
}

// updateVSphere updates the vSphere datacenter and machine configs in the
// cluster.Config by applying all the fillers.
func updateVSphere(config *cluster.Config, fillers ...VSphereFiller) {
	vc := VSphereConfig{
		datacenterConfig: config.VSphereDatacenter,
		machineConfigs:   config.VSphereMachineConfigs,
	}

	for _, f := range fillers {
		f(vc)
	}
}

func WithOsFamilyForAllMachines(value anywherev1.OSFamily) VSphereFiller {
	return func(config VSphereConfig) {
		for _, m := range config.machineConfigs {
			m.Spec.OSFamily = value
		}
	}
}

// WithTagsForAllMachines add provided tags to all machines.
func WithTagsForAllMachines(value []string) VSphereFiller {
	return func(config VSphereConfig) {
		for _, m := range config.machineConfigs {
			m.Spec.TagIDs = value
		}
	}
}

func WithNumCPUsForAllMachines(value int) VSphereFiller {
	return func(config VSphereConfig) {
		for _, m := range config.machineConfigs {
			m.Spec.NumCPUs = value
		}
	}
}

func WithDiskGiBForAllMachines(value int) VSphereFiller {
	return func(config VSphereConfig) {
		for _, m := range config.machineConfigs {
			m.Spec.DiskGiB = value
		}
	}
}

func WithMemoryMiBForAllMachines(value int) VSphereFiller {
	return func(config VSphereConfig) {
		for _, m := range config.machineConfigs {
			m.Spec.MemoryMiB = value
		}
	}
}

func WithTLSInsecure(value bool) VSphereFiller {
	return func(config VSphereConfig) {
		config.datacenterConfig.Spec.Insecure = value
	}
}

func WithTLSThumbprint(value string) VSphereFiller {
	return func(config VSphereConfig) {
		config.datacenterConfig.Spec.Thumbprint = value
	}
}

func WithTemplateForAllMachines(value string) VSphereFiller {
	return func(config VSphereConfig) {
		for _, m := range config.machineConfigs {
			m.Spec.Template = value
		}
	}
}

func WithStoragePolicyNameForAllMachines(value string) VSphereFiller {
	return func(config VSphereConfig) {
		for _, m := range config.machineConfigs {
			m.Spec.StoragePolicyName = value
		}
	}
}

func WithVSphereConfigNamespaceForAllMachinesAndDatacenter(ns string) VSphereFiller {
	return func(config VSphereConfig) {
		config.datacenterConfig.Namespace = ns
		for _, m := range config.machineConfigs {
			m.Namespace = ns
		}
	}
}

func WithSSHAuthorizedKeyForAllMachines(key string) VSphereFiller {
	return func(config VSphereConfig) {
		for _, m := range config.machineConfigs {
			setSSHKeyForFirstUser(m, key)
		}
	}
}

func WithServer(value string) VSphereFiller {
	return func(config VSphereConfig) {
		config.datacenterConfig.Spec.Server = value
	}
}

func WithResourcePoolForAllMachines(value string) VSphereFiller {
	return func(config VSphereConfig) {
		for _, m := range config.machineConfigs {
			m.Spec.ResourcePool = value
		}
	}
}

func WithNetwork(value string) VSphereFiller {
	return func(config VSphereConfig) {
		config.datacenterConfig.Spec.Network = value
	}
}

func WithFolderForAllMachines(value string) VSphereFiller {
	return func(config VSphereConfig) {
		for _, m := range config.machineConfigs {
			m.Spec.Folder = value
		}
	}
}

func WithDatastoreForAllMachines(value string) VSphereFiller {
	return func(config VSphereConfig) {
		for _, m := range config.machineConfigs {
			m.Spec.Datastore = value
		}
	}
}

func WithDatacenter(value string) VSphereFiller {
	return func(config VSphereConfig) {
		config.datacenterConfig.Spec.Datacenter = value
	}
}

// WithDisableCSI sets the value for DisableCSI in VSphereDatacenterConfig.
func WithDisableCSI(value bool) VSphereFiller {
	return func(config VSphereConfig) {
		config.datacenterConfig.Spec.DisableCSI = value
	}
}

func WithVSphereStringFromEnvVar(envVar string, opt func(string) VSphereFiller) VSphereFiller {
	return opt(os.Getenv(envVar))
}

func WithVSphereBoolFromEnvVar(envVar string, opt func(bool) VSphereFiller) VSphereFiller {
	return opt(os.Getenv(envVar) == "true")
}

func WithVSphereMachineConfig(name string, fillers ...VSphereMachineConfigFiller) VSphereFiller {
	return func(config VSphereConfig) {
		m, ok := config.machineConfigs[name]
		if !ok {
			m = &anywherev1.VSphereMachineConfig{
				TypeMeta: metav1.TypeMeta{
					Kind:       anywherev1.VSphereMachineConfigKind,
					APIVersion: anywherev1.SchemeBuilder.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			}
			config.machineConfigs[name] = m
		}

		FillVSphereMachineConfig(m, fillers...)
	}
}
