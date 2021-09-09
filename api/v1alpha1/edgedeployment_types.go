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

// EdgeDeploymentSpec defines the desired state of EdgeDeployment
type EdgeDeploymentSpec struct {
	DeviceSelector *metav1.LabelSelector `json:"deviceSelector,omitempty"`
	Device         string                `json:"device,omitempty"`
	Type           EdgeDeploymentType    `json:"type"`
	Pod            Pod                   `json:"pod,omitempty"`
	Data           *DataConfiguration    `json:"data,omitempty"`
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

type EdgeDeploymentType string

const (
	PodDeploymentType EdgeDeploymentType = "pod"
)

// EdgeDeploymentStatus defines the observed state of EdgeDeployment
type EdgeDeploymentStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EdgeDeployment is the Schema for the edgedeployments API
type EdgeDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeDeploymentSpec   `json:"spec,omitempty"`
	Status EdgeDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EdgeDeploymentList contains a list of EdgeDeployment
type EdgeDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeDeployment{}, &EdgeDeploymentList{})
}
