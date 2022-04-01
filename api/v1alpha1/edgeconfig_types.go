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

// EdgeConfigSpec defines the desired state of EdgeConfig
type EdgeConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The ansible playbook command to execute
	EdgePlaybook *EdgePlaybookSpec `json:"edgePlaybook,omitempty"`

	//TODO: Add EdgeDeviceGroup. Depends on https://github.com/project-flotta/flotta-operator/pull/161
}

// EdgeConfigStatus defines the observed state of EdgeConfig
type EdgeConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file.
	EdgePlaybookStatus *EdgePlaybookStatus `json:"edgePlaybookStatus,omitempty"`
}

// EdgePlaybookSpec defines the desired state of EdgePlaybook
type EdgePlaybookSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The ansible playbook command to execute
	AnsiblePlaybookCmd *AnsiblePlaybookCmd `json:"ansiblePlaybookCmd,omitempty"`
	//Execution strategy for each playbook
	ExecutionStrategy map[string]ExecutionStrategy `json:"playbooksPriorityMap,omitempty"`
}

// EdgePlaybookStatus defines the observed state of EdgePlaybook
type EdgePlaybookStatus struct {
	Condition                 EdgePlaybookCondition `json:"condition,omitempty"`
	LastSeenTime              metav1.Time           `json:"lastSeenTime,omitempty"`
	LastSyncedResourceVersion string                `json:"lastSyncedResourceVersion,omitempty"`
}

type AnsiblePlaybookCmd struct {
	// username who execute the playbook
	User string `json:"user,omitempty"`
	// the ansible's playbooks list with priority to be used
	// +kubebuilder:validation:MinProperties=1
	Playbooks map[string]Playbook `json:"playbooksPriorityMap,omitempty"`
	// the ansible's playbook options for each playbook
	Options map[string]*AnsibleOptions `json:"ansibleOptions,omitempty"`
	// the ansible's playbook privilege escalation options for each playbook
	PrivilegeEscalationOptions map[string]*PrivilegeEscalationOptions `json:"privilegeEscalationOptions,omitempty"`
}

type ExecutionStrategy string

const (
	StopAtFailuire    ExecutionStrategy = "Stop at first failure"
	ContinueOnFailure ExecutionStrategy = "Continue on failure"
)

type AnsibleOptions struct {
	// don't make any changes; instead, try to predict some of the changes that may occur
	Check bool `json:"check,omitempty"`
}

type PrivilegeEscalationOptions struct {
	Become bool `json:"become,omitempty"`
	// +kubebuilder:validation:Enum=sudo;su
	BecomeMethod string `json:"becomeMethod,omitempty"`
	BecomeUser   string `json:"becomeUser,omitempty"`
}

type Playbook struct {
	// Link to an arichived playbook content (tar.gz)
	URL string `json:"url"`
	// The connection timeout on ansible-playbook
	Timeout uint64 `json:"timeout,omitempty"`
	// TODO: Enum like linux capabilities ?
	// The required privelege level necessary to execute the playbook
	RequiredPrivilegeLevel []string `json:"requiredPrivilegeLevel,omitempty"`
}

type EdgePlaybookCondition struct {
	Type   EdgePlaybookConditionType   `json:"type" description:"type of EdgePlaybookCondition condition"`
	Status EdgePlaybookConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// +optional
	Reason *string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`
	// +optional
	Message *string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty" description:"last time the condition transit from one status to another"`
}

type EdgePlaybookConditionType string

const (
	TargetVerification EdgePlaybookConditionType = "TargetVerification"
	PlaybookDeploying  EdgePlaybookConditionType = "Deploying"
	PlaybookExecuting  EdgePlaybookConditionType = "Executing"
	Completed          EdgePlaybookConditionType = "Completed"
)

type EdgePlaybookConditionStatus string

const (
	EdgePlaybookConditionStatusTrue    EdgePlaybookConditionStatus = "True"
	EdgePlaybookConditionStatusFalse   EdgePlaybookConditionStatus = "False"
	EdgePlaybookConditionStatusUnknown EdgePlaybookConditionStatus = "Unknown"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EdgeConfig is the Schema for the edgeconfigs API
type EdgeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeConfigSpec   `json:"spec,omitempty"`
	Status EdgeConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EdgeConfigList contains a list of EdgeConfig
type EdgeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeConfig{}, &EdgeConfigList{})
}
