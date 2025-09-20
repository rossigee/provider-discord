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

// UserParameters defines the desired state of a Discord user
type UserParameters struct {
	// UserID is the Discord user ID to retrieve/manage
	// For current user operations, use "@me"
	// +kubebuilder:validation:Required
	UserID string `json:"userId"`

	// Username is the user's username (only for modifying current user)
	// +optional
	Username *string `json:"username,omitempty"`

	// Avatar is the user's avatar image data (base64 encoded)
	// Only applicable when modifying current user
	// +optional
	Avatar *string `json:"avatar,omitempty"`

	// Banner is the user's banner image data (base64 encoded)
	// Only applicable when modifying current user
	// +optional
	Banner *string `json:"banner,omitempty"`
}

// UserObservation represents the observed state of a Discord user
type UserObservation struct {
	// ID is the user's unique Discord ID
	ID string `json:"id,omitempty"`

	// Username is the user's username
	Username string `json:"username,omitempty"`

	// Discriminator is the user's 4-digit discriminator
	Discriminator string `json:"discriminator,omitempty"`

	// GlobalName is the user's display name
	GlobalName *string `json:"globalName,omitempty"`

	// Avatar is the user's avatar hash
	Avatar *string `json:"avatar,omitempty"`

	// Bot indicates whether the user is a bot
	Bot *bool `json:"bot,omitempty"`

	// System indicates whether the user is a system user
	System *bool `json:"system,omitempty"`

	// MFAEnabled indicates whether the user has MFA enabled
	MFAEnabled *bool `json:"mfaEnabled,omitempty"`

	// Banner is the user's banner hash
	Banner *string `json:"banner,omitempty"`

	// AccentColor is the user's banner color encoded as integer
	AccentColor *int `json:"accentColor,omitempty"`

	// Locale is the user's chosen language option
	Locale *string `json:"locale,omitempty"`

	// Verified indicates whether the email is verified
	Verified *bool `json:"verified,omitempty"`

	// Email is the user's email (only visible for current user)
	Email *string `json:"email,omitempty"`

	// Flags are the user's public flags
	Flags *int `json:"flags,omitempty"`

	// PremiumType is the user's Nitro subscription type
	PremiumType *int `json:"premiumType,omitempty"`

	// PublicFlags are the user's public flags
	PublicFlags *int `json:"publicFlags,omitempty"`

	// AvatarDecorationData contains avatar decoration information
	AvatarDecorationData *string `json:"avatarDecorationData,omitempty"`
}

// A UserSpec defines the desired state of a User.
type UserSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       UserParameters `json:"forProvider"`
}

// A UserStatus represents the observed state of a User.
type UserStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// A User is a managed resource that represents a Discord user
// +kubebuilder:printcolumn:name="USER_ID",type="string",JSONPath=".spec.forProvider.userId"
// +kubebuilder:printcolumn:name="USERNAME",type="string",JSONPath=".status.atProvider.username"
// +kubebuilder:printcolumn:name="DISCRIMINATOR",type="string",JSONPath=".status.atProvider.discriminator"
// +kubebuilder:printcolumn:name="BOT",type="boolean",JSONPath=".status.atProvider.bot"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,discord}
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true

// UserList contains a list of Users.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
