package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Organization",type="string",JSONPath=".spec.organization"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// GithubOrganizationWatch defines the desired state for GitHub secrets configuration
type GithubOrganizationWatch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GithubOrganizationWatchSpec   `json:"spec"`
	Status            GithubOrganizationWatchStatus `json:"status,omitempty"`
}

// GithubOrganizationWatchSpec defines the desired state
type GithubOrganizationWatchSpec struct {
	// Organization is the GitHub organization name
	Organization string `json:"organization"`
	// Topics is a list of repository topics to match
	Topics []string `json:"topics"`
	// ConfigSetRef is the name of the ConfigMap containing variables
	ConfigSetRef string `json:"configSetRef"`
	// SecretRef is the name of the Secret containing sensitive data
	SecretRef string `json:"secretRef,omitempty"`
}

// GithubOrganizationWatchStatus defines the observed state
type GithubOrganizationWatchStatus struct {
	// LastSyncTime is the last time the secrets were synchronized
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
	// SyncedRepositories is the list of repositories that were synchronized
	SyncedRepositories []string `json:"syncedRepositories,omitempty"`
	// Conditions represent the latest available observations of the config's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// GithubOrganizationWatchList contains a list of GithubOrganizationWatch
type GithubOrganizationWatchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GithubOrganizationWatch `json:"items"`
}
