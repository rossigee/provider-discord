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

func TestChannelDeepCopy(t *testing.T) {
	original := &Channel{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Channel",
			APIVersion: "discord.crossplane.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-channel",
			Namespace: "default",
		},
		Spec: ChannelSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy: xpv1.DeletionDelete,
			},
			ForProvider: ChannelParameters{
				Name:             "test-channel",
				Type:             0, // Text channel
				GuildID:          "123456789",
				Topic:            stringPtr("Test channel topic"),
				Position:         intPtr(1),
				ParentID:         stringPtr("987654321"),
				NSFW:             boolPtr(false),
				Bitrate:          intPtr(64000),
				UserLimit:        intPtr(10),
				RateLimitPerUser: intPtr(5),
			},
		},
		Status: ChannelStatus{
			ResourceStatus: xpv1.ResourceStatus{},
			AtProvider: ChannelObservation{
				ID:       "111222333",
				Name:     "test-channel",
				Type:     0,
				GuildID:  "123456789",
				Position: 1,
				ParentID: "987654321",
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
	copied.Spec.ForProvider.Name = "modified-channel"
	assert.NotEqual(t, original.Spec.ForProvider.Name, copied.Spec.ForProvider.Name)
}

func TestChannelListDeepCopy(t *testing.T) {
	original := &ChannelList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ChannelList",
			APIVersion: "discord.crossplane.io/v1alpha1",
		},
		ListMeta: metav1.ListMeta{
			ResourceVersion: "1",
		},
		Items: []Channel{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "channel1",
				},
				Spec: ChannelSpec{
					ForProvider: ChannelParameters{
						Name:    "Channel 1",
						Type:    0,
						GuildID: "123456789",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "channel2",
				},
				Spec: ChannelSpec{
					ForProvider: ChannelParameters{
						Name:    "Channel 2",
						Type:    2, // Voice channel
						GuildID: "123456789",
					},
				},
			},
		},
	}

	// Test DeepCopyObject
	obj := original.DeepCopyObject()
	copied, ok := obj.(*ChannelList)
	require.True(t, ok)
	
	// Verify they're not the same object
	assert.NotSame(t, original, copied)
	
	// Verify the content is the same
	assert.Equal(t, original.TypeMeta, copied.TypeMeta)
	assert.Equal(t, original.ListMeta, copied.ListMeta)
	assert.Len(t, copied.Items, len(original.Items))
	
	// Verify deep copy - modifying one shouldn't affect the other
	copied.Items[0].Spec.ForProvider.Name = "Modified Channel"
	assert.NotEqual(t, original.Items[0].Spec.ForProvider.Name, copied.Items[0].Spec.ForProvider.Name)
}

func TestChannelJSONMarshaling(t *testing.T) {
	channel := &Channel{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Channel",
			APIVersion: "discord.crossplane.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-channel",
		},
		Spec: ChannelSpec{
			ForProvider: ChannelParameters{
				Name:             "general",
				Type:             0, // Text channel
				GuildID:          "123456789",
				Topic:            stringPtr("General discussion"),
				Position:         intPtr(0),
				NSFW:             boolPtr(false),
				RateLimitPerUser: intPtr(0),
			},
		},
		Status: ChannelStatus{
			AtProvider: ChannelObservation{
				ID:       "111222333",
				Name:     "general",
				Type:     0,
				GuildID:  "123456789",
				Position: 0,
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(channel)
	require.NoError(t, err)
	assert.Contains(t, string(data), "general")
	assert.Contains(t, string(data), "111222333")
	assert.Contains(t, string(data), "123456789")

	// Test unmarshaling
	var unmarshaled Channel
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	
	// Verify the unmarshaled object matches
	assert.Equal(t, channel.TypeMeta, unmarshaled.TypeMeta)
	assert.Equal(t, channel.Name, unmarshaled.Name)
	assert.Equal(t, channel.Spec.ForProvider.Name, unmarshaled.Spec.ForProvider.Name)
	assert.Equal(t, channel.Spec.ForProvider.Type, unmarshaled.Spec.ForProvider.Type)
	assert.Equal(t, channel.Status.AtProvider.ID, unmarshaled.Status.AtProvider.ID)
	
	// Verify pointer fields are handled correctly
	require.NotNil(t, unmarshaled.Spec.ForProvider.Topic)
	assert.Equal(t, "General discussion", *unmarshaled.Spec.ForProvider.Topic)
	require.NotNil(t, unmarshaled.Spec.ForProvider.Position)
	assert.Equal(t, 0, *unmarshaled.Spec.ForProvider.Position)
	require.NotNil(t, unmarshaled.Spec.ForProvider.NSFW)
	assert.Equal(t, false, *unmarshaled.Spec.ForProvider.NSFW)
}

func TestChannelParametersValidation(t *testing.T) {
	// Test creating valid channel parameters for different channel types
	
	// Text channel
	textParams := ChannelParameters{
		Name:             "general-chat",
		Type:             0, // Text
		GuildID:          "123456789",
		Topic:            stringPtr("General discussion channel"),
		Position:         intPtr(1),
		ParentID:         stringPtr("category123"),
		NSFW:             boolPtr(false),
		RateLimitPerUser: intPtr(5),
	}
	
	assert.Equal(t, "general-chat", textParams.Name)
	assert.Equal(t, 0, textParams.Type)
	assert.Equal(t, "123456789", textParams.GuildID)
	assert.Equal(t, "General discussion channel", *textParams.Topic)
	assert.Equal(t, 1, *textParams.Position)
	assert.Equal(t, false, *textParams.NSFW)
	assert.Equal(t, 5, *textParams.RateLimitPerUser)

	// Voice channel
	voiceParams := ChannelParameters{
		Name:      "General Voice",
		Type:      2, // Voice
		GuildID:   "123456789",
		Position:  intPtr(2),
		ParentID:  stringPtr("category123"),
		Bitrate:   intPtr(64000),
		UserLimit: intPtr(10),
	}
	
	assert.Equal(t, "General Voice", voiceParams.Name)
	assert.Equal(t, 2, voiceParams.Type)
	assert.Equal(t, "123456789", voiceParams.GuildID)
	assert.Equal(t, 2, *voiceParams.Position)
	assert.Equal(t, 64000, *voiceParams.Bitrate)
	assert.Equal(t, 10, *voiceParams.UserLimit)

	// Category channel
	categoryParams := ChannelParameters{
		Name:     "General Category",
		Type:     4, // Category
		GuildID:  "123456789",
		Position: intPtr(0),
	}
	
	assert.Equal(t, "General Category", categoryParams.Name)
	assert.Equal(t, 4, categoryParams.Type)
	assert.Equal(t, "123456789", categoryParams.GuildID)
	assert.Equal(t, 0, *categoryParams.Position)
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