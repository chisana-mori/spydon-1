package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NodeSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Node `json:"items"`
}

type NodeSpec struct {
	BGP      *NodeBGPSpec `json:"bgp,omitempty"`
	OrchRefs []OrchRef    `json:"orchRefs,omitempty"`
}

type NodeBGPSpec struct {
	ASNumber    *int   `json:"asNumber,omitempty"`
	IPv4Address string `json:"ipv4Address,omitempty"`
	IPv6Address string `json:"ipv6Address,omitempty"`
}

type OrchRef struct {
	NodeName     string `json:"nodeName"`
	Orchestrator string `json:"orchestrator"`
}

// Manual DeepCopy implementations

func (in *Node) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *Node) DeepCopy() *Node {
	if in == nil {
		return nil
	}
	out := new(Node)
	in.DeepCopyInto(out)
	return out
}
func (in *Node) DeepCopyInto(out *Node) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}
func (in *NodeSpec) DeepCopyInto(out *NodeSpec) {
	*out = *in
	if in.BGP != nil {
		in, out := &in.BGP, &out.BGP
		*out = new(NodeBGPSpec)
		**out = **in
		if (*in).ASNumber != nil {
			in, out := &(*in).ASNumber, &(*out).ASNumber
			*out = new(int)
			**out = **in
		}
	}
	if in.OrchRefs != nil {
		in, out := &in.OrchRefs, &out.OrchRefs
		*out = make([]OrchRef, len(*in))
		copy(*out, *in)
	}
}

func (in *NodeList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *NodeList) DeepCopy() *NodeList {
	if in == nil {
		return nil
	}
	out := new(NodeList)
	in.DeepCopyInto(out)
	return out
}
func (in *NodeList) DeepCopyInto(out *NodeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Node, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
