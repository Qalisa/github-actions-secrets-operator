/*
Copyright 2025 Guillaume Vara.

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

type ResourceRef struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SecretRef defines a reference to a Kubernetes Secret and how to map it to a GitHub Secret
type SecretRef struct {
	// SecretRef is the name of the Kubernetes Secret containing the value
	SecretRef ResourceRef `json:"secretRef"`
	// Key is the key in the Kubernetes Secret to use
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
	// GithubSecretName is the name to use for the GitHub Secret (defaults to Key if not set)
	// +optional
	GithubSecretName string `json:"githubSecretName,omitempty"`
}

// VariableRef defines a reference to a Kubernetes ConfigMap and how to map it to a GitHub Variable
type VariableRef struct {
	// ConfigMapRef is the name of the Kubernetes ConfigMap containing the value
	ConfigMapRef ResourceRef `json:"configMapRef"`
	// Key is the key in the Kubernetes ConfigMap to use
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
	// GithubVariableName is the name to use for the GitHub Variable (defaults to Key if not set)
	// +optional
	GithubVariableName string `json:"githubVariableName,omitempty"`
}

// GithubActionSecretsSyncSpec defines the desired state of GithubActionSecretsSync
type GithubActionSecretsSyncSpec struct {
	// Secrets is a list of Kubernetes Secrets to sync to GitHub Secrets
	// +optional
	Secrets []SecretRef `json:"secrets,omitempty"`
	// Variables is a list of Kubernetes ConfigMaps to sync to GitHub Variables
	// +optional
	Variables []VariableRef `json:"variables,omitempty"`
}

// GithubActionSecretsSyncStatus defines the observed state of GithubActionSecretsSync
type GithubActionSecretsSyncStatus struct {
	// LastSyncTime is the last time the secrets were synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
	// Conditions represent the latest available observations of the sync state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// ErrorMessage contains the last error message if sync failed
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Synced",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="Last Sync",type="date",JSONPath=".status.lastSyncTime"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// GithubActionSecretsSync is the Schema for the githubactionsecretssyncs API.
type GithubActionSecretsSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GithubActionSecretsSyncSpec   `json:"spec,omitempty"`
	Status GithubActionSecretsSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GithubActionSecretsSyncList contains a list of GithubActionSecretsSync.
type GithubActionSecretsSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GithubActionSecretsSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GithubActionSecretsSync{}, &GithubActionSecretsSyncList{})
}
