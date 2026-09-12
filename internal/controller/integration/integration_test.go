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

package integration

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/pkg/errors"
	guildv1beta1 "github.com/rossigee/provider-discord/apis/guild/v1beta1"
	integrationv1beta1 "github.com/rossigee/provider-discord/apis/integration/v1beta1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MockIntegrationClient implements a mock Discord client for testing
type MockIntegrationClient struct {
	GetGuildIntegrationsFunc   func(ctx context.Context, guildID string) ([]discordclient.GuildIntegration, error)
	DeleteGuildIntegrationFunc func(ctx context.Context, guildID, integrationID string) error
}

// Ensure MockIntegrationClient implements IntegrationClient interface
var _ discordclient.IntegrationClient = (*MockIntegrationClient)(nil)

func (m *MockIntegrationClient) GetGuildIntegrations(ctx context.Context, guildID string) ([]discordclient.GuildIntegration, error) {
	if m.GetGuildIntegrationsFunc != nil {
		return m.GetGuildIntegrationsFunc(ctx, guildID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockIntegrationClient) DeleteGuildIntegration(ctx context.Context, guildID, integrationID string) error {
	if m.DeleteGuildIntegrationFunc != nil {
		return m.DeleteGuildIntegrationFunc(ctx, guildID, integrationID)
	}
	return errors.New("not implemented")
}

func TestObserve(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789012345678"
	integrationID := "987654321098765432"

	tests := []struct {
		name           string
		integration    *integrationv1beta1.Integration
		mockSetup      func(*MockIntegrationClient)
		expectedExists bool
		expectError    bool
	}{
		{
			name: "observe_integration_found",
			integration: &integrationv1beta1.Integration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: integrationID,
					},
				},
				Spec: integrationv1beta1.IntegrationSpec{
					ForProvider: integrationv1beta1.IntegrationParameters{
						GuildID:       guildID,
						IntegrationID: integrationID,
					},
				},
			},
			mockSetup: func(m *MockIntegrationClient) {
				m.GetGuildIntegrationsFunc = func(ctx context.Context, gid string) ([]discordclient.GuildIntegration, error) {
					return []discordclient.GuildIntegration{
						{
							ID:   integrationID,
							Name: "Test Integration",
							Type: "twitch",
						},
					}, nil
				}
			},
			expectedExists: true,
			expectError:    false,
		},
		{
			name: "observe_integration_not_found",
			integration: &integrationv1beta1.Integration{
				Spec: integrationv1beta1.IntegrationSpec{
					ForProvider: integrationv1beta1.IntegrationParameters{
						GuildID:       guildID,
						IntegrationID: integrationID,
					},
				},
			},
			mockSetup: func(m *MockIntegrationClient) {
				m.GetGuildIntegrationsFunc = func(ctx context.Context, gid string) ([]discordclient.GuildIntegration, error) {
					return []discordclient.GuildIntegration{}, nil
				}
			},
			expectedExists: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockIntegrationClient{}
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			e := &external{discord: mock}
			obs, err := e.Observe(ctx, tt.integration)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedExists, obs.ResourceExists)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	integration := &integrationv1beta1.Integration{
		Spec: integrationv1beta1.IntegrationSpec{
			ForProvider: integrationv1beta1.IntegrationParameters{
				GuildID: "123456789012345678",
			},
		},
	}

	e := &external{discord: &MockIntegrationClient{}}
	_, err := e.Create(ctx, integration)

	// Integrations can't be created via API
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be created")
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	integration := &integrationv1beta1.Integration{
		Spec: integrationv1beta1.IntegrationSpec{
			ForProvider: integrationv1beta1.IntegrationParameters{
				GuildID: "123456789012345678",
			},
		},
	}

	e := &external{discord: &MockIntegrationClient{}}
	_, err := e.Update(ctx, integration)

	// Integrations can't be modified via API
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be modified")
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789012345678"
	integrationID := "987654321098765432"

	tests := []struct {
		name        string
		integration *integrationv1beta1.Integration
		mockSetup   func(*MockIntegrationClient)
		expectError bool
	}{
		{
			name: "delete_success",
			integration: &integrationv1beta1.Integration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: integrationID,
					},
				},
				Spec: integrationv1beta1.IntegrationSpec{
					ForProvider: integrationv1beta1.IntegrationParameters{
						GuildID:       guildID,
						IntegrationID: integrationID,
					},
				},
			},
			mockSetup: func(m *MockIntegrationClient) {
				m.DeleteGuildIntegrationFunc = func(ctx context.Context, gid, iid string) error {
					return nil
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockIntegrationClient{}
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			e := &external{discord: mock}
			_, err := e.Delete(ctx, tt.integration)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDisconnect(t *testing.T) {
	e := &external{discord: &MockIntegrationClient{}}
	err := e.Disconnect(context.Background())
	assert.NoError(t, err)
}

func TestObservePermissionDenied(t *testing.T) {
	ctx := context.Background()

	integration := &integrationv1beta1.Integration{
		Spec: integrationv1beta1.IntegrationSpec{
			ForProvider: integrationv1beta1.IntegrationParameters{
				GuildID:       "123456789012345678",
				IntegrationID: "987654321098765432",
			},
		},
	}

	mockClient := &MockIntegrationClient{
		GetGuildIntegrationsFunc: func(ctx context.Context, guildID string) ([]discordclient.GuildIntegration, error) {
			return nil, errors.New("Discord API error: 403 - Missing Permissions")
		},
	}

	e := &external{discord: mockClient}
	_, err := e.Observe(ctx, integration)

	// Should return nil error (no retry on permission error)
	assert.NoError(t, err)
}

func TestDeletePermissionDenied(t *testing.T) {
	ctx := context.Background()

	integration := &integrationv1beta1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				meta.AnnotationKeyExternalName: "987654321098765432",
			},
		},
		Spec: integrationv1beta1.IntegrationSpec{
			ForProvider: integrationv1beta1.IntegrationParameters{
				GuildID:       "123456789012345678",
				IntegrationID: "987654321098765432",
			},
		},
	}

	mockClient := &MockIntegrationClient{
		DeleteGuildIntegrationFunc: func(ctx context.Context, guildID, integrationID string) error {
			return errors.New("Discord API error: 403 - Missing Permissions")
		},
	}

	e := &external{discord: mockClient}
	_, err := e.Delete(ctx, integration)

	// Should return nil error (no retry on permission error)
	assert.NoError(t, err)
}

func TestTypeAssertions(t *testing.T) {
	ctx := context.Background()

	// Test with wrong type
	wrongType := &guildv1beta1.Guild{}

	e := &external{discord: &MockIntegrationClient{}}

	_, err := e.Observe(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotIntegration)

	_, err = e.Create(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotIntegration)

	_, err = e.Update(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotIntegration)

	_, err = e.Delete(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotIntegration)
}
