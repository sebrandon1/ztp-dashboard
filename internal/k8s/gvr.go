package k8s

import "k8s.io/apimachinery/pkg/runtime/schema"

var (
	GVRManagedCluster = schema.GroupVersionResource{
		Group: "cluster.open-cluster-management.io", Version: "v1", Resource: "managedclusters",
	}
	GVRClusterDeployment = schema.GroupVersionResource{
		Group: "hive.openshift.io", Version: "v1", Resource: "clusterdeployments",
	}
	GVRClusterInstance = schema.GroupVersionResource{
		Group: "siteconfig.open-cluster-management.io", Version: "v1alpha1", Resource: "clusterinstances",
	}
	GVRInfraEnv = schema.GroupVersionResource{
		Group: "agent-install.openshift.io", Version: "v1beta1", Resource: "infraenvs",
	}
	GVRBareMetalHost = schema.GroupVersionResource{
		Group: "metal3.io", Version: "v1alpha1", Resource: "baremetalhosts",
	}
	GVRAgentClusterInstall = schema.GroupVersionResource{
		Group: "extensions.hive.openshift.io", Version: "v1beta1", Resource: "agentclusterinstalls",
	}
	GVRAgent = schema.GroupVersionResource{
		Group: "agent-install.openshift.io", Version: "v1beta1", Resource: "agents",
	}
	GVRPolicy = schema.GroupVersionResource{
		Group: "policy.open-cluster-management.io", Version: "v1", Resource: "policies",
	}
	GVRArgoApplication = schema.GroupVersionResource{
		Group: "argoproj.io", Version: "v1alpha1", Resource: "applications",
	}
)
