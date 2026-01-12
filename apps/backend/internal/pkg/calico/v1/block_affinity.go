package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BlockAffinity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BlockAffinitySpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BlockAffinityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BlockAffinity `json:"items"`
}

type BlockAffinitySpec struct {
	State string `json:"state"`
	Node  string `json:"node"`
	CIDR  string `json:"cidr"`
}

// Manual DeepCopy implementations

func (in *BlockAffinity) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *BlockAffinity) DeepCopy() *BlockAffinity {
	if in == nil {
		return nil
	}
	out := new(BlockAffinity)
	in.DeepCopyInto(out)
	return out
}
func (in *BlockAffinity) DeepCopyInto(out *BlockAffinity) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

func (in *BlockAffinityList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *BlockAffinityList) DeepCopy() *BlockAffinityList {
	if in == nil {
		return nil
	}
	out := new(BlockAffinityList)
	in.DeepCopyInto(out)
	return out
}
func (in *BlockAffinityList) DeepCopyInto(out *BlockAffinityList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BlockAffinity, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
