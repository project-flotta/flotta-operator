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
	HardwareProfile *HardwareProfileConfiguration `json:"hardwareProfile,omitempty"`

	// period seconds
	PeriodSeconds int64 `json:"periodSeconds,omitempty"`
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
	Hardware                  *Hardware   `json:"hardware,omitempty"`
}

type Hardware struct {

	// boot
	Boot *Boot `json:"boot,omitempty"`

	// cpu
	CPU *CPU `json:"cpu,omitempty"`

	// disks
	Disks []*Disk `json:"disks"`

	// gpus
	Gpus []*Gpu `json:"gpus"`

	// hostname
	Hostname string `json:"hostname,omitempty"`

	// interfaces
	Interfaces []*Interface `json:"interfaces"`

	// memory
	Memory *Memory `json:"memory,omitempty"`

	// system vendor
	SystemVendor *SystemVendor `json:"system_vendor,omitempty"`
}

type Boot struct {

	// current boot mode
	CurrentBootMode string `json:"current_boot_mode,omitempty"`

	// pxe interface
	PxeInterface string `json:"pxe_interface,omitempty"`
}

type ClockMhz float64

type CPU struct {

	// architecture
	Architecture string `json:"architecture,omitempty"`

	// count
	Count int64 `json:"count,omitempty"`

	// flags
	Flags []string `json:"flags"`

	// frequency
	Frequency string `json:"frequency,omitempty"`

	// model name
	ModelName string `json:"model_name,omitempty"`
}

type Disk struct {

	// bootable
	Bootable bool `json:"bootable,omitempty"`

	// by-id is the World Wide Number of the device which guaranteed to be unique for every storage device
	ByID string `json:"by_id,omitempty"`

	// by-path is the shortest physical path to the device
	ByPath string `json:"by_path,omitempty"`

	// drive type
	DriveType string `json:"drive_type,omitempty"`

	// hctl
	Hctl string `json:"hctl,omitempty"`

	// Determine the disk's unique identifier which is the by-id field if it exists and fallback to the by-path field otherwise
	ID string `json:"id,omitempty"`

	// io perf
	IoPerf *IoPerf `json:"io_perf,omitempty"`

	// Whether the disk appears to be an installation media or not
	IsInstallationMedia bool `json:"is_installation_media,omitempty"`

	// model
	Model string `json:"model,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// path
	Path string `json:"path,omitempty"`

	// serial
	Serial string `json:"serial,omitempty"`

	// size bytes
	SizeBytes int64 `json:"size_bytes,omitempty"`

	// smart
	Smart string `json:"smart,omitempty"`

	// vendor
	Vendor string `json:"vendor,omitempty"`

	// wwn
	Wwn string `json:"wwn,omitempty"`
}

type Gpu struct {

	// Device address (for example "0000:00:02.0")
	Address string `json:"address,omitempty"`

	// ID of the device (for example "3ea0")
	DeviceID string `json:"device_id,omitempty"`

	// Product name of the device (for example "UHD Graphics 620 (Whiskey Lake)")
	Name string `json:"name,omitempty"`

	// The name of the device vendor (for example "Intel Corporation")
	Vendor string `json:"vendor,omitempty"`

	// ID of the vendor (for example "8086")
	VendorID string `json:"vendor_id,omitempty"`
}

type IoPerf struct {

	// 99th percentile of fsync duration in milliseconds
	SyncDuration int64 `json:"sync_duration,omitempty"`
}

type Interface struct {

	// biosdevname
	Biosdevname string `json:"biosdevname,omitempty"`

	// client id
	ClientID string `json:"client_id,omitempty"`

	// flags
	Flags []string `json:"flags"`

	// has carrier
	HasCarrier bool `json:"has_carrier,omitempty"`

	// ipv4 addresses
	IPV4Addresses []string `json:"ipv4_addresses"`

	// ipv6 addresses
	IPV6Addresses []string `json:"ipv6_addresses"`

	// mac address
	MacAddress string `json:"mac_address,omitempty"`

	// mtu
	Mtu int64 `json:"mtu,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// product
	Product string `json:"product,omitempty"`

	// speed mbps
	SpeedMbps int64 `json:"speed_mbps,omitempty"`

	// vendor
	Vendor string `json:"vendor,omitempty"`
}

type Memory struct {

	// physical bytes
	PhysicalBytes int64 `json:"physical_bytes,omitempty"`

	// usable bytes
	UsableBytes int64 `json:"usable_bytes,omitempty"`
}

type Route struct {

	// The destination network or destination host
	Destination string `json:"destination,omitempty"`

	// Defines whether this is an IPv4 (4) or IPv6 route (6)
	Family int32 `json:"family,omitempty"`

	// Gateway address where the packets are sent
	Gateway string `json:"gateway,omitempty"`

	// Interface to which packets for this route will be sent
	Interface string `json:"interface,omitempty"`
}

type SystemVendor struct {

	// manufacturer
	Manufacturer string `json:"manufacturer,omitempty"`

	// product name
	ProductName string `json:"product_name,omitempty"`

	// serial number
	SerialNumber string `json:"serial_number,omitempty"`

	// Whether the machine appears to be a virtual machine or not
	Virtual bool `json:"virtual,omitempty"`
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
