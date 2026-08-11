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

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	channelv1alpha1 "github.com/rossigee/provider-discord/apis/channel/v1alpha1"
	deduplicationv1alpha1 "github.com/rossigee/provider-discord/apis/deduplication/v1alpha1"
	rolev1alpha1 "github.com/rossigee/provider-discord/apis/role/v1alpha1"
	webhookv1alpha1 "github.com/rossigee/provider-discord/apis/webhook/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// discordDeleteDelay is the minimum spacing between channel DELETE calls to stay within Discord's
	// per-route rate limit (typically ~5 req/s for channel deletes).
	discordDeleteDelay = 500 * time.Millisecond

	// discordDefaultRetryAfter is used when a 429 response carries no Retry-After header.
	discordDefaultRetryAfter = 2 * time.Second
)

// DeduplicationService provides methods for analyzing and deduplicating Discord channels.
type DeduplicationService struct {
	httpClient *http.Client
	baseURL    string
	botToken   string
	kubeClient client.Client
}

// NewDeduplicationService creates a new DeduplicationService.
func NewDeduplicationService(httpClient *http.Client, baseURL, botToken string, kubeClient client.Client) *DeduplicationService {
	if baseURL == "" {
		baseURL = "https://discord.com/api/v10"
	}
	return &DeduplicationService{
		httpClient: httpClient,
		baseURL:    baseURL,
		botToken:   botToken,
		kubeClient: kubeClient,
	}
}

// Guild represents a Discord guild.
type Guild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Channel represents a Discord channel.
type Channel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	GuildID  string `json:"guild_id"`
	Position int    `json:"position"`
	ParentID string `json:"parent_id"`
}

// Webhook represents a Discord webhook.
type Webhook struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id"`
}

// Role represents a Discord guild role.
type Role struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

// DuplicateGroup represents a group of duplicate channels with the same name.
//
// Keep-selection is message-history-based, not ID- or Position-based: a channel with
// zero sampled messages is safe to delete (it's either a fresh duplicate from the
// create-race or genuinely unused); a channel with any messages is never auto-deleted.
// This means "keep" isn't necessarily a single channel - see KeepIndices.
type DuplicateGroup struct {
	Name     string
	Channels []Channel

	// MessageCounts holds a message-history sample count per channel in Channels
	// (same index), capped at 100 (Discord's per-request page limit - this is "at
	// least N messages" for N==100, exact for N<100).
	MessageCounts []int

	// KeepIndices are the indices (into Channels) of channels that must NOT be
	// deleted: every channel with a non-zero MessageCounts entry, or - only when
	// every channel in the group has zero messages - the single oldest (lowest
	// snowflake ID) channel as a last-resort tie-breaker.
	KeepIndices []int

	// DeleteIndices are the indices (into Channels) of channels safe to delete:
	// exactly the ones with zero sampled messages, excluding whichever index the
	// all-zero fallback chose to keep.
	DeleteIndices []int

	// NeedsManualReview is true when more than one channel in the group has message
	// history - i.e. real activity happened in more than one duplicate before anyone
	// noticed. When true, DeleteIndices only contains the zero-message channels;
	// the multiple non-empty channels are left for a human to resolve.
	NeedsManualReview bool
}

// WebhookDuplicateGroup represents a group of duplicate webhooks with the same name.
type WebhookDuplicateGroup struct {
	Name      string
	Webhooks  []Webhook
	KeepIndex int // Index of the webhook to keep (oldest by ID)
}

// RoleDuplicateGroup represents a group of duplicate roles with the same name.
type RoleDuplicateGroup struct {
	Name      string
	Roles     []Role
	KeepIndex int // Index of the role to keep (oldest by ID)
}

