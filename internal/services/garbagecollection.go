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
	"fmt"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	channelv1alpha1 "github.com/rossigee/provider-discord/apis/channel/v1alpha1"
	discordv1alpha1 "github.com/rossigee/provider-discord/apis/v1alpha1"
)

// GarbageCollectionService handles autonomous cleanup of duplicate channels.
type GarbageCollectionService struct {
	botToken   string
	baseURL    string
	httpClient *http.Client
	k8sClient  client.Client
}

// GarbageCollectionResult contains the results of a garbage collection run.
type GarbageCollectionResult struct {
	DuplicatesPrevented      int
	DuplicatesDeleted        int
	OrphanedResourcesDeleted int
	UnmanagedChannelsDeleted int
	HasErrors                bool
	Errors                   []string
}

// NewGarbageCollectionService creates a new garbage collection service.
func NewGarbageCollectionService(botToken string, baseURL string, k8sClient client.Client) *GarbageCollectionService {
	return &GarbageCollectionService{
		botToken:   botToken,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		k8sClient:  k8sClient,
	}
}

// RunGarbageCollection performs autonomous cleanup based on the GC spec.
func (s *GarbageCollectionService) RunGarbageCollection(ctx context.Context, spec *discordv1alpha1.GarbageCollectionSpec) (*GarbageCollectionResult, error) {
	if spec == nil {
		return &GarbageCollectionResult{}, nil
	}

	result := &GarbageCollectionResult{
		Errors: make([]string, 0),
	}

	// Get targeted guilds (empty = all guilds)
	targetGuilds := spec.TargetGuilds
	if len(targetGuilds) == 0 {
		guilds, err := s.getGuilds(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to list guilds: %v", err))
			result.HasErrors = true
			return result, err
		}
		for _, g := range guilds {
			targetGuilds = append(targetGuilds, g.ID)
		}
	}

	deleteUnmanaged := false // default: safe, opt-in only
	if spec.DeleteUnmanagedChannels != nil {
		deleteUnmanaged = *spec.DeleteUnmanagedChannels
	}

	deleteOrphaned := true // default
	if spec.DeleteOrphanedResources != nil {
		deleteOrphaned = *spec.DeleteOrphanedResources
	}

	// Process each guild
	for _, guildID := range targetGuilds {
		duplicatesDeleted, orphanedCleaned, err := s.processGuildDuplicates(ctx, guildID, deleteOrphaned)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("guild %s duplicates: %v", guildID, err))
			result.HasErrors = true
			continue
		}
		result.DuplicatesDeleted += duplicatesDeleted
		result.OrphanedResourcesDeleted += orphanedCleaned

		if deleteUnmanaged {
			unmanagedDeleted, err := s.deleteUnmanagedChannels(ctx, guildID)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("guild %s unmanaged: %v", guildID, err))
				result.HasErrors = true
				continue
			}
			result.UnmanagedChannelsDeleted += unmanagedDeleted
		}
	}

	return result, nil
}

// processGuildDuplicates finds and deletes duplicate channels in a specific guild.
func (s *GarbageCollectionService) processGuildDuplicates(ctx context.Context, guildID string, deleteOrphaned bool) (int, int, error) {
	channels, err := s.getChannels(ctx, guildID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list channels: %w", err)
	}

	// Group channels by name to identify duplicates
	channelsByName := make(map[string][]Channel)
	for _, ch := range channels {
		channelsByName[ch.Name] = append(channelsByName[ch.Name], ch)
	}

	var duplicatesDeleted int
	var orphanedCleaned int

	// Process each duplicate group
	for name, group := range channelsByName {
		if len(group) <= 1 {
			continue // Not a duplicate
		}

		// Find the channel to keep (lowest position = oldest/highest priority)
		keepIndex := 0
		minPosition := group[0].Position
		for i, ch := range group {
			if ch.Position < minPosition {
				minPosition = ch.Position
				keepIndex = i
			}
		}

		// Delete all duplicates except the one to keep
		for i, ch := range group {
			if i == keepIndex {
				continue
			}

			if err := s.deleteChannel(ctx, ch.ID); err != nil {
				return duplicatesDeleted, orphanedCleaned, fmt.Errorf("failed to delete duplicate %q (%s): %w", name, ch.ID, err)
			}
			duplicatesDeleted++

			// Clean up orphaned Crossplane resource if requested
			if deleteOrphaned {
				if deleted := s.deleteOrphanedChannelResource(ctx, ch.ID); deleted {
					orphanedCleaned++
				}
			}
		}
	}

	return duplicatesDeleted, orphanedCleaned, nil
}

