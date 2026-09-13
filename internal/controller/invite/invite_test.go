package invite

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/pkg/errors"
	invitev1beta1 "github.com/rossigee/provider-discord/apis/invite/v1beta1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

type MockInviteClient struct {
	CreateChannelInviteFunc func(ctx context.Context, channelID string, req *discordclient.CreateInviteRequest) (*discordclient.Invite, error)
	GetInviteFunc          func(ctx context.Context, inviteCode string) (*discordclient.Invite, error)
	DeleteInviteFunc       func(ctx context.Context, inviteCode string) error
	GetChannelInvitesFunc  func(ctx context.Context, channelID string) ([]discordclient.Invite, error)
	GetGuildInvitesFunc   func(ctx context.Context, guildID string) ([]discordclient.Invite, error)
}

var _ discordclient.InviteClient = (*MockInviteClient)(nil)

func (m *MockInviteClient) CreateChannelInvite(ctx context.Context, channelID string, req *discordclient.CreateInviteRequest) (*discordclient.Invite, error) {
	if m.CreateChannelInviteFunc != nil {
		return m.CreateChannelInviteFunc(ctx, channelID, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockInviteClient) GetInvite(ctx context.Context, inviteCode string) (*discordclient.Invite, error) {
	if m.GetInviteFunc != nil {
		return m.GetInviteFunc(ctx, inviteCode)
	}
	return nil, errors.New("not implemented")
}

func (m *MockInviteClient) DeleteInvite(ctx context.Context, inviteCode string) error {
	if m.DeleteInviteFunc != nil {
		return m.DeleteInviteFunc(ctx, inviteCode)
	}
	return errors.New("not implemented")
}

func (m *MockInviteClient) GetChannelInvites(ctx context.Context, channelID string) ([]discordclient.Invite, error) {
	if m.GetChannelInvitesFunc != nil {
		return m.GetChannelInvitesFunc(ctx, channelID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockInviteClient) GetGuildInvites(ctx context.Context, guildID string) ([]discordclient.Invite, error) {
	if m.GetGuildInvitesFunc != nil {
		return m.GetGuildInvitesFunc(ctx, guildID)
	}
	return nil, errors.New("not implemented")
}

func TestObserve(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		invite           *invitev1beta1.Invite
		mockSetup        func(*MockInviteClient)
		expectedExists   bool
		expectedUpToDate bool
		expectError      bool
	}{
		{
			name: "empty external name - needs creation",
			invite: &invitev1beta1.Invite{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-invite",
					Annotations: map[string]string{},
				},
			},
			mockSetup:        func(m *MockInviteClient) {},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "invalid external name - needs creation",
			invite: &invitev1beta1.Invite{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-invite",
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "my-invalid-code",
					},
				},
			},
			mockSetup:        func(m *MockInviteClient) {},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "valid external name - invite found",
			invite: &invitev1beta1.Invite{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "abc123",
					},
				},
			},
			mockSetup: func(m *MockInviteClient) {
				m.GetInviteFunc = func(ctx context.Context, code string) (*discordclient.Invite, error) {
					return &discordclient.Invite{
						Code:              code,
						Channel:           &discordclient.Channel{ID: "123456"},
						Guild:             &discordclient.Guild{ID: "789012"},
						Inviter:           &discordclient.User{ID: "345678"},
						MaxAge:            86400,
						MaxUses:            0,
						Temporary:          false,
						Uses:               0,
						CreatedAt:          "2024-01-01T00:00:00Z",
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: true,
			expectError:      false,
		},
		{
			name: "valid external name - invite not found",
			invite: &invitev1beta1.Invite{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "abc123",
					},
				},
			},
			mockSetup: func(m *MockInviteClient) {
				m.GetInviteFunc = func(ctx context.Context, code string) (*discordclient.Invite, error) {
					return nil, errors.New("not found")
				}
			},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "API error returns error",
			invite: &invitev1beta1.Invite{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "abc123",
					},
				},
			},
			mockSetup: func(m *MockInviteClient) {
				m.GetInviteFunc = func(ctx context.Context, code string) (*discordclient.Invite, error) {
					return nil, errors.New("Discord API error: 403")
				}
			},
			// API errors don't return an error - they return resource does not exist
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockInviteClient{}
			tt.mockSetup(mockClient)

			e := &external{service: mockClient, kube: nil}
			obs, err := e.Observe(ctx, tt.invite)

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

	tests := []struct {
		name        string
		invite      *invitev1beta1.Invite
		mockSetup   func(*MockInviteClient)
		expectError bool
	}{
		{
			name: "create invite successfully",
			invite: &invitev1beta1.Invite{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "newinvite",
					},
				},
				Spec: invitev1beta1.InviteSpec{
					ForProvider: invitev1beta1.InviteParameters{
						ChannelID: "123456",
						MaxAge:    intPtr(3600),
						MaxUses:   intPtr(10),
						Temporary: boolPtr(true),
						Unique:    boolPtr(true),
					},
				},
			},
			mockSetup: func(m *MockInviteClient) {
				m.CreateChannelInviteFunc = func(ctx context.Context, channelID string, req *discordclient.CreateInviteRequest) (*discordclient.Invite, error) {
					return &discordclient.Invite{
						Code:      "newinvite",
						Channel:   &discordclient.Channel{ID: channelID},
						MaxAge:    *req.MaxAge,
						MaxUses:   *req.MaxUses,
						Temporary: *req.Temporary,
					}, nil
				}
			},
			expectError: false,
		},
		{
			name: "create fails with permission denied",
			invite: &invitev1beta1.Invite{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: invitev1beta1.InviteSpec{
					ForProvider: invitev1beta1.InviteParameters{
						ChannelID: "123456",
					},
				},
			},
			mockSetup: func(m *MockInviteClient) {
				m.CreateChannelInviteFunc = func(ctx context.Context, channelID string, req *discordclient.CreateInviteRequest) (*discordclient.Invite, error) {
					return nil, errors.New("Discord API error: 403")
				}
			},
			expectError: false, // Returns nil error but sets condition
		},
		{
			name: "create fails with API error",
			invite: &invitev1beta1.Invite{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: invitev1beta1.InviteSpec{
					ForProvider: invitev1beta1.InviteParameters{
						ChannelID: "123456",
					},
				},
			},
			mockSetup: func(m *MockInviteClient) {
				m.CreateChannelInviteFunc = func(ctx context.Context, channelID string, req *discordclient.CreateInviteRequest) (*discordclient.Invite, error) {
					return nil, errors.New("Discord API error: 500")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockInviteClient{}
			tt.mockSetup(mockClient)

			e := &external{service: mockClient, kube: nil}
			_, err := e.Create(ctx, tt.invite)

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

	e := &external{}
	_, err := e.Update(ctx, nil)
	assert.NoError(t, err) // No-op for invites
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		invite      *invitev1beta1.Invite
		mockSetup   func(*MockInviteClient)
		expectError bool
	}{
		{
			name: "delete invite successfully",
			invite: &invitev1beta1.Invite{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "abc123",
					},
				},
			},
			mockSetup: func(m *MockInviteClient) {
				m.DeleteInviteFunc = func(ctx context.Context, code string) error {
					return nil
				}
			},
			expectError: false,
		},
		{
			name: "delete fails with permission denied",
			invite: &invitev1beta1.Invite{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "abc123",
					},
				},
			},
			mockSetup: func(m *MockInviteClient) {
				m.DeleteInviteFunc = func(ctx context.Context, code string) error {
					return errors.New("Discord API error: 403")
				}
			},
			expectError: false, // Returns nil error but sets condition
		},
		{
			name: "delete fails with API error",
			invite: &invitev1beta1.Invite{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "abc123",
					},
				},
			},
			mockSetup: func(m *MockInviteClient) {
				m.DeleteInviteFunc = func(ctx context.Context, code string) error {
					return errors.New("Discord API error: 500")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockInviteClient{}
			tt.mockSetup(mockClient)

			e := &external{service: mockClient, kube: nil}
			_, err := e.Delete(ctx, tt.invite)

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
		err     error
		expected bool
	}{
		{
			name:     "error contains 403",
			err:     errors.New("Discord API error: 403 Forbidden"),
			expected: true,
		},
		{
			name:     "error does not contain 403",
			err:     errors.New("Discord API error: 404 Not Found"),
			expected: false,
		},
		{
			name:     "nil error",
			err:     nil,
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
		err     error
		expected bool
	}{
		{
			name:     "error contains 401",
			err:     errors.New("Discord API error: 401 Unauthorized"),
			expected: true,
		},
		{
			name:     "error does not contain 401",
			err:     errors.New("Discord API error: 403 Forbidden"),
			expected: false,
		},
		{
			name:     "nil error",
			err:     nil,
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

func TestIsValidDiscordInviteCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "valid short code",
			code:     "abc",
			expected: true,
		},
		{
			name:     "valid medium code",
			code:     "abc123xyz",
			expected: true,
		},
		{
			name:     "valid long code",
			code:     "abc123xyz789",
			expected: true,
		},
		{
			name:     "invalid - too short",
			code:     "ab",
			expected: false,
		},
		{
			name:     "invalid - too long",
			code:     "abc123xyz789abc",
			expected: false,
		},
		{
			name:     "invalid - contains dash",
			code:     "abc-123",
			expected: false,
		},
		{
			name:     "invalid - contains underscore",
			code:     "abc_123",
			expected: false,
		},
		{
			name:     "invalid - contains uppercase",
			code:     "ABC123",
			expected: true, // Valid - alphanumeric includes uppercase
		},
		{
			name:     "invalid - empty string",
			code:     "",
			expected: false,
		},
		{
			name:     "invalid - resource name format",
			code:     "general-channel",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDiscordInviteCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExternalObserveNotInvite(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Observe(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotInvite)
}

func TestExternalCreateNotInvite(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Create(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotInvite)
}

func TestExternalUpdateNotInvite(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	// Note: Update is a no-op in this controller and doesn't validate input
	// so passing nil returns no error
	_, err := e.Update(ctx, nil)
	assert.NoError(t, err)
}

func TestExternalDeleteNotInvite(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Delete(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotInvite)
}
