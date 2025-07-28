package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

//+kubebuilder:object:generate=true

// RoleParameters are the configurable fields of a Role.
type RoleParameters struct {
	// Name of the role
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// GuildID is the ID of the guild this role belongs to
	// +kubebuilder:validation:Required
	GuildID string `json:"guildId"`

	// Color integer representation of hexadecimal color code
	// +optional
	Color *int `json:"color,omitempty"`

	// Whether to display role members separately from other members
	// +optional
	Hoist *bool `json:"hoist,omitempty"`

	// Whether the role can be mentioned
	// +optional
	Mentionable *bool `json:"mentionable,omitempty"`

	// Permission bit set
	// +optional
	Permissions *string `json:"permissions,omitempty"`

	// Position of the role in the role hierarchy
	// +optional
	Position *int `json:"position,omitempty"`
}

// RoleObservation are the observable fields of a Role.
type RoleObservation struct {
	// ID of the role on Discord
	ID string `json:"id,omitempty"`

	// Whether this role is managed by an integration
	Managed bool `json:"managed,omitempty"`
}

// A RoleSpec defines the desired state of a Role.
type RoleSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RoleParameters `json:"forProvider"`
}

// A RoleStatus represents the observed state of a Role.
type RoleStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RoleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="GUILD",type="string",JSONPath=".spec.forProvider.guildId"
// +kubebuilder:printcolumn:name="POSITION",type="integer",JSONPath=".spec.forProvider.position"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,discord}

// A Role is an example API type.
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoleSpec   `json:"spec"`
	Status RoleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RoleList contains a list of Role
type RoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Role `json:"items"`
}

// Role type metadata.
var (
	RoleKind             = reflect.TypeOf(Role{}).Name()
	RoleGroupKind        = schema.GroupKind{Group: Group, Kind: RoleKind}.String()
	RoleKindAPIVersion   = RoleKind + "." + SchemeGroupVersion.String()
	RoleGroupVersionKind = SchemeGroupVersion.WithKind(RoleKind)
)

func init() {
	SchemeBuilder.Register(&Role{}, &RoleList{})
}