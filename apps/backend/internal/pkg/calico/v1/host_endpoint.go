package v1

import (
	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type HostEndpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              calicov3.HostEndpointSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type HostEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HostEndpoint `json:"items"`
}

// Manual DeepCopy implementations

func (in *HostEndpoint) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *HostEndpoint) DeepCopy() *HostEndpoint {
	if in == nil {
		return nil
	}
	out := new(HostEndpoint)
	in.DeepCopyInto(out)
	return out
}
func (in *HostEndpoint) DeepCopyInto(out *HostEndpoint) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}
func (in *HostEndpointList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *HostEndpointList) DeepCopy() *HostEndpointList {
	if in == nil {
		return nil
	}
	out := new(HostEndpointList)
	in.DeepCopyInto(out)
	return out
}
func (in *HostEndpointList) DeepCopyInto(out *HostEndpointList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]HostEndpoint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
