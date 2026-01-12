package v1

import (
	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NetworkSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              calicov3.NetworkSetSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NetworkSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkSet `json:"items"`
}

// Manual DeepCopy implementations

func (in *NetworkSet) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *NetworkSet) DeepCopy() *NetworkSet {
	if in == nil {
		return nil
	}
	out := new(NetworkSet)
	in.DeepCopyInto(out)
	return out
}
func (in *NetworkSet) DeepCopyInto(out *NetworkSet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}
func (in *NetworkSetList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *NetworkSetList) DeepCopy() *NetworkSetList {
	if in == nil {
		return nil
	}
	out := new(NetworkSetList)
	in.DeepCopyInto(out)
	return out
}
func (in *NetworkSetList) DeepCopyInto(out *NetworkSetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NetworkSet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
