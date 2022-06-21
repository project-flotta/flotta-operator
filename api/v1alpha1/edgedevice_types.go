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

	// OsInformation carries information about commit ID of the OS Image deployed to the device
	OsInformation *OsInformation `json:"osInformation,omitempty"`

	// RequestTime is the time of device registration request
	RequestTime *metav1.Time `json:"requestTime,omitempty"`

	Heartbeat     *HeartbeatConfiguration         `json:"heartbeat,omitempty"`
	Storage       *Storage                        `json:"storage,omitempty"`
	Metrics       *MetricsConfiguration           `json:"metrics,omitempty"`
	LogCollection map[string]*LogCollectionConfig `json:"logCollection,omitempty"`
	Mounts        []*Mount                        `json:"mounts,omitempty"`
}

type MetricsReceiverConfiguration struct {
	RequestNumSamples int64  `json:"requestNumSamples,omitempty"`
	TimeoutSeconds    int64  `json:"timeoutSeconds,omitempty"`
	URL               string `json:"url,omitempty"`
	CaSecretName      string `json:"caSecretName,omitempty"`
}

type LogCollectionConfig struct {

	// Kind is the type of log collection to be used
	// +kubebuilder:validation:Enum=syslog
	Kind string `json:"kind,omitempty"`

	// +kubebuilder:default=12
	// +kubebuilder:validation:Minimum=1
	BufferSize int32 `json:"bufferSize,omitempty"`

	// SyslogConfig is the pointer to the configMap to be used to load the config
	SyslogConfig *NameRef `json:"syslogConfig,omitempty"`
}

type MetricsConfiguration struct {
	Retention             *Retention                     `json:"retention,omitempty"`
	SystemMetrics         *ComponentMetricsConfiguration `json:"system,omitempty"`
	DataTransferMetrics   *ComponentMetricsConfiguration `json:"dataTransfer,omitempty"`
	ReceiverConfiguration *MetricsReceiverConfiguration  `json:"receiverConfiguration,omitempty"`
}

type ComponentMetricsConfiguration struct {
	// Interval(in seconds) to scrape system metrics.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=60
	Interval int32 `json:"interval,omitempty"`

	// AllowList defines name of a ConfigMap containing
	// list of system metrics that should be scraped
	AllowList *NameRef `json:"allowList,omitempty"`

	// Disabled when set to true instructs the device to turn off system metrics collection
	Disabled bool `json:"disabled,omitempty"`
}

type Retention struct {
	// MaxMiB specifies how much disk space should be used for storing persisted metrics on the device
	// +kubebuilder:validation:Minimum=0
	MaxMiB int32 `json:"maxMiB,omitempty"`
	// MaxHours specifies how long should persisted metrics be stored on the device disk
	// +kubebuilder:validation:Minimum=0
	MaxHours int32 `json:"maxHours,omitempty"`
}

type Storage struct {
	S3 *S3Storage `json:"s3,omitempty"`
}

type OsInformation struct {

	//Automatically upgrade the OS image
	AutomaticallyUpgrade bool `json:"automaticallyUpgrade,omitempty"`

	//CommitID carries information about commit of the OS Image
	CommitID string `json:"commitID,omitempty"`

	//HostedObjectsURL carries the URL of the hosted commits web server
	HostedObjectsURL string `json:"hostedObjectsURL,omitempty"`
}

type S3Storage struct {
	// secret name
	SecretName string `json:"secretName,omitempty"`
	// configMap name
	ConfigMapName string `json:"configMapName,omitempty"`
	// createOBC. if the configuration above is empty and this bool is true then create OBC
	CreateOBC bool `json:"createOBC,omitempty"`
}

type HeartbeatConfiguration struct {

	// hardware profile
	HardwareProfile *HardwareProfileConfiguration `json:"hardwareProfile,omitempty"`

	// period seconds
	// +kubebuilder:validation:Minimum=1
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
	Phase                     string              `json:"phase,omitempty"`
	LastSyncedResourceVersion string              `json:"lastSyncedResourceVersion,omitempty"`
	Hardware                  *Hardware           `json:"hardware,omitempty"`
	Workloads                 []Workload          `json:"workloads,omitempty"`
	DataOBC                   *string             `json:"dataObc,omitempty"`
	UpgradeInformation        *UpgradeInformation `json:"upgradeInformation,omitempty"`
}

type EdgeWorkloadPhase string

const (
	Deploying EdgeWorkloadPhase = "Deploying"
	Running   EdgeWorkloadPhase = "Running"
	Exited    EdgeWorkloadPhase = "Exited"
)

type Workload struct {
	Name               string            `json:"name"`
	Phase              EdgeWorkloadPhase `json:"phase,omitempty"`
	LastTransitionTime metav1.Time       `json:"lastTransitionTime,omitempty"`
	LastDataUpload     metav1.Time       `json:"lastDataUpload,omitempty"`
}

