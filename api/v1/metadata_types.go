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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetadataSpec defines the desired state of Metadata
type MetadataSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Metadata. Edit metadata_types.go to remove/update
	PVReference        string `json:"pvReference"`
	BackingPVReference string `json:backingPVReference,omitempty`
	RefCount           int64  `json:refCount,omitempty`
}

// MetadataStatus defines the observed state of Metadata
//type MetadataStatus struct {
//	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
//	// Important: Run "make" to regenerate code after modifying this file
//}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Metadata is the Schema for the metadata API
type Metadata struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetadataSpec   `json:"spec,omitempty"`
	Status MetadataStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MetadataList contains a list of Metadata
type MetadataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Metadata `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Metadata{}, &MetadataList{})
}
