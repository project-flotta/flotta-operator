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

// EdgeDeviceGroupSpec defines the desired state of EdgeDeviceGroup
type EdgeDeviceGroupSpec struct {
	// Heartbeat contains heartbeat messages configuration
	Heartbeat *HeartbeatConfiguration `json:"heartbeat,omitempty"`
	// Storage contains data upload configuration
	Storage *Storage `json:"storage,omitempty"`
	// Metrics contain metric collection and upload configuration
	Metrics *MetricsConfiguration `json:"metrics,omitempty"`
	// LogCollection contains configuration for device log collection
	LogCollection map[string]*LogCollectionConfig `json:"logCollection,omitempty"`
	// OsInformation carries information about commit ID of the OS Image deployed to the device
	OsInformation *OsInformation `json:"osInformation,omitempty"`
}

// EdgeDeviceGroupStatus defines the observed state of EdgeDeviceGroup
type EdgeDeviceGroupStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EdgeDeviceGroup is the Schema for the edgedevicegroups API
type EdgeDeviceGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeDeviceGroupSpec   `json:"spec,omitempty"`
	Status EdgeDeviceGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EdgeDeviceGroupList contains a list of EdgeDeviceGroup
type EdgeDeviceGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeDeviceGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeDeviceGroup{}, &EdgeDeviceGroupList{})
}
