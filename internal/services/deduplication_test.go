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
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAnalyzeAndDeduplicate_NoDuplicates tests the service when no duplicates are found.
func TestAnalyzeAndDeduplicate_NoDuplicates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/@me/guilds":
			guilds := []Guild{
				{ID: "guild1", Name: "Test Guild"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(guilds)
		case "/guilds/guild1/channels":
			channels := []Channel{
				{ID: "ch1", Name: "general", Type: 0, GuildID: "guild1", Position: 0},
				{ID: "ch2", Name: "announcements", Type: 0, GuildID: "guild1", Position: 1},
				{ID: "ch3", Name: "random", Type: 0, GuildID: "guild1", Position: 2},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(channels)
		}
	}))
	defer server.Close()

	svc := NewDeduplicationService(server.Client(), server.URL, "fake-token", nil)

	result, err := svc.AnalyzeAndDeduplicate(context.Background(), "report", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.TotalGuildsAnalyzed != 1 {
		t.Errorf("expected 1 guild analyzed, got %d", result.Summary.TotalGuildsAnalyzed)
	}

	if result.Summary.TotalDuplicateChannelsFound != 0 {
		t.Errorf("expected 0 duplicates, got %d", result.Summary.TotalDuplicateChannelsFound)
	}

	if result.Summary.DuplicateGroupsFound != 0 {
		t.Errorf("expected 0 duplicate groups, got %d", result.Summary.DuplicateGroupsFound)
	}
}

// TestAnalyzeAndDeduplicate_WithDuplicates tests the service when duplicates are found.
func TestAnalyzeAndDeduplicate_WithDuplicates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/@me/guilds":
			guilds := []Guild{
				{ID: "guild1", Name: "Test Guild"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(guilds)
		case "/guilds/guild1/channels":
			channels := []Channel{
				{ID: "ch1", Name: "general", Type: 0, GuildID: "guild1", Position: 0},
				{ID: "ch2", Name: "general", Type: 0, GuildID: "guild1", Position: 1}, // duplicate
				{ID: "ch3", Name: "general", Type: 0, GuildID: "guild1", Position: 2}, // duplicate
				{ID: "ch4", Name: "announcements", Type: 0, GuildID: "guild1", Position: 3},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(channels)
		}
	}))
	defer server.Close()

	svc := NewDeduplicationService(server.Client(), server.URL, "fake-token", nil)

	result, err := svc.AnalyzeAndDeduplicate(context.Background(), "report", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.TotalGuildsAnalyzed != 1 {
		t.Errorf("expected 1 guild analyzed, got %d", result.Summary.TotalGuildsAnalyzed)
	}

	if result.Summary.TotalDuplicateChannelsFound != 2 {
		t.Errorf("expected 2 duplicates, got %d", result.Summary.TotalDuplicateChannelsFound)
	}

	if result.Summary.DuplicateGroupsFound != 1 {
		t.Errorf("expected 1 duplicate group, got %d", result.Summary.DuplicateGroupsFound)
	}

	if result.Summary.ChannelsDeleted != 0 {
		t.Errorf("expected 0 channels deleted in report mode, got %d", result.Summary.ChannelsDeleted)
	}
}

// TestAnalyzeAndDeduplicate_ActionMode_DeletesDuplicates tests deletion in action mode.
func TestAnalyzeAndDeduplicate_ActionMode_DeletesDuplicates(t *testing.T) {
	deleteCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/@me/guilds":
			guilds := []Guild{
				{ID: "guild1", Name: "Test Guild"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(guilds)
		case "/guilds/guild1/channels":
			channels := []Channel{
				{ID: "ch1", Name: "general", Type: 0, GuildID: "guild1", Position: 0},
				{ID: "ch2", Name: "general", Type: 0, GuildID: "guild1", Position: 1}, // will be deleted
				{ID: "ch3", Name: "announcements", Type: 0, GuildID: "guild1", Position: 2},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(channels)
		default:
			if r.Method == "DELETE" {
				deleteCount++
				w.WriteHeader(204)
			}
		}
	}))
	defer server.Close()

	svc := NewDeduplicationService(server.Client(), server.URL, "fake-token", nil)

	result, err := svc.AnalyzeAndDeduplicate(context.Background(), "action", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.ChannelsDeleted != 1 {
		t.Errorf("expected 1 channel deleted, got %d", result.Summary.ChannelsDeleted)
	}

	if deleteCount != 1 {
		t.Errorf("expected 1 DELETE request, got %d", deleteCount)
	}
}

