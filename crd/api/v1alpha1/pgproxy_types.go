/*
Copyright 2025.

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

// PgProxySpec defines the desired state of PgProxy
type PgProxySpec struct {
	DeploymentName string `json:"deploymentName"`
	// Namespace du Deployment et du service PostgreSQL
	Namespace string `json:"namespace"`
	// Host/IP du service PostgreSQL
	PostgresServiceHost string `json:"postgresServiceHost"`
	// Port du service PostgreSQL
	PostgresServicePort int32 `json:"postgresServicePort"`
}

// PgProxyStatus defines the observed state of PgProxy.
type PgProxyStatus struct {
	// état du proxy (Ready, Error…)
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PgProxy is the Schema for the pgproxies API
type PgProxy struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of PgProxy
	// +required
	Spec PgProxySpec `json:"spec"`

	// status defines the observed state of PgProxy
	// +optional
	Status PgProxyStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// PgProxyList contains a list of PgProxy
type PgProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PgProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PgProxy{}, &PgProxyList{})
}