// olderSnowflake reports whether a is an earlier-created Discord snowflake ID than b
// (lower numeric value = created first). Discord IDs are 64-bit unsigned integers
// encoded as decimal strings; comparing them as strings only happens to work while
// all IDs share the same digit length; parse-and-compare numerically to be correct
// regardless of length or Discord's ID epoch. Falls back to string comparison if
// either value fails to parse, which fails safe (never panics) though it should not
// occur for genuine Discord-issued IDs.
func olderSnowflake(a, b string) bool {
	aNum, aErr := strconv.ParseUint(a, 10, 64)
	bNum, bErr := strconv.ParseUint(b, 10, 64)
	if aErr != nil || bErr != nil {
		return a < b
	}
	return aNum < bNum
}

// AnalyzeAndDeduplicateResult contains the results of a deduplication operation.
type AnalyzeAndDeduplicateResult struct {
	Mode     string
	Guilds   map[string]*GuildResult
	Summary  *deduplicationv1alpha1.DeduplicationSummary
	HasError bool
	Error    string
}

// GuildResult contains results for a specific guild.
type GuildResult struct {
	GuildID                  string
	GuildName                string
	TotalChannels            int
	DuplicateGroups          []DuplicateGroup
	ChannelsDeleted          int
	TotalWebhooks            int
	WebhookDuplicateGroups   []WebhookDuplicateGroup
	WebhooksDeleted          int
	TotalRoles               int
	RoleDuplicateGroups      []RoleDuplicateGroup
	RolesDeleted             int
	OrphanedResourcesDeleted int
	Errors                   []string
}

// AnalyzeAndDeduplicate analyzes guilds for duplicate channels and optionally deletes them.
func (s *DeduplicationService) AnalyzeAndDeduplicate(ctx context.Context, mode string, targetGuilds []string) (*AnalyzeAndDeduplicateResult, error) {
	return s.AnalyzeAndDeduplicateWithCleanup(ctx, mode, targetGuilds, true)
}

// AnalyzeAndDeduplicateWithCleanup analyzes guilds for duplicate channels and optionally cleans up Crossplane resources.
func (s *DeduplicationService) AnalyzeAndDeduplicateWithCleanup(ctx context.Context, mode string, targetGuilds []string, deleteOrphanedResources bool) (*AnalyzeAndDeduplicateResult, error) {
	result := &AnalyzeAndDeduplicateResult{
		Mode:   mode,
		Guilds: make(map[string]*GuildResult),
		Summary: &deduplicationv1alpha1.DeduplicationSummary{
			TotalGuildsAnalyzed:         0,
			TotalChannelsAnalyzed:       0,
			DuplicateGroupsFound:        0,
			TotalDuplicateChannelsFound: 0,
			ChannelsDeleted:             0,
			OrphanedResourcesDeleted:    0,
		},
	}

	// Fetch all guilds the bot is a member of
	guilds, err := s.getGuilds(ctx)
	if err != nil {
		result.HasError = true
		result.Error = fmt.Sprintf("failed to fetch guilds: %v", err)
		return result, err
	}

	// Filter by target guilds if specified
	if len(targetGuilds) > 0 {
		filtered := make([]Guild, 0)
		targetMap := make(map[string]bool)
		for _, gid := range targetGuilds {
			targetMap[gid] = true
		}
		for _, guild := range guilds {
			if targetMap[guild.ID] {
				filtered = append(filtered, guild)
			}
		}
		guilds = filtered
	}

	// Process each guild
	for _, guild := range guilds {
		guildResult := s.analyzeGuild(ctx, guild, mode, deleteOrphanedResources)
		result.Guilds[guild.ID] = guildResult

		// Update summary
		result.Summary.TotalGuildsAnalyzed++
		result.Summary.TotalChannelsAnalyzed += guildResult.TotalChannels
		result.Summary.DuplicateGroupsFound += len(guildResult.DuplicateGroups)
		result.Summary.ChannelsDeleted += guildResult.ChannelsDeleted
		result.Summary.TotalWebhooksAnalyzed += guildResult.TotalWebhooks
		result.Summary.WebhookDuplicateGroupsFound += len(guildResult.WebhookDuplicateGroups)
		result.Summary.WebhooksDeleted += guildResult.WebhooksDeleted
		result.Summary.TotalRolesAnalyzed += guildResult.TotalRoles
		result.Summary.RoleDuplicateGroupsFound += len(guildResult.RoleDuplicateGroups)
		result.Summary.RolesDeleted += guildResult.RolesDeleted
		result.Summary.OrphanedResourcesDeleted += guildResult.OrphanedResourcesDeleted

		// Count total duplicates (each group has len-1 duplicates)
		for _, group := range guildResult.DuplicateGroups {
			result.Summary.TotalDuplicateChannelsFound += len(group.Channels) - 1
		}
		for _, group := range guildResult.WebhookDuplicateGroups {
			result.Summary.TotalDuplicateWebhooksFound += len(group.Webhooks) - 1
		}
		for _, group := range guildResult.RoleDuplicateGroups {
			result.Summary.TotalDuplicateRolesFound += len(group.Roles) - 1
		}

		// Log any errors encountered during processing
		if len(guildResult.Errors) > 0 {
			result.HasError = true
		}
	}

	return result, nil
}

