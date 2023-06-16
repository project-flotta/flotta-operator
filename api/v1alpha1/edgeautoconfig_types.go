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

const (
	EdgeDeviceStatePending EdgeDeviceState = "pending"
	EdgeDeviceStateRunning EdgeDeviceState = "running"
)

type EdgeDeviceState string

type EdgeDeviceProperties struct {
	// ModelName is the model name from the OS.
	// The output of:
	// cat /sys/firmware/devicetree/base/model
	OsModelName string `json:"osmodelname,omitempty"`

	// Hardware defines the hardware that devices has
	Hardware *PrefHardware `json:"hardware,omitempty"`
}

type PrefHardware struct {

	// boot
	Boot *Boot `json:"boot,omitempty"`

	// cpu
	CPU *PrefCPU `json:"cpu,omitempty"`

	// disks
	Disks []*Disk `json:"disks,omitempty"`

	// gpus
	Gpus []*Gpu `json:"gpus,omitempty"`

	// hostname
	Hostname string `json:"hostname,omitempty"`

	// interfaces
	Interfaces []*Interface `json:"interfaces,omitempty"`

	// memory
	Memory *Memory `json:"memory,omitempty"`

	// system vendor
	SystemVendor *SystemVendor `json:"systemVendor,omitempty"`

	// list of devices present on the edgedevice
	HostDevices []*HostDevice `json:"hostDevices,omitempty"`

	// list of all mounts found on edgedevice
	Mounts []*Mount `json:"mounts,omitempty"`
}
type PrefCPU struct {

	// architecture
	Architecture string `json:"architecture,omitempty"`

	// count
	Count int64 `json:"count,omitempty"`

	// flags
	Flags []string `json:"flags,omitempty"`

	// frequency
	Frequency string `json:"frequency,omitempty"`

	// model name
	ModelName string `json:"modelName,omitempty"`
}
type EdgeDeviceWorkloads struct {
	Containers []Containers `json:"containers"`
}

type Containers struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EdgeAutoConfigSpec defines the desired state of EdgeAutoConfig
type EdgeAutoConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of EdgeAutoConfig. Edit edgeautoconfig_types.go to remove/update
	EdgeDeviceProperties *EdgeDeviceProperties `json:"edgedeviceproperties,omitempty"`
	EdgeDeviceWorkloads  *EdgeDeviceWorkloads  `json:"edgedeviceworkloads,omitempty"`
}

// EdgeAutoConfigStatus defines the observed state of EdgeAutoConfig
type EdgeAutoConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	EdgeDevices []EdgeDevices `json:"edgedevices,omitempty"`
}

type EdgeDevices struct {
	Name            string          `json:"name"`
	EdgeDeviceState EdgeDeviceState `json:"edgedevicestate"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EdgeAutoConfig is the Schema for the edgeautoconfigs API
type EdgeAutoConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeAutoConfigSpec   `json:"spec,omitempty"`
	Status EdgeAutoConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EdgeAutoConfigList contains a list of EdgeAutoConfig
type EdgeAutoConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeAutoConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeAutoConfig{}, &EdgeAutoConfigList{})
}
