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

package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// DiscordAPIBaseURL is the base URL for the Discord API
	DiscordAPIBaseURL = "https://discord.com/api/v10"
)

// RoleClient defines the interface for role-related Discord operations
type RoleClient interface {
	CreateRole(ctx context.Context, guildID string, req CreateRoleRequest) (*Role, error)
	GetRole(ctx context.Context, guildID, roleID string) (*Role, error)
	ModifyRole(ctx context.Context, guildID, roleID string, req ModifyRoleRequest) (*Role, error)
	DeleteRole(ctx context.Context, guildID, roleID string) error
}

// GuildClient defines the interface for guild-related Discord operations
type GuildClient interface {
	CreateGuild(ctx context.Context, req *CreateGuildRequest) (*Guild, error)
	GetGuild(ctx context.Context, guildID string) (*Guild, error)
	ModifyGuild(ctx context.Context, guildID string, req *ModifyGuildRequest) (*Guild, error)
	DeleteGuild(ctx context.Context, guildID string) error
	ListGuilds(ctx context.Context) ([]Guild, error)
}

// ChannelClient defines the interface for channel-related Discord operations
type ChannelClient interface {
	CreateChannel(ctx context.Context, req *CreateChannelRequest) (*Channel, error)
	GetChannel(ctx context.Context, channelID string) (*Channel, error)
	ModifyChannel(ctx context.Context, channelID string, req *ModifyChannelRequest) (*Channel, error)
	DeleteChannel(ctx context.Context, channelID string) error
}

// WebhookClient defines the interface for webhook-related Discord operations
type WebhookClient interface {
	CreateWebhook(ctx context.Context, channelID string, req *CreateWebhookRequest) (*Webhook, error)
	GetWebhook(ctx context.Context, webhookID string) (*Webhook, error)
	ModifyWebhook(ctx context.Context, webhookID string, req *ModifyWebhookRequest) (*Webhook, error)
	DeleteWebhook(ctx context.Context, webhookID string) error
	GetChannelWebhooks(ctx context.Context, channelID string) ([]Webhook, error)
	GetGuildWebhooks(ctx context.Context, guildID string) ([]Webhook, error)
}

// InviteClient defines the interface for invite-related Discord operations
type InviteClient interface {
	CreateChannelInvite(ctx context.Context, channelID string, req *CreateInviteRequest) (*Invite, error)
	GetInvite(ctx context.Context, inviteCode string) (*Invite, error)
	DeleteInvite(ctx context.Context, inviteCode string) error
	GetChannelInvites(ctx context.Context, channelID string) ([]Invite, error)
	GetGuildInvites(ctx context.Context, guildID string) ([]Invite, error)
}

// MemberClient defines the interface for member-related Discord operations
type MemberClient interface {
	GetGuildMember(ctx context.Context, guildID, userID string) (*GuildMember, error)
	ListGuildMembers(ctx context.Context, guildID string, req *ListGuildMembersRequest) ([]GuildMember, error)
	SearchGuildMembers(ctx context.Context, guildID string, req *SearchGuildMembersRequest) ([]GuildMember, error)
	AddGuildMember(ctx context.Context, guildID, userID string, req *AddGuildMemberRequest) (*GuildMember, error)
	ModifyGuildMember(ctx context.Context, guildID, userID string, req *ModifyGuildMemberRequest) (*GuildMember, error)
	ModifyCurrentMember(ctx context.Context, guildID string, req *ModifyCurrentMemberRequest) (*GuildMember, error)
	RemoveGuildMember(ctx context.Context, guildID, userID string) error
	AddGuildMemberRole(ctx context.Context, guildID, userID, roleID string) error
	RemoveGuildMemberRole(ctx context.Context, guildID, userID, roleID string) error
}

// UserClient defines the interface for user-related Discord operations
type UserClient interface {
	GetUser(ctx context.Context, userID string) (*DiscordUser, error)
	GetCurrentUser(ctx context.Context) (*DiscordUser, error)
	ModifyCurrentUser(ctx context.Context, req *ModifyCurrentUserRequest) (*DiscordUser, error)
	GetCurrentUserGuilds(ctx context.Context, req *GetCurrentUserGuildsRequest) ([]Guild, error)
	LeaveGuild(ctx context.Context, guildID string) error
}

// ApplicationClient defines the interface for application-related Discord operations
type ApplicationClient interface {
	GetApplication(ctx context.Context, applicationID string) (*DiscordApplication, error)
	GetCurrentApplication(ctx context.Context) (*DiscordApplication, error)
	ModifyCurrentApplication(ctx context.Context, req *ModifyCurrentApplicationRequest) (*DiscordApplication, error)
}

// IntegrationClient defines the interface for integration-related Discord operations
type IntegrationClient interface {
	GetGuildIntegrations(ctx context.Context, guildID string) ([]GuildIntegration, error)
	DeleteGuildIntegration(ctx context.Context, guildID, integrationID string) error
}

// DiscordClient is a client for the Discord API
type DiscordClient struct {
	httpClient *http.Client
	token      string
	baseURL    string
	logger     logr.Logger
}

// Ensure DiscordClient implements all client interfaces
var _ RoleClient = (*DiscordClient)(nil)
var _ GuildClient = (*DiscordClient)(nil)
var _ ChannelClient = (*DiscordClient)(nil)
var _ WebhookClient = (*DiscordClient)(nil)
var _ InviteClient = (*DiscordClient)(nil)
var _ MemberClient = (*DiscordClient)(nil)
var _ UserClient = (*DiscordClient)(nil)
var _ ApplicationClient = (*DiscordClient)(nil)
var _ IntegrationClient = (*DiscordClient)(nil)

// NewDiscordClient creates a new Discord API client
func NewDiscordClient(token string) *DiscordClient {
	return &DiscordClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token:   token,
		baseURL: DiscordAPIBaseURL,
		logger:  ctrl.Log.WithName("discord-client"),
	}
}