// analyzeGuild analyzes a single guild for duplicates.
func (s *DeduplicationService) analyzeGuild(ctx context.Context, guild Guild, mode string, deleteOrphanedResources bool) *GuildResult {
	result := &GuildResult{
		GuildID:   guild.ID,
		GuildName: guild.Name,
		Errors:    make([]string, 0),
	}

	// Get channels for this guild
	channels, err := s.getChannels(ctx, guild.ID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to fetch channels: %v", err))
		return result
	}

	result.TotalChannels = len(channels)

	// Group channels by name to find duplicates
	nameGroups := make(map[string][]Channel)
	for _, channel := range channels {
		nameGroups[channel.Name] = append(nameGroups[channel.Name], channel)
	}

	// Find and process duplicate groups. Keep-selection is message-history-based:
	// a channel with zero sampled messages is safe to delete; a channel with any
	// messages is never auto-deleted. See DuplicateGroup's doc comment.
	for name, group := range nameGroups {
		if len(group) <= 1 {
			continue
		}

		messageCounts := make([]int, len(group))
		for i, channel := range group {
			if i > 0 {
				select {
				case <-time.After(discordDeleteDelay):
				case <-ctx.Done():
					result.Errors = append(result.Errors, "context cancelled during message sampling")
					return result
				}
			}
			count, err := s.getChannelMessageCount(ctx, channel.ID)
			if err != nil {
				// Fail safe: if we can't determine message history, treat as non-empty
				// so it's never auto-deleted based on incomplete information.
				result.Errors = append(result.Errors, fmt.Sprintf("failed to sample messages for channel %s (%s): %v — treating as non-empty", channel.ID, name, err))
				messageCounts[i] = -1
			} else {
				messageCounts[i] = count
			}
		}

		var keepIndices, deleteIndices []int
		nonEmptyCount := 0
		for _, count := range messageCounts {
			if count != 0 {
				nonEmptyCount++
			}
		}

		needsManualReview := false
		if nonEmptyCount == 0 {
			// Every channel is empty: nothing to lose either way, fall back to
			// lowest snowflake ID (earliest created) as a tie-breaker.
			keep := 0
			for i, channel := range group {
				if olderSnowflake(channel.ID, group[keep].ID) {
					keep = i
				}
			}
			for i := range group {
				if i == keep {
					keepIndices = append(keepIndices, i)
				} else {
					deleteIndices = append(deleteIndices, i)
				}
			}
		} else {
			// Keep every non-empty channel; delete only the empty ones. If more
			// than one is non-empty, flag for manual review rather than guessing.
			needsManualReview = nonEmptyCount > 1
			for i, count := range messageCounts {
				if count != 0 {
					keepIndices = append(keepIndices, i)
				} else {
					deleteIndices = append(deleteIndices, i)
				}
			}
		}

		dupGroup := DuplicateGroup{
			Name:              name,
			Channels:          group,
			MessageCounts:     messageCounts,
			KeepIndices:       keepIndices,
			DeleteIndices:     deleteIndices,
			NeedsManualReview: needsManualReview,
		}
		result.DuplicateGroups = append(result.DuplicateGroups, dupGroup)

		if needsManualReview {
			result.Errors = append(result.Errors, fmt.Sprintf("manual review needed: %d channels named %q have message history — only the %d empty duplicate(s) will be auto-deleted", nonEmptyCount, name, len(deleteIndices)))
		}

		// If in action mode, delete only the channels marked safe to delete
		if mode == "action" {
			deletesMade := 0
			duplicatesToDelete := len(deleteIndices)

			for _, i := range deleteIndices {
				channel := group[i]

				// Rate-limit: space out DELETE calls to avoid Discord 429 responses
				if deletesMade > 0 {
					select {
					case <-time.After(discordDeleteDelay):
					case <-ctx.Done():
						result.Errors = append(result.Errors, "context cancelled during channel deletion")
						return result
					}
				}

				err := s.deleteChannel(ctx, channel.ID)
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("failed to delete channel %s (%s): %v", channel.ID, channel.Name, err))
				} else {
					deletesMade++
					result.ChannelsDeleted++

					// Clean up corresponding Crossplane resources if requested
					if deleteOrphanedResources {
						deletedCount := s.deleteOrphanedResources(ctx, channel.ID)
						result.OrphanedResourcesDeleted += deletedCount
					}
				}
			}

			// Log if some duplicates failed to delete
			if deletesMade < duplicatesToDelete {
				result.Errors = append(result.Errors, fmt.Sprintf("partial deletion: %d/%d duplicates of %q deleted", deletesMade, duplicatesToDelete, name))
			}
		}
	}

	s.analyzeGuildWebhooks(ctx, guild, mode, deleteOrphanedResources, result)
	s.analyzeGuildRoles(ctx, guild, mode, deleteOrphanedResources, result)

	return result
}

