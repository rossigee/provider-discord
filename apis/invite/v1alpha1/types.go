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

// InviteParameters are the configurable fields of an Invite.
type InviteParameters struct {
	// ChannelID is the ID of the channel this invite is for.
	// +kubebuilder:validation:Required
	ChannelID string `json:"channelId"`

	// MaxAge is the duration of invite in seconds before expiry, or 0 for never.
	// Default is 86400 (24 hours).
	// +optional
	// +kubebuilder:default=86400
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=604800
	MaxAge *int `json:"maxAge,omitempty"`

	// MaxUses is the max number of uses, or 0 for unlimited.
	// Default is 0 (unlimited).
	// +optional
	// +kubebuilder:default=0
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	MaxUses *int `json:"maxUses,omitempty"`

	// Temporary specifies whether this invite only grants temporary membership.
	// Default is false.
	// +optional
	// +kubebuilder:default=false
	Temporary *bool `json:"temporary,omitempty"`

	// Unique specifies whether this invite should be unique.
	// If true, don't try to reuse a similar invite.
	// Default is false.
	// +optional
	// +kubebuilder:default=false
	Unique *bool `json:"unique,omitempty"`
}

// InviteObservation are the observable fields of an Invite.
type InviteObservation struct {
	// Code is the invite code.
	Code string `json:"code,omitempty"`

	// GuildID is the ID of the guild this invite is for.
	GuildID string `json:"guildId,omitempty"`

	// ChannelID is the ID of the channel this invite is for.
	ChannelID string `json:"channelId,omitempty"`

	// InviterID is the ID of the user who created the invite.
	InviterID string `json:"inviterId,omitempty"`

	// TargetType is the type of target for this voice channel invite.
	TargetType *int `json:"targetType,omitempty"`

	// TargetUserID is the ID of the target user for this invite.
	TargetUserID *string `json:"targetUserId,omitempty"`

	// TargetApplicationID is the ID of the embedded application to open for this invite.
	TargetApplicationID *string `json:"targetApplicationId,omitempty"`

	// ApproximatePresenceCount is the approximate count of online members.
	ApproximatePresenceCount *int `json:"approximatePresenceCount,omitempty"`

	// ApproximateMemberCount is the approximate count of total members.
	ApproximateMemberCount *int `json:"approximateMemberCount,omitempty"`

	// ExpiresAt is the expiration date of this invite.
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// CreatedAt is the timestamp when the invite was created.
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// Uses is the number of times this invite has been used.
	Uses int `json:"uses,omitempty"`

	// MaxAge is the max age of the invite in seconds.
	MaxAge int `json:"maxAge,omitempty"`

	// MaxUses is the max number of uses for the invite.
	MaxUses int `json:"maxUses,omitempty"`

	// Temporary indicates whether the invite grants temporary membership.
	Temporary bool `json:"temporary,omitempty"`

	// URL is the full invite URL (stored in connection secret).
	URL string `json:"-"`
}

// An InviteSpec defines the desired state of an Invite.
type InviteSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       InviteParameters `json:"forProvider"`
}

// An InviteStatus represents the observed state of an Invite.
type InviteStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          InviteObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// An Invite is a managed resource that represents a Discord invite.
// +kubebuilder:printcolumn:name="CODE",type="string",JSONPath=".status.atProvider.code"
// +kubebuilder:printcolumn:name="CHANNEL",type="string",JSONPath=".spec.forProvider.channelId"
// +kubebuilder:printcolumn:name="USES",type="integer",JSONPath=".status.atProvider.uses"
// +kubebuilder:printcolumn:name="MAX-USES",type="integer",JSONPath=".status.atProvider.maxUses"
// +kubebuilder:printcolumn:name="EXPIRES",type="date",JSONPath=".status.atProvider.expiresAt"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,discord}
type Invite struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InviteSpec   `json:"spec"`
	Status InviteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// InviteList contains a list of Invite
type InviteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Invite `json:"items"`
}
