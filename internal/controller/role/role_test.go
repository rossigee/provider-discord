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

package role

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/crossplane-runtime/pkg/meta"

	"github.com/crossplane-contrib/provider-discord/apis/role/v1alpha1"
	guildv1alpha1 "github.com/crossplane-contrib/provider-discord/apis/guild/v1alpha1"
	discordclient "github.com/crossplane-contrib/provider-discord/internal/clients"
)

// MockDiscordClient implements a mock Discord client for testing
type MockDiscordClient struct {
	CreateRoleFunc func(ctx context.Context, guildID string, req discordclient.CreateRoleRequest) (*discordclient.Role, error)
	GetRoleFunc    func(ctx context.Context, guildID, roleID string) (*discordclient.Role, error)
	ModifyRoleFunc func(ctx context.Context, guildID, roleID string, req discordclient.ModifyRoleRequest) (*discordclient.Role, error)
	DeleteRoleFunc func(ctx context.Context, guildID, roleID string) error
}

// Ensure MockDiscordClient implements RoleClient interface
var _ discordclient.RoleClient = (*MockDiscordClient)(nil)

func (m *MockDiscordClient) CreateRole(ctx context.Context, guildID string, req discordclient.CreateRoleRequest) (*discordclient.Role, error) {
	if m.CreateRoleFunc != nil {
		return m.CreateRoleFunc(ctx, guildID, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDiscordClient) GetRole(ctx context.Context, guildID, roleID string) (*discordclient.Role, error) {
	if m.GetRoleFunc != nil {
		return m.GetRoleFunc(ctx, guildID, roleID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDiscordClient) ModifyRole(ctx context.Context, guildID, roleID string, req discordclient.ModifyRoleRequest) (*discordclient.Role, error) {
	if m.ModifyRoleFunc != nil {
		return m.ModifyRoleFunc(ctx, guildID, roleID, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockDiscordClient) DeleteRole(ctx context.Context, guildID, roleID string) error {
	if m.DeleteRoleFunc != nil {
		return m.DeleteRoleFunc(ctx, guildID, roleID)
	}
	return errors.New("not implemented")
}

func TestObserve(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789"
	roleID := "987654321"
	
	tests := []struct {
		name           string
		role           *v1alpha1.Role
		mockSetup      func(*MockDiscordClient)
		expectedExists bool
		expectedUpToDate bool
		expectError    bool
	}{
		{
			name: "role exists and up to date",
			role: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: roleID,
					},
				},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						Name:    "Test Role",
						GuildID: guildID,
						Color:   intPtr(16711680),
						Hoist:   boolPtr(true),
					},
				},
			},
			mockSetup: func(m *MockDiscordClient) {
				m.GetRoleFunc = func(ctx context.Context, gID, rID string) (*discordclient.Role, error) {
					assert.Equal(t, guildID, gID)
					assert.Equal(t, roleID, rID)
					return &discordclient.Role{
						ID:       roleID,
						Name:     "Test Role",
						Color:    16711680,
						Hoist:    true,
						Position: 1,
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: true,
			expectError:      false,
		},
		{
			name: "role exists but needs update",
			role: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: roleID,
					},
				},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						Name:    "Updated Role",
						GuildID: guildID,
						Color:   intPtr(255),
					},
				},
			},
			mockSetup: func(m *MockDiscordClient) {
				m.GetRoleFunc = func(ctx context.Context, gID, rID string) (*discordclient.Role, error) {
					return &discordclient.Role{
						ID:    roleID,
						Name:  "Test Role",
						Color: 16711680,
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "role does not exist",
			role: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: roleID,
					},
				},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						Name:    "Test Role",
						GuildID: guildID,
					},
				},
			},
			mockSetup: func(m *MockDiscordClient) {
				m.GetRoleFunc = func(ctx context.Context, gID, rID string) (*discordclient.Role, error) {
					return nil, errors.New("role not found")
				}
			},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "no external name set",
			role: &v1alpha1.Role{
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						Name:    "Test Role",
						GuildID: guildID,
					},
				},
			},
			mockSetup: func(m *MockDiscordClient) {
				// No setup needed
			},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockDiscordClient{}
			tt.mockSetup(mockClient)
			
			e := &external{discord: mockClient}
			
			obs, err := e.Observe(ctx, tt.role)
			
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.Equal(t, tt.expectedExists, obs.ResourceExists)
			assert.Equal(t, tt.expectedUpToDate, obs.ResourceUpToDate)
		})
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789"
	roleID := "987654321"

	role := &v1alpha1.Role{
		Spec: v1alpha1.RoleSpec{
			ForProvider: v1alpha1.RoleParameters{
				Name:        "Test Role",
				GuildID:     guildID,
				Color:       intPtr(16711680),
				Hoist:       boolPtr(true),
				Mentionable: boolPtr(false),
				Permissions: stringPtr("1234567890"),
			},
		},
	}

	mockClient := &MockDiscordClient{
		CreateRoleFunc: func(ctx context.Context, gID string, req discordclient.CreateRoleRequest) (*discordclient.Role, error) {
			assert.Equal(t, guildID, gID)
			assert.Equal(t, "Test Role", req.Name)
			assert.Equal(t, 16711680, *req.Color)
			assert.Equal(t, true, *req.Hoist)
			assert.Equal(t, false, *req.Mentionable)
			assert.Equal(t, "1234567890", *req.Permissions)
			
			return &discordclient.Role{
				ID:          roleID,
				Name:        "Test Role",
				Color:       16711680,
				Hoist:       true,
				Mentionable: false,
				Permissions: "1234567890",
			}, nil
		},
	}

	e := &external{discord: mockClient}

	_, err := e.Create(ctx, role)
	require.NoError(t, err)

	// Check that external name was set
	assert.Equal(t, roleID, meta.GetExternalName(role))
	assert.Equal(t, roleID, role.Status.AtProvider.ID)
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789"
	roleID := "987654321"

	role := &v1alpha1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				meta.AnnotationKeyExternalName: roleID,
			},
		},
		Spec: v1alpha1.RoleSpec{
			ForProvider: v1alpha1.RoleParameters{
				Name:     "Updated Role",
				GuildID:  guildID,
				Color:    intPtr(255),
				Position: intPtr(2),
			},
		},
	}

	mockClient := &MockDiscordClient{
		ModifyRoleFunc: func(ctx context.Context, gID, rID string, req discordclient.ModifyRoleRequest) (*discordclient.Role, error) {
			assert.Equal(t, guildID, gID)
			assert.Equal(t, roleID, rID)
			assert.Equal(t, "Updated Role", *req.Name)
			assert.Equal(t, 255, *req.Color)
			assert.Equal(t, 2, *req.Position)
			
			return &discordclient.Role{
				ID:       roleID,
				Name:     "Updated Role",
				Color:    255,
				Position: 2,
			}, nil
		},
	}

	e := &external{discord: mockClient}

	_, err := e.Update(ctx, role)
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	guildID := "123456789"
	roleID := "987654321"

	tests := []struct {
		name        string
		role        *v1alpha1.Role
		mockSetup   func(*MockDiscordClient)
		expectError bool
	}{
		{
			name: "successful delete",
			role: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: roleID,
					},
				},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						GuildID: guildID,
					},
				},
			},
			mockSetup: func(m *MockDiscordClient) {
				m.DeleteRoleFunc = func(ctx context.Context, gID, rID string) error {
					assert.Equal(t, guildID, gID)
					assert.Equal(t, roleID, rID)
					return nil
				}
			},
			expectError: false,
		},
		{
			name: "delete non-existent role",
			role: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: roleID,
					},
				},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						GuildID: guildID,
					},
				},
			},
			mockSetup: func(m *MockDiscordClient) {
				m.DeleteRoleFunc = func(ctx context.Context, gID, rID string) error {
					return errors.New("role not found")
				}
			},
			expectError: false, // Should not error for non-existent role
		},
		{
			name: "no external name",
			role: &v1alpha1.Role{
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						GuildID: guildID,
					},
				},
			},
			mockSetup: func(m *MockDiscordClient) {
				// No setup needed
			},
			expectError: false, // Should not error if no external name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockDiscordClient{}
			tt.mockSetup(mockClient)
			
			e := &external{discord: mockClient}
			
			_, err := e.Delete(ctx, tt.role)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDisconnect(t *testing.T) {
	e := &external{}
	err := e.Disconnect(context.Background())
	assert.NoError(t, err)
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

// Test type assertions
func TestTypeAssertions(t *testing.T) {
	ctx := context.Background()
	
	// Test with wrong type
	wrongType := &guildv1alpha1.Guild{}
	
	e := &external{discord: &MockDiscordClient{}}
	
	_, err := e.Observe(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotRole)
	
	_, err = e.Create(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotRole)
	
	_, err = e.Update(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotRole)
	
	_, err = e.Delete(ctx, wrongType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotRole)
}