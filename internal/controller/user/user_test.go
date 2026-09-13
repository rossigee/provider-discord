package user

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/pkg/errors"
	userv1beta1 "github.com/rossigee/provider-discord/apis/user/v1beta1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func stringPtr(s string) *string {
	return &s
}

type MockUserClient struct {
	GetUserFunc              func(ctx context.Context, userID string) (*discordclient.DiscordUser, error)
	GetCurrentUserFunc       func(ctx context.Context) (*discordclient.DiscordUser, error)
	ModifyCurrentUserFunc    func(ctx context.Context, req *discordclient.ModifyCurrentUserRequest) (*discordclient.DiscordUser, error)
	GetCurrentUserGuildsFunc func(ctx context.Context, req *discordclient.GetCurrentUserGuildsRequest) ([]discordclient.Guild, error)
	LeaveGuildFunc           func(ctx context.Context, guildID string) error
}

var _ discordclient.UserClient = (*MockUserClient)(nil)

func (m *MockUserClient) GetUser(ctx context.Context, userID string) (*discordclient.DiscordUser, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockUserClient) GetCurrentUser(ctx context.Context) (*discordclient.DiscordUser, error) {
	if m.GetCurrentUserFunc != nil {
		return m.GetCurrentUserFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *MockUserClient) ModifyCurrentUser(ctx context.Context, req *discordclient.ModifyCurrentUserRequest) (*discordclient.DiscordUser, error) {
	if m.ModifyCurrentUserFunc != nil {
		return m.ModifyCurrentUserFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockUserClient) GetCurrentUserGuilds(ctx context.Context, req *discordclient.GetCurrentUserGuildsRequest) ([]discordclient.Guild, error) {
	if m.GetCurrentUserGuildsFunc != nil {
		return m.GetCurrentUserGuildsFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockUserClient) LeaveGuild(ctx context.Context, guildID string) error {
	if m.LeaveGuildFunc != nil {
		return m.LeaveGuildFunc(ctx, guildID)
	}
	return errors.New("not implemented")
}

func TestObserve(t *testing.T) {
	ctx := context.Background()
	userID := "123456789"

	tests := []struct {
		name             string
		user             *userv1beta1.User
		mockSetup        func(*MockUserClient)
		expectedExists   bool
		expectedUpToDate bool
		expectError      bool
	}{
		{
			name: "user exists and up to date",
			user: &userv1beta1.User{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: userID,
					},
				},
				Spec: userv1beta1.UserSpec{
					ForProvider: userv1beta1.UserParameters{
						UserID: userID,
					},
				},
			},
			mockSetup: func(m *MockUserClient) {
				m.GetUserFunc = func(ctx context.Context, uid string) (*discordclient.DiscordUser, error) {
					return &discordclient.DiscordUser{
						ID:         uid,
						Username:   "testuser",
						GlobalName: stringPtr("Test User"),
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: true,
			expectError:      false,
		},
		{
			name: "user exists but needs update - @me only",
			user: &userv1beta1.User{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "@me",
					},
				},
				Spec: userv1beta1.UserSpec{
					ForProvider: userv1beta1.UserParameters{
						UserID:   "@me",
						Username: stringPtr("newusername"),
					},
				},
			},
			mockSetup: func(m *MockUserClient) {
				m.GetCurrentUserFunc = func(ctx context.Context) (*discordclient.DiscordUser, error) {
					return &discordclient.DiscordUser{
						ID:       "currentuser",
						Username: "oldusername",
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "user does not exist",
			user: &userv1beta1.User{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: userID,
					},
				},
				Spec: userv1beta1.UserSpec{
					ForProvider: userv1beta1.UserParameters{
						UserID: userID,
					},
				},
			},
			mockSetup: func(m *MockUserClient) {
				m.GetUserFunc = func(ctx context.Context, uid string) (*discordclient.DiscordUser, error) {
					return nil, errors.New("user not found")
				}
			},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "@me returns current user",
			user: &userv1beta1.User{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "@me",
					},
				},
				Spec: userv1beta1.UserSpec{
					ForProvider: userv1beta1.UserParameters{
						UserID: "@me",
					},
				},
			},
			mockSetup: func(m *MockUserClient) {
				m.GetCurrentUserFunc = func(ctx context.Context) (*discordclient.DiscordUser, error) {
					return &discordclient.DiscordUser{
						ID:       "currentuser",
						Username: "botuser",
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: true,
			expectError:      false,
		},
		{
			name: "API error returns observation without error",
			user: &userv1beta1.User{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: userID,
					},
				},
				Spec: userv1beta1.UserSpec{
					ForProvider: userv1beta1.UserParameters{
						UserID: userID,
					},
				},
			},
			mockSetup: func(m *MockUserClient) {
				m.GetUserFunc = func(ctx context.Context, uid string) (*discordclient.DiscordUser, error) {
					return nil, errors.New("Discord API error: 403")
				}
			},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUserClient{}
			tt.mockSetup(mockClient)

			e := &external{discord: mockClient}
			obs, err := e.Observe(ctx, tt.user)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedExists, obs.ResourceExists)
			assert.Equal(t, tt.expectedUpToDate, obs.ResourceUpToDate)
		})
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	userID := "123456789"

	tests := []struct {
		name        string
		user        *userv1beta1.User
		mockSetup   func(*MockUserClient)
		expectError bool
	}{
		{
			name: "create not supported - user is read-only",
			user: &userv1beta1.User{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: userID,
					},
				},
			},
			mockSetup:   func(m *MockUserClient) {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUserClient{}
			tt.mockSetup(mockClient)

			e := &external{discord: mockClient}
			_, err := e.Create(ctx, tt.user)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		user        *userv1beta1.User
		mockSetup   func(*MockUserClient)
		expectError bool
	}{
		{
			name: "update username successfully",
			user: &userv1beta1.User{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "@me",
					},
				},
				Spec: userv1beta1.UserSpec{
					ForProvider: userv1beta1.UserParameters{
						UserID:   "@me",
						Username: stringPtr("newusername"),
					},
				},
			},
			mockSetup: func(m *MockUserClient) {
				m.ModifyCurrentUserFunc = func(ctx context.Context, req *discordclient.ModifyCurrentUserRequest) (*discordclient.DiscordUser, error) {
					return &discordclient.DiscordUser{
						ID:       "currentuser",
						Username: *req.Username,
					}, nil
				}
			},
			expectError: false,
		},
		{
			name: "update fails with API error",
			user: &userv1beta1.User{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "@me",
					},
				},
				Spec: userv1beta1.UserSpec{
					ForProvider: userv1beta1.UserParameters{
						UserID:   "@me",
						Username: stringPtr("newusername"),
					},
				},
			},
			mockSetup: func(m *MockUserClient) {
				m.ModifyCurrentUserFunc = func(ctx context.Context, req *discordclient.ModifyCurrentUserRequest) (*discordclient.DiscordUser, error) {
					return nil, errors.New("Discord API error: 400")
				}
			},
			expectError: true,
		},
		{
			name: "update fails - not @me",
			user: &userv1beta1.User{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "123456789",
					},
				},
				Spec: userv1beta1.UserSpec{
					ForProvider: userv1beta1.UserParameters{
						UserID: "123456789",
					},
				},
			},
			mockSetup:   func(m *MockUserClient) {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUserClient{}
			tt.mockSetup(mockClient)

			e := &external{discord: mockClient}
			_, err := e.Update(ctx, tt.user)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	userID := "123456789"

	tests := []struct {
		name        string
		user        *userv1beta1.User
		mockSetup   func(*MockUserClient)
		expectError bool
	}{
		{
			name: "delete is no-op for user",
			user: &userv1beta1.User{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: userID,
					},
				},
			},
			mockSetup:   func(m *MockUserClient) {},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUserClient{}
			tt.mockSetup(mockClient)

			e := &external{discord: mockClient}
			_, err := e.Delete(ctx, tt.user)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsDiscordPermissionDenied(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "error contains 403",
			err:      errors.New("Discord API error: 403 Forbidden"),
			expected: true,
		},
		{
			name:     "error does not contain 403",
			err:      errors.New("Discord API error: 404 Not Found"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "error does not contain Discord",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDiscordPermissionDenied(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDiscordUnauthorized(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "error contains 401",
			err:      errors.New("Discord API error: 401 Unauthorized"),
			expected: true,
		},
		{
			name:     "error does not contain 401",
			err:      errors.New("Discord API error: 403 Forbidden"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDiscordUnauthorized(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExternalObserveNotUser(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Observe(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotUser)
}

func TestExternalCreateNotUser(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Create(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotUser)
}

func TestExternalUpdateNotUser(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Update(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotUser)
}

func TestExternalDeleteNotUser(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Delete(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotUser)
}