// analyzeGuildWebhooks analyzes a single guild's webhooks for duplicates, mirroring
// analyzeGuild's logic (group by name, keep lowest snowflake ID, delete the rest in
// action mode), and writes its findings into the shared GuildResult.
func (s *DeduplicationService) analyzeGuildWebhooks(ctx context.Context, guild Guild, mode string, deleteOrphanedResources bool, result *GuildResult) {
	webhooks, err := s.getGuildWebhooks(ctx, guild.ID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to fetch webhooks: %v", err))
		return
	}

	result.TotalWebhooks = len(webhooks)

	nameGroups := make(map[string][]Webhook)
	for _, wh := range webhooks {
		nameGroups[wh.Name] = append(nameGroups[wh.Name], wh)
	}

	for name, group := range nameGroups {
		if len(group) <= 1 {
			continue
		}

		// Keep the webhook with the lowest (earliest-created) snowflake ID - see
		// olderSnowflake's doc comment for why Position-like fields must not be used.
		keepIndex := 0
		for i, wh := range group {
			if olderSnowflake(wh.ID, group[keepIndex].ID) {
				keepIndex = i
			}
		}

		result.WebhookDuplicateGroups = append(result.WebhookDuplicateGroups, WebhookDuplicateGroup{
			Name:      name,
			Webhooks:  group,
			KeepIndex: keepIndex,
		})

		if mode != "action" {
			continue
		}

		deletesMade := 0
		duplicatesToDelete := len(group) - 1

		for i, wh := range group {
			if i == keepIndex {
				continue
			}

			if deletesMade > 0 {
				select {
				case <-time.After(discordDeleteDelay):
				case <-ctx.Done():
					result.Errors = append(result.Errors, "context cancelled during webhook deletion")
					return
				}
			}

			if err := s.deleteWebhook(ctx, wh.ID); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to delete webhook %s (%s): %v", wh.ID, wh.Name, err))
			} else {
				deletesMade++
				result.WebhooksDeleted++

				if deleteOrphanedResources {
					result.OrphanedResourcesDeleted += s.deleteOrphanedWebhookResources(ctx, wh.ID)
				}
			}
		}

		if deletesMade < duplicatesToDelete {
			result.Errors = append(result.Errors, fmt.Sprintf("partial deletion: %d/%d duplicates of webhook %q deleted", deletesMade, duplicatesToDelete, name))
		}
	}
}

