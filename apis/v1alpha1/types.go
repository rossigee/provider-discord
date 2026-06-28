/*
Copyright 2025 The Crossplane Authors.

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

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// A ProviderConfigSpec defines the desired state of a ProviderConfig.
type ProviderConfigSpec struct {
	// Credentials required to authenticate to this provider.
	Credentials ProviderCredentials `json:"credentials"`

	// BaseURL is the base URL of the Discord API.
	// Defaults to https://discord.com/api/v10 if not specified.
	// +optional
	BaseURL *string `json:"baseURL,omitempty"`

	// Deduplication configuration for channel deduplication.
	// +optional
	Deduplication *DeduplicationSpec `json:"deduplication,omitempty"`

	// GarbageCollection configuration for autonomous cleanup.
	// +optional
	GarbageCollection *GarbageCollectionSpec `json:"garbageCollection,omitempty"`
}

// ProviderCredentials required to authenticate.
type ProviderCredentials struct {
	// Source of the provider credentials.
	// +kubebuilder:validation:Enum=Secret
	Source xpv1.CredentialsSource `json:"source"`

	xpv1.CommonCredentialSelectors `json:",inline"`
}

// A ProviderConfigStatus reflects the observed state of a ProviderConfig.
type ProviderConfigStatus struct {
	xpv1.ProviderConfigStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// A ProviderConfig configures a Discord provider.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.credentials.secretRef.name",priority=1
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,discord}
// +kubebuilder:storageversion
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// ProviderConfigList contains a list of ProviderConfig.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

// DeduplicationSpec defines the desired state of channel deduplication.
type DeduplicationSpec struct {
	// Enabled indicates if deduplication is active.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Mode defines the deduplication behavior.
	// "report" - analyze and report duplicates via Kubernetes Events
	// "action" - delete duplicate channels and corresponding Crossplane resources
	// +kubebuilder:validation:Enum=report;action
	// +optional
	Mode DeduplicationMode `json:"mode,omitempty"`

	// DeleteOrphanedResources indicates whether to delete Crossplane resources
	// for deleted Discord channels. Only applies in "action" mode.
	// +optional
	DeleteOrphanedResources bool `json:"deleteOrphanedResources,omitempty"`

	// TargetGuilds limits deduplication to specific guild IDs.
	// If empty, all guilds the bot is a member of will be processed.
	// +optional
	TargetGuilds []string `json:"targetGuilds,omitempty"`
}

// DeduplicationMode defines how deduplication should be performed.
// +kubebuilder:validation:Enum=report;action
type DeduplicationMode string

const (
	// DeduplicationModeReport performs analysis only and reports via Kubernetes Events.
	DeduplicationModeReport DeduplicationMode = "report"

	// DeduplicationModeAction deletes duplicate channels and related Crossplane resources.
	DeduplicationModeAction DeduplicationMode = "action"
)

// GarbageCollectionSpec defines autonomous cleanup configuration.
type GarbageCollectionSpec struct {
	// Enabled indicates if garbage collection is active.
	// When enabled, the provider automatically prevents and cleans up duplicates.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// PreventDuplicatesOnCreate blocks channel creation if a channel with the same name
	// already exists in the guild. When false, duplicate channels are allowed at creation.
	// Default: true
	// +optional
	PreventDuplicatesOnCreate *bool `json:"preventDuplicatesOnCreate,omitempty"`

	// PollIntervalSeconds is the interval in seconds for periodic duplicate cleanup.
	// Minimum: 60 (1 minute), Maximum: 3600 (1 hour)
	// Default: 300 (5 minutes)
	// +kubebuilder:validation:Minimum=60
	// +kubebuilder:validation:Maximum=3600
	// +optional
	PollIntervalSeconds *int32 `json:"pollIntervalSeconds,omitempty"`

	// DeleteOrphanedResources indicates whether to delete Crossplane Channel resources
	// when their corresponding Discord channels are deleted during garbage collection.
	// Default: true
	// +optional
	DeleteOrphanedResources *bool `json:"deleteOrphanedResources,omitempty"`

	// TargetGuilds limits garbage collection to specific guild IDs.
	// If empty, all guilds the bot is a member of will be monitored.
	// +optional
	TargetGuilds []string `json:"targetGuilds,omitempty"`
}
