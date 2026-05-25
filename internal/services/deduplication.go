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

	"sigs.k8s.io/controller-runtime/pkg/client"

	channelv1alpha1 "github.com/rossigee/provider-discord/apis/channel/v1alpha1"
	deduplicationv1alpha1 "github.com/rossigee/provider-discord/apis/deduplication/v1alpha1"
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

// DuplicateGroup represents a group of duplicate channels with the same name.
type DuplicateGroup struct {
	Name      string
	Channels  []Channel
	KeepIndex int // Index of the channel to keep (oldest by position)
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
		result.Summary.OrphanedResourcesDeleted += guildResult.OrphanedResourcesDeleted

		// Count total duplicates (each group has len-1 duplicates)
		for _, group := range guildResult.DuplicateGroups {
			result.Summary.TotalDuplicateChannelsFound += len(group.Channels) - 1
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

	// Find and process duplicate groups
	for name, group := range nameGroups {
		if len(group) > 1 {
			// Find the channel to keep (lowest position = oldest/highest priority)
			keepIndex := 0
			minPosition := group[0].Position

			for i, channel := range group {
				if channel.Position < minPosition {
					minPosition = channel.Position
					keepIndex = i
				}
			}

			dupGroup := DuplicateGroup{
				Name:      name,
				Channels:  group,
				KeepIndex: keepIndex,
			}
			result.DuplicateGroups = append(result.DuplicateGroups, dupGroup)

			// If in action mode, delete the duplicate channels
			if mode == "action" {
				deletesMade := 0
				for i, channel := range group {
					if i == keepIndex {
						continue
					}
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
						result.Errors = append(result.Errors, fmt.Sprintf("failed to delete channel %s: %v", channel.ID, err))
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
			}
		}
	}

	return result
}

// deleteChannel deletes a Discord channel by ID.
// It respects Discord's 429 rate-limit response and blocks for the Retry-After duration.
func (s *DeduplicationService) deleteChannel(ctx context.Context, channelID string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/channels/%s", s.baseURL, channelID), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bot "+s.botToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
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
		return fmt.Errorf("discord API rate limited (429); waited %s — caller should retry", retryAfter)
	}

	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord API error: %d - %s", resp.StatusCode, string(body))
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