// analyzeGuildRoles analyzes a single guild's roles for duplicates, mirroring
// analyzeGuild's logic (group by name, keep lowest snowflake ID, delete the rest in
// action mode), and writes its findings into the shared GuildResult.
func (s *DeduplicationService) analyzeGuildRoles(ctx context.Context, guild Guild, mode string, deleteOrphanedResources bool, result *GuildResult) {
	roles, err := s.getGuildRoles(ctx, guild.ID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to fetch roles: %v", err))
		return
	}

	result.TotalRoles = len(roles)

	nameGroups := make(map[string][]Role)
	for _, role := range roles {
		nameGroups[role.Name] = append(nameGroups[role.Name], role)
	}

	for name, group := range nameGroups {
		if len(group) <= 1 {
			continue
		}

		// Keep the role with the lowest (earliest-created) snowflake ID. Roles also
		// carry a Position (hierarchy order), which has the exact same pitfall as
		// Channel.Position - never use it to decide which one is "original".
		keepIndex := 0
		for i, role := range group {
			if olderSnowflake(role.ID, group[keepIndex].ID) {
				keepIndex = i
			}
		}

		result.RoleDuplicateGroups = append(result.RoleDuplicateGroups, RoleDuplicateGroup{
			Name:      name,
			Roles:     group,
			KeepIndex: keepIndex,
		})

		if mode != "action" {
			continue
		}

		deletesMade := 0
		duplicatesToDelete := len(group) - 1

		for i, role := range group {
			if i == keepIndex {
				continue
			}

			if deletesMade > 0 {
				select {
				case <-time.After(discordDeleteDelay):
				case <-ctx.Done():
					result.Errors = append(result.Errors, "context cancelled during role deletion")
					return
				}
			}

			if err := s.deleteRole(ctx, guild.ID, role.ID); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to delete role %s (%s): %v", role.ID, role.Name, err))
			} else {
				deletesMade++
				result.RolesDeleted++

				if deleteOrphanedResources {
					result.OrphanedResourcesDeleted += s.deleteOrphanedRoleResources(ctx, role.ID)
				}
			}
		}

		if deletesMade < duplicatesToDelete {
			result.Errors = append(result.Errors, fmt.Sprintf("partial deletion: %d/%d duplicates of role %q deleted", deletesMade, duplicatesToDelete, name))
		}
	}
}

// deleteChannel deletes a Discord channel by ID.
// It respects Discord's 429 rate-limit response and blocks for the Retry-After duration.
func (s *DeduplicationService) deleteChannel(ctx context.Context, channelID string) error {
	url := fmt.Sprintf("%s/channels/%s", s.baseURL, channelID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request for channel %s: %w", channelID, err)
	}

	req.Header.Set("Authorization", "Bot "+s.botToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE request failed for channel %s: %w", channelID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 429 {
		// Rate limited — respect the Retry-After header before returning an error
		// so the caller can retry without an immediate 429 storm.
		retryAfter := discordDefaultRetryAfter
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, parseErr := strconv.ParseFloat(ra, 64); parseErr == nil && secs > 0 {
				retryAfter = time.Duration(secs * float64(time.Second))
			}
		}
		select {
		case <-time.After(retryAfter):
		case <-ctx.Done():
			return ctx.Err()
		}
		return fmt.Errorf("discord API rate limited (429) for channel %s; waited %s — caller should retry", channelID, retryAfter)
	}

	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord API error (status %d) deleting channel %s: %s", resp.StatusCode, channelID, string(body))
	}

	return nil
}