// Guild represents a Discord guild
type Guild struct {
	ID                          string    `json:"id"`
	Name                        string    `json:"name"`
	Icon                        *string   `json:"icon"`
	IconHash                    *string   `json:"icon_hash"`
	Splash                      *string   `json:"splash"`
	DiscoverySplash             *string   `json:"discovery_splash"`
	Owner                       *bool     `json:"owner,omitempty"`
	OwnerID                     string    `json:"owner_id"`
	Permissions                 *string   `json:"permissions,omitempty"`
	Region                      *string   `json:"region"`
	AFKChannelID                *string   `json:"afk_channel_id"`
	AFKTimeout                  int       `json:"afk_timeout"`
	WidgetEnabled               *bool     `json:"widget_enabled,omitempty"`
	WidgetChannelID             *string   `json:"widget_channel_id,omitempty"`
	VerificationLevel           int       `json:"verification_level"`
	DefaultMessageNotifications int       `json:"default_message_notifications"`
	ExplicitContentFilter       int       `json:"explicit_content_filter"`
	Roles                       []Role    `json:"roles,omitempty"`
	Emojis                      []Emoji   `json:"emojis,omitempty"`
	Features                    []string  `json:"features"`
	MFALevel                    int       `json:"mfa_level"`
	ApplicationID               *string   `json:"application_id"`
	SystemChannelID             *string   `json:"system_channel_id"`
	SystemChannelFlags          int       `json:"system_channel_flags"`
	RulesChannelID              *string   `json:"rules_channel_id"`
	MaxPresences                *int      `json:"max_presences,omitempty"`
	MaxMembers                  *int      `json:"max_members,omitempty"`
	VanityURLCode               *string   `json:"vanity_url_code"`
	Description                 *string   `json:"description"`
	Banner                      *string   `json:"banner"`
	PremiumTier                 int       `json:"premium_tier"`
	PremiumSubscriptionCount    *int      `json:"premium_subscription_count,omitempty"`
	PreferredLocale             string    `json:"preferred_locale"`
	PublicUpdatesChannelID      *string   `json:"public_updates_channel_id"`
	MaxVideoChannelUsers        *int      `json:"max_video_channel_users,omitempty"`
	ApproximateMemberCount      *int      `json:"approximate_member_count,omitempty"`
	ApproximatePresenceCount    *int      `json:"approximate_presence_count,omitempty"`
	WelcomeScreen               *struct{} `json:"welcome_screen,omitempty"`
	NSFWLevel                   int       `json:"nsfw_level"`
	Stickers                    []struct{} `json:"stickers,omitempty"`
	PremiumProgressBarEnabled   bool      `json:"premium_progress_bar_enabled"`
}

// Role represents a Discord role
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Hoist       bool   `json:"hoist"`
	Icon        string `json:"icon,omitempty"`
	UnicodeEmoji string `json:"unicode_emoji,omitempty"`
	Position    int    `json:"position"`
	Permissions string `json:"permissions"`
	Managed     bool   `json:"managed"`
	Mentionable bool   `json:"mentionable"`
}

// Emoji represents a Discord emoji
type Emoji struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Roles         []string `json:"roles,omitempty"`
	User          *struct{} `json:"user,omitempty"`
	RequireColons bool     `json:"require_colons,omitempty"`
	Managed       bool     `json:"managed,omitempty"`
	Animated      bool     `json:"animated,omitempty"`
	Available     bool     `json:"available,omitempty"`
}

// CreateGuildRequest represents a request to create a guild
type CreateGuildRequest struct {
	Name                        string    `json:"name"`
	Region                      *string   `json:"region,omitempty"`
	Icon                        *string   `json:"icon,omitempty"`
	VerificationLevel           *int      `json:"verification_level,omitempty"`
	DefaultMessageNotifications *int      `json:"default_message_notifications,omitempty"`
	ExplicitContentFilter       *int      `json:"explicit_content_filter,omitempty"`
	Roles                       []Role    `json:"roles,omitempty"`
	Channels                    []Channel `json:"channels,omitempty"`
	AFKChannelID                *string   `json:"afk_channel_id,omitempty"`
	AFKTimeout                  *int      `json:"afk_timeout,omitempty"`
	SystemChannelID             *string   `json:"system_channel_id,omitempty"`
	SystemChannelFlags          *int      `json:"system_channel_flags,omitempty"`
}

