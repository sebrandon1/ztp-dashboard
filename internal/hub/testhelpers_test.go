package hub

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/sebrandon1/ztp-dashboard/internal/k8s"
)

// testGVRKinds maps every watched GVR to its list kind. The fake dynamic client
// panics on List unless each GVR has a registered list kind.
var testGVRKinds = map[schema.GroupVersionResource]string{
	k8s.GVRManagedCluster:      "ManagedClusterList",
	k8s.GVRClusterDeployment:   "ClusterDeploymentList",
	k8s.GVRClusterInstance:     "ClusterInstanceList",
	k8s.GVRInfraEnv:            "InfraEnvList",
	k8s.GVRBareMetalHost:       "BareMetalHostList",
	k8s.GVRAgentClusterInstall: "AgentClusterInstallList",
	k8s.GVRAgent:               "AgentList",
	k8s.GVRPolicy:              "PolicyList",
	k8s.GVRArgoApplication:     "ApplicationList",
}

// newFakeClient builds a k8s.Client backed by fake typed and dynamic clients.
func newFakeClient(objs ...runtime.Object) *k8s.Client {
	scheme := runtime.NewScheme()
	for gvr, listKind := range testGVRKinds {
		kind := strings.TrimSuffix(listKind, "List")
		gvk := gvr.GroupVersion().WithKind(kind)
		scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
		scheme.AddKnownTypeWithName(gvr.GroupVersion().WithKind(listKind), &unstructured.UnstructuredList{})
	}

	return &k8s.Client{
		Clientset:     fake.NewClientset(),
		Dynamic:       dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, testGVRKinds, objs...),
		ServerVersion: "v1.30.0",
	}
}

func newFakeManager(objs ...runtime.Object) *Manager {
	return NewManager(newFakeClient(objs...))
}

// unstructuredObj builds an unstructured object with the GVR's apiVersion,
// kind and metadata.
func unstructuredObj(gvr schema.GroupVersionResource, kind, namespace, name string) *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": gvr.GroupVersion().String(),
		"kind":       kind,
		"metadata": map[string]interface{}{
			"name": name,
		},
	}
	if namespace != "" {
		obj["metadata"].(map[string]interface{})["namespace"] = namespace
	}
	return &unstructured.Unstructured{Object: obj}
}

func managedCluster(name string, conditions []interface{}, labels map[string]string) *unstructured.Unstructured {
	obj := unstructuredObj(k8s.GVRManagedCluster, "ManagedCluster", "", name)
	if conditions != nil {
		_ = unstructured.SetNestedSlice(obj.Object, conditions, "status", "conditions")
	}
	if labels != nil {
		obj.SetLabels(labels)
	}
	return obj
}

func condition(condType, status, reason string) map[string]interface{} {
	c := map[string]interface{}{"type": condType, "status": status}
	if reason != "" {
		c["reason"] = reason
	}
	return c
}

func namespacedResource(gvr schema.GroupVersionResource, kind, namespace, name string, conditions []interface{}, extra map[string]interface{}) *unstructured.Unstructured {
	obj := unstructuredObj(gvr, kind, namespace, name)
	if conditions != nil {
		_ = unstructured.SetNestedSlice(obj.Object, conditions, "status", "conditions")
	}
	for path, value := range extra {
		parts := strings.Split(path, ".")
		_ = unstructured.SetNestedField(obj.Object, value, parts...)
	}
	return obj
}