// getGuilds retrieves all guilds the bot is a member of, handling pagination.
// Discord returns at most 200 guilds per request; this function follows the
// `after` cursor until all pages are exhausted.
func (s *DeduplicationService) getGuilds(ctx context.Context) ([]Guild, error) {
	var all []Guild
	after := ""

	for {
		url := fmt.Sprintf("%s/users/@me/guilds?limit=200", s.baseURL)
		if after != "" {
			url += "&after=" + after
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bot "+s.botToken)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("discord API error: %d - %s", resp.StatusCode, string(body))
		}

		var page []Guild
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			_ = resp.Body.Close()
			return nil, err
		}
		_ = resp.Body.Close()

		all = append(all, page...)

		// Discord signals the last page by returning fewer than the requested limit
		if len(page) < 200 {
			break
		}
		// Advance cursor to the last guild ID on this page
		after = page[len(page)-1].ID
	}

	return all, nil
}

// getChannels retrieves all channels for a specific guild.
func (s *DeduplicationService) getChannels(ctx context.Context, guildID string) ([]Channel, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/guilds/%s/channels", s.baseURL, guildID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bot "+s.botToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord API error: %d - %s", resp.StatusCode, string(body))
	}

	var channels []Channel
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		return nil, err
	}

	return channels, nil
}

// getChannelMessageCount samples up to 100 messages from a channel and returns how many
// exist. Discord's API has no direct total-message-count endpoint - an exact count would
// require paginating the channel's entire history, which is expensive and heavy on rate
// limits. This is a cheap proxy sufficient to distinguish "empty duplicate" (0) from "has
// real history" (>0); for channels with over 100 messages the result is simply capped at
// 100 rather than being exact, which is fine since only the zero/non-zero distinction
// drives keep-selection.
func (s *DeduplicationService) getChannelMessageCount(ctx context.Context, channelID string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/channels/%s/messages?limit=100", s.baseURL, channelID), nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bot "+s.botToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 429 {
		retryAfter := discordDefaultRetryAfter
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, parseErr := strconv.ParseFloat(ra, 64); parseErr == nil && secs > 0 {
				retryAfter = time.Duration(secs * float64(time.Second))
			}
		}
		select {
		case <-time.After(retryAfter):
		case <-ctx.Done():
			return 0, ctx.Err()
		}
		return 0, fmt.Errorf("discord API rate limited (429) sampling messages for channel %s; waited %s — caller should retry", channelID, retryAfter)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("discord API error: %d - %s", resp.StatusCode, string(body))
	}

	var messages []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return 0, err
	}

	return len(messages), nil
}

// getGuildWebhooks retrieves every webhook across every channel in a guild in one call.
func (s *DeduplicationService) getGuildWebhooks(ctx context.Context, guildID string) ([]Webhook, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/guilds/%s/webhooks", s.baseURL, guildID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bot "+s.botToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord API error: %d - %s", resp.StatusCode, string(body))
	}

	var webhooks []Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhooks); err != nil {
		return nil, err
	}

	return webhooks, nil
}

// getGuildRoles retrieves all roles for a specific guild.
func (s *DeduplicationService) getGuildRoles(ctx context.Context, guildID string) ([]Role, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/guilds/%s/roles", s.baseURL, guildID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bot "+s.botToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord API error: %d - %s", resp.StatusCode, string(body))
	}

	var roles []Role
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, err
	}

	return roles, nil
}

// deleteWebhook deletes a Discord webhook by ID.
// It respects Discord's 429 rate-limit response and blocks for the Retry-After duration.
func (s *DeduplicationService) deleteWebhook(ctx context.Context, webhookID string) error {
	url := fmt.Sprintf("%s/webhooks/%s", s.baseURL, webhookID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request for webhook %s: %w", webhookID, err)
	}

	req.Header.Set("Authorization", "Bot "+s.botToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE request failed for webhook %s: %w", webhookID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 429 {
		retryAfter := discordDefaultRetryAfter
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, parseErr := strconv.ParseFloat(ra, 64); parseErr == nil && secs > 0 {
				retryAfter = time.Duration(secs * float64(time.Second))
			}
		}
		select {
		case <-time.After(retryAfter):
		case <-ctx.Done():
			return ctx.Err()
		}
		return fmt.Errorf("discord API rate limited (429) for webhook %s; waited %s — caller should retry", webhookID, retryAfter)
	}

	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord API error (status %d) deleting webhook %s: %s", resp.StatusCode, webhookID, string(body))
	}

	return nil
}

