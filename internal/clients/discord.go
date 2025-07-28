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
	"time"

	"github.com/pkg/errors"
)

const (
	// DiscordAPIBaseURL is the base URL for the Discord API
	DiscordAPIBaseURL = "https://discord.com/api/v10"
)

// DiscordClient is a client for the Discord API
type DiscordClient struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

// NewDiscordClient creates a new Discord API client
func NewDiscordClient(token string) *DiscordClient {
	return &DiscordClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token:   token,
		baseURL: DiscordAPIBaseURL,
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
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal request body")
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Authorization", "Bot "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Crossplane Discord Provider/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform request")
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	return nil
}

// ListGuilds lists all guilds the bot is a member of
func (c *DiscordClient) ListGuilds(ctx context.Context) ([]Guild, error) {
	resp, err := c.makeRequest(ctx, "GET", "/users/@me/guilds", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list guilds")
	}
	defer resp.Body.Close()

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
	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal create role request")
	}

	resp, err := c.makeRequest(ctx, "POST", fmt.Sprintf("/guilds/%s/roles", guildID), body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create role")
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal modify role request")
	}

	resp, err := c.makeRequest(ctx, "PATCH", fmt.Sprintf("/guilds/%s/roles/%s", guildID, roleID), body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to modify role")
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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

// GetChannel retrieves a channel by ID
func (c *DiscordClient) GetChannel(ctx context.Context, channelID string) (*Channel, error) {
	resp, err := c.makeRequest(ctx, "GET", "/channels/"+channelID, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get channel")
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	return nil
}