// Channel represents a Discord channel
type Channel struct {
	ID       string `json:"id,omitempty"`
	Type     int    `json:"type"`
	GuildID  string `json:"guild_id,omitempty"`
	Name     string `json:"name"`
	Position int    `json:"position,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
}

// ModifyGuildRequest represents a request to modify a guild
type ModifyGuildRequest struct {
	Name                        *string `json:"name,omitempty"`
	Region                      *string `json:"region,omitempty"`
	VerificationLevel           *int    `json:"verification_level,omitempty"`
	DefaultMessageNotifications *int    `json:"default_message_notifications,omitempty"`
	ExplicitContentFilter       *int    `json:"explicit_content_filter,omitempty"`
	AFKChannelID                *string `json:"afk_channel_id,omitempty"`
	AFKTimeout                  *int    `json:"afk_timeout,omitempty"`
	Icon                        *string `json:"icon,omitempty"`
	OwnerID                     *string `json:"owner_id,omitempty"`
	Splash                      *string `json:"splash,omitempty"`
	DiscoverySplash             *string `json:"discovery_splash,omitempty"`
	Banner                      *string `json:"banner,omitempty"`
	SystemChannelID             *string `json:"system_channel_id,omitempty"`
	SystemChannelFlags          *int    `json:"system_channel_flags,omitempty"`
	RulesChannelID              *string `json:"rules_channel_id,omitempty"`
	PublicUpdatesChannelID      *string `json:"public_updates_channel_id,omitempty"`
	PreferredLocale             *string `json:"preferred_locale,omitempty"`
	Features                    []string `json:"features,omitempty"`
	Description                 *string `json:"description,omitempty"`
	PremiumProgressBarEnabled   *bool   `json:"premium_progress_bar_enabled,omitempty"`
}

// makeRequest performs an HTTP request to the Discord API
func (c *DiscordClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	var bodyStr string
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			c.logger.Error(err, "Failed to marshal request body", "endpoint", endpoint)
			return nil, errors.Wrap(err, "failed to marshal request body")
		}
		reqBody = bytes.NewReader(jsonBody)
		bodyStr = string(jsonBody)
	}

	url := c.baseURL + endpoint
	c.logger.Info("Making Discord API request",
		"method", method,
		"url", url,
		"body", bodyStr)

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		c.logger.Error(err, "Failed to create request", "url", url)
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Authorization", "Bot "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Crossplane Discord Provider/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error(err, "Failed to perform request", "url", url)
		return nil, errors.Wrap(err, "failed to perform request")
	}

	c.logger.Info("Discord API response",
		"method", method,
		"url", url,
		"status", resp.StatusCode)

	if resp.StatusCode >= 400 {
		defer func() { _ = resp.Body.Close() }()
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.logger.Error(nil, "Discord API error",
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"response", string(bodyBytes))
		return nil, errors.Errorf("Discord API error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// GetGuild retrieves a guild by ID
func (c *DiscordClient) GetGuild(ctx context.Context, guildID string) (*Guild, error) {
	resp, err := c.makeRequest(ctx, "GET", "/guilds/"+guildID+"?with_counts=true", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get guild")
	}
	defer func() { _ = resp.Body.Close() }()

	var guild Guild
	if err := json.NewDecoder(resp.Body).Decode(&guild); err != nil {
		return nil, errors.Wrap(err, "failed to decode guild response")
	}

	return &guild, nil
}

// CreateGuild creates a new guild
func (c *DiscordClient) CreateGuild(ctx context.Context, req *CreateGuildRequest) (*Guild, error) {
	resp, err := c.makeRequest(ctx, "POST", "/guilds", req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create guild")
	}
	defer func() { _ = resp.Body.Close() }()

	var guild Guild
	if err := json.NewDecoder(resp.Body).Decode(&guild); err != nil {
		return nil, errors.Wrap(err, "failed to decode created guild response")
	}

	return &guild, nil
}

// ModifyGuild modifies an existing guild
func (c *DiscordClient) ModifyGuild(ctx context.Context, guildID string, req *ModifyGuildRequest) (*Guild, error) {
	resp, err := c.makeRequest(ctx, "PATCH", "/guilds/"+guildID, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to modify guild")
	}
	defer func() { _ = resp.Body.Close() }()

	var guild Guild
	if err := json.NewDecoder(resp.Body).Decode(&guild); err != nil {
		return nil, errors.Wrap(err, "failed to decode modified guild response")
	}

	return &guild, nil
}

// DeleteGuild deletes a guild
func (c *DiscordClient) DeleteGuild(ctx context.Context, guildID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/guilds/"+guildID, nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete guild")
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// ListGuilds lists all guilds the bot is a member of
func (c *DiscordClient) ListGuilds(ctx context.Context) ([]Guild, error) {
	resp, err := c.makeRequest(ctx, "GET", "/users/@me/guilds", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list guilds")
	}
	defer func() { _ = resp.Body.Close() }()

	var guilds []Guild
	if err := json.NewDecoder(resp.Body).Decode(&guilds); err != nil {
		return nil, errors.Wrap(err, "failed to decode guilds response")
	}

	return guilds, nil
}

// CreateRoleRequest represents a request to create a role
type CreateRoleRequest struct {
	Name        string  `json:"name"`
	Permissions *string `json:"permissions,omitempty"`
	Color       *int    `json:"color,omitempty"`
	Hoist       *bool   `json:"hoist,omitempty"`
	Mentionable *bool   `json:"mentionable,omitempty"`
}

// ModifyRoleRequest represents a request to modify a role
type ModifyRoleRequest struct {
	Name        *string `json:"name,omitempty"`
	Permissions *string `json:"permissions,omitempty"`
	Color       *int    `json:"color,omitempty"`
	Hoist       *bool   `json:"hoist,omitempty"`
	Position    *int    `json:"position,omitempty"`
	Mentionable *bool   `json:"mentionable,omitempty"`
}

// CreateRole creates a new role in a guild
func (c *DiscordClient) CreateRole(ctx context.Context, guildID string, req CreateRoleRequest) (*Role, error) {
	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/guilds/%s/roles", guildID), req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create role")
	}
	defer func() { _ = resp.Body.Close() }()

	var role Role
	if err := json.NewDecoder(resp.Body).Decode(&role); err != nil {
		return nil, errors.Wrap(err, "failed to decode role response")
	}

	return &role, nil
}

// GetRole gets a role by ID
func (c *DiscordClient) GetRole(ctx context.Context, guildID, roleID string) (*Role, error) {
	resp, err := c.makeRequest(ctx, "GET", fmt.Sprintf("/guilds/%s/roles", guildID), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get roles")
	}
	defer func() { _ = resp.Body.Close() }()

	var roles []Role
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, errors.Wrap(err, "failed to decode roles response")
	}

	for _, role := range roles {
		if role.ID == roleID {
			return &role, nil
		}
	}

	return nil, errors.New("role not found")
}

// ModifyRole modifies an existing role
func (c *DiscordClient) ModifyRole(ctx context.Context, guildID, roleID string, req ModifyRoleRequest) (*Role, error) {
	resp, err := c.makeRequest(ctx, "PATCH", fmt.Sprintf("/guilds/%s/roles/%s", guildID, roleID), req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to modify role")
	}
	defer func() { _ = resp.Body.Close() }()

	var role Role
	if err := json.NewDecoder(resp.Body).Decode(&role); err != nil {
		return nil, errors.Wrap(err, "failed to decode modified role response")
	}

	return &role, nil
}

// DeleteRole deletes a role
func (c *DiscordClient) DeleteRole(ctx context.Context, guildID, roleID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/guilds/%s/roles/%s", guildID, roleID), nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete role")
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// CreateChannelRequest represents a request to create a channel
type CreateChannelRequest struct {
	Name             string  `json:"name"`
	Type             int     `json:"type"`
	GuildID          string  `json:"-"` // Not in JSON, used in URL
	Topic            *string `json:"topic,omitempty"`
	Bitrate          *int    `json:"bitrate,omitempty"`
	UserLimit        *int    `json:"user_limit,omitempty"`
	RateLimitPerUser *int    `json:"rate_limit_per_user,omitempty"`
	Position         *int    `json:"position,omitempty"`
	ParentID         *string `json:"parent_id,omitempty"`
	NSFW             *bool   `json:"nsfw,omitempty"`
}

// ModifyChannelRequest represents a request to modify a channel
type ModifyChannelRequest struct {
	Name             *string `json:"name,omitempty"`
	Type             *int    `json:"type,omitempty"`
	Position         *int    `json:"position,omitempty"`
	Topic            *string `json:"topic,omitempty"`
	NSFW             *bool   `json:"nsfw,omitempty"`
	RateLimitPerUser *int    `json:"rate_limit_per_user,omitempty"`
	Bitrate          *int    `json:"bitrate,omitempty"`
	UserLimit        *int    `json:"user_limit,omitempty"`
	ParentID         *string `json:"parent_id,omitempty"`
}

// Webhook represents a Discord webhook
type Webhook struct {
	ID            string  `json:"id,omitempty"`
	Type          int     `json:"type,omitempty"`
	GuildID       string  `json:"guild_id,omitempty"`
	ChannelID     string  `json:"channel_id,omitempty"`
	User          *User   `json:"user,omitempty"`
	Name          string  `json:"name,omitempty"`
	Avatar        *string `json:"avatar,omitempty"`
	Token         string  `json:"token,omitempty"`
	ApplicationID *string `json:"application_id,omitempty"`
	SourceGuild   *Guild  `json:"source_guild,omitempty"`
	SourceChannel *Channel `json:"source_channel,omitempty"`
	URL           string  `json:"url,omitempty"`
}

// CreateWebhookRequest represents a request to create a webhook
type CreateWebhookRequest struct {
	Name   string  `json:"name"`
	Avatar *string `json:"avatar,omitempty"`
}

// ModifyWebhookRequest represents a request to modify a webhook
type ModifyWebhookRequest struct {
	Name      *string `json:"name,omitempty"`
	Avatar    *string `json:"avatar,omitempty"`
	ChannelID *string `json:"channel_id,omitempty"`
}

// Invite represents a Discord invite
type Invite struct {
	Code                     string     `json:"code"`
	Guild                    *Guild     `json:"guild,omitempty"`
	Channel                  *Channel   `json:"channel,omitempty"`
	Inviter                  *User      `json:"inviter,omitempty"`
	TargetType               *int       `json:"target_type,omitempty"`
	TargetUser               *User      `json:"target_user,omitempty"`
	TargetApplication        *Application `json:"target_application,omitempty"`
	ApproximatePresenceCount *int       `json:"approximate_presence_count,omitempty"`
	ApproximateMemberCount   *int       `json:"approximate_member_count,omitempty"`
	ExpiresAt                *string    `json:"expires_at,omitempty"`
	StageInstance            *StageInstance `json:"stage_instance,omitempty"`
	GuildScheduledEvent      *GuildScheduledEvent `json:"guild_scheduled_event,omitempty"`
	Uses                     int        `json:"uses"`
	MaxUses                  int        `json:"max_uses"`
	MaxAge                   int        `json:"max_age"`
	Temporary                bool       `json:"temporary"`
	CreatedAt                string     `json:"created_at"`
}

// CreateInviteRequest represents a request to create an invite
type CreateInviteRequest struct {
	MaxAge              *int    `json:"max_age,omitempty"`
	MaxUses             *int    `json:"max_uses,omitempty"`
	Temporary           *bool   `json:"temporary,omitempty"`
	Unique              *bool   `json:"unique,omitempty"`
	TargetType          *int    `json:"target_type,omitempty"`
	TargetUserID        *string `json:"target_user_id,omitempty"`
	TargetApplicationID *string `json:"target_application_id,omitempty"`
}

// Member-related request structures

// ListGuildMembersRequest represents a request to list guild members
type ListGuildMembersRequest struct {
	Limit *int    `json:"limit,omitempty"`
	After *string `json:"after,omitempty"`
}

// SearchGuildMembersRequest represents a request to search guild members
type SearchGuildMembersRequest struct {
	Query string `json:"query"`
	Limit *int   `json:"limit,omitempty"`
}

// AddGuildMemberRequest represents a request to add a guild member
type AddGuildMemberRequest struct {
	AccessToken string    `json:"access_token"`
	Nick        *string   `json:"nick,omitempty"`
	Roles       []string  `json:"roles,omitempty"`
	Mute        *bool     `json:"mute,omitempty"`
	Deaf        *bool     `json:"deaf,omitempty"`
}

// ModifyGuildMemberRequest represents a request to modify a guild member
type ModifyGuildMemberRequest struct {
	Nick                       *string  `json:"nick,omitempty"`
	Roles                      []string `json:"roles,omitempty"`
	Mute                       *bool    `json:"mute,omitempty"`
	Deaf                       *bool    `json:"deaf,omitempty"`
	ChannelID                  *string  `json:"channel_id,omitempty"`
	CommunicationDisabledUntil *string  `json:"communication_disabled_until,omitempty"`
	Flags                      *int     `json:"flags,omitempty"`
}

// ModifyCurrentMemberRequest represents a request to modify the current member
type ModifyCurrentMemberRequest struct {
	Nick *string `json:"nick,omitempty"`
}

// User-related request structures

// ModifyCurrentUserRequest represents a request to modify the current user
type ModifyCurrentUserRequest struct {
	Username *string `json:"username,omitempty"`
	Avatar   *string `json:"avatar,omitempty"`
	Banner   *string `json:"banner,omitempty"`
}

// GetCurrentUserGuildsRequest represents a request to get current user guilds
type GetCurrentUserGuildsRequest struct {
	Before     *string `json:"before,omitempty"`
	After      *string `json:"after,omitempty"`
	Limit      *int    `json:"limit,omitempty"`
	WithCounts *bool   `json:"with_counts,omitempty"`
}

// Application-related request structures

// ModifyCurrentApplicationRequest represents a request to modify the current application
type ModifyCurrentApplicationRequest struct {
	Name                  *string  `json:"name,omitempty"`
	Description           *string  `json:"description,omitempty"`
	Icon                  *string  `json:"icon,omitempty"`
	CoverImage            *string  `json:"cover_image,omitempty"`
	RPCOrigins            []string `json:"rpc_origins,omitempty"`
	BotPublic             *bool    `json:"bot_public,omitempty"`
	BotRequireCodeGrant   *bool    `json:"bot_require_code_grant,omitempty"`
	TermsOfServiceURL     *string  `json:"terms_of_service_url,omitempty"`
	PrivacyPolicyURL      *string  `json:"privacy_policy_url,omitempty"`
	CustomInstallURL      *string  `json:"custom_install_url,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
}

