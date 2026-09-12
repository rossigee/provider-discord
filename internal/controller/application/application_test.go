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

package application

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/pkg/errors"
	applicationv1beta1 "github.com/rossigee/provider-discord/apis/application/v1beta1"
	guildv1beta1 "github.com/rossigee/provider-discord/apis/guild/v1beta1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MockApplicationClient implements a mock Discord client for testing
type MockApplicationClient struct {
	GetCurrentApplicationFunc    func(ctx context.Context) (*discordclient.DiscordApplication, error)
	GetApplicationFunc           func(ctx context.Context, applicationID string) (*discordclient.DiscordApplication, error)
	ModifyCurrentApplicationFunc func(ctx context.Context, req *discordclient.ModifyCurrentApplicationRequest) (*discordclient.DiscordApplication, error)
}

// Ensure MockApplicationClient implements ApplicationClient interface
var _ discordclient.ApplicationClient = (*MockApplicationClient)(nil)

func (m *MockApplicationClient) GetCurrentApplication(ctx context.Context) (*discordclient.DiscordApplication, error) {
	if m.GetCurrentApplicationFunc != nil {
		return m.GetCurrentApplicationFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *MockApplicationClient) GetApplication(ctx context.Context, applicationID string) (*discordclient.DiscordApplication, error) {
	if m.GetApplicationFunc != nil {
		return m.GetApplicationFunc(ctx, applicationID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockApplicationClient) ModifyCurrentApplication(ctx context.Context, req *discordclient.ModifyCurrentApplicationRequest) (*discordclient.DiscordApplication, error) {
	if m.ModifyCurrentApplicationFunc != nil {
		return m.ModifyCurrentApplicationFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func TestObserve(t *testing.T) {
	ctx := context.Background()
	appID := "123456789012345678"

	tests := []struct {
		name             string
		app              *applicationv1beta1.Application
		mockSetup        func(*MockApplicationClient)
		expectedExists   bool
		expectedUpToDate bool
		expectError      bool
	}{
		{
			name: "observe_current_application_up_to_date",
			app: &applicationv1beta1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: appID,
					},
				},
				Spec: applicationv1beta1.ApplicationSpec{
					ForProvider: applicationv1beta1.ApplicationParameters{
						ApplicationID: "@me",
						Name:          strPtr("My App"),
						Description:   strPtr("Test app"),
					},
				},
			},
			mockSetup: func(m *MockApplicationClient) {
				m.GetCurrentApplicationFunc = func(ctx context.Context) (*discordclient.DiscordApplication, error) {
					return &discordclient.DiscordApplication{
						ID:          appID,
						Name:        "My App",
						Description: "Test app",
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: true,
			expectError:      false,
		},
		{
			name: "observe_current_application_needs_update",
			app: &applicationv1beta1.Application{
				Spec: applicationv1beta1.ApplicationSpec{
					ForProvider: applicationv1beta1.ApplicationParameters{
						ApplicationID: "@me",
						Name:          strPtr("My App"),
						Description:   strPtr("Updated description"),
					},
				},
			},
			mockSetup: func(m *MockApplicationClient) {
				m.GetCurrentApplicationFunc = func(ctx context.Context) (*discordclient.DiscordApplication, error) {
					return &discordclient.DiscordApplication{
						ID:          appID,
						Name:        "My App",
						Description: "Test app",
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "observe_application_by_id",
			app: &applicationv1beta1.Application{
				Spec: applicationv1beta1.ApplicationSpec{
					ForProvider: applicationv1beta1.ApplicationParameters{
						ApplicationID: appID,
					},
				},
			},
			mockSetup: func(m *MockApplicationClient) {
				m.GetApplicationFunc = func(ctx context.Context, id string) (*discordclient.DiscordApplication, error) {
					return &discordclient.DiscordApplication{
						ID:   id,
						Name: "Test App",
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: true,
			expectError:      false,
		},
		{
			name: "observe_application_not_found",
			app: &applicationv1beta1.Application{
				Spec: applicationv1beta1.ApplicationSpec{
					ForProvider: applicationv1beta1.ApplicationParameters{
						ApplicationID: appID,
					},
				},
			},
			mockSetup: func(m *MockApplicationClient) {
				m.GetApplicationFunc = func(ctx context.Context, id string) (*discordclient.DiscordApplication, error) {
					return nil, errors.New("application not found")
				}
			},
			expectedExists: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockApplicationClient{}
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			e := &external{discord: mock}
			obs, err := e.Observe(ctx, tt.app)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedExists, obs.ResourceExists)
				assert.Equal(t, tt.expectedUpToDate, obs.ResourceUpToDate)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	app := &applicationv1beta1.Application{
		Spec: applicationv1beta1.ApplicationSpec{
			ForProvider: applicationv1beta1.ApplicationParameters{
				ApplicationID: "@me",
			},
		},
	}

	e := &external{discord: &MockApplicationClient{}}
	_, err := e.Create(ctx, app)

	// Applications can't be created via API
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be created")
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	appID := "123456789012345678"

	tests := []struct {
		name        string
		app         *applicationv1beta1.Application
		mockSetup   func(*MockApplicationClient)
		expectError bool
	}{
		{
			name: "update_current_application_success",
			app: &applicationv1beta1.Application{
				Spec: applicationv1beta1.ApplicationSpec{
					ForProvider: applicationv1beta1.ApplicationParameters{
						ApplicationID: "@me",
						Name:          strPtr("Updated Name"),
						Description:   strPtr("Updated Description"),
					},
				},
			},
			mockSetup: func(m *MockApplicationClient) {
				m.ModifyCurrentApplicationFunc = func(ctx context.Context, req *discordclient.ModifyCurrentApplicationRequest) (*discordclient.DiscordApplication, error) {
					return &discordclient.DiscordApplication{
						ID:          appID,
						Name:        "Updated Name",
						Description: "Updated Description",
					}, nil
				}
			},
			expectError: false,
		},
		{
			name: "update_specific_application_fails",
			app: &applicationv1beta1.Application{
				Spec: applicationv1beta1.ApplicationSpec{
					ForProvider: applicationv1beta1.ApplicationParameters{
						ApplicationID: appID,
						Name:          strPtr("Name"),
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockApplicationClient{}
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			e := &external{discord: mock}
			_, err := e.Update(ctx, tt.app)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	app := &applicationv1beta1.Application{
		Spec: applicationv1beta1.ApplicationSpec{
			ForProvider: applicationv1beta1.ApplicationParameters{
				ApplicationID: "@me",
			},
		},
	}

	e := &external{discord: &MockApplicationClient{}}
	_, err := e.Delete(ctx, app)

	// Delete is a no-op
	assert.NoError(t, err)
}

func TestDisconnect(t *testing.T) {
	e := &external{discord: &MockApplicationClient{}}
	err := e.Disconnect(context.Background())
	assert.NoError(t, err)
}

func TestObservePermissionDenied(t *testing.T) {
	ctx := context.Background()

	app := &applicationv1beta1.Application{
		Spec: applicationv1beta1.ApplicationSpec{
			ForProvider: applicationv1beta1.ApplicationParameters{
				ApplicationID: "@me",
			},
		},
	}

	mockClient := &MockApplicationClient{
		GetCurrentApplicationFunc: func(ctx context.Context) (*discordclient.DiscordApplication, error) {
			return nil, errors.New("Discord API error: 403 - Missing Permissions")
		},
	}

	e := &external{discord: mockClient}
	_, err := e.Observe(ctx, app)

	// Should return nil error (no retry on permission error)
	assert.NoError(t, err)
	// Condition should be set to Unavailable
	assert.NotNil(t, app.Status.Conditions)
}

func TestObserveUnauthorized(t *testing.T) {
	ctx := context.Background()

	app := &applicationv1beta1.Application{
		Spec: applicationv1beta1.ApplicationSpec{
			ForProvider: applicationv1beta1.ApplicationParameters{
				ApplicationID: "@me",
			},
		},
	}

	mockClient := &MockApplicationClient{
		GetCurrentApplicationFunc: func(ctx context.Context) (*discordclient.DiscordApplication, error) {
			return nil, errors.New("Discord API error: 401 - Unauthorized")
		},
	}

	e := &external{discord: mockClient}
	_, err := e.Observe(ctx, app)

	// Should return nil error (no retry on auth error)
	assert.NoError(t, err)
}

func TestUpdatePermissionDenied(t *testing.T) {
	ctx := context.Background()

	app := &applicationv1beta1.Application{
		Spec: applicationv1beta1.ApplicationSpec{
			ForProvider: applicationv1beta1.ApplicationParameters{
				ApplicationID: "@me",
				Name:          strPtr("New Name"),
			},
		},
	}

	mockClient := &MockApplicationClient{
		ModifyCurrentApplicationFunc: func(ctx context.Context, req *discordclient.ModifyCurrentApplicationRequest) (*discordclient.DiscordApplication, error) {
			return nil, errors.New("Discord API error: 403 - Missing Permissions")
		},
	}

	e := &external{discord: mockClient}
	_, err := e.Update(ctx, app)

	// Should return nil error (no retry on permission error)
	assert.NoError(t, err)
}

func TestTypeAssertions(t *testing.T) {
	ctx := context.Background()

	// Test with wrong type
	wrongType := &guildv1beta1.Guild{}

	e := &external{discord: &MockApplicationClient{}}

	_, err := e.Observe(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotApplication)

	_, err = e.Create(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotApplication)

	_, err = e.Update(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotApplication)

	_, err = e.Delete(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotApplication)
}

// Helper functions
func strPtr(s string) *string {
	return &s
}