// deleteOrphanedChannelResource deletes the Crossplane Channel CR whose external-name matches the deleted Discord channel ID.
func (s *GarbageCollectionService) deleteOrphanedChannelResource(ctx context.Context, discordChannelID string) bool {
	if s.k8sClient == nil {
		return false
	}
	list := &channelv1alpha1.ChannelList{}
	if err := s.k8sClient.List(ctx, list); err != nil {
		return false
	}
	for i := range list.Items {
		ch := &list.Items[i]
		if ch.GetAnnotations()["crossplane.io/external-name"] == discordChannelID {
			if err := s.k8sClient.Delete(ctx, ch); err == nil {
				return true
			}
		}
	}
	return false
}

// deleteUnmanagedChannels deletes Discord channels in a guild that have no corresponding Crossplane Channel CR.
// Safety guard: only runs if at least one Channel CR exists for the guild, to avoid wiping
// guilds where Crossplane management has not been established.
func (s *GarbageCollectionService) deleteUnmanagedChannels(ctx context.Context, guildID string) (int, error) {
	if s.k8sClient == nil {
		return 0, nil
	}

	// List all Crossplane Channel CRs
	list := &channelv1alpha1.ChannelList{}
	if err := s.k8sClient.List(ctx, list); err != nil {
		return 0, fmt.Errorf("failed to list Channel resources: %w", err)
	}

	// Build set of managed Discord channel IDs for this guild
	managedIDs := make(map[string]struct{})
	for _, ch := range list.Items {
		if ch.Spec.ForProvider.GuildID == guildID {
			if externalName := ch.GetAnnotations()["crossplane.io/external-name"]; externalName != "" {
				managedIDs[externalName] = struct{}{}
			}
		}
	}

	// Safety guard: don't delete anything if no channels are managed in this guild
	if len(managedIDs) == 0 {
		return 0, nil
	}

	// Get all Discord channels in the guild
	channels, err := s.getChannels(ctx, guildID)
	if err != nil {
		return 0, fmt.Errorf("failed to list Discord channels: %w", err)
	}

	var deleted int
	for _, ch := range channels {
		if _, managed := managedIDs[ch.ID]; managed {
			continue
		}
		// Category channels (type 4) are never auto-deleted to avoid losing structure
		if ch.Type == 4 {
			continue
		}
		if err := s.deleteChannel(ctx, ch.ID); err != nil {
			return deleted, fmt.Errorf("failed to delete unmanaged channel %q (%s): %w", ch.Name, ch.ID, err)
		}
		deleted++
	}

	return deleted, nil
}

// getGuilds retrieves all guilds the bot is a member of.
func (s *GarbageCollectionService) getGuilds(ctx context.Context) ([]Guild, error) {
	// Reuse deduplication service logic
	dedupService := NewDeduplicationService(s.httpClient, s.baseURL, s.botToken, s.k8sClient)
	return dedupService.getGuilds(ctx)
}

// getChannels retrieves all channels in a guild.
func (s *GarbageCollectionService) getChannels(ctx context.Context, guildID string) ([]Channel, error) {
	// Reuse deduplication service logic
	dedupService := NewDeduplicationService(s.httpClient, s.baseURL, s.botToken, s.k8sClient)
	return dedupService.getChannels(ctx, guildID)
}

// deleteChannel deletes a Discord channel by ID.
func (s *GarbageCollectionService) deleteChannel(ctx context.Context, channelID string) error {
	// Reuse deduplication service logic
	dedupService := NewDeduplicationService(s.httpClient, s.baseURL, s.botToken, s.k8sClient)
	return dedupService.deleteChannel(ctx, channelID)
}

// ValidateChannelNameAvailable checks if a channel name is available in a guild.
// Returns an error if a channel with the same name already exists.
func (s *GarbageCollectionService) ValidateChannelNameAvailable(ctx context.Context, guildID, channelName string) error {
	channels, err := s.getChannels(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to check existing channels: %w", err)
	}

	for _, ch := range channels {
		if ch.Name == channelName {
			return fmt.Errorf("channel with name %q already exists in guild (ID: %s)", channelName, guildID)
		}
	}

	return nil
}
