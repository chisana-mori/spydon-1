package v1

import (
	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FelixConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              calicov3.FelixConfigurationSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FelixConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FelixConfiguration `json:"items"`
}

// Manual DeepCopy implementations

func (in *FelixConfiguration) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *FelixConfiguration) DeepCopy() *FelixConfiguration {
	if in == nil {
		return nil
	}
	out := new(FelixConfiguration)
	in.DeepCopyInto(out)
	return out
}
func (in *FelixConfiguration) DeepCopyInto(out *FelixConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}
func (in *FelixConfigurationList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *FelixConfigurationList) DeepCopy() *FelixConfigurationList {
	if in == nil {
		return nil
	}
	out := new(FelixConfigurationList)
	in.DeepCopyInto(out)
	return out
}
func (in *FelixConfigurationList) DeepCopyInto(out *FelixConfigurationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]FelixConfiguration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
