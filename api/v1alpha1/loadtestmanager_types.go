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

// LoadTestManagerSpec defines the desired state of LoadTestManager
type LoadTestManagerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Replicas is the number of load tester instances.
	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// UserIncreaseStep is the number of user increase between load tests.
	// +kubebuilder:default=10
	// +optional
	UserIncreaseStep int64 `json:"userIncreaseStep,omitempty"`

	// LoadDuration is the number of seconds to keep the load in the same level.
	// +kubebuilder:default=30
	// +optional
	LoadDuration int64 `json:"loadDuration,omitempty"`

	// SpawnDuration is the number of user increase in a second during the load increasing phase.
	// +kubebuilder:default=5
	// +optional
	SpawnDuration int64 `json:"spawnDuration,omitempty"`

	// TargetHost is the host name to load.
	TargetHost map[string]string `json:"targetHost,omitempty"`

	// Scripts is the load test scripts written in Python format.
	Scripts map[string]string `json:"scripts,omitempty"`
}

// LoadTestManagerStatus defines the observed state of LoadTestManager
// +kubebuilder:validation:Enum=NotReady;Available;Testing
type LoadTestManagerStatus string

const (
	LoadTestManagerNotReady  = LoadTestManagerStatus("NotReady")
	LoadTestManagerAvailable = LoadTestManagerStatus("Available")
	LoadTestManagerTesting   = LoadTestManagerStatus("Testing")
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status"

// LoadTestManager is the Schema for the loadtestmanagers API
type LoadTestManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoadTestManagerSpec   `json:"spec,omitempty"`
	Status LoadTestManagerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LoadTestManagerList contains a list of LoadTestManager
type LoadTestManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoadTestManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LoadTestManager{}, &LoadTestManagerList{})
}
