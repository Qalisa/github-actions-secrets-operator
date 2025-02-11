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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GithubSyncRepoSpec defines the desired state of GithubSyncRepo
type GithubSyncRepoSpec struct {
	// Repository is the full name of the GitHub repository (org/repo)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9-_]+/[a-zA-Z0-9-_\.]+$`
	Repository string `json:"repository"`
	// SecretsSyncRefs is a list of GithubActionSecretsSync names to apply to this repository
	// +optional
	SecretsSyncRefs []string `json:"secretsSyncRefs,omitempty"`
}

// GithubSyncRepoStatus defines the observed state of GithubSyncRepo
type GithubSyncRepoStatus struct {
	// LastSyncTime is the last time the secrets were synced to this repository
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
// +kubebuilder:printcolumn:name="Repository",type="string",JSONPath=".spec.repository"
// +kubebuilder:printcolumn:name="Synced",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="Last Sync",type="date",JSONPath=".status.lastSyncTime"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// GithubSyncRepo is the Schema for the githubsyncrepoes API.
type GithubSyncRepo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GithubSyncRepoSpec   `json:"spec,omitempty"`
	Status GithubSyncRepoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GithubSyncRepoList contains a list of GithubSyncRepo.
type GithubSyncRepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GithubSyncRepo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GithubSyncRepo{}, &GithubSyncRepoList{})
}
