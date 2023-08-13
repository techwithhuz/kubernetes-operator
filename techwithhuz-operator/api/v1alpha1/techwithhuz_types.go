/*
Copyright 2023.

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

// TechwithhuzSpec defines the desired state of Techwithhuz
type TechwithhuzSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Techwithhuz. Edit techwithhuz_types.go to remove/update
	//Foo string `json:"foo,omitempty"`
	//Add size and containerport property for which values can be passed through Custom Resource(CR) file.
	Size          int32 `json:"size,omitempty"`
	ContainerPort int32 `json:"containerPort,omitempty"`
}

// TechwithhuzStatus defines the observed state of Techwithhuz
type TechwithhuzStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Conditions store the status conditions of the TechWithHuz instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Techwithhuz is the Schema for the techwithhuzs API
type Techwithhuz struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TechwithhuzSpec   `json:"spec,omitempty"`
	Status TechwithhuzStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TechwithhuzList contains a list of Techwithhuz
type TechwithhuzList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Techwithhuz `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Techwithhuz{}, &TechwithhuzList{})
}
