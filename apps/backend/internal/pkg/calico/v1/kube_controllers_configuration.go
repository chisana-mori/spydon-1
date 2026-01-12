package v1

import (
	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KubeControllersConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              calicov3.KubeControllersConfigurationSpec   `json:"spec,omitempty"`
	Status            calicov3.KubeControllersConfigurationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KubeControllersConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeControllersConfiguration `json:"items"`
}

// Manual DeepCopy implementations

func (in *KubeControllersConfiguration) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *KubeControllersConfiguration) DeepCopy() *KubeControllersConfiguration {
	if in == nil {
		return nil
	}
	out := new(KubeControllersConfiguration)
	in.DeepCopyInto(out)
	return out
}
func (in *KubeControllersConfiguration) DeepCopyInto(out *KubeControllersConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}
func (in *KubeControllersConfigurationList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *KubeControllersConfigurationList) DeepCopy() *KubeControllersConfigurationList {
	if in == nil {
		return nil
	}
	out := new(KubeControllersConfigurationList)
	in.DeepCopyInto(out)
	return out
}
func (in *KubeControllersConfigurationList) DeepCopyInto(out *KubeControllersConfigurationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KubeControllersConfiguration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
