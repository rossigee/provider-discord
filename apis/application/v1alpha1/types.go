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

// ApplicationParameters defines the desired state of a Discord application
type ApplicationParameters struct {
	// ApplicationID is the Discord application ID to retrieve/manage
	// For current application operations, use "@me"
	// +kubebuilder:validation:Required
	ApplicationID string `json:"applicationId"`

	// Name is the application name (only for editing current application)
	// +optional
	Name *string `json:"name,omitempty"`

	// Description is the application description
	// +optional
	Description *string `json:"description,omitempty"`

	// Icon is the application icon image data (base64 encoded)
	// Only applicable when editing current application
	// +optional
	Icon *string `json:"icon,omitempty"`

	// CoverImage is the application cover image (base64 encoded)
	// +optional
	CoverImage *string `json:"coverImage,omitempty"`

	// RPCOrigins are RPC origin URLs
	// +optional
	RPCOrigins []string `json:"rpcOrigins,omitempty"`

	// BotPublic indicates whether the bot is public
	// +optional
	BotPublic *bool `json:"botPublic,omitempty"`

	// BotRequireCodeGrant indicates if the bot requires OAuth2 code grant
	// +optional
	BotRequireCodeGrant *bool `json:"botRequireCodeGrant,omitempty"`

	// TermsOfServiceURL is the URL to the application's terms of service
	// +optional
	TermsOfServiceURL *string `json:"termsOfServiceUrl,omitempty"`

	// PrivacyPolicyURL is the URL to the application's privacy policy
	// +optional
	PrivacyPolicyURL *string `json:"privacyPolicyUrl,omitempty"`

	// CustomInstallURL is a custom URL for OAuth2 authorization
	// +optional
	CustomInstallURL *string `json:"customInstallUrl,omitempty"`

	// Tags are tags describing the application
	// +optional
	Tags []string `json:"tags,omitempty"`
}

// ApplicationObservation represents the observed state of a Discord application
type ApplicationObservation struct {
	// ID is the application's unique Discord ID
	ID string `json:"id,omitempty"`

	// Name is the application name
	Name string `json:"name,omitempty"`

	// Icon is the application icon hash
	Icon *string `json:"icon,omitempty"`

	// Description is the application description
	Description string `json:"description,omitempty"`

	// RPCOrigins are RPC origin URLs
	RPCOrigins []string `json:"rpcOrigins,omitempty"`

	// BotPublic indicates whether the bot is public
	BotPublic bool `json:"botPublic,omitempty"`

	// BotRequireCodeGrant indicates if bot requires OAuth2 code grant
	BotRequireCodeGrant bool `json:"botRequireCodeGrant,omitempty"`

	// BotUserID is the ID of the bot user associated with this application
	BotUserID *string `json:"botUserId,omitempty"`

	// TermsOfServiceURL is the URL to the terms of service
	TermsOfServiceURL *string `json:"termsOfServiceUrl,omitempty"`

	// PrivacyPolicyURL is the URL to the privacy policy
	PrivacyPolicyURL *string `json:"privacyPolicyUrl,omitempty"`

	// OwnerID is the ID of the application owner
	OwnerID *string `json:"ownerId,omitempty"`

	// Summary is a summary of the application (deprecated)
	Summary string `json:"summary,omitempty"`

	// VerifyKey is the hex-encoded key for GameSDK's GetTicket
	VerifyKey string `json:"verifyKey,omitempty"`

	// TeamID is the ID of the team if the application belongs to a team
	TeamID *string `json:"teamId,omitempty"`

	// GuildID is the guild ID associated with the application
	GuildID *string `json:"guildId,omitempty"`

	// PrimarySkuID is the ID of the "Game SKU" created for the application
	PrimarySkuID *string `json:"primarySkuId,omitempty"`

	// Slug is the URL slug that links to the application's store page
	Slug *string `json:"slug,omitempty"`

	// CoverImage is the application's cover image hash
	CoverImage *string `json:"coverImage,omitempty"`

	// Flags are the application's public flags
	Flags *int `json:"flags,omitempty"`

	// ApproximateGuildCount is the approximate count of guilds the app is in
	ApproximateGuildCount *int `json:"approximateGuildCount,omitempty"`

	// RedirectURIs are array of redirect URIs for OAuth2
	RedirectURIs []string `json:"redirectUris,omitempty"`

	// InteractionsEndpointURL is the URL for receiving interactions
	InteractionsEndpointURL *string `json:"interactionsEndpointUrl,omitempty"`

	// RoleConnectionsVerificationURL is the URL for role connections
	RoleConnectionsVerificationURL *string `json:"roleConnectionsVerificationUrl,omitempty"`

	// Tags are tags describing the application
	Tags []string `json:"tags,omitempty"`

	// InstallParamsScopes are the OAuth2 scopes for installation
	InstallParamsScopes []string `json:"installParamsScopes,omitempty"`

	// InstallParamsPermissions are the permissions for installation
	InstallParamsPermissions *string `json:"installParamsPermissions,omitempty"`

	// CustomInstallURL is the custom URL for OAuth2 authorization
	CustomInstallURL *string `json:"customInstallUrl,omitempty"`
}

// A ApplicationSpec defines the desired state of a Application.
type ApplicationSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ApplicationParameters `json:"forProvider"`
}

// A ApplicationStatus represents the observed state of a Application.
type ApplicationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ApplicationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// A Application is a managed resource that represents a Discord application
// +kubebuilder:printcolumn:name="APP_ID",type="string",JSONPath=".spec.forProvider.applicationId"
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".status.atProvider.name"
// +kubebuilder:printcolumn:name="BOT_PUBLIC",type="boolean",JSONPath=".status.atProvider.botPublic"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,discord}
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// ApplicationList contains a list of Applications.
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
