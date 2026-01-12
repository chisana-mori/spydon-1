package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type IPAMBlock struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              IPAMBlockSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type IPAMBlockList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPAMBlock `json:"items"`
}

type IPAMBlockSpec struct {
	CIDR           string               `json:"cidr"`
	Affinity       string               `json:"affinity"`
	Allocations    []*int               `json:"allocations"`
	Unallocated    []int                `json:"unallocated"`
	Attributes     []IPAMBlockAttribute `json:"attributes"`
	StrictAffinity bool                 `json:"strictAffinity"`
	Deleted        bool                 `json:"deleted"`
}

type IPAMBlockAttribute struct {
	HandleID  string            `json:"handle_id"`
	Secondary map[string]string `json:"secondary"`
}

// Manual DeepCopy implementations

func (in *IPAMBlock) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *IPAMBlock) DeepCopy() *IPAMBlock {
	if in == nil {
		return nil
	}
	out := new(IPAMBlock)
	in.DeepCopyInto(out)
	return out
}
func (in *IPAMBlock) DeepCopyInto(out *IPAMBlock) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}
func (in *IPAMBlockSpec) DeepCopyInto(out *IPAMBlockSpec) {
	*out = *in
	out.CIDR = in.CIDR
	out.Affinity = in.Affinity
	if in.Allocations != nil {
		in, out := &in.Allocations, &out.Allocations
		*out = make([]*int, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(int)
				**out = **in
			}
		}
	}
	if in.Unallocated != nil {
		in, out := &in.Unallocated, &out.Unallocated
		*out = make([]int, len(*in))
		copy(*out, *in)
	}
	if in.Attributes != nil {
		in, out := &in.Attributes, &out.Attributes
		*out = make([]IPAMBlockAttribute, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.StrictAffinity = in.StrictAffinity
	out.Deleted = in.Deleted
}

func (in *IPAMBlockAttribute) DeepCopyInto(out *IPAMBlockAttribute) {
	*out = *in
	if in.Secondary != nil {
		in, out := &in.Secondary, &out.Secondary
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}
func (in *IPAMBlockList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *IPAMBlockList) DeepCopy() *IPAMBlockList {
	if in == nil {
		return nil
	}
	out := new(IPAMBlockList)
	in.DeepCopyInto(out)
	return out
}
func (in *IPAMBlockList) DeepCopyInto(out *IPAMBlockList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IPAMBlock, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
