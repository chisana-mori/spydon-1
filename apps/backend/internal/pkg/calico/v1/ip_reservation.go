package v1

import (
	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type IPReservation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              calicov3.IPReservationSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type IPReservationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPReservation `json:"items"`
}

// Manual DeepCopy implementations

func (in *IPReservation) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *IPReservation) DeepCopy() *IPReservation {
	if in == nil {
		return nil
	}
	out := new(IPReservation)
	in.DeepCopyInto(out)
	return out
}
func (in *IPReservation) DeepCopyInto(out *IPReservation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

func (in *IPReservationList) DeepCopyObject() runtime.Object { return in.DeepCopy() }
func (in *IPReservationList) DeepCopy() *IPReservationList {
	if in == nil {
		return nil
	}
	out := new(IPReservationList)
	in.DeepCopyInto(out)
	return out
}
func (in *IPReservationList) DeepCopyInto(out *IPReservationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IPReservation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
