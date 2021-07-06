/*
Copyright 2021.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EdgeDeviceSpec defines the desired state of EdgeDevice
type EdgeDeviceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// OsImageId carries information about ID of the OS Image deployed to the device
	OsImageId string `json:"osImageId,omitempty"`

	// RequestTime is the time of device registration request
	RequestTime *metav1.Time `json:"requestTime,omitempty"`

	Heartbeat *HeartbeatConfiguration `json:"heartbeat,omitempty"`
}

type DeviceConfiguration struct {

	// heartbeat
	Heartbeat *HeartbeatConfiguration `json:"heartbeat,omitempty"`
}

type HeartbeatConfiguration struct {

	// hardware profile
	HardwareProfile *HardwareProfileConfiguration `json:"hardware_profile,omitempty"`

	// period seconds
	PeriodSeconds int64 `json:"period_seconds,omitempty"`
}

type HardwareProfileConfiguration struct {

	// include
	Include bool `json:"include,omitempty"`

	// scope
	// Enum: [full delta]
	Scope string `json:"scope,omitempty"`
}

// EdgeDeviceStatus defines the observed state of EdgeDevice
type EdgeDeviceStatus struct {
	Phase                     string      `json:"phase,omitempty"`
	LastSeenTime              metav1.Time `json:"lastSeenTime,omitempty"`
	LastSyncedResourceVersion string      `json:"lastSyncedResourceVersion,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EdgeDevice is the Schema for the edgedevices API
type EdgeDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeDeviceSpec   `json:"spec,omitempty"`
	Status EdgeDeviceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EdgeDeviceList contains a list of EdgeDevice
type EdgeDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeDevice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeDevice{}, &EdgeDeviceList{})
}
