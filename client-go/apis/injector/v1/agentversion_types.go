/*
Copyright 2024.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentVersionSpec defines the desired state of AgentVersion
type AgentVersionSpec struct {
	// Version of JoyLive Agent release
	Version string `json:"version"`
	// ConfigMapName record storage version configuration file
	ConfigMapName string `json:"configMapName,omitempty"`
	// Enable indicates whether this version is enabled
	Enable bool `json:"enable"`
}

// AgentVersionStatus defines the observed state of AgentVersion
type AgentVersionStatus struct {
}

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// AgentVersion is the Schema for the agentversions API
type AgentVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentVersionSpec   `json:"spec,omitempty"`
	Status AgentVersionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AgentVersionList contains a list of AgentVersion
type AgentVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentVersion `json:"items"`
}

func init() {
	//SchemeBuilder.Register(&AgentVersion{}, &AgentVersionList{})
}
