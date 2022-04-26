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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EdgeWorkloadSpec defines the desired state of EdgeWorkload
type EdgeWorkloadSpec struct {
	DeviceSelector  *metav1.LabelSelector          `json:"deviceSelector,omitempty"`
	Device          string                         `json:"device,omitempty"`
	Type            EdgeWorkloadType               `json:"type"`
	Pod             Pod                            `json:"pod,omitempty"`
	Data            *DataConfiguration             `json:"data,omitempty"`
	ImageRegistries *ImageRegistriesConfiguration  `json:"imageRegistries,omitempty"`
	Metrics         *ContainerMetricsConfiguration `json:"metrics,omitempty"`

	// LogCollection is the logCollection property to be used to collect logs
	// from this endpoint. This key is what is defined on the edgedevice
	// logCollection property
	LogCollection string `json:"logCollection,omitempty"`
}

type ImageRegistriesConfiguration struct {
	AuthFileSecret *NameRef `json:"secretRef,omitempty"`
}

type MetricsConfigEntity struct {
	// Path to use when retrieving metrics
	// +kubebuilder:default=/
	Path string `json:"path,omitempty"`

	// Port to use when retrieve the metrics
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port,omitempty"`

	Disabled bool `json:"disabled,omitempty"`
}

type ContainerMetricsConfiguration struct {
	// Path to use when retrieving metrics
	// +kubebuilder:default=/
	Path string `json:"path,omitempty"`

	// Port to use when retrieve the metrics
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port,omitempty"`

	// Interval(in seconds) to scrape metrics endpoint.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=60
	Interval int32 `json:"interval,omitempty"`

	// Specification of workload metrics to be collected
	AllowList *NameRef `json:"allowList,omitempty"`

	Containers map[string]*MetricsConfigEntity `json:"containers,omitempty"`
}

type NameRef struct {
	Name string `json:"name"`
}

type DataConfiguration struct {
	Paths []DataPath `json:"paths,omitempty"`
}

type DataPath struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Pod struct {
	Spec v1.PodSpec `json:"spec"`
}

type EdgeWorkloadType string

const (
	PodWorkloadType EdgeWorkloadType = "pod"
)

// EdgeWorkloadStatus defines the observed state of EdgeWorkload
type EdgeWorkloadStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+genclient

// EdgeWorkload is the Schema for the EdgeWorkloads API
type EdgeWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeWorkloadSpec   `json:"spec,omitempty"`
	Status EdgeWorkloadStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EdgeWorkloadList contains a list of EdgeWorkload
type EdgeWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeWorkload{}, &EdgeWorkloadList{})
}