// deleteRole deletes a Discord guild role by ID.
// It respects Discord's 429 rate-limit response and blocks for the Retry-After duration.
func (s *DeduplicationService) deleteRole(ctx context.Context, guildID, roleID string) error {
	url := fmt.Sprintf("%s/guilds/%s/roles/%s", s.baseURL, guildID, roleID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request for role %s: %w", roleID, err)
	}

	req.Header.Set("Authorization", "Bot "+s.botToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE request failed for role %s: %w", roleID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 429 {
		retryAfter := discordDefaultRetryAfter
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, parseErr := strconv.ParseFloat(ra, 64); parseErr == nil && secs > 0 {
				retryAfter = time.Duration(secs * float64(time.Second))
			}
		}
		select {
		case <-time.After(retryAfter):
		case <-ctx.Done():
			return ctx.Err()
		}
		return fmt.Errorf("discord API rate limited (429) for role %s; waited %s — caller should retry", roleID, retryAfter)
	}

	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord API error (status %d) deleting role %s: %s", resp.StatusCode, roleID, string(body))
	}

	return nil
}

// deleteOrphanedResources finds and deletes Crossplane Channel resources whose
// status.atProvider.id matches the Discord channel that was just deleted.
// Returns the count of Crossplane resources successfully deleted.
func (s *DeduplicationService) deleteOrphanedResources(ctx context.Context, channelID string) int {
	if s.kubeClient == nil {
		return 0
	}

	deletedCount := 0

	// List all Crossplane Channel resources (cluster-scoped) and match by observed Discord channel ID.
	// We filter in-memory because Kubernetes field selectors are not supported on custom CRD status fields.
	channelList := &channelv1alpha1.ChannelList{}
	if err := s.kubeClient.List(ctx, channelList); err != nil {
		// Non-fatal: log suppressed here since we have no logger; caller tracks orphan count.
		return 0
	}

	for i := range channelList.Items {
		ch := &channelList.Items[i]
		if ch.Status.AtProvider.ID == channelID {
			if err := s.kubeClient.Delete(ctx, ch); err == nil {
				deletedCount++
			}
		}
	}

	return deletedCount
}

// deleteOrphanedWebhookResources finds and deletes Crossplane Webhook resources whose
// status.atProvider.id matches the Discord webhook that was just deleted. Mirrors
// deleteOrphanedResources for Channels. Returns the count deleted.
func (s *DeduplicationService) deleteOrphanedWebhookResources(ctx context.Context, webhookID string) int {
	if s.kubeClient == nil {
		return 0
	}

	deletedCount := 0

	webhookList := &webhookv1alpha1.WebhookList{}
	if err := s.kubeClient.List(ctx, webhookList); err != nil {
		return 0
	}

	for i := range webhookList.Items {
		wh := &webhookList.Items[i]
		if wh.Status.AtProvider.ID == webhookID {
			if err := s.kubeClient.Delete(ctx, wh); err == nil {
				deletedCount++
			}
		}
	}

	return deletedCount
}

// deleteOrphanedRoleResources finds and deletes Crossplane Role resources whose
// status.atProvider.id matches the Discord role that was just deleted. Mirrors
// deleteOrphanedResources for Channels. Returns the count deleted.
func (s *DeduplicationService) deleteOrphanedRoleResources(ctx context.Context, roleID string) int {
	if s.kubeClient == nil {
		return 0
	}

	deletedCount := 0

	roleList := &rolev1alpha1.RoleList{}
	if err := s.kubeClient.List(ctx, roleList); err != nil {
		return 0
	}

	for i := range roleList.Items {
		r := &roleList.Items[i]
		if r.Status.AtProvider.ID == roleID {
			if err := s.kubeClient.Delete(ctx, r); err == nil {
				deletedCount++
			}
		}
	}

	return deletedCount
}
