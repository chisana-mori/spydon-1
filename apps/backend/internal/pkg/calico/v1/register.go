package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName is the group name use in this package
const GroupName = "crd.projectcalico.org"
const GroupVersion = "v1"

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&IPPool{},
		&IPPoolList{},
		&IPReservation{},
		&IPReservationList{},
		&BGPConfiguration{},
		&BGPConfigurationList{},
		&BGPPeer{},
		&BGPPeerList{},
		&GlobalNetworkPolicy{},
		&GlobalNetworkPolicyList{},
		&NetworkPolicy{},
		&NetworkPolicyList{},
		&GlobalNetworkSet{},
		&GlobalNetworkSetList{},
		&NetworkSet{},
		&NetworkSetList{},
		&HostEndpoint{},
		&HostEndpointList{},
		&FelixConfiguration{},
		&FelixConfigurationList{},
		&ClusterInformation{},
		&ClusterInformationList{},
		&KubeControllersConfiguration{},
		&KubeControllersConfigurationList{},
		&Tier{},
		&TierList{},
		&CalicoNodeStatus{},
		&CalicoNodeStatusList{},
		&Node{},
		&NodeList{},
		&BlockAffinity{},
		&BlockAffinityList{},
		&IPAMBlock{},
		&IPAMBlockList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