// TestAnalyzeAndDeduplicate_MultipleGuilds tests deduplication across multiple guilds.
func TestAnalyzeAndDeduplicate_MultipleGuilds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/@me/guilds":
			guilds := []Guild{
				{ID: "guild1", Name: "Guild 1"},
				{ID: "guild2", Name: "Guild 2"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(guilds)
		case "/guilds/guild1/channels":
			channels := []Channel{
				{ID: "ch1", Name: "general", Type: 0, GuildID: "guild1", Position: 0},
				{ID: "ch2", Name: "general", Type: 0, GuildID: "guild1", Position: 1}, // duplicate
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(channels)
		case "/guilds/guild2/channels":
			channels := []Channel{
				{ID: "ch3", Name: "announcements", Type: 0, GuildID: "guild2", Position: 0},
				{ID: "ch4", Name: "announcements", Type: 0, GuildID: "guild2", Position: 1}, // duplicate
				{ID: "ch5", Name: "announcements", Type: 0, GuildID: "guild2", Position: 2}, // duplicate
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(channels)
		}
	}))
	defer server.Close()

	svc := NewDeduplicationService(server.Client(), server.URL, "fake-token", nil)

	result, err := svc.AnalyzeAndDeduplicate(context.Background(), "report", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.TotalGuildsAnalyzed != 2 {
		t.Errorf("expected 2 guilds analyzed, got %d", result.Summary.TotalGuildsAnalyzed)
	}

	if result.Summary.TotalDuplicateChannelsFound != 3 {
		t.Errorf("expected 3 total duplicates (1 from guild1 + 2 from guild2), got %d", result.Summary.TotalDuplicateChannelsFound)
	}

	if result.Summary.DuplicateGroupsFound != 2 {
		t.Errorf("expected 2 duplicate groups, got %d", result.Summary.DuplicateGroupsFound)
	}

	// Check guild1 results
	if guild1, ok := result.Guilds["guild1"]; !ok {
		t.Error("expected guild1 in results")
	} else {
		if len(guild1.DuplicateGroups) != 1 {
			t.Errorf("expected 1 duplicate group in guild1, got %d", len(guild1.DuplicateGroups))
		}
	}

	// Check guild2 results
	if guild2, ok := result.Guilds["guild2"]; !ok {
		t.Error("expected guild2 in results")
	} else {
		if len(guild2.DuplicateGroups) != 1 {
			t.Errorf("expected 1 duplicate group in guild2, got %d", len(guild2.DuplicateGroups))
		}
	}
}

// TestAnalyzeAndDeduplicate_TargetGuilds tests filtering by target guilds.
func TestAnalyzeAndDeduplicate_TargetGuilds(t *testing.T) {
	guildsCalled := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/@me/guilds":
			guilds := []Guild{
				{ID: "guild1", Name: "Guild 1"},
				{ID: "guild2", Name: "Guild 2"},
				{ID: "guild3", Name: "Guild 3"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(guilds)
		case "/guilds/guild1/channels":
			guildsCalled["guild1"] = true
			channels := []Channel{
				{ID: "ch1", Name: "general", Type: 0, GuildID: "guild1", Position: 0},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(channels)
		case "/guilds/guild2/channels":
			guildsCalled["guild2"] = true
			channels := []Channel{
				{ID: "ch2", Name: "general", Type: 0, GuildID: "guild2", Position: 0},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(channels)
		case "/guilds/guild3/channels":
			guildsCalled["guild3"] = true
			channels := []Channel{
				{ID: "ch3", Name: "general", Type: 0, GuildID: "guild3", Position: 0},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(channels)
		}
	}))
	defer server.Close()

	svc := NewDeduplicationService(server.Client(), server.URL, "fake-token", nil)

	// Only analyze guild1 and guild3
	result, err := svc.AnalyzeAndDeduplicate(context.Background(), "report", []string{"guild1", "guild3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.TotalGuildsAnalyzed != 2 {
		t.Errorf("expected 2 guilds analyzed, got %d", result.Summary.TotalGuildsAnalyzed)
	}

	if !guildsCalled["guild1"] {
		t.Error("expected guild1 to be queried")
	}

	if guildsCalled["guild2"] {
		t.Error("expected guild2 to NOT be queried")
	}

	if !guildsCalled["guild3"] {
		t.Error("expected guild3 to be queried")
	}
}

// TestAnalyzeAndDeduplicate_KeepsOldestChannel tests that the oldest (lowest position) channel is kept.
func TestAnalyzeAndDeduplicate_KeepsOldestChannel(t *testing.T) {
	deleteCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/@me/guilds":
			guilds := []Guild{
				{ID: "guild1", Name: "Test Guild"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(guilds)
		case "/guilds/guild1/channels":
			// Create channels with positions: 5, 0, 3
			// Position 0 should be kept (oldest)
			channels := []Channel{
				{ID: "ch1", Name: "general", Type: 0, GuildID: "guild1", Position: 5},
				{ID: "ch2", Name: "general", Type: 0, GuildID: "guild1", Position: 0}, // keeper (lowest position)
				{ID: "ch3", Name: "general", Type: 0, GuildID: "guild1", Position: 3},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(channels)
		default:
			if r.Method == "DELETE" {
				deleteCount++
				w.WriteHeader(204) // Success response for channel deletion
			}
		}
	}))
	defer server.Close()

	svc := NewDeduplicationService(server.Client(), server.URL, "fake-token", nil)

	result, err := svc.AnalyzeAndDeduplicate(context.Background(), "action", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the result shows the correct kept channel
	guild := result.Guilds["guild1"]
	if len(guild.DuplicateGroups) != 1 {
		t.Fatalf("expected 1 duplicate group, got %d", len(guild.DuplicateGroups))
	}

	dupGroup := guild.DuplicateGroups[0]
	if dupGroup.Channels[dupGroup.KeepIndex].ID != "ch2" {
		t.Errorf("expected to keep channel ch2 (position 0), but kept %s (position %d)",
			dupGroup.Channels[dupGroup.KeepIndex].ID,
			dupGroup.Channels[dupGroup.KeepIndex].Position)
	}

	if guild.ChannelsDeleted != 2 {
		t.Errorf("expected 2 channels deleted, got %d", guild.ChannelsDeleted)
	}

	if deleteCount != 2 {
		t.Errorf("expected 2 DELETE requests, got %d", deleteCount)
	}
}

// TestAnalyzeAndDeduplicate_APIError tests handling of Discord API errors.
func TestAnalyzeAndDeduplicate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/@me/guilds" {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"error": "Internal Server Error"}`))
		}
	}))
	defer server.Close()

	svc := NewDeduplicationService(server.Client(), server.URL, "fake-token", nil)

	result, err := svc.AnalyzeAndDeduplicate(context.Background(), "report", []string{})
	if err == nil {
		t.Error("expected error when Discord API fails")
	}

	if !result.HasError {
		t.Error("expected result to indicate error")
	}
}

