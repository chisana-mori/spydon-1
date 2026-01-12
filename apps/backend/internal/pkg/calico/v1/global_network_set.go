package v1

import (
	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GlobalNetworkSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              calicov3.GlobalNetworkSetSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GlobalNetworkSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GlobalNetworkSet `json:"items"`
}

// Manual DeepCopy implementations

func (in *GlobalNetworkSet) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *GlobalNetworkSet) DeepCopy() *GlobalNetworkSet {
	if in == nil {
		return nil
	}
	out := new(GlobalNetworkSet)
	in.DeepCopyInto(out)
	return out
}
func (in *GlobalNetworkSet) DeepCopyInto(out *GlobalNetworkSet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}
func (in *GlobalNetworkSetList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *GlobalNetworkSetList) DeepCopy() *GlobalNetworkSetList {
	if in == nil {
		return nil
	}
	out := new(GlobalNetworkSetList)
	in.DeepCopyInto(out)
	return out
}
func (in *GlobalNetworkSetList) DeepCopyInto(out *GlobalNetworkSetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]GlobalNetworkSet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