// User represents a Discord user (basic fields for webhook/invite context)
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        *string `json:"avatar"`
}

// DiscordUser represents a full Discord user object
type DiscordUser struct {
	ID               string                 `json:"id"`
	Username         string                 `json:"username"`
	Discriminator    string                 `json:"discriminator"`
	GlobalName       *string                `json:"global_name"`
	Avatar           *string                `json:"avatar"`
	Bot              *bool                  `json:"bot,omitempty"`
	System           *bool                  `json:"system,omitempty"`
	MFAEnabled       *bool                  `json:"mfa_enabled,omitempty"`
	Banner           *string                `json:"banner"`
	AccentColor      *int                   `json:"accent_color"`
	Locale           *string                `json:"locale,omitempty"`
	Verified         *bool                  `json:"verified,omitempty"`
	Email            *string                `json:"email,omitempty"`
	Flags            *int                   `json:"flags,omitempty"`
	PremiumType      *int                   `json:"premium_type,omitempty"`
	PublicFlags      *int                   `json:"public_flags,omitempty"`
	AvatarDecoration map[string]interface{} `json:"avatar_decoration_data,omitempty"`
}

// GuildMember represents a Discord guild member
type GuildMember struct {
	User                       *DiscordUser           `json:"user,omitempty"`
	Nick                       *string                `json:"nick"`
	Avatar                     *string                `json:"avatar"`
	Banner                     *string                `json:"banner"`
	Roles                      []string               `json:"roles"`
	JoinedAt                   *string                `json:"joined_at"`
	PremiumSince               *string                `json:"premium_since"`
	Deaf                       bool                   `json:"deaf"`
	Mute                       bool                   `json:"mute"`
	Flags                      int                    `json:"flags"`
	Pending                    *bool                  `json:"pending,omitempty"`
	Permissions                *string                `json:"permissions,omitempty"`
	CommunicationDisabledUntil *string                `json:"communication_disabled_until"`
	AvatarDecorationData       map[string]interface{} `json:"avatar_decoration_data,omitempty"`
}

