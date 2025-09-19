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

package channel

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	channelv1alpha1 "github.com/rossigee/provider-discord/apis/channel/v1alpha1"
	guildv1alpha1 "github.com/rossigee/provider-discord/apis/guild/v1alpha1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
)

// MockChannelClient implements a mock Discord client for testing
type MockChannelClient struct {
	CreateChannelFunc func(ctx context.Context, req *discordclient.CreateChannelRequest) (*discordclient.Channel, error)
	GetChannelFunc    func(ctx context.Context, channelID string) (*discordclient.Channel, error)
	ModifyChannelFunc func(ctx context.Context, channelID string, req *discordclient.ModifyChannelRequest) (*discordclient.Channel, error)
	DeleteChannelFunc func(ctx context.Context, channelID string) error
}

// Ensure MockChannelClient implements ChannelClient interface
var _ discordclient.ChannelClient = (*MockChannelClient)(nil)

func (m *MockChannelClient) CreateChannel(ctx context.Context, req *discordclient.CreateChannelRequest) (*discordclient.Channel, error) {
	if m.CreateChannelFunc != nil {
		return m.CreateChannelFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockChannelClient) GetChannel(ctx context.Context, channelID string) (*discordclient.Channel, error) {
	if m.GetChannelFunc != nil {
		return m.GetChannelFunc(ctx, channelID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockChannelClient) ModifyChannel(ctx context.Context, channelID string, req *discordclient.ModifyChannelRequest) (*discordclient.Channel, error) {
	if m.ModifyChannelFunc != nil {
		return m.ModifyChannelFunc(ctx, channelID, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockChannelClient) DeleteChannel(ctx context.Context, channelID string) error {
	if m.DeleteChannelFunc != nil {
		return m.DeleteChannelFunc(ctx, channelID)
	}
	return errors.New("not implemented")
}

func TestObserve(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789012345678"  // Valid Discord snowflake ID (18 digits)
	channelID := "987654321098765432" // Valid Discord snowflake ID (18 digits)
	
	tests := []struct {
		name                string
		channel             *channelv1alpha1.Channel
		mockSetup           func(*MockChannelClient)
		expectedExists      bool
		expectedUpToDate    bool
		expectError         bool
	}{
		{
			name: "channel exists and up to date",
			channel: &channelv1alpha1.Channel{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: channelID,
					},
				},
				Spec: channelv1alpha1.ChannelSpec{
					ForProvider: channelv1alpha1.ChannelParameters{
						Name:    "test-channel",
						Type:    0, // Text channel
						GuildID: guildID,
					},
				},
			},
			mockSetup: func(m *MockChannelClient) {
				m.GetChannelFunc = func(ctx context.Context, channelID string) (*discordclient.Channel, error) {
					return &discordclient.Channel{
						ID:      channelID,
						Name:    "test-channel",
						Type:    0,
						GuildID: guildID,
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: true,
			expectError:      false,
		},
		{
			name: "channel exists but needs update",
			channel: &channelv1alpha1.Channel{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: channelID,
					},
				},
				Spec: channelv1alpha1.ChannelSpec{
					ForProvider: channelv1alpha1.ChannelParameters{
						Name:    "updated-channel",
						Type:    0,
						GuildID: guildID,
					},
				},
			},
			mockSetup: func(m *MockChannelClient) {
				m.GetChannelFunc = func(ctx context.Context, channelID string) (*discordclient.Channel, error) {
					return &discordclient.Channel{
						ID:      channelID,
						Name:    "old-channel",
						Type:    0,
						GuildID: guildID,
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "channel does not exist",
			channel: &channelv1alpha1.Channel{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: channelID,
					},
				},
				Spec: channelv1alpha1.ChannelSpec{
					ForProvider: channelv1alpha1.ChannelParameters{
						Name:    "test-channel",
						Type:    0,
						GuildID: guildID,
					},
				},
			},
			mockSetup: func(m *MockChannelClient) {
				m.GetChannelFunc = func(ctx context.Context, channelID string) (*discordclient.Channel, error) {
					return nil, nil // Return nil for not found
				}
			},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "no external name set",
			channel: &channelv1alpha1.Channel{
				Spec: channelv1alpha1.ChannelSpec{
					ForProvider: channelv1alpha1.ChannelParameters{
						Name:    "test-channel",
						Type:    0,
						GuildID: guildID,
					},
				},
			},
			mockSetup: func(m *MockChannelClient) {
				// No setup needed for this test
			},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "invalid external name (not Discord ID)",
			channel: &channelv1alpha1.Channel{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-resource-name",
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "test-resource-name", // Invalid - not a Discord snowflake
					},
				},
				Spec: channelv1alpha1.ChannelSpec{
					ForProvider: channelv1alpha1.ChannelParameters{
						Name:    "test-channel",
						Type:    0,
						GuildID: guildID,
					},
				},
			},
			mockSetup: func(m *MockChannelClient) {
				// No setup needed - should not call GetChannel for invalid IDs
			},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockChannelClient{}
			tc.mockSetup(mockClient)

			e := &external{service: mockClient, kube: nil}
			obs, err := e.Observe(ctx, tc.channel)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedExists, obs.ResourceExists)
				assert.Equal(t, tc.expectedUpToDate, obs.ResourceUpToDate)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789012345678"  // Valid Discord snowflake ID
	channelID := "987654321098765432" // Valid Discord snowflake ID

	mockClient := &MockChannelClient{
		CreateChannelFunc: func(ctx context.Context, req *discordclient.CreateChannelRequest) (*discordclient.Channel, error) {
			return &discordclient.Channel{
				ID:      channelID,
				Name:    req.Name,
				Type:    req.Type,
				GuildID: guildID,
			}, nil
		},
	}

	channel := &channelv1alpha1.Channel{
		Spec: channelv1alpha1.ChannelSpec{
			ForProvider: channelv1alpha1.ChannelParameters{
				Name:    "test-channel",
				Type:    0,
				GuildID: guildID,
			},
		},
	}

	e := &external{service: mockClient, kube: nil}
	_, err := e.Create(ctx, channel)

	require.NoError(t, err)
	assert.Equal(t, channelID, meta.GetExternalName(channel))
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789012345678"  // Valid Discord snowflake ID
	channelID := "987654321098765432" // Valid Discord snowflake ID

	mockClient := &MockChannelClient{
		ModifyChannelFunc: func(ctx context.Context, channelID string, req *discordclient.ModifyChannelRequest) (*discordclient.Channel, error) {
			return &discordclient.Channel{
				ID:      channelID,
				Name:    *req.Name,
				Type:    0,
				GuildID: guildID,
			}, nil
		},
	}

	channel := &channelv1alpha1.Channel{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				meta.AnnotationKeyExternalName: channelID,
			},
		},
		Spec: channelv1alpha1.ChannelSpec{
			ForProvider: channelv1alpha1.ChannelParameters{
				Name:    "updated-channel",
				Type:    0,
				GuildID: guildID,
			},
		},
	}

	e := &external{service: mockClient, kube: nil}
	_, err := e.Update(ctx, channel)

	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	channelID := "987654321098765432" // Valid Discord snowflake ID

	tests := []struct {
		name        string
		channel     *channelv1alpha1.Channel
		mockSetup   func(*MockChannelClient)
		expectError bool
	}{
		{
			name: "successful delete",
			channel: &channelv1alpha1.Channel{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: channelID,
					},
				},
			},
			mockSetup: func(m *MockChannelClient) {
				m.DeleteChannelFunc = func(ctx context.Context, channelID string) error {
					return nil
				}
			},
			expectError: false,
		},
		{
			name: "delete non-existent channel",
			channel: &channelv1alpha1.Channel{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: channelID,
					},
				},
			},
			mockSetup: func(m *MockChannelClient) {
				m.DeleteChannelFunc = func(ctx context.Context, channelID string) error {
					return nil // Should not error for non-existent channel
				}
			},
			expectError: false, // Should not error for non-existent channel
		},
		{
			name: "no external name",
			channel: &channelv1alpha1.Channel{
				ObjectMeta: metav1.ObjectMeta{},
			},
			mockSetup: func(m *MockChannelClient) {
				m.DeleteChannelFunc = func(ctx context.Context, channelID string) error {
					return nil // Should be called with empty string
				}
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockChannelClient{}
			tc.mockSetup(mockClient)

			e := &external{service: mockClient, kube: nil}
			_, err := e.Delete(ctx, tc.channel)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDisconnect(t *testing.T) {
	e := &external{service: &MockChannelClient{}, kube: nil}
	err := e.Disconnect(context.Background())
	assert.NoError(t, err)
}

func TestTypeAssertions(t *testing.T) {
	tests := []struct {
		name     string
		resource interface{}
		method   string
	}{
		{
			name:     "invalid resource type in Observe",
			resource: &channelv1alpha1.Channel{},
			method:   "Observe",
		},
		{
			name:     "invalid resource type in Create",
			resource: &channelv1alpha1.Channel{},
			method:   "Create",
		},
		{
			name:     "invalid resource type in Update",
			resource: &channelv1alpha1.Channel{},
			method:   "Update",
		},
		{
			name:     "invalid resource type in Delete",
			resource: &channelv1alpha1.Channel{},
			method:   "Delete",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockChannelClient{}
			e := &external{service: mockClient, kube: nil}

			// Test with a non-Channel resource (should fail type assertion)
			invalidResource := &guildv1alpha1.Guild{} // Use Guild instead of Channel

			ctx := context.Background()
			switch tc.method {
			case "Observe":
				_, err := e.Observe(ctx, invalidResource)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errNotChannel)
			case "Create":
				_, err := e.Create(ctx, invalidResource)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errNotChannel)
			case "Update":
				_, err := e.Update(ctx, invalidResource)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errNotChannel)
			case "Delete":
				_, err := e.Delete(ctx, invalidResource)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errNotChannel)
			}
		})
	}
}

// Helper functions
// Helper functions removed - unused