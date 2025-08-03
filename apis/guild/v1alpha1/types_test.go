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

func TestGuildDeepCopy(t *testing.T) {
	original := &Guild{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Guild",
			APIVersion: "discord.crossplane.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-guild",
			Namespace: "default",
		},
		Spec: GuildSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy: xpv1.DeletionDelete,
			},
			ForProvider: GuildParameters{
				Name:                        "Test Guild",
				Region:                      stringPtr("us-east"),
				VerificationLevel:           intPtr(1),
				DefaultMessageNotifications: intPtr(1),
				ExplicitContentFilter:       intPtr(1),
				AFKTimeout:                  intPtr(300),
			},
		},
		Status: GuildStatus{
			ResourceStatus: xpv1.ResourceStatus{},
			AtProvider: GuildObservation{
				ID:            "123456789",
				Name:          "Test Guild",
				Region:        "us-east",
				MemberCount:   10,
				VerificationLevel: 1,
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
	copied.Spec.ForProvider.Name = "Modified Guild"
	assert.NotEqual(t, original.Spec.ForProvider.Name, copied.Spec.ForProvider.Name)
}

func TestGuildListDeepCopy(t *testing.T) {
	original := &GuildList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "GuildList",
			APIVersion: "discord.crossplane.io/v1alpha1",
		},
		ListMeta: metav1.ListMeta{
			ResourceVersion: "1",
		},
		Items: []Guild{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "guild1",
				},
				Spec: GuildSpec{
					ForProvider: GuildParameters{
						Name: "Guild 1",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "guild2",
				},
				Spec: GuildSpec{
					ForProvider: GuildParameters{
						Name: "Guild 2",
					},
				},
			},
		},
	}

	// Test DeepCopyObject
	obj := original.DeepCopyObject()
	copied, ok := obj.(*GuildList)
	require.True(t, ok)
	
	// Verify they're not the same object
	assert.NotSame(t, original, copied)
	
	// Verify the content is the same
	assert.Equal(t, original.TypeMeta, copied.TypeMeta)
	assert.Equal(t, original.ListMeta, copied.ListMeta)
	assert.Len(t, copied.Items, len(original.Items))
	
	// Verify deep copy - modifying one shouldn't affect the other
	copied.Items[0].Spec.ForProvider.Name = "Modified Guild"
	assert.NotEqual(t, original.Items[0].Spec.ForProvider.Name, copied.Items[0].Spec.ForProvider.Name)
}

func TestGuildJSONMarshaling(t *testing.T) {
	guild := &Guild{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Guild",
			APIVersion: "discord.crossplane.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-guild",
		},
		Spec: GuildSpec{
			ForProvider: GuildParameters{
				Name:                        "Test Guild",
				Region:                      stringPtr("us-east"),
				VerificationLevel:           intPtr(2),
				DefaultMessageNotifications: intPtr(1),
				ExplicitContentFilter:       intPtr(1),
				AFKTimeout:                  intPtr(600),
				SystemChannelFlags:          intPtr(0),
			},
		},
		Status: GuildStatus{
			AtProvider: GuildObservation{
				ID:          "123456789",
				Name:        "Test Guild",
				Region:      "us-east",
				MemberCount: 25,
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(guild)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Test Guild")
	assert.Contains(t, string(data), "123456789")

	// Test unmarshaling
	var unmarshaled Guild
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	
	// Verify the unmarshaled object matches
	assert.Equal(t, guild.TypeMeta, unmarshaled.TypeMeta)
	assert.Equal(t, guild.ObjectMeta.Name, unmarshaled.ObjectMeta.Name)
	assert.Equal(t, guild.Spec.ForProvider.Name, unmarshaled.Spec.ForProvider.Name)
	assert.Equal(t, guild.Status.AtProvider.ID, unmarshaled.Status.AtProvider.ID)
	
	// Verify pointer fields are handled correctly
	require.NotNil(t, unmarshaled.Spec.ForProvider.Region)
	assert.Equal(t, "us-east", *unmarshaled.Spec.ForProvider.Region)
	require.NotNil(t, unmarshaled.Spec.ForProvider.VerificationLevel)
	assert.Equal(t, 2, *unmarshaled.Spec.ForProvider.VerificationLevel)
}

func TestGuildParametersValidation(t *testing.T) {
	// Test creating valid guild parameters
	params := GuildParameters{
		Name:                        "Valid Guild Name",
		Region:                      stringPtr("us-west"),
		VerificationLevel:           intPtr(3),
		DefaultMessageNotifications: intPtr(0),
		ExplicitContentFilter:       intPtr(2),
		AFKTimeout:                  intPtr(300),
		SystemChannelFlags:          intPtr(0),
	}
	
	// Verify parameters can be created and accessed
	assert.Equal(t, "Valid Guild Name", params.Name)
	assert.Equal(t, "us-west", *params.Region)
	assert.Equal(t, 3, *params.VerificationLevel)
	assert.Equal(t, 0, *params.DefaultMessageNotifications)
	assert.Equal(t, 2, *params.ExplicitContentFilter)
	assert.Equal(t, 300, *params.AFKTimeout)
	assert.Equal(t, 0, *params.SystemChannelFlags)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}