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

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// GuildParameters are the configurable fields of a Guild.
type GuildParameters struct {
	// Name is the name of the Discord guild (server).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=100
	Name string `json:"name"`

	// Region is the voice region for the guild.
	// +optional
	Region *string `json:"region,omitempty"`

	// Icon is the icon hash for the guild.
	// +optional
	Icon *string `json:"icon,omitempty"`

	// VerificationLevel is the verification level for the guild.
	// 0 = None, 1 = Low, 2 = Medium, 3 = High, 4 = Very High
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=4
	VerificationLevel *int `json:"verificationLevel,omitempty"`

	// DefaultMessageNotifications is the default message notification level.
	// 0 = All messages, 1 = Only mentions
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1
	DefaultMessageNotifications *int `json:"defaultMessageNotifications,omitempty"`

	// ExplicitContentFilter is the explicit content filter level.
	// 0 = Disabled, 1 = Members without roles, 2 = All members
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=2
	ExplicitContentFilter *int `json:"explicitContentFilter,omitempty"`

	// AFKChannelID is the ID of the AFK channel.
	// +optional
	AFKChannelID *string `json:"afkChannelId,omitempty"`

	// AFKTimeout is the AFK timeout in seconds.
	// +optional
	// +kubebuilder:validation:Minimum=60
	// +kubebuilder:validation:Maximum=3600
	AFKTimeout *int `json:"afkTimeout,omitempty"`

	// SystemChannelID is the ID of the system channel.
	// +optional
	SystemChannelID *string `json:"systemChannelId,omitempty"`

	// SystemChannelFlags are the system channel flags.
	// +optional
	SystemChannelFlags *int `json:"systemChannelFlags,omitempty"`
}

// GuildObservation are the observable fields of a Guild.
type GuildObservation struct {
	// ID is the unique identifier of the guild in Discord.
	ID string `json:"id,omitempty"`

	// Name is the current name of the guild.
	Name string `json:"name,omitempty"`

	// Region is the voice region of the guild.
	Region string `json:"region,omitempty"`

	// Icon is the icon hash of the guild.
	Icon string `json:"icon,omitempty"`

	// OwnerID is the ID of the guild owner.
	OwnerID string `json:"ownerId,omitempty"`

	// MemberCount is the total number of members in the guild.
	MemberCount int `json:"memberCount,omitempty"`

	// VerificationLevel is the verification level of the guild.
	VerificationLevel int `json:"verificationLevel,omitempty"`

	// DefaultMessageNotifications is the default message notification level.
	DefaultMessageNotifications int `json:"defaultMessageNotifications,omitempty"`

	// ExplicitContentFilter is the explicit content filter level.
	ExplicitContentFilter int `json:"explicitContentFilter,omitempty"`

	// Features are the features enabled for the guild.
	Features []string `json:"features,omitempty"`

	// AFKChannelID is the ID of the AFK channel.
	AFKChannelID string `json:"afkChannelId,omitempty"`

	// AFKTimeout is the AFK timeout in seconds.
	AFKTimeout int `json:"afkTimeout,omitempty"`

	// SystemChannelID is the ID of the system channel.
	SystemChannelID string `json:"systemChannelId,omitempty"`

	// SystemChannelFlags are the system channel flags.
	SystemChannelFlags int `json:"systemChannelFlags,omitempty"`

	// CreatedAt is the timestamp when the guild was created.
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// UpdatedAt is the timestamp when the guild was last updated.
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`
}

// A GuildSpec defines the desired state of a Guild.
type GuildSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       GuildParameters `json:"forProvider"`
}

// A GuildStatus represents the observed state of a Guild.
type GuildStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          GuildObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// A Guild is a managed resource that represents a Discord guild (server).
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="GUILD-ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="MEMBERS",type="integer",JSONPath=".status.atProvider.memberCount"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,discord}
type Guild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GuildSpec   `json:"spec"`
	Status GuildStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// GuildList contains a list of Guild
type GuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Guild `json:"items"`
}

