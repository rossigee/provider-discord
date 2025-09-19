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

package guild

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	guildv1alpha1 "github.com/rossigee/provider-discord/apis/guild/v1alpha1"
	channelv1alpha1 "github.com/rossigee/provider-discord/apis/channel/v1alpha1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
)

// MockGuildClient implements a mock Discord client for testing
type MockGuildClient struct {
	CreateGuildFunc func(ctx context.Context, req *discordclient.CreateGuildRequest) (*discordclient.Guild, error)
	GetGuildFunc    func(ctx context.Context, guildID string) (*discordclient.Guild, error)
	ModifyGuildFunc func(ctx context.Context, guildID string, req *discordclient.ModifyGuildRequest) (*discordclient.Guild, error)
	DeleteGuildFunc func(ctx context.Context, guildID string) error
	ListGuildsFunc  func(ctx context.Context) ([]discordclient.Guild, error)
}

// Ensure MockGuildClient implements GuildClient interface
var _ discordclient.GuildClient = (*MockGuildClient)(nil)

func (m *MockGuildClient) CreateGuild(ctx context.Context, req *discordclient.CreateGuildRequest) (*discordclient.Guild, error) {
	if m.CreateGuildFunc != nil {
		return m.CreateGuildFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockGuildClient) GetGuild(ctx context.Context, guildID string) (*discordclient.Guild, error) {
	if m.GetGuildFunc != nil {
		return m.GetGuildFunc(ctx, guildID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockGuildClient) ModifyGuild(ctx context.Context, guildID string, req *discordclient.ModifyGuildRequest) (*discordclient.Guild, error) {
	if m.ModifyGuildFunc != nil {
		return m.ModifyGuildFunc(ctx, guildID, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockGuildClient) DeleteGuild(ctx context.Context, guildID string) error {
	if m.DeleteGuildFunc != nil {
		return m.DeleteGuildFunc(ctx, guildID)
	}
	return errors.New("not implemented")
}

func (m *MockGuildClient) ListGuilds(ctx context.Context) ([]discordclient.Guild, error) {
	if m.ListGuildsFunc != nil {
		return m.ListGuildsFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

func TestObserve(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789"
	
	tests := []struct {
		name                string
		guild               *guildv1alpha1.Guild
		mockSetup           func(*MockGuildClient)
		expectedExists      bool
		expectedUpToDate    bool
		expectError         bool
	}{
		{
			name: "guild exists and up to date",
			guild: &guildv1alpha1.Guild{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: guildID,
					},
				},
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name:                        "Test Guild",
						Region:                      strPtr("us-east"),
						VerificationLevel:           intPtr(1),
						DefaultMessageNotifications: intPtr(1),
						ExplicitContentFilter:       intPtr(1),
						AFKTimeout:                  intPtr(300),
						SystemChannelFlags:          intPtr(0),
					},
				},
			},
			mockSetup: func(m *MockGuildClient) {
				m.GetGuildFunc = func(ctx context.Context, guildID string) (*discordclient.Guild, error) {
					return &discordclient.Guild{
						ID:                          guildID,
						Name:                        "Test Guild",
						Region:                      strPtr("us-east"),
						VerificationLevel:           1,
						DefaultMessageNotifications: 1,
						ExplicitContentFilter:       1,
						AFKTimeout:                  300,
						SystemChannelFlags:          0,
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: true,
			expectError:      false,
		},
		{
			name: "guild exists but needs update",
			guild: &guildv1alpha1.Guild{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: guildID,
					},
				},
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name:                        "Updated Guild",
						Region:                      strPtr("us-west"),
						VerificationLevel:           intPtr(2),
						DefaultMessageNotifications: intPtr(0),
					},
				},
			},
			mockSetup: func(m *MockGuildClient) {
				m.GetGuildFunc = func(ctx context.Context, guildID string) (*discordclient.Guild, error) {
					return &discordclient.Guild{
						ID:                          guildID,
						Name:                        "Old Guild",
						Region:                      strPtr("us-east"),
						VerificationLevel:           1,
						DefaultMessageNotifications: 1,
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "guild does not exist",
			guild: &guildv1alpha1.Guild{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: guildID,
					},
				},
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name: "Test Guild",
					},
				},
			},
			mockSetup: func(m *MockGuildClient) {
				m.GetGuildFunc = func(ctx context.Context, guildID string) (*discordclient.Guild, error) {
					return nil, nil // Return nil for not found
				}
			},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "no external name set",
			guild: &guildv1alpha1.Guild{
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name: "Test Guild",
					},
				},
			},
			mockSetup: func(m *MockGuildClient) {
				// No setup needed for this test
			},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockGuildClient{}
			tc.mockSetup(mockClient)

			e := &external{service: mockClient, kube: nil}
			obs, err := e.Observe(ctx, tc.guild)

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
	guildID := "123456789"

	mockClient := &MockGuildClient{
		CreateGuildFunc: func(ctx context.Context, req *discordclient.CreateGuildRequest) (*discordclient.Guild, error) {
			return &discordclient.Guild{
				ID:                          guildID,
				Name:                        req.Name,
				Region:                      req.Region,
				VerificationLevel:           *req.VerificationLevel,
				DefaultMessageNotifications: *req.DefaultMessageNotifications,
				ExplicitContentFilter:       *req.ExplicitContentFilter,
				AFKTimeout:                  *req.AFKTimeout,
				SystemChannelFlags:          *req.SystemChannelFlags,
			}, nil
		},
	}

	guild := &guildv1alpha1.Guild{
		Spec: guildv1alpha1.GuildSpec{
			ForProvider: guildv1alpha1.GuildParameters{
				Name:                        "Test Guild",
				Region:                      strPtr("us-east"),
				VerificationLevel:           intPtr(1),
				DefaultMessageNotifications: intPtr(1),
				ExplicitContentFilter:       intPtr(1),
				AFKTimeout:                  intPtr(300),
				SystemChannelFlags:          intPtr(0),
			},
		},
	}

	e := &external{service: mockClient, kube: nil}
	_, err := e.Create(ctx, guild)

	require.NoError(t, err)
	assert.Equal(t, guildID, meta.GetExternalName(guild))
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789"
	
	tests := []struct {
		name        string
		guild       *guildv1alpha1.Guild
		mockSetup   func(*MockGuildClient)
		expectError bool
		expectUpdate bool
	}{
		{
			name: "update name",
			guild: &guildv1alpha1.Guild{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: guildID,
					},
				},
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name: "Updated Guild",
					},
				},
				Status: guildv1alpha1.GuildStatus{
					AtProvider: guildv1alpha1.GuildObservation{
						Name: "Old Guild", // Different from spec, so update needed
					},
				},
			},
			mockSetup: func(m *MockGuildClient) {
				m.ModifyGuildFunc = func(ctx context.Context, guildID string, req *discordclient.ModifyGuildRequest) (*discordclient.Guild, error) {
					assert.NotNil(t, req.Name)
					assert.Equal(t, "Updated Guild", *req.Name)
					return &discordclient.Guild{
						ID:   guildID,
						Name: *req.Name,
					}, nil
				}
			},
			expectError: false,
			expectUpdate: true,
		},
		{
			name: "update region",
			guild: &guildv1alpha1.Guild{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: guildID,
					},
				},
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name:   "Test Guild",
						Region: strPtr("us-west"),
					},
				},
				Status: guildv1alpha1.GuildStatus{
					AtProvider: guildv1alpha1.GuildObservation{
						Name:   "Test Guild",
						Region: "us-east", // Different from spec
					},
				},
			},
			mockSetup: func(m *MockGuildClient) {
				m.ModifyGuildFunc = func(ctx context.Context, guildID string, req *discordclient.ModifyGuildRequest) (*discordclient.Guild, error) {
					assert.NotNil(t, req.Region)
					assert.Equal(t, "us-west", *req.Region)
					return &discordclient.Guild{
						ID:     guildID,
						Name:   "Test Guild",
						Region: req.Region,
					}, nil
				}
			},
			expectError: false,
			expectUpdate: true,
		},
		{
			name: "update multiple fields",
			guild: &guildv1alpha1.Guild{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: guildID,
					},
				},
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name:                        "Updated Guild",
						VerificationLevel:           intPtr(2),
						DefaultMessageNotifications: intPtr(1),
						ExplicitContentFilter:       intPtr(2),
						AFKTimeout:                  intPtr(600),
						SystemChannelFlags:          intPtr(1),
					},
				},
				Status: guildv1alpha1.GuildStatus{
					AtProvider: guildv1alpha1.GuildObservation{
						Name:                        "Old Guild",
						VerificationLevel:           1,
						DefaultMessageNotifications: 0,
						ExplicitContentFilter:       1,
						AFKTimeout:                  300,
						SystemChannelFlags:          0,
					},
				},
			},
			mockSetup: func(m *MockGuildClient) {
				m.ModifyGuildFunc = func(ctx context.Context, guildID string, req *discordclient.ModifyGuildRequest) (*discordclient.Guild, error) {
					assert.NotNil(t, req.Name)
					assert.Equal(t, "Updated Guild", *req.Name)
					assert.NotNil(t, req.VerificationLevel)
					assert.Equal(t, 2, *req.VerificationLevel)
					assert.NotNil(t, req.DefaultMessageNotifications)
					assert.Equal(t, 1, *req.DefaultMessageNotifications)
					assert.NotNil(t, req.ExplicitContentFilter)
					assert.Equal(t, 2, *req.ExplicitContentFilter)
					assert.NotNil(t, req.AFKTimeout)
					assert.Equal(t, 600, *req.AFKTimeout)
					assert.NotNil(t, req.SystemChannelFlags)
					assert.Equal(t, 1, *req.SystemChannelFlags)
					return &discordclient.Guild{
						ID:   guildID,
						Name: *req.Name,
					}, nil
				}
			},
			expectError: false,
			expectUpdate: true,
		},
		{
			name: "no update needed",
			guild: &guildv1alpha1.Guild{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: guildID,
					},
				},
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name: "Test Guild",
					},
				},
				Status: guildv1alpha1.GuildStatus{
					AtProvider: guildv1alpha1.GuildObservation{
						Name: "Test Guild", // Same as spec, no update needed
					},
				},
			},
			mockSetup: func(m *MockGuildClient) {
				// ModifyGuildFunc should not be called
			},
			expectError: false,
			expectUpdate: false,
		},
		{
			name: "update fails",
			guild: &guildv1alpha1.Guild{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: guildID,
					},
				},
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name: "Updated Guild",
					},
				},
				Status: guildv1alpha1.GuildStatus{
					AtProvider: guildv1alpha1.GuildObservation{
						Name: "Old Guild",
					},
				},
			},
			mockSetup: func(m *MockGuildClient) {
				m.ModifyGuildFunc = func(ctx context.Context, guildID string, req *discordclient.ModifyGuildRequest) (*discordclient.Guild, error) {
					return nil, errors.New("update failed")
				}
			},
			expectError: true,
			expectUpdate: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockGuildClient{}
			tc.mockSetup(mockClient)

			e := &external{service: mockClient, kube: nil}
			_, err := e.Update(ctx, tc.guild)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789"

	tests := []struct {
		name        string
		guild       *guildv1alpha1.Guild
		mockSetup   func(*MockGuildClient)
		expectError bool
	}{
		{
			name: "successful delete",
			guild: &guildv1alpha1.Guild{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: guildID,
					},
				},
			},
			mockSetup: func(m *MockGuildClient) {
				m.DeleteGuildFunc = func(ctx context.Context, guildID string) error {
					return nil
				}
			},
			expectError: false,
		},
		{
			name: "delete non-existent guild", 
			guild: &guildv1alpha1.Guild{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: guildID,
					},
				},
			},
			mockSetup: func(m *MockGuildClient) {
				m.DeleteGuildFunc = func(ctx context.Context, guildID string) error {
					return errors.New("guild not found")
				}
			},
			expectError: true, // Guild delete should error if API returns error
		},
		{
			name: "no external name",
			guild: &guildv1alpha1.Guild{
				ObjectMeta: metav1.ObjectMeta{},
			},
			mockSetup: func(m *MockGuildClient) {
				m.DeleteGuildFunc = func(ctx context.Context, guildID string) error {
					return nil // Should be called with empty string
				}
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockGuildClient{}
			tc.mockSetup(mockClient)

			e := &external{service: mockClient}
			_, err := e.Delete(ctx, tc.guild)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDisconnect(t *testing.T) {
	e := &external{service: &MockGuildClient{}}
	err := e.Disconnect(context.Background())
	assert.NoError(t, err)
}

func TestTypeAssertions(t *testing.T) {
	tests := []struct {
		name     string
		method   string
	}{
		{
			name:   "invalid resource type in Observe",
			method: "Observe",
		},
		{
			name:   "invalid resource type in Create",
			method: "Create",
		},
		{
			name:   "invalid resource type in Update",
			method: "Update",
		},
		{
			name:   "invalid resource type in Delete", 
			method: "Delete",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockGuildClient{}
			e := &external{service: mockClient}

			// Test with a non-Guild resource (should fail type assertion)
			invalidResource := &channelv1alpha1.Channel{}

			ctx := context.Background()
			switch tc.method {
			case "Observe":
				_, err := e.Observe(ctx, invalidResource)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errNotGuild)
			case "Create":
				_, err := e.Create(ctx, invalidResource)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errNotGuild)
			case "Update":
				_, err := e.Update(ctx, invalidResource)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errNotGuild)
			case "Delete":
				_, err := e.Delete(ctx, invalidResource)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errNotGuild)
			}
		})
	}
}

