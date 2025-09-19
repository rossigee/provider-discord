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

//+kubebuilder:object:generate=true

// MemberParameters defines the desired state of a Discord guild member
type MemberParameters struct {
	// GuildID is the ID of the Discord guild
	// +kubebuilder:validation:Required
	GuildID string `json:"guildId"`

	// UserID is the ID of the Discord user to manage
	// +kubebuilder:validation:Required
	UserID string `json:"userId"`

	// Nick is the user's nickname in the guild
	// +optional
	Nick *string `json:"nick,omitempty"`

	// Roles is an array of role IDs assigned to the member
	// +optional
	Roles []string `json:"roles,omitempty"`

	// Mute indicates whether the user is muted in voice channels
	// +optional
	Mute *bool `json:"mute,omitempty"`

	// Deaf indicates whether the user is deafened in voice channels
	// +optional
	Deaf *bool `json:"deaf,omitempty"`

	// ChannelID is the ID of the voice channel to move the user to
	// +optional
	ChannelID *string `json:"channelId,omitempty"`

	// CommunicationDisabledUntil sets when the user's timeout expires
	// Set to null/empty to remove timeout
	// +optional
	CommunicationDisabledUntil *string `json:"communicationDisabledUntil,omitempty"`

	// Flags represents guild member flags as a bit set
	// +optional
	Flags *int `json:"flags,omitempty"`

	// AccessToken is required for adding new members via OAuth2
	// Only used for PUT operations when adding a user to the guild
	// +optional
	AccessToken *string `json:"accessToken,omitempty"`
}

// DiscordUser represents basic user information
type DiscordUser struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	Discriminator string  `json:"discriminator"`
	Avatar        *string `json:"avatar,omitempty"`
}

// MemberObservation represents the observed state of a Discord guild member
type MemberObservation struct {
	// ID is the user ID (same as UserID parameter)
	ID string `json:"id,omitempty"`

	// Username is the Discord username
	Username string `json:"username,omitempty"`

	// Discriminator is the user's 4-digit discriminator
	Discriminator string `json:"discriminator,omitempty"`

	// Avatar is the user's avatar hash
	Avatar *string `json:"avatar,omitempty"`

	// Banner is the user's banner hash
	Banner *string `json:"banner,omitempty"`

	// GuildAvatar is the member's guild-specific avatar hash
	GuildAvatar *string `json:"guildAvatar,omitempty"`

	// GuildBanner is the member's guild-specific banner hash
	GuildBanner *string `json:"guildBanner,omitempty"`

	// JoinedAt is when the user joined the guild
	JoinedAt *string `json:"joinedAt,omitempty"`

	// PremiumSince is when the user started boosting the guild
	PremiumSince *string `json:"premiumSince,omitempty"`

	// Pending indicates if the user has passed Membership Screening
	Pending *bool `json:"pending,omitempty"`

	// Permissions are the total permissions in the guild
	Permissions *string `json:"permissions,omitempty"`

	// AvatarDecorationData contains avatar decoration information
	AvatarDecorationData *string `json:"avatarDecorationData,omitempty"`

	// User is the user object for this member
	User *DiscordUser `json:"user,omitempty"`

	// Nick is the member's nickname in the guild
	Nick *string `json:"nick,omitempty"`

	// Roles are the role IDs assigned to this member
	Roles []string `json:"roles,omitempty"`

	// Deaf indicates if the member is deafened in voice channels
	Deaf *bool `json:"deaf,omitempty"`

	// Mute indicates if the member is muted in voice channels
	Mute *bool `json:"mute,omitempty"`

	// Flags represents guild member flags
	Flags *int `json:"flags,omitempty"`

	// CommunicationDisabledUntil is when the member's timeout expires
	CommunicationDisabledUntil *string `json:"communicationDisabledUntil,omitempty"`
}

// A MemberSpec defines the desired state of a Member.
type MemberSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       MemberParameters `json:"forProvider"`
}

// A MemberStatus represents the observed state of a Member.
type MemberStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          MemberObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// A Member is a managed resource that represents a Discord guild member
// +kubebuilder:printcolumn:name="GUILD",type="string",JSONPath=".spec.forProvider.guildId"
// +kubebuilder:printcolumn:name="USER",type="string",JSONPath=".spec.forProvider.userId"
// +kubebuilder:printcolumn:name="NICK",type="string",JSONPath=".spec.forProvider.nick"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,discord}
type Member struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemberSpec   `json:"spec"`
	Status MemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// MemberList contains a list of Members.
type MemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Member `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Member{}, &MemberList{})
}