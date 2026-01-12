package v1

import (
	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterInformation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              calicov3.ClusterInformationSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterInformationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterInformation `json:"items"`
}

// Manual DeepCopy implementations

func (in *ClusterInformation) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *ClusterInformation) DeepCopy() *ClusterInformation {
	if in == nil {
		return nil
	}
	out := new(ClusterInformation)
	in.DeepCopyInto(out)
	return out
}
func (in *ClusterInformation) DeepCopyInto(out *ClusterInformation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}
func (in *ClusterInformationList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *ClusterInformationList) DeepCopy() *ClusterInformationList {
	if in == nil {
		return nil
	}
	out := new(ClusterInformationList)
	in.DeepCopyInto(out)
	return out
}
func (in *ClusterInformationList) DeepCopyInto(out *ClusterInformationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterInformation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
