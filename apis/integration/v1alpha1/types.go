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

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

//+kubebuilder:object:generate=true

// IntegrationParameters defines the desired state of a Discord guild integration
type IntegrationParameters struct {
	// GuildID is the ID of the Discord guild
	// +kubebuilder:validation:Required
	GuildID string `json:"guildId"`

	// IntegrationID is the ID of the Discord integration to manage
	// This is mainly used for deletion operations since integrations
	// are typically created externally through Discord's OAuth2 flow
	// +kubebuilder:validation:Required
	IntegrationID string `json:"integrationId"`
}

// IntegrationObservation represents the observed state of a Discord integration
type IntegrationObservation struct {
	// ID is the integration's unique Discord ID
	ID string `json:"id,omitempty"`

	// Name is the integration name
	Name string `json:"name,omitempty"`

	// Type is the integration type (twitch, youtube, discord, etc.)
	Type string `json:"type,omitempty"`

	// Enabled indicates whether this integration is enabled
	Enabled bool `json:"enabled,omitempty"`

	// Syncing indicates whether this integration is syncing
	Syncing *bool `json:"syncing,omitempty"`

	// RoleID is the ID of the role this integration uses for subscribers
	RoleID *string `json:"roleId,omitempty"`

	// EnableEmoticons indicates whether emoticons are enabled
	EnableEmoticons *bool `json:"enableEmoticons,omitempty"`

	// ExpireBehavior is the behavior used for expired subscribers
	ExpireBehavior *int `json:"expireBehavior,omitempty"`

	// ExpireGracePeriod is the grace period (in days) for expired subscribers
	ExpireGracePeriod *int `json:"expireGracePeriod,omitempty"`

	// UserID is the ID of the user for this integration
	UserID *string `json:"userId,omitempty"`

	// AccountID is the integration account ID
	AccountID *string `json:"accountId,omitempty"`

	// AccountName is the integration account name
	AccountName *string `json:"accountName,omitempty"`

	// SyncedAt is when this integration was last synced
	SyncedAt *string `json:"syncedAt,omitempty"`

	// SubscriberCount is how many subscribers this integration has
	SubscriberCount *int `json:"subscriberCount,omitempty"`

	// Revoked indicates whether the integration was revoked
	Revoked *bool `json:"revoked,omitempty"`

	// ApplicationID is the ID of the application for Discord integrations
	ApplicationID *string `json:"applicationId,omitempty"`

	// Scopes are the OAuth2 scopes the application has been authorized for
	Scopes []string `json:"scopes,omitempty"`
}

// A IntegrationSpec defines the desired state of a Integration.
type IntegrationSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       IntegrationParameters `json:"forProvider"`
}

// A IntegrationStatus represents the observed state of a Integration.
type IntegrationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          IntegrationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// A Integration is a managed resource that represents a Discord guild integration
// +kubebuilder:printcolumn:name="GUILD",type="string",JSONPath=".spec.forProvider.guildId"
// +kubebuilder:printcolumn:name="INTEGRATION",type="string",JSONPath=".spec.forProvider.integrationId"
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".status.atProvider.name"
// +kubebuilder:printcolumn:name="TYPE",type="string",JSONPath=".status.atProvider.type"
// +kubebuilder:printcolumn:name="ENABLED",type="boolean",JSONPath=".status.atProvider.enabled"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,discord}
type Integration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntegrationSpec   `json:"spec"`
	Status IntegrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// IntegrationList contains a list of Integrations.
type IntegrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Integration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Integration{}, &IntegrationList{})
}