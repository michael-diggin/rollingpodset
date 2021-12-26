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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&RollingPodSet{}, &RollingPodSetList{})
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RollingPodSetSpec defines the desired state of RollingPodSet
type RollingPodSetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Replicas is the number of Pods of the RollingPodSet to be deployed at any one time.
	Replicas int `json:"replicas,omitempty"`

	// PodTemplate specifies the Pod that will be created by the RollingPodSet.
	PodTemplate corev1.PodTemplateSpec `json:"podTemplate,omitempty"`

	//+kubebuilder:validation:Minimum=30

	// Specifies the duration that pods will be cycled after.
	CycleTime int64 `json:"cycleTime,omitempty"`

	// Selector is a label query over pods to match the replica count.
	Selector metav1.LabelSelector `json:"selector,omitempty"`
}

// RollingPodSetStatus defines the observed state of RollingPodSet
type RollingPodSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Replicas int `json:"replicas,omitempty"`

	AvailableReplicas int `json:"availableReplicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RollingPodSet is the Schema for the rollingpodsets API
type RollingPodSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RollingPodSetSpec   `json:"spec,omitempty"`
	Status RollingPodSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RollingPodSetList contains a list of RollingPodSet
type RollingPodSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RollingPodSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RollingPodSet{}, &RollingPodSetList{})
}
