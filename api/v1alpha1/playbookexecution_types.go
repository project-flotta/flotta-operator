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

// PlaybookExecutionSpec defines the desired state of PlaybookExecution
type PlaybookExecutionSpec struct {
	Playbooks Playbook `json:"playbook,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Default=0
	ExecutionAttempt uint8 `json:"executionAttempt,omitempty" description:"the number of times the playbook has been executed" default:"0"`
}

// PlaybookExecutionStatus defines the observed state of PlaybookExecution
type PlaybookExecutionStatus struct {
	Conditions []PlaybookExecutionCondition `json:"conditions,omitempty"`
}

type PlaybookExecutionCondition struct {
	Type PlaybookExecutionConditionType `json:"type" description:"type of PlaybookExecutionCondition condition"`
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

type PlaybookExecutionConditionType string

const (
	PlaybookExecutionDeploying             PlaybookExecutionConditionType = "Deploying"
	PlaybookExecutionTargetVerification    PlaybookExecutionConditionType = "TargetVerification"
	PlaybookExecutionRunning               PlaybookExecutionConditionType = "Running"
	PlaybookExecutionSuccessfullyCompleted PlaybookExecutionConditionType = "SuccessfullyCompleted"
	PlaybookExecutionCompletedWithError    PlaybookExecutionConditionType = "CompletedWithError"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PlaybookExecution is the Schema for the playbookexecutions API
type PlaybookExecution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlaybookExecutionSpec   `json:"spec,omitempty"`
	Status PlaybookExecutionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PlaybookExecutionList contains a list of PlaybookExecution
type PlaybookExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlaybookExecution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlaybookExecution{}, &PlaybookExecutionList{})
}
