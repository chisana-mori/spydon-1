package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type V3Node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              V3NodeSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type V3NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []V3Node `json:"items"`
}

type V3NodeSpec struct {
	BGP      *V3NodeBGPSpec `json:"bgp,omitempty"`
	OrchRefs []V3OrchRef    `json:"orchRefs,omitempty"`
}

type V3NodeBGPSpec struct {
	ASNumber    *int   `json:"asNumber,omitempty"`
	IPv4Address string `json:"ipv4Address,omitempty"`
	IPv6Address string `json:"ipv6Address,omitempty"`
}

type V3OrchRef struct {
	NodeName     string `json:"nodeName"`
	Orchestrator string `json:"orchestrator"`
}

// Manual DeepCopy

func (in *V3Node) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *V3Node) DeepCopy() *V3Node {
	if in == nil {
		return nil
	}
	out := new(V3Node)
	in.DeepCopyInto(out)
	return out
}
func (in *V3Node) DeepCopyInto(out *V3Node) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

func (in *V3NodeList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *V3NodeList) DeepCopy() *V3NodeList {
	if in == nil {
		return nil
	}
	out := new(V3NodeList)
	in.DeepCopyInto(out)
	return out
}
func (in *V3NodeList) DeepCopyInto(out *V3NodeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]V3Node, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *V3NodeSpec) DeepCopyInto(out *V3NodeSpec) {
	*out = *in
	if in.BGP != nil {
		in, out := &in.BGP, &out.BGP
		*out = new(V3NodeBGPSpec)
		**out = **in
		if (*in).ASNumber != nil {
			in, out := &(*in).ASNumber, &(*out).ASNumber
			*out = new(int)
			**out = **in
		}
	}
	if in.OrchRefs != nil {
		in, out := &in.OrchRefs, &out.OrchRefs
		*out = make([]V3OrchRef, len(*in))
		copy(*out, *in)
	}
}
