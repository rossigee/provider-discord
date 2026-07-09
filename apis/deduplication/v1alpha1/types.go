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
)

// A DeduplicationSpec defines the desired state of a Deduplication operation.
type DeduplicationSpec struct {
	// ProviderConfigRef references the ProviderConfig that triggered this deduplication.
	ProviderConfigRef ProviderConfigReference `json:"providerConfigRef"`

	// Mode defines the deduplication behavior that was requested.
	// +kubebuilder:validation:Enum=report;action
	Mode string `json:"mode"`

	// DeleteOrphanedResources indicates whether to delete Crossplane resources
	// for deleted Discord channels.
	// +optional
	DeleteOrphanedResources bool `json:"deleteOrphanedResources,omitempty"`

	// TargetGuilds limits deduplication to specific guild IDs.
	// If empty, all guilds were processed.
	// +optional
	TargetGuilds []string `json:"targetGuilds,omitempty"`
}

// ProviderConfigReference references a ProviderConfig resource.
type ProviderConfigReference struct {
	// Name of the ProviderConfig.
	Name string `json:"name"`
}

// A DeduplicationStatus reflects the observed state of a Deduplication operation.
type DeduplicationStatus struct {
	// Phase indicates the current phase of the deduplication operation.
	// +kubebuilder:validation:Enum=pending;analyzing;completed;failed
	Phase string `json:"phase,omitempty"`

	// StartTime is when the deduplication operation was initiated.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the deduplication operation completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Summary provides aggregate statistics about the deduplication operation.
	// +optional
	Summary *DeduplicationSummary `json:"summary,omitempty"`

	// Results contains per-guild deduplication results.
	// +optional
	Results map[string]GuildDeduplicationResult `json:"results,omitempty"`

	// DeepCopyObject is implemented by zz_generated.deepcopy.go for DeduplicationSummary and GuildDeduplicationResult
	// (these are stored inline and do not need to be registered as root types).

	// Conditions represent the latest available observations of the deduplication's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastError describes the last error encountered during deduplication (if any).
	// +optional
	LastError string `json:"lastError,omitempty"`
}

// DeduplicationSummary provides aggregate statistics.
type DeduplicationSummary struct {
	// TotalGuildsAnalyzed is the number of guilds that were analyzed.
	TotalGuildsAnalyzed int `json:"totalGuildsAnalyzed"`

	// TotalChannelsAnalyzed is the total number of channels examined.
	TotalChannelsAnalyzed int `json:"totalChannelsAnalyzed"`

	// DuplicateGroupsFound is the number of distinct duplicate channel groups.
	DuplicateGroupsFound int `json:"duplicateGroupsFound"`

	// TotalDuplicateChannelsFound is the total count of duplicate channels (excluding the kept one in each group).
	TotalDuplicateChannelsFound int `json:"totalDuplicateChannelsFound"`

	// ChannelsDeleted is the count of Discord channels actually deleted (action mode only).
	ChannelsDeleted int `json:"channelsDeleted"`

	// OrphanedResourcesDeleted is the count of Crossplane resources deleted (action mode only).
	OrphanedResourcesDeleted int `json:"orphanedResourcesDeleted"`
}

// GuildDeduplicationResult contains deduplication results for a specific guild.
type GuildDeduplicationResult struct {
	// GuildID is the Discord guild ID.
	GuildID string `json:"guildId"`

	// GuildName is the Discord guild name.
	GuildName string `json:"guildName"`

	// TotalChannels is the total number of channels in the guild.
	TotalChannels int `json:"totalChannels"`

	// DuplicateGroups contains information about each duplicate group found.
	// +optional
	DuplicateGroups []DuplicateGroupInfo `json:"duplicateGroups,omitempty"`

	// ChannelsDeleted is the count of channels deleted in this guild (action mode only).
	ChannelsDeleted int `json:"channelsDeleted"`

	// OrphanedResourcesDeleted is the count of resources deleted in this guild (action mode only).
	OrphanedResourcesDeleted int `json:"orphanedResourcesDeleted"`

	// Errors contains any errors encountered while processing this guild.
	// +optional
	Errors []string `json:"errors,omitempty"`
}

// DuplicateGroupInfo contains information about a group of duplicate channels.
type DuplicateGroupInfo struct {
	// ChannelName is the name shared by all channels in this group.
	ChannelName string `json:"channelName"`

	// Count is the number of duplicate channels with this name.
	Count int `json:"count"`

	// KeptChannelID is the ID of the channel that was kept.
	// +optional
	KeptChannelID string `json:"keptChannelId,omitempty"`

	// DeletedChannelIDs are the IDs of channels that were deleted.
	// +optional
	DeletedChannelIDs []string `json:"deletedChannelIds,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// A Deduplication tracks a channel deduplication operation.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PHASE",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="MODE",type="string",JSONPath=".spec.mode"
// +kubebuilder:printcolumn:name="DUPLICATES-FOUND",type="integer",JSONPath=".status.summary.totalDuplicateChannelsFound"
// +kubebuilder:printcolumn:name="DELETED",type="integer",JSONPath=".status.summary.channelsDeleted"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,discord}
// +kubebuilder:storageversion
type Deduplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeduplicationSpec   `json:"spec"`
	Status DeduplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// DeduplicationList contains a list of Deduplication.
type DeduplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Deduplication `json:"items"`
}
