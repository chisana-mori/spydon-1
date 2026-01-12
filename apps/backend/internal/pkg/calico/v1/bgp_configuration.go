package v1

import (
	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BGPConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              calicov3.BGPConfigurationSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BGPConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BGPConfiguration `json:"items"`
}

// Manual DeepCopy implementations

func (in *BGPConfiguration) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *BGPConfiguration) DeepCopy() *BGPConfiguration {
	if in == nil {
		return nil
	}
	out := new(BGPConfiguration)
	in.DeepCopyInto(out)
	return out
}
func (in *BGPConfiguration) DeepCopyInto(out *BGPConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}
func (in *BGPConfigurationList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *BGPConfigurationList) DeepCopy() *BGPConfigurationList {
	if in == nil {
		return nil
	}
	out := new(BGPConfigurationList)
	in.DeepCopyInto(out)
	return out
}
func (in *BGPConfigurationList) DeepCopyInto(out *BGPConfigurationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BGPConfiguration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
