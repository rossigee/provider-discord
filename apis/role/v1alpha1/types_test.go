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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

func TestRoleDeepCopy(t *testing.T) {
	original := &Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "discord.crossplane.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-role",
			Namespace: "default",
		},
		Spec: RoleSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy: xpv1.DeletionDelete,
			},
			ForProvider: RoleParameters{
				Name:        "Admin Role",
				GuildID:     "123456789",
				Color:       intPtr(16711680), // Red color
				Hoist:       boolPtr(true),
				Position:    intPtr(5),
				Permissions: stringPtr("8"), // Administrator permission
				Mentionable: boolPtr(false),
			},
		},
		Status: RoleStatus{
			ResourceStatus: xpv1.ResourceStatus{},
			AtProvider: RoleObservation{
				ID:      "role123456",
				Managed: true,
			},
		},
	}

	// Test DeepCopy
	copied := original.DeepCopy()
	
	// Verify they're not the same object
	assert.NotSame(t, original, copied)
	
	// Verify the content is the same
	assert.Equal(t, original.TypeMeta, copied.TypeMeta)
	assert.Equal(t, original.ObjectMeta, copied.ObjectMeta)
	assert.Equal(t, original.Spec, copied.Spec)
	assert.Equal(t, original.Status, copied.Status)
	
	// Verify deep copy - modifying one shouldn't affect the other
	copied.Spec.ForProvider.Name = "Modified Role"
	assert.NotEqual(t, original.Spec.ForProvider.Name, copied.Spec.ForProvider.Name)
}

func TestRoleListDeepCopy(t *testing.T) {
	original := &RoleList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleList",
			APIVersion: "discord.crossplane.io/v1alpha1",
		},
		ListMeta: metav1.ListMeta{
			ResourceVersion: "1",
		},
		Items: []Role{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "role1",
				},
				Spec: RoleSpec{
					ForProvider: RoleParameters{
						Name:    "Admin",
						GuildID: "123456789",
						Color:   intPtr(16711680),
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "role2",
				},
				Spec: RoleSpec{
					ForProvider: RoleParameters{
						Name:    "Moderator",
						GuildID: "123456789",
						Color:   intPtr(255),
					},
				},
			},
		},
	}

	// Test DeepCopyObject
	obj := original.DeepCopyObject()
	copied, ok := obj.(*RoleList)
	require.True(t, ok)
	
	// Verify they're not the same object
	assert.NotSame(t, original, copied)
	
	// Verify the content is the same
	assert.Equal(t, original.TypeMeta, copied.TypeMeta)
	assert.Equal(t, original.ListMeta, copied.ListMeta)
	assert.Len(t, copied.Items, len(original.Items))
	
	// Verify deep copy - modifying one shouldn't affect the other
	copied.Items[0].Spec.ForProvider.Name = "Super Admin"
	assert.NotEqual(t, original.Items[0].Spec.ForProvider.Name, copied.Items[0].Spec.ForProvider.Name)
}

func TestRoleJSONMarshaling(t *testing.T) {
	role := &Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "discord.crossplane.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-role",
		},
		Spec: RoleSpec{
			ForProvider: RoleParameters{
				Name:        "Moderator",
				GuildID:     "123456789",
				Color:       intPtr(255), // Blue color
				Hoist:       boolPtr(true),
				Position:    intPtr(3),
				Permissions: stringPtr("32"), // Manage messages permission
				Mentionable: boolPtr(true),
			},
		},
		Status: RoleStatus{
			AtProvider: RoleObservation{
				ID:      "role987654",
				Managed: true,
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(role)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Moderator")
	assert.Contains(t, string(data), "role987654")
	assert.Contains(t, string(data), "123456789")

	// Test unmarshaling
	var unmarshaled Role
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	
	// Verify the unmarshaled object matches
	assert.Equal(t, role.TypeMeta, unmarshaled.TypeMeta)
	assert.Equal(t, role.Name, unmarshaled.Name)
	assert.Equal(t, role.Spec.ForProvider.Name, unmarshaled.Spec.ForProvider.Name)
	assert.Equal(t, role.Spec.ForProvider.GuildID, unmarshaled.Spec.ForProvider.GuildID)
	assert.Equal(t, role.Status.AtProvider.ID, unmarshaled.Status.AtProvider.ID)
	
	// Verify pointer fields are handled correctly
	require.NotNil(t, unmarshaled.Spec.ForProvider.Color)
	assert.Equal(t, 255, *unmarshaled.Spec.ForProvider.Color)
	require.NotNil(t, unmarshaled.Spec.ForProvider.Hoist)
	assert.Equal(t, true, *unmarshaled.Spec.ForProvider.Hoist)
	require.NotNil(t, unmarshaled.Spec.ForProvider.Position)
	assert.Equal(t, 3, *unmarshaled.Spec.ForProvider.Position)
	require.NotNil(t, unmarshaled.Spec.ForProvider.Permissions)
	assert.Equal(t, "32", *unmarshaled.Spec.ForProvider.Permissions)
	require.NotNil(t, unmarshaled.Spec.ForProvider.Mentionable)
	assert.Equal(t, true, *unmarshaled.Spec.ForProvider.Mentionable)
}

func TestRoleParametersValidation(t *testing.T) {
	// Test creating valid role parameters for different role types
	
	// Admin role with all permissions
	adminParams := RoleParameters{
		Name:        "Administrator",
		GuildID:     "123456789",
		Color:       intPtr(16711680), // Red
		Hoist:       boolPtr(true),
		Position:    intPtr(10),
		Permissions: stringPtr("8"), // Administrator permission
		Mentionable: boolPtr(false),
	}
	
	assert.Equal(t, "Administrator", adminParams.Name)
	assert.Equal(t, "123456789", adminParams.GuildID)
	assert.Equal(t, 16711680, *adminParams.Color)
	assert.Equal(t, true, *adminParams.Hoist)
	assert.Equal(t, 10, *adminParams.Position)
	assert.Equal(t, "8", *adminParams.Permissions)
	assert.Equal(t, false, *adminParams.Mentionable)

	// Basic member role
	memberParams := RoleParameters{
		Name:        "Member",
		GuildID:     "123456789",
		Color:       intPtr(0), // Default color
		Hoist:       boolPtr(false),
		Position:    intPtr(1),
		Permissions: stringPtr("104197632"), // Basic read/send messages
		Mentionable: boolPtr(true),
	}
	
	assert.Equal(t, "Member", memberParams.Name)
	assert.Equal(t, "123456789", memberParams.GuildID)
	assert.Equal(t, 0, *memberParams.Color)
	assert.Equal(t, false, *memberParams.Hoist)
	assert.Equal(t, 1, *memberParams.Position)
	assert.Equal(t, "104197632", *memberParams.Permissions)
	assert.Equal(t, true, *memberParams.Mentionable)

	// Minimal role with only required fields
	minimalParams := RoleParameters{
		Name:    "Basic Role",
		GuildID: "123456789",
	}
	
	assert.Equal(t, "Basic Role", minimalParams.Name)
	assert.Equal(t, "123456789", minimalParams.GuildID)
	assert.Nil(t, minimalParams.Color)
	assert.Nil(t, minimalParams.Hoist)
	assert.Nil(t, minimalParams.Position)
	assert.Nil(t, minimalParams.Permissions)
	assert.Nil(t, minimalParams.Mentionable)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}