// DiscordApplication represents a Discord application
type DiscordApplication struct {
	ID                             string                 `json:"id"`
	Name                           string                 `json:"name"`
	Icon                           *string                `json:"icon"`
	Description                    string                 `json:"description"`
	RPCOrigins                     []string               `json:"rpc_origins"`
	BotPublic                      bool                   `json:"bot_public"`
	BotRequireCodeGrant            bool                   `json:"bot_require_code_grant"`
	Bot                            map[string]interface{} `json:"bot,omitempty"`
	TermsOfServiceURL              *string                `json:"terms_of_service_url"`
	PrivacyPolicyURL               *string                `json:"privacy_policy_url"`
	Owner                          map[string]interface{} `json:"owner,omitempty"`
	Summary                        string                 `json:"summary"`
	VerifyKey                      string                 `json:"verify_key"`
	Team                           map[string]interface{} `json:"team"`
	GuildID                        *string                `json:"guild_id"`
	PrimarySkuID                   *string                `json:"primary_sku_id"`
	Slug                           *string                `json:"slug"`
	CoverImage                     *string                `json:"cover_image"`
	Flags                          *int                   `json:"flags"`
	ApproximateGuildCount          *int                   `json:"approximate_guild_count"`
	RedirectURIs                   []string               `json:"redirect_uris"`
	InteractionsEndpointURL        *string                `json:"interactions_endpoint_url"`
	RoleConnectionsVerificationURL *string                `json:"role_connections_verification_url"`
	Tags                           []string               `json:"tags"`
	InstallParams                  map[string]interface{} `json:"install_params"`
	CustomInstallURL               *string                `json:"custom_install_url"`
}

// GuildIntegration represents a Discord guild integration
type GuildIntegration struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Type              string                 `json:"type"`
	Enabled           bool                   `json:"enabled"`
	Syncing           *bool                  `json:"syncing,omitempty"`
	RoleID            *string                `json:"role_id"`
	EnableEmoticons   *bool                  `json:"enable_emoticons,omitempty"`
	ExpireBehavior    *int                   `json:"expire_behavior,omitempty"`
	ExpireGracePeriod *int                   `json:"expire_grace_period,omitempty"`
	User              map[string]interface{} `json:"user,omitempty"`
	Account           map[string]interface{} `json:"account"`
	SyncedAt          *string                `json:"synced_at"`
	SubscriberCount   *int                   `json:"subscriber_count,omitempty"`
	Revoked           *bool                  `json:"revoked,omitempty"`
	Application       map[string]interface{} `json:"application,omitempty"`
	Scopes            []string               `json:"scopes,omitempty"`
}

// Application represents a Discord application (basic fields for invite context)
type Application struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Icon        *string `json:"icon"`
	Description string  `json:"description"`
}

// StageInstance represents a Discord stage instance (basic fields for invite context)
type StageInstance struct {
	ID                    string `json:"id"`
	GuildID               string `json:"guild_id"`
	ChannelID             string `json:"channel_id"`
	Topic                 string `json:"topic"`
	PrivacyLevel          int    `json:"privacy_level"`
	DiscoverableDisabled  bool   `json:"discoverable_disabled"`
	GuildScheduledEventID *string `json:"guild_scheduled_event_id"`
}

// GuildScheduledEvent represents a Discord scheduled event (basic fields for invite context)
type GuildScheduledEvent struct {
	ID                 string  `json:"id"`
	GuildID            string  `json:"guild_id"`
	ChannelID          *string `json:"channel_id"`
	CreatorID          *string `json:"creator_id"`
	Name               string  `json:"name"`
	Description        *string `json:"description"`
	ScheduledStartTime string  `json:"scheduled_start_time"`
	ScheduledEndTime   *string `json:"scheduled_end_time"`
	PrivacyLevel       int     `json:"privacy_level"`
	Status             int     `json:"status"`
	EntityType         int     `json:"entity_type"`
	EntityID           *string `json:"entity_id"`
}

// GetChannel retrieves a channel by ID
func (c *DiscordClient) GetChannel(ctx context.Context, channelID string) (*Channel, error) {
	resp, err := c.makeRequest(ctx, "GET", "/channels/"+channelID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get channel")
	}
	defer func() { _ = resp.Body.Close() }()

	var channel Channel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil, errors.Wrap(err, "failed to decode channel response")
	}

	return &channel, nil
}

// CreateChannel creates a new channel in a guild
func (c *DiscordClient) CreateChannel(ctx context.Context, req *CreateChannelRequest) (*Channel, error) {
	resp, err := c.makeRequest(ctx, "POST", "/guilds/"+req.GuildID+"/channels", req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create channel")
	}
	defer func() { _ = resp.Body.Close() }()

	var channel Channel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil, errors.Wrap(err, "failed to decode created channel response")
	}

	return &channel, nil
}

// ModifyChannel modifies an existing channel
func (c *DiscordClient) ModifyChannel(ctx context.Context, channelID string, req *ModifyChannelRequest) (*Channel, error) {
	resp, err := c.makeRequest(ctx, "PATCH", "/channels/"+channelID, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to modify channel")
	}
	defer func() { _ = resp.Body.Close() }()

	var channel Channel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil, errors.Wrap(err, "failed to decode modified channel response")
	}

	return &channel, nil
}

// DeleteChannel deletes a channel
func (c *DiscordClient) DeleteChannel(ctx context.Context, channelID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/channels/"+channelID, nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete channel")
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// Webhook methods

// CreateWebhook creates a new webhook in a channel
func (c *DiscordClient) CreateWebhook(ctx context.Context, channelID string, req *CreateWebhookRequest) (*Webhook, error) {
	resp, err := c.makeRequest(ctx, "POST", "/channels/"+channelID+"/webhooks", req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create webhook")
	}
	defer func() { _ = resp.Body.Close() }()

	var webhook Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return nil, errors.Wrap(err, "failed to decode created webhook response")
	}

	return &webhook, nil
}

// GetWebhook retrieves a webhook by ID
func (c *DiscordClient) GetWebhook(ctx context.Context, webhookID string) (*Webhook, error) {
	resp, err := c.makeRequest(ctx, "GET", "/webhooks/"+webhookID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get webhook")
	}
	defer func() { _ = resp.Body.Close() }()

	var webhook Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return nil, errors.Wrap(err, "failed to decode webhook response")
	}

	return &webhook, nil
}

// ModifyWebhook modifies an existing webhook
func (c *DiscordClient) ModifyWebhook(ctx context.Context, webhookID string, req *ModifyWebhookRequest) (*Webhook, error) {
	resp, err := c.makeRequest(ctx, "PATCH", "/webhooks/"+webhookID, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to modify webhook")
	}
	defer func() { _ = resp.Body.Close() }()

	var webhook Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return nil, errors.Wrap(err, "failed to decode modified webhook response")
	}

	return &webhook, nil
}

// DeleteWebhook deletes a webhook
func (c *DiscordClient) DeleteWebhook(ctx context.Context, webhookID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/webhooks/"+webhookID, nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete webhook")
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// GetChannelWebhooks gets all webhooks for a channel
func (c *DiscordClient) GetChannelWebhooks(ctx context.Context, channelID string) ([]Webhook, error) {
	resp, err := c.makeRequest(ctx, "GET", "/channels/"+channelID+"/webhooks", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get channel webhooks")
	}
	defer func() { _ = resp.Body.Close() }()

	var webhooks []Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhooks); err != nil {
		return nil, errors.Wrap(err, "failed to decode channel webhooks response")
	}

	return webhooks, nil
}

// GetGuildWebhooks gets all webhooks for a guild
func (c *DiscordClient) GetGuildWebhooks(ctx context.Context, guildID string) ([]Webhook, error) {
	resp, err := c.makeRequest(ctx, "GET", "/guilds/"+guildID+"/webhooks", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get guild webhooks")
	}
	defer func() { _ = resp.Body.Close() }()

	var webhooks []Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhooks); err != nil {
		return nil, errors.Wrap(err, "failed to decode guild webhooks response")
	}

	return webhooks, nil
}

// Invite methods

// CreateChannelInvite creates a new invite for a channel
func (c *DiscordClient) CreateChannelInvite(ctx context.Context, channelID string, req *CreateInviteRequest) (*Invite, error) {
	resp, err := c.makeRequest(ctx, "POST", "/channels/"+channelID+"/invites", req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create channel invite")
	}
	defer func() { _ = resp.Body.Close() }()

	var invite Invite
	if err := json.NewDecoder(resp.Body).Decode(&invite); err != nil {
		return nil, errors.Wrap(err, "failed to decode created invite response")
	}

	return &invite, nil
}

// GetInvite retrieves an invite by code
func (c *DiscordClient) GetInvite(ctx context.Context, inviteCode string) (*Invite, error) {
	resp, err := c.makeRequest(ctx, "GET", "/invites/"+inviteCode+"?with_counts=true&with_expiration=true", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get invite")
	}
	defer func() { _ = resp.Body.Close() }()

	var invite Invite
	if err := json.NewDecoder(resp.Body).Decode(&invite); err != nil {
		return nil, errors.Wrap(err, "failed to decode invite response")
	}

	return &invite, nil
}

// DeleteInvite deletes an invite
func (c *DiscordClient) DeleteInvite(ctx context.Context, inviteCode string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/invites/"+inviteCode, nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete invite")
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// GetChannelInvites gets all invites for a channel
func (c *DiscordClient) GetChannelInvites(ctx context.Context, channelID string) ([]Invite, error) {
	resp, err := c.makeRequest(ctx, "GET", "/channels/"+channelID+"/invites", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get channel invites")
	}
	defer func() { _ = resp.Body.Close() }()

	var invites []Invite
	if err := json.NewDecoder(resp.Body).Decode(&invites); err != nil {
		return nil, errors.Wrap(err, "failed to decode channel invites response")
	}

	return invites, nil
}

// GetGuildInvites gets all invites for a guild
func (c *DiscordClient) GetGuildInvites(ctx context.Context, guildID string) ([]Invite, error) {
	resp, err := c.makeRequest(ctx, "GET", "/guilds/"+guildID+"/invites", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get guild invites")
	}
	defer func() { _ = resp.Body.Close() }()

	var invites []Invite
	if err := json.NewDecoder(resp.Body).Decode(&invites); err != nil {
		return nil, errors.Wrap(err, "failed to decode guild invites response")
	}

	return invites, nil
}

// Member Client Methods

// GetGuildMember retrieves a guild member by user ID
func (c *DiscordClient) GetGuildMember(ctx context.Context, guildID, userID string) (*GuildMember, error) {
	resp, err := c.makeRequest(ctx, "GET", "/guilds/"+guildID+"/members/"+userID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get guild member")
	}
	defer func() { _ = resp.Body.Close() }()

	var member GuildMember
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, errors.Wrap(err, "failed to decode member response")
	}

	return &member, nil
}

// ListGuildMembers lists guild members
func (c *DiscordClient) ListGuildMembers(ctx context.Context, guildID string, req *ListGuildMembersRequest) ([]GuildMember, error) {
	query := ""
	if req != nil {
		params := make([]string, 0)
		if req.Limit != nil {
			params = append(params, fmt.Sprintf("limit=%d", *req.Limit))
		}
		if req.After != nil {
			params = append(params, fmt.Sprintf("after=%s", *req.After))
		}
		if len(params) > 0 {
			query = "?" + strings.Join(params, "&")
		}
	}

	resp, err := c.makeRequest(ctx, "GET", "/guilds/"+guildID+"/members"+query, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list guild members")
	}
	defer func() { _ = resp.Body.Close() }()

	var members []GuildMember
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, errors.Wrap(err, "failed to decode members response")
	}

	return members, nil
}

// AddGuildMember adds a user to a guild (requires OAuth2 access token)
func (c *DiscordClient) AddGuildMember(ctx context.Context, guildID, userID string, req *AddGuildMemberRequest) (*GuildMember, error) {
	resp, err := c.makeRequest(ctx, "PUT", "/guilds/"+guildID+"/members/"+userID, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add guild member")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 204 {
		// Member was already in the guild
		return c.GetGuildMember(ctx, guildID, userID)
	}

	var member GuildMember
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, errors.Wrap(err, "failed to decode added member response")
	}

	return &member, nil
}

// ModifyGuildMember modifies a guild member
func (c *DiscordClient) ModifyGuildMember(ctx context.Context, guildID, userID string, req *ModifyGuildMemberRequest) (*GuildMember, error) {
	resp, err := c.makeRequest(ctx, "PATCH", "/guilds/"+guildID+"/members/"+userID, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to modify guild member")
	}
	defer func() { _ = resp.Body.Close() }()

	var member GuildMember
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, errors.Wrap(err, "failed to decode modified member response")
	}

	return &member, nil
}

// ModifyCurrentMember modifies the current user's member in a guild
func (c *DiscordClient) ModifyCurrentMember(ctx context.Context, guildID string, req *ModifyCurrentMemberRequest) (*GuildMember, error) {
	resp, err := c.makeRequest(ctx, "PATCH", "/guilds/"+guildID+"/members/@me", req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to modify current member")
	}
	defer func() { _ = resp.Body.Close() }()

	var member GuildMember
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, errors.Wrap(err, "failed to decode current member response")
	}

	return &member, nil
}

// AddGuildMemberRole adds a role to a guild member
func (c *DiscordClient) AddGuildMemberRole(ctx context.Context, guildID, userID, roleID string) error {
	resp, err := c.makeRequest(ctx, "PUT", "/guilds/"+guildID+"/members/"+userID+"/roles/"+roleID, nil)
	if err != nil {
		return errors.Wrap(err, "failed to add guild member role")
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// RemoveGuildMemberRole removes a role from a guild member
func (c *DiscordClient) RemoveGuildMemberRole(ctx context.Context, guildID, userID, roleID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/guilds/"+guildID+"/members/"+userID+"/roles/"+roleID, nil)
	if err != nil {
		return errors.Wrap(err, "failed to remove guild member role")
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// RemoveGuildMember removes/kicks a member from a guild
func (c *DiscordClient) RemoveGuildMember(ctx context.Context, guildID, userID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/guilds/"+guildID+"/members/"+userID, nil)
	if err != nil {
		return errors.Wrap(err, "failed to remove guild member")
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// SearchGuildMembers searches for guild members by username or nickname
func (c *DiscordClient) SearchGuildMembers(ctx context.Context, guildID string, req *SearchGuildMembersRequest) ([]GuildMember, error) {
	query := fmt.Sprintf("?query=%s", req.Query)
	if req.Limit != nil {
		query += fmt.Sprintf("&limit=%d", *req.Limit)
	}

	resp, err := c.makeRequest(ctx, "GET", "/guilds/"+guildID+"/members/search"+query, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to search guild members")
	}
	defer func() { _ = resp.Body.Close() }()

	var members []GuildMember
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, errors.Wrap(err, "failed to decode search members response")
	}

	return members, nil
}

// User Client Methods

// GetUser retrieves a user by ID
func (c *DiscordClient) GetUser(ctx context.Context, userID string) (*DiscordUser, error) {
	resp, err := c.makeRequest(ctx, "GET", "/users/"+userID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user")
	}
	defer func() { _ = resp.Body.Close() }()

	var user DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, errors.Wrap(err, "failed to decode user response")
	}

	return &user, nil
}

// GetCurrentUser retrieves the current authenticated user
func (c *DiscordClient) GetCurrentUser(ctx context.Context) (*DiscordUser, error) {
	resp, err := c.makeRequest(ctx, "GET", "/users/@me", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current user")
	}
	defer func() { _ = resp.Body.Close() }()

	var user DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, errors.Wrap(err, "failed to decode current user response")
	}

	return &user, nil
}

// ModifyCurrentUser modifies the current authenticated user
func (c *DiscordClient) ModifyCurrentUser(ctx context.Context, req *ModifyCurrentUserRequest) (*DiscordUser, error) {
	resp, err := c.makeRequest(ctx, "PATCH", "/users/@me", req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to modify current user")
	}
	defer func() { _ = resp.Body.Close() }()

	var user DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, errors.Wrap(err, "failed to decode modified user response")
	}

	return &user, nil
}

// GetCurrentUserGuilds gets the current user's guilds
func (c *DiscordClient) GetCurrentUserGuilds(ctx context.Context, req *GetCurrentUserGuildsRequest) ([]Guild, error) {
	query := ""
	if req != nil {
		params := make([]string, 0)
		if req.Before != nil {
			params = append(params, fmt.Sprintf("before=%s", *req.Before))
		}
		if req.After != nil {
			params = append(params, fmt.Sprintf("after=%s", *req.After))
		}
		if req.Limit != nil {
			params = append(params, fmt.Sprintf("limit=%d", *req.Limit))
		}
		if len(params) > 0 {
			query = "?" + strings.Join(params, "&")
		}
	}

	resp, err := c.makeRequest(ctx, "GET", "/users/@me/guilds"+query, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current user guilds")
	}
	defer func() { _ = resp.Body.Close() }()

	var guilds []Guild
	if err := json.NewDecoder(resp.Body).Decode(&guilds); err != nil {
		return nil, errors.Wrap(err, "failed to decode current user guilds response")
	}

	return guilds, nil
}

// LeaveGuild leaves a guild
func (c *DiscordClient) LeaveGuild(ctx context.Context, guildID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/users/@me/guilds/"+guildID, nil)
	if err != nil {
		return errors.Wrap(err, "failed to leave guild")
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// Application Client Methods

// GetApplication retrieves an application by ID
func (c *DiscordClient) GetApplication(ctx context.Context, applicationID string) (*DiscordApplication, error) {
	resp, err := c.makeRequest(ctx, "GET", "/applications/"+applicationID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get application")
	}
	defer func() { _ = resp.Body.Close() }()

	var application DiscordApplication
	if err := json.NewDecoder(resp.Body).Decode(&application); err != nil {
		return nil, errors.Wrap(err, "failed to decode application response")
	}

	return &application, nil
}

// GetCurrentApplication retrieves the current application
func (c *DiscordClient) GetCurrentApplication(ctx context.Context) (*DiscordApplication, error) {
	resp, err := c.makeRequest(ctx, "GET", "/applications/@me", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current application")
	}
	defer func() { _ = resp.Body.Close() }()

	var application DiscordApplication
	if err := json.NewDecoder(resp.Body).Decode(&application); err != nil {
		return nil, errors.Wrap(err, "failed to decode current application response")
	}

	return &application, nil
}

// ModifyCurrentApplication modifies the current application
func (c *DiscordClient) ModifyCurrentApplication(ctx context.Context, req *ModifyCurrentApplicationRequest) (*DiscordApplication, error) {
	resp, err := c.makeRequest(ctx, "PATCH", "/applications/@me", req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to edit current application")
	}
	defer func() { _ = resp.Body.Close() }()

	var application DiscordApplication
	if err := json.NewDecoder(resp.Body).Decode(&application); err != nil {
		return nil, errors.Wrap(err, "failed to decode edited application response")
	}

	return &application, nil
}

// Integration Client Methods

// GetGuildIntegrations retrieves integrations for a guild
func (c *DiscordClient) GetGuildIntegrations(ctx context.Context, guildID string) ([]GuildIntegration, error) {
	resp, err := c.makeRequest(ctx, "GET", "/guilds/"+guildID+"/integrations", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get guild integrations")
	}
	defer func() { _ = resp.Body.Close() }()

	var integrations []GuildIntegration
	if err := json.NewDecoder(resp.Body).Decode(&integrations); err != nil {
		return nil, errors.Wrap(err, "failed to decode integrations response")
	}

	return integrations, nil
}

// DeleteGuildIntegration deletes a guild integration
func (c *DiscordClient) DeleteGuildIntegration(ctx context.Context, guildID, integrationID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/guilds/"+guildID+"/integrations/"+integrationID, nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete guild integration")
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}