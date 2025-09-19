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

// ChannelParameters are the configurable fields of a Channel.
type ChannelParameters struct {
	// Name is the name of the Discord channel.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=100
	Name string `json:"name"`

	// Type is the type of channel.
	// 0 = Text, 1 = DM, 2 = Voice, 3 = Group DM, 4 = Category, 5 = News, 10 = News Thread, 11 = Public Thread, 12 = Private Thread, 13 = Stage Voice, 15 = Forum
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=0;2;4;5;13;15
	Type int `json:"type"`

	// GuildID is the ID of the guild this channel belongs to.
	// +kubebuilder:validation:Required
	GuildID string `json:"guildId"`

	// Topic is the channel topic (text channels only).
	// +optional
	// +kubebuilder:validation:MaxLength=1024
	Topic *string `json:"topic,omitempty"`

	// Position is the sorting position of the channel.
	// +optional
	Position *int `json:"position,omitempty"`

	// ParentID is the ID of the parent category for a channel.
	// +optional
	ParentID *string `json:"parentId,omitempty"`

	// NSFW indicates whether the channel is NSFW.
	// +optional
	NSFW *bool `json:"nsfw,omitempty"`

	// Bitrate is the bitrate (in bits) of the voice channel.
	// Voice channels only, 8000 to 96000 (128000 for VIP servers).
	// +optional
	// +kubebuilder:validation:Minimum=8000
	// +kubebuilder:validation:Maximum=128000
	Bitrate *int `json:"bitrate,omitempty"`

	// UserLimit is the user limit of the voice channel.
	// Voice channels only, 0 refers to no limit, 1 to 99 refers to a user limit.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=99
	UserLimit *int `json:"userLimit,omitempty"`

	// RateLimitPerUser is the amount of seconds a user has to wait before sending another message.
	// Text channels only, 0-21600.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=21600
	RateLimitPerUser *int `json:"rateLimitPerUser,omitempty"`

	// DefaultAutoArchiveDuration is the default duration for newly created threads.
	// +optional
	// +kubebuilder:validation:Enum=60;1440;4320;10080
	DefaultAutoArchiveDuration *int `json:"defaultAutoArchiveDuration,omitempty"`
}

// ChannelObservation are the observable fields of a Channel.
type ChannelObservation struct {
	// ID is the unique identifier of the channel in Discord.
	ID string `json:"id,omitempty"`

	// Name is the current name of the channel.
	Name string `json:"name,omitempty"`

	// Type is the type of channel.
	Type int `json:"type,omitempty"`

	// GuildID is the ID of the guild this channel belongs to.
	GuildID string `json:"guildId,omitempty"`

	// Topic is the channel topic.
	Topic string `json:"topic,omitempty"`

	// Position is the sorting position of the channel.
	Position int `json:"position,omitempty"`

	// ParentID is the ID of the parent category.
	ParentID string `json:"parentId,omitempty"`

	// NSFW indicates whether the channel is NSFW.
	NSFW bool `json:"nsfw,omitempty"`

	// Bitrate is the bitrate of the voice channel.
	Bitrate int `json:"bitrate,omitempty"`

	// UserLimit is the user limit of the voice channel.
	UserLimit int `json:"userLimit,omitempty"`

	// RateLimitPerUser is the rate limit per user.
	RateLimitPerUser int `json:"rateLimitPerUser,omitempty"`

	// DefaultAutoArchiveDuration is the default auto archive duration.
	DefaultAutoArchiveDuration int `json:"defaultAutoArchiveDuration,omitempty"`

	// LastMessageID is the ID of the last message sent in this channel.
	LastMessageID string `json:"lastMessageId,omitempty"`

	// CreatedAt is the timestamp when the channel was created.
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// UpdatedAt is the timestamp when the channel was last updated.
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`
}

// A ChannelSpec defines the desired state of a Channel.
type ChannelSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ChannelParameters `json:"forProvider"`
}

// A ChannelStatus represents the observed state of a Channel.
type ChannelStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ChannelObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// A Channel is a managed resource that represents a Discord channel.
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="TYPE",type="integer",JSONPath=".spec.forProvider.type"
// +kubebuilder:printcolumn:name="GUILD",type="string",JSONPath=".spec.forProvider.guildId"
// +kubebuilder:printcolumn:name="CHANNEL-ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,discord}
type Channel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChannelSpec   `json:"spec"`
	Status ChannelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// ChannelList contains a list of Channel
type ChannelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Channel `json:"items"`
}

