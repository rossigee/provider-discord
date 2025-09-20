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

// WebhookParameters are the configurable fields of a Webhook.
type WebhookParameters struct {
	// Name is the name of the Discord webhook.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=80
	Name string `json:"name"`

	// ChannelID is the ID of the channel this webhook will post to.
	// +kubebuilder:validation:Required
	ChannelID string `json:"channelId"`

	// Avatar is the avatar image data for the webhook (base64 encoded image).
	// +optional
	Avatar *string `json:"avatar,omitempty"`
}

// WebhookObservation are the observable fields of a Webhook.
type WebhookObservation struct {
	// ID is the unique identifier of the webhook in Discord.
	ID string `json:"id,omitempty"`

	// Type is the type of webhook.
	// 1 = Incoming, 2 = Channel Follower, 3 = Application
	Type int `json:"type,omitempty"`

	// Name is the current name of the webhook.
	Name string `json:"name,omitempty"`

	// Avatar is the webhook's avatar hash.
	Avatar string `json:"avatar,omitempty"`

	// ChannelID is the ID of the channel this webhook posts to.
	ChannelID string `json:"channelId,omitempty"`

	// GuildID is the ID of the guild this webhook belongs to.
	GuildID string `json:"guildId,omitempty"`

	// ApplicationID is the bot/OAuth2 application that created this webhook.
	ApplicationID string `json:"applicationId,omitempty"`

	// Token is the secure token of the webhook (returned only on creation).
	// This is stored in the connection secret and not exposed in status.
	Token string `json:"-"`

	// URL is the webhook URL (returned only on creation).
	// This is stored in the connection secret and not exposed in status.
	URL string `json:"-"`

	// CreatedAt is the timestamp when the webhook was created.
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// UpdatedAt is the timestamp when the webhook was last updated.
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`
}

// A WebhookSpec defines the desired state of a Webhook.
type WebhookSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       WebhookParameters `json:"forProvider"`
}

// A WebhookStatus represents the observed state of a Webhook.
type WebhookStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          WebhookObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// A Webhook is a managed resource that represents a Discord webhook.
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="CHANNEL",type="string",JSONPath=".spec.forProvider.channelId"
// +kubebuilder:printcolumn:name="WEBHOOK-ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="TYPE",type="integer",JSONPath=".status.atProvider.type"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,discord}
type Webhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebhookSpec   `json:"spec"`
	Status WebhookStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// WebhookList contains a list of Webhook
type WebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Webhook `json:"items"`
}