// TestAnalyzeAndDeduplicate_DeleteError tests handling of channel deletion errors.
func TestAnalyzeAndDeduplicate_DeleteError(t *testing.T) {
	deleteCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/@me/guilds":
			guilds := []Guild{
				{ID: "guild1", Name: "Test Guild"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(guilds)
		case "/guilds/guild1/channels":
			channels := []Channel{
				{ID: "ch1", Name: "general", Type: 0, GuildID: "guild1", Position: 0},
				{ID: "ch2", Name: "general", Type: 0, GuildID: "guild1", Position: 1}, // will fail to delete
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(channels)
		default:
			if r.Method == "DELETE" {
				deleteCount++
				// Fail the delete with 403 Forbidden
				w.WriteHeader(403)
				_, _ = w.Write([]byte(`{"error": "Forbidden"}`))
			}
		}
	}))
	defer server.Close()

	svc := NewDeduplicationService(server.Client(), server.URL, "fake-token", nil)

	result, err := svc.AnalyzeAndDeduplicate(context.Background(), "action", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	guild := result.Guilds["guild1"]
	if guild.ChannelsDeleted != 0 {
		t.Errorf("expected 0 channels deleted due to error, got %d", guild.ChannelsDeleted)
	}

	if len(guild.Errors) == 0 {
		t.Error("expected errors to be recorded")
	}
}

// TestEmptyGuild tests handling of guilds with no channels.
func TestEmptyGuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/@me/guilds":
			guilds := []Guild{
				{ID: "guild1", Name: "Empty Guild"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(guilds)
		case "/guilds/guild1/channels":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]Channel{})
		}
	}))
	defer server.Close()

	svc := NewDeduplicationService(server.Client(), server.URL, "fake-token", nil)

	result, err := svc.AnalyzeAndDeduplicate(context.Background(), "report", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.TotalGuildsAnalyzed != 1 {
		t.Errorf("expected 1 guild analyzed, got %d", result.Summary.TotalGuildsAnalyzed)
	}

	if result.Summary.TotalChannelsAnalyzed != 0 {
		t.Errorf("expected 0 channels analyzed, got %d", result.Summary.TotalChannelsAnalyzed)
	}

	if result.Summary.DuplicateGroupsFound != 0 {
		t.Errorf("expected 0 duplicate groups, got %d", result.Summary.DuplicateGroupsFound)
	}
}

// TestNoGuilds tests handling when bot is in no guilds.
func TestNoGuilds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/@me/guilds" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]Guild{})
		}
	}))
	defer server.Close()

	svc := NewDeduplicationService(server.Client(), server.URL, "fake-token", nil)

	result, err := svc.AnalyzeAndDeduplicate(context.Background(), "report", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.TotalGuildsAnalyzed != 0 {
		t.Errorf("expected 0 guilds analyzed, got %d", result.Summary.TotalGuildsAnalyzed)
	}
}
