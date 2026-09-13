package webhook

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/pkg/errors"
	webhookv1beta1 "github.com/rossigee/provider-discord/apis/webhook/v1beta1"
	discordclient "github.com/rossigee/provider-discord/internal/clients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func stringPtr(s string) *string {
	return &s
}

type MockWebhookClient struct {
	CreateWebhookFunc      func(ctx context.Context, channelID string, req *discordclient.CreateWebhookRequest) (*discordclient.Webhook, error)
	GetWebhookFunc         func(ctx context.Context, webhookID string) (*discordclient.Webhook, error)
	ModifyWebhookFunc      func(ctx context.Context, webhookID string, req *discordclient.ModifyWebhookRequest) (*discordclient.Webhook, error)
	DeleteWebhookFunc      func(ctx context.Context, webhookID string) error
	GetChannelWebhooksFunc func(ctx context.Context, channelID string) ([]discordclient.Webhook, error)
	GetGuildWebhooksFunc   func(ctx context.Context, guildID string) ([]discordclient.Webhook, error)
}

var _ discordclient.WebhookClient = (*MockWebhookClient)(nil)

func (m *MockWebhookClient) CreateWebhook(ctx context.Context, channelID string, req *discordclient.CreateWebhookRequest) (*discordclient.Webhook, error) {
	if m.CreateWebhookFunc != nil {
		return m.CreateWebhookFunc(ctx, channelID, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockWebhookClient) GetWebhook(ctx context.Context, webhookID string) (*discordclient.Webhook, error) {
	if m.GetWebhookFunc != nil {
		return m.GetWebhookFunc(ctx, webhookID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockWebhookClient) ModifyWebhook(ctx context.Context, webhookID string, req *discordclient.ModifyWebhookRequest) (*discordclient.Webhook, error) {
	if m.ModifyWebhookFunc != nil {
		return m.ModifyWebhookFunc(ctx, webhookID, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockWebhookClient) DeleteWebhook(ctx context.Context, webhookID string) error {
	if m.DeleteWebhookFunc != nil {
		return m.DeleteWebhookFunc(ctx, webhookID)
	}
	return errors.New("not implemented")
}

func (m *MockWebhookClient) GetChannelWebhooks(ctx context.Context, channelID string) ([]discordclient.Webhook, error) {
	if m.GetChannelWebhooksFunc != nil {
		return m.GetChannelWebhooksFunc(ctx, channelID)
	}
	return nil, errors.New("not implemented")
}

func (m *MockWebhookClient) GetGuildWebhooks(ctx context.Context, guildID string) ([]discordclient.Webhook, error) {
	if m.GetGuildWebhooksFunc != nil {
		return m.GetGuildWebhooksFunc(ctx, guildID)
	}
	return nil, errors.New("not implemented")
}

func TestObserve(t *testing.T) {
	ctx := context.Background()
	webhookID := "123456789012345678"

	tests := []struct {
		name             string
		webhook          *webhookv1beta1.Webhook
		mockSetup        func(*MockWebhookClient)
		expectedExists   bool
		expectedUpToDate bool
		expectError      bool
	}{
		{
			name: "empty external name - needs creation",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-webhook",
					Annotations: map[string]string{},
				},
			},
			mockSetup:        func(m *MockWebhookClient) {},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "invalid external name - needs creation",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-webhook",
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: "invalid-id",
					},
				},
			},
			mockSetup:        func(m *MockWebhookClient) {},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "valid external name - webhook found",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: webhookID,
					},
				},
				Spec: webhookv1beta1.WebhookSpec{
					ForProvider: webhookv1beta1.WebhookParameters{
						Name:      "test-webhook", // Match the mock
						ChannelID: "123456",
					},
				},
			},
			mockSetup: func(m *MockWebhookClient) {
				m.GetWebhookFunc = func(ctx context.Context, id string) (*discordclient.Webhook, error) {
					return &discordclient.Webhook{
						ID:        id,
						ChannelID: "123456",
						GuildID:   "789012",
						Name:      "test-webhook",
						Avatar:    stringPtr("abc123"),
						Token:     "",
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: true,
			expectError:      false,
		},
		{
			name: "valid external name - webhook not found",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: webhookID,
					},
				},
			},
			mockSetup: func(m *MockWebhookClient) {
				m.GetWebhookFunc = func(ctx context.Context, id string) (*discordclient.Webhook, error) {
					return nil, errors.New("not found")
				}
			},
			expectedExists:   false,
			expectedUpToDate: false,
			expectError:      false,
		},
		{
			name: "webhook found but needs update",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: webhookID,
					},
				},
				Spec: webhookv1beta1.WebhookSpec{
					ForProvider: webhookv1beta1.WebhookParameters{
						Name: "newname",
					},
				},
			},
			mockSetup: func(m *MockWebhookClient) {
				m.GetWebhookFunc = func(ctx context.Context, id string) (*discordclient.Webhook, error) {
					return &discordclient.Webhook{
						ID:        id,
						Name:      "oldname",
						ChannelID: "123456",
						GuildID:   "789012",
					}, nil
				}
			},
			expectedExists:   true,
			expectedUpToDate: false,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWebhookClient{}
			tt.mockSetup(mockClient)

			e := &external{service: mockClient, kube: nil}
			obs, err := e.Observe(ctx, tt.webhook)

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
		webhook     *webhookv1beta1.Webhook
		mockSetup   func(*MockWebhookClient)
		expectError bool
	}{
		{
			name: "create webhook successfully",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: webhookv1beta1.WebhookSpec{
					ForProvider: webhookv1beta1.WebhookParameters{
						ChannelID: "123456",
						Name:      "my-webhook",
					},
				},
			},
			mockSetup: func(m *MockWebhookClient) {
				m.CreateWebhookFunc = func(ctx context.Context, channelID string, req *discordclient.CreateWebhookRequest) (*discordclient.Webhook, error) {
					return &discordclient.Webhook{
						ID:        "newwebhook123",
						ChannelID: channelID,
						Name:      req.Name,
					}, nil
				}
			},
			expectError: false,
		},
		{
			name: "create fails with permission denied",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: webhookv1beta1.WebhookSpec{
					ForProvider: webhookv1beta1.WebhookParameters{
						ChannelID: "123456",
					},
				},
			},
			mockSetup: func(m *MockWebhookClient) {
				m.CreateWebhookFunc = func(ctx context.Context, channelID string, req *discordclient.CreateWebhookRequest) (*discordclient.Webhook, error) {
					return nil, errors.New("Discord API error: 403")
				}
			},
			expectError: false, // Returns nil error but sets condition
		},
		{
			name: "create fails with API error",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: webhookv1beta1.WebhookSpec{
					ForProvider: webhookv1beta1.WebhookParameters{
						ChannelID: "123456",
					},
				},
			},
			mockSetup: func(m *MockWebhookClient) {
				m.CreateWebhookFunc = func(ctx context.Context, channelID string, req *discordclient.CreateWebhookRequest) (*discordclient.Webhook, error) {
					return nil, errors.New("Discord API error: 500")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWebhookClient{}
			tt.mockSetup(mockClient)

			e := &external{service: mockClient, kube: nil}
			_, err := e.Create(ctx, tt.webhook)

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
	webhookID := "123456789012345678"

	tests := []struct {
		name        string
		webhook     *webhookv1beta1.Webhook
		mockSetup   func(*MockWebhookClient)
		expectError bool
	}{
		{
			name: "update webhook successfully",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: webhookID,
					},
				},
				Spec: webhookv1beta1.WebhookSpec{
					ForProvider: webhookv1beta1.WebhookParameters{
						Name: "newname",
					},
				},
			},
			mockSetup: func(m *MockWebhookClient) {
				m.ModifyWebhookFunc = func(ctx context.Context, id string, req *discordclient.ModifyWebhookRequest) (*discordclient.Webhook, error) {
					return &discordclient.Webhook{
						ID:        id,
						Name:      *req.Name,
						ChannelID: "123456",
					}, nil
				}
			},
			expectError: false,
		},
		{
			name: "update fails with API error",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: webhookID,
					},
				},
				Spec: webhookv1beta1.WebhookSpec{
					ForProvider: webhookv1beta1.WebhookParameters{
						Name: "newname",
					},
				},
			},
			mockSetup: func(m *MockWebhookClient) {
				m.ModifyWebhookFunc = func(ctx context.Context, id string, req *discordclient.ModifyWebhookRequest) (*discordclient.Webhook, error) {
					return nil, errors.New("Discord API error: 500")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWebhookClient{}
			tt.mockSetup(mockClient)

			e := &external{service: mockClient, kube: nil}
			_, err := e.Update(ctx, tt.webhook)

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
	webhookID := "123456789012345678"

	tests := []struct {
		name        string
		webhook     *webhookv1beta1.Webhook
		mockSetup   func(*MockWebhookClient)
		expectError bool
	}{
		{
			name: "delete webhook successfully",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: webhookID,
					},
				},
			},
			mockSetup: func(m *MockWebhookClient) {
				m.DeleteWebhookFunc = func(ctx context.Context, id string) error {
					return nil
				}
			},
			expectError: false,
		},
		{
			name: "delete fails with permission denied",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: webhookID,
					},
				},
			},
			mockSetup: func(m *MockWebhookClient) {
				m.DeleteWebhookFunc = func(ctx context.Context, id string) error {
					return errors.New("Discord API error: 403")
				}
			},
			expectError: false, // Returns nil error but sets condition
		},
		{
			name: "delete fails with API error",
			webhook: &webhookv1beta1.Webhook{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: webhookID,
					},
				},
			},
			mockSetup: func(m *MockWebhookClient) {
				m.DeleteWebhookFunc = func(ctx context.Context, id string) error {
					return errors.New("Discord API error: 500")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWebhookClient{}
			tt.mockSetup(mockClient)

			e := &external{service: mockClient, kube: nil}
			_, err := e.Delete(ctx, tt.webhook)

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

func TestIsValidDiscordID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{
			name:     "valid 18 digit ID",
			id:       "123456789012345678",
			expected: true,
		},
		{
			name:     "valid 19 digit ID",
			id:       "1234567890123456789",
			expected: true,
		},
		{
			name:     "invalid - too short",
			id:       "12345678901234567",
			expected: false,
		},
		{
			name:     "invalid - too long",
			id:       "12345678901234567890",
			expected: false,
		},
		{
			name:     "invalid - contains letters",
			id:       "12345678901234567a",
			expected: false,
		},
		{
			name:     "invalid - empty string",
			id:       "",
			expected: false,
		},
		{
			name:     "invalid - not all digits",
			id:       "12345678901234567_",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDiscordID(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExternalObserveNotWebhook(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Observe(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotWebhook)
}

func TestExternalCreateNotWebhook(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Create(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotWebhook)
}

func TestExternalUpdateNotWebhook(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Update(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotWebhook)
}

func TestExternalDeleteNotWebhook(t *testing.T) {
	ctx := context.Background()
	e := &external{}

	_, err := e.Delete(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errNotWebhook)
}