func TestIsUpToDate(t *testing.T) {
	tests := []struct {
		name     string
		cr       *guildv1alpha1.Guild
		guild    *discordclient.Guild
		expected bool
	}{
		{
			name: "up to date",
			cr: &guildv1alpha1.Guild{
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name:                        "Test Guild",
						Region:                      strPtr("us-east"),
						VerificationLevel:           intPtr(1),
						DefaultMessageNotifications: intPtr(1),
						ExplicitContentFilter:       intPtr(1),
						AFKTimeout:                  intPtr(300),
						SystemChannelFlags:          intPtr(0),
					},
				},
			},
			guild: &discordclient.Guild{
				Name:                        "Test Guild",
				Region:                      strPtr("us-east"),
				VerificationLevel:           1,
				DefaultMessageNotifications: 1,
				ExplicitContentFilter:       1,
				AFKTimeout:                  300,
				SystemChannelFlags:          0,
			},
			expected: true,
		},
		{
			name: "name needs update",
			cr: &guildv1alpha1.Guild{
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name: "New Name",
					},
				},
			},
			guild: &discordclient.Guild{
				Name: "Old Name",
			},
			expected: false,
		},
		{
			name: "region needs update",
			cr: &guildv1alpha1.Guild{
				Spec: guildv1alpha1.GuildSpec{
					ForProvider: guildv1alpha1.GuildParameters{
						Name:   "Test Guild",
						Region: strPtr("us-west"),
					},
				},
			},
			guild: &discordclient.Guild{
				Name:   "Test Guild",
				Region: strPtr("us-east"),
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := &external{}
			result := e.isUpToDate(tc.cr, tc.guild)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}