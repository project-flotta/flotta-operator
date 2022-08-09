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
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EdgeConfigSpec defines the desired state of EdgeConfig
type EdgeConfigSpec struct {
	// Definition of the custom playbook workload to execute
	EdgePlaybook *EdgePlaybookSpec `json:"edgePlaybook,omitempty"`

	// IgnitionConfig is the config used by flotta config. By default only
	// user/systemd and a subset of filesystem are in place(directories, links,
	// files) The main reason to use in this way is because ignition is only
	// valid for startup, but some functions(described before)can be used without
	// issues
	IgnitionConfig json.RawMessage `json:"IgnitionConfig,omitempty"`
}

// EdgeConfigStatus defines the observed state of EdgeConfig
type EdgeConfigStatus struct {
	EdgePlaybookStatus *EdgePlaybookStatus `json:"edgePlaybookStatus,omitempty"`
}

// EdgePlaybookSpec defines the desired state of EdgePlaybook
type EdgePlaybookSpec struct {
	// username who execute the playbook
	User string `json:"user,omitempty"`
	// The ansible's playbooks list. The first element is the playbook with the highest priority.
	// +kubebuilder:validation:MinItems=1
	Playbooks []Playbook `json:"playbooks,omitempty"`
}

// EdgePlaybookStatus defines the observed state of EdgePlaybook
type EdgePlaybookStatus struct {
	Conditions []EdgePlaybookCondition `json:"conditions,omitempty"`
}

type Playbook struct {
	// Playbook content
	Content []byte `json:"content"`
	// The connection timeout on ansible-playbook
	// +kubernetes:validation:Minimum=0
	TimeoutSeconds int64 `json:"timeoutSeconds,omitempty"`
	// The required privelege level necessary to execute the playbook
	RequiredPrivilegeLevel *RequiredPrivilegeLevel `json:"requiredPrivilegeLevel,omitempty"`
	// the ansible's playbook options for the playbook
	Options *AnsibleOptions `json:"ansibleOptions,omitempty"`
	// the ansible's playbook privilege escalation options for the playbook
	PrivilegeEscalationOptions *PrivilegeEscalationOptions `json:"privilegeEscalationOptions,omitempty"`
	//Execution strategy for the playbook
	ExecutionStrategy ExecutionStrategy `json:"executionStrategy,omitempty"`
}

type ExecutionStrategy string

const (
	StopAtFailuire ExecutionStrategy = "StopOnFailure"
	RetryOnFailure ExecutionStrategy = "RetryOnFailure"
	Once           ExecutionStrategy = "ExecuteOnce"
)

type AnsibleOptions struct {
	// don't make any changes; instead, try to predict some of the changes that may occur
	Check bool `json:"check,omitempty"`
}

type RequiredPrivilegeLevel struct {
	// See https://man7.org/linux/man-pages/man7/capabilities.7.html
	CapAdd  []CapType `json:"capAdd,omitempty" description:"Capabilities to add"`
	CapDrop []CapType `json:"capDrop,omitempty" description:"Capabilities to drop"`
}
type PrivilegeEscalationOptions struct {
	Become bool `json:"become,omitempty"`
	// +kubebuilder:validation:Enum=sudo;su
	// +kubebuilder:validation:default=sudo
	BecomeMethod string `json:"becomeMethod,omitempty"`
	BecomeUser   string `json:"becomeUser,omitempty"`
}

type CapType int8

const (
	CHOWN CapType = iota
	DAC_OVERRIDE
	DAC_READ_SEARCH
	FOWNER
	FSETID
	KILL
	SETGID
	SETUID
	SETPCAP
	LINUX_IMMUTABLE
	NET_BIND_SERVICE
	NET_BROADCAST
	NET_ADMIN
	NET_RAW
	IPC_LOCK
	IPC_OWNER
	SYS_MODULE
	SYS_RAWIO
	SYS_CHROOT
	SYS_PTRACE
	SYS_PACCT
	SYS_ADMIN
	SYS_BOOT
	SYS_NICE
	SYS_RESOURCE
	SYS_TIME
	SYS_TTY_CONFIG
	MKNOD
	LEASE
	AUDIT_WRITE
	AUDIT_CONTROL
	SETFCAP
	MAC_OVERRIDE
	MAC_ADMIN
	SYSLOG
	WAKE_ALARM
	BLOCK_SUSPEND
	AUDIT_READ
)

type EdgePlaybookCondition struct {
	Type EdgePlaybookConditionType `json:"type" description:"type of EdgePlaybookCondition condition"`
	// Indicates whether that condition is applicable, with possible values "True", "False", or "Unknown"
	// The absence of a condition should be interpreted the same as Unknown
	Status metav1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// +optional
	Reason *string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`
	// +optional
	Message *string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty" description:"last time the condition transit from one status to another"`
}

type EdgePlaybookConditionType string

const (
	PlaybookDeploying EdgePlaybookConditionType = "Deploying"
	Completed         EdgePlaybookConditionType = "Completed"
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
