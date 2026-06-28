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

	// Process each guild for duplicates
	for _, guildID := range targetGuilds {
		deleteOrphaned := true // default
		if spec.DeleteOrphanedResources != nil {
			deleteOrphaned = *spec.DeleteOrphanedResources
		}

		duplicatesDeleted, orphanedCleaned, err := s.processGuildDuplicates(ctx, guildID, deleteOrphaned)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("guild %s: %v", guildID, err))
			result.HasErrors = true
			continue
		}
		result.DuplicatesDeleted += duplicatesDeleted
		result.OrphanedResourcesDeleted += orphanedCleaned
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

// deleteOrphanedChannelResource deletes the Crossplane Channel resource for a deleted Discord channel.
func (s *GarbageCollectionService) deleteOrphanedChannelResource(ctx context.Context, discordChannelID string) bool {
	// TODO: Implement resource lookup and deletion
	// This requires querying Kubernetes for Channel resources with external-name = discordChannelID
	// and deleting them.
	return false
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
