package v1

import (
	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CalicoNodeStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              calicov3.CalicoNodeStatusSpec   `json:"spec,omitempty"`
	Status            calicov3.CalicoNodeStatusStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CalicoNodeStatusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CalicoNodeStatus `json:"items"`
}

// Manual DeepCopy implementations

func (in *CalicoNodeStatus) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *CalicoNodeStatus) DeepCopy() *CalicoNodeStatus {
	if in == nil {
		return nil
	}
	out := new(CalicoNodeStatus)
	in.DeepCopyInto(out)
	return out
}
func (in *CalicoNodeStatus) DeepCopyInto(out *CalicoNodeStatus) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}
func (in *CalicoNodeStatusList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *CalicoNodeStatusList) DeepCopy() *CalicoNodeStatusList {
	if in == nil {
		return nil
	}
	out := new(CalicoNodeStatusList)
	in.DeepCopyInto(out)
	return out
}
func (in *CalicoNodeStatusList) DeepCopyInto(out *CalicoNodeStatusList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CalicoNodeStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