type UpgradeInformation struct {
	// Current commit
	CurrentCommitID string `json:"currentCommitID"`
	// last upgrade status
	LastUpgradeStatus string `json:"lastUpgradeStatus,omitempty"`
	// last upgrade time
	LastUpgradeTime string `json:"lastUpgradeTime,omitempty"`
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
	SystemVendor *SystemVendor `json:"systemVendor,omitempty"`

	// list of devices present on the edgedevice
	HostDevices []*HostDevice `json:"hostDevices,omitempty"`

	// list of all mounts found on edgedevice
	Mounts []*Mount `json:"mounts,omitempty"`
}

type Boot struct {

	// current boot mode
	CurrentBootMode string `json:"currentBootMode,omitempty"`

	// pxe interface
	PxeInterface string `json:"pxeInterface,omitempty"`
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
	ModelName string `json:"modelName,omitempty"`
}

type Disk struct {

	// bootable
	Bootable bool `json:"bootable,omitempty"`

	// by-id is the World Wide Number of the device which guaranteed to be unique for every storage device
	ByID string `json:"byId,omitempty"`

	// by-path is the shortest physical path to the device
	ByPath string `json:"byPath,omitempty"`

	// drive type
	DriveType string `json:"driveType,omitempty"`

	// hctl
	Hctl string `json:"hctl,omitempty"`

	// Determine the disk's unique identifier which is the by-id field if it exists and fallback to the by-path field otherwise
	ID string `json:"id,omitempty"`

	// io perf
	IoPerf *IoPerf `json:"ioPerf,omitempty"`

	// Whether the disk appears to be an installation media or not
	IsInstallationMedia bool `json:"isInstallationMedia,omitempty"`

	// model
	Model string `json:"model,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// path
	Path string `json:"path,omitempty"`

	// serial
	Serial string `json:"serial,omitempty"`

	// size bytes
	SizeBytes int64 `json:"sizeBytes,omitempty"`

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
	DeviceID string `json:"deviceId,omitempty"`

	// Product name of the device (for example "UHD Graphics 620 (Whiskey Lake)")
	Name string `json:"name,omitempty"`

	// The name of the device vendor (for example "Intel Corporation")
	Vendor string `json:"vendor,omitempty"`

	// ID of the vendor (for example "8086")
	VendorID string `json:"vendorId,omitempty"`
}

type IoPerf struct {

	// 99th percentile of fsync duration in milliseconds
	SyncDuration int64 `json:"syncDuration,omitempty"`
}

type Interface struct {

	// biosdevname
	Biosdevname string `json:"biosdevname,omitempty"`

	// client id
	ClientID string `json:"clientId,omitempty"`

	// flags
	Flags []string `json:"flags"`

	// has carrier
	HasCarrier bool `json:"hasCarrier,omitempty"`

	// ipv4 addresses
	IPV4Addresses []string `json:"ipv4Addresses,omitempty"`

	// ipv6 addresses
	IPV6Addresses []string `json:"ipv6Addresses,omitempty"`

	// mac address
	MacAddress string `json:"macAddress,omitempty"`

	// mtu
	Mtu int64 `json:"mtu,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// product
	Product string `json:"product,omitempty"`

	// speed mbps
	SpeedMbps int64 `json:"speedMbps,omitempty"`

	// vendor
	Vendor string `json:"vendor,omitempty"`
}

type Memory struct {

	// physical bytes
	PhysicalBytes int64 `json:"physicalBytes,omitempty"`

	// usable bytes
	UsableBytes int64 `json:"usableBytes,omitempty"`
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
	ProductName string `json:"productName,omitempty"`

	// serial number
	SerialNumber string `json:"serialNumber,omitempty"`

	// Whether the machine appears to be a virtual machine or not
	Virtual bool `json:"virtual,omitempty"`
}

type HostDevice struct {

	// path of the device (i.e. /dev/loop)
	Path string `json:"path,omitempty"`

	// Device type block or character
	DeviceType string `json:"deviceType,omitempty"`

	// owner id
	UID uint32 `json:"owner,omitempty"`

	// group id
	GID uint32 `json:"group,omitempty"`

	// Major ID identifying the class of the device
	Major uint32 `json:"major,omitempty"`

	// Minor ID identifying the instance of the device in the class
	Minor uint32 `json:"minor,omitempty"`
}

type Mount struct {
	// Device path to be mounted
	Device string `json:"device,omitempty"`

	// Destination directory path
	Directory string `json:"folder,omitempty"`

	// Mount type: (i.e ext4)
	Type string `json:"type,omitempty"`

	// Mount options (i.e. rw, suid, dev)
	Options string `json:"options,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+genclient
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

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
