package v1

import (
	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GlobalNetworkPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              calicov3.GlobalNetworkPolicySpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GlobalNetworkPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GlobalNetworkPolicy `json:"items"`
}

// Manual DeepCopy implementations

func (in *GlobalNetworkPolicy) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *GlobalNetworkPolicy) DeepCopy() *GlobalNetworkPolicy {
	if in == nil {
		return nil
	}
	out := new(GlobalNetworkPolicy)
	in.DeepCopyInto(out)
	return out
}
func (in *GlobalNetworkPolicy) DeepCopyInto(out *GlobalNetworkPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}
func (in *GlobalNetworkPolicyList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *GlobalNetworkPolicyList) DeepCopy() *GlobalNetworkPolicyList {
	if in == nil {
		return nil
	}
	out := new(GlobalNetworkPolicyList)
	in.DeepCopyInto(out)
	return out
}
func (in *GlobalNetworkPolicyList) DeepCopyInto(out *GlobalNetworkPolicyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]GlobalNetworkPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
