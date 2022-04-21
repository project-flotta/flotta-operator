/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EdgeDeviceSignedRequestLabelName  = "edgedeviceSignedRequest"
	EdgeDeviceSignedRequestLabelValue = "true"

	EdgeDeviceSignedRequestStatusApproved EdgeDeviceSignedRequestStatusType = "approved"
	EdgeDeviceSignedRequestStatusPending  EdgeDeviceSignedRequestStatusType = "pending"
	EdgeDeviceSignedRequestStatusDeclined EdgeDeviceSignedRequestStatusType = "declined"
)

type EdgeDeviceSignedRequestSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// TargetNamespace is the namespace where device will land
	TargetNamespace string `json:"targetNamespace"`

	// TargetSet is set that edgedevice will use.
	TargetSet string `json:"targetSet,omitempty"`

	// Approved is set to true if the device should be approved to register
	// +kubebuilder:default=false
	Approved bool `json:"approved,omitempty"`

	// Features lists features that the registering edge device has
	Features *Features `json:"features,omitempty"`
}

type Features struct {
	// ModelName is the model name from the OS.
	// The output of:
	// cat /sys/firmware/devicetree/base/model
	ModelName string `json:"modelName,omitempty"`

	// Hardware defines the hardware that devices has
	Hardware *Hardware `json:"hardware,omitempty"`
}

type EdgeDeviceSignedRequestStatusType string

type EdgeDeviceSignedRequestCondition struct {
	// Type of status
	// +kubebuilder:validation:Enum=declined;approved;pending
	// +kubebuilder:default=pending
	Type EdgeDeviceSignedRequestStatusType `json:"type"`

	// Status of the condition, one of True, False, Unknown
	Status metav1.ConditionStatus `json:"status"`

	// A human-readable message indicating details about last transition
	// +kubebuilder:optional
	Message *string `json:"message,omitempty"`

	// The last time the condition transit from one status to another
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
}

type EdgeDeviceSignedRequestStatus struct {
	Conditions []EdgeDeviceSignedRequestCondition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EdgeDeviceSignedRequest is the Schema for the edgedevice enrolment options
// +kubebuilder:resource:singular="edgedevicesignedrequest",path="edgedevicesignedrequest",scope="Namespaced",shortName={edsr}
// +kubebuilder:printcolumn:JSONPath=".metadata.name",description="DeviceID",name="deviceid",type=string
// +kubebuilder:printcolumn:JSONPath=".spec.targetNamespace",description="Target Namespace to land",name="targetNamespace",type=string
// +kubebuilder:printcolumn:JSONPath=".spec.approved",description="Approved",name="Approved",type=string
// +kubebuilder:printcolumn:JSONPath=".status.phase",description="Status",name="Status",type=string
type EdgeDeviceSignedRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeDeviceSignedRequestSpec   `json:"spec,omitempty"`
	Status EdgeDeviceSignedRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
// EdgeDeviceSignedRequestList contains a list of EdgeDeviceSignedRequest
type EdgeDeviceSignedRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeDeviceSignedRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeDeviceSignedRequest{}, &EdgeDeviceSignedRequestList{})
}
