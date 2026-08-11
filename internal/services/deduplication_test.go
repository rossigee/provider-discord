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
	"strings"
	"testing"
)

// respondEmptyForAncillaryEndpoints handles the /channels/*/messages, /guilds/*/webhooks,
// and /guilds/*/roles GET requests that every analyzeGuild call now makes (message-history
// sampling, webhook dedup, role dedup), so per-test mock servers below - which only care
// about channel behavior - don't need to redefine them. Returns true if it handled the
// request. Tests that specifically want non-empty message history call
// json.NewEncoder(w).Encode(...) themselves for that path before falling back to this.
func channelIDsAt(group DuplicateGroup, indices []int) []string {
	ids := make([]string, len(indices))
	for i, idx := range indices {
		ids[i] = group.Channels[idx].ID
	}
	return ids
}

func respondEmptyForAncillaryEndpoints(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != "GET" {
		return false
	}
	if strings.HasSuffix(r.URL.Path, "/messages") || strings.HasSuffix(r.URL.Path, "/webhooks") || strings.HasSuffix(r.URL.Path, "/roles") {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
		return true
	}
	return false
}

// TestAnalyzeAndDeduplicate_NoDuplicates tests the service when no duplicates are found.
func TestAnalyzeAndDeduplicate_NoDuplicates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
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
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
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
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
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
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
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
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
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

// TestAnalyzeAndDeduplicate_KeepsChannelWithMessageHistory tests that keep-selection is
// message-history-based, not ID- or position-based: the channel with real message history
// must be kept even though it has neither the lowest position nor the lowest (earliest)
// snowflake ID - both of which point at a different, empty channel. This is the core
// safety property of the deduplication fix: never discard a channel with history.
func TestAnalyzeAndDeduplicate_KeepsChannelWithMessageHistory(t *testing.T) {
	deleteCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ch3 has the highest ID/position of the three but is the one with real
		// message history - it must be the one kept.
		if r.URL.Path == "/channels/ch3/messages" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]string{{"id": "msg1"}})
			return
		}
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
		switch r.URL.Path {
		case "/users/@me/guilds":
			guilds := []Guild{
				{ID: "guild1", Name: "Test Guild"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(guilds)
		case "/guilds/guild1/channels":
			// ch1 has the lowest ID and lowest position - under the old (removed)
			// position/ID-based logic it would incorrectly be kept, but it's empty.
			channels := []Channel{
				{ID: "ch1", Name: "general", Type: 0, GuildID: "guild1", Position: 0},
				{ID: "ch2", Name: "general", Type: 0, GuildID: "guild1", Position: 1},
				{ID: "ch3", Name: "general", Type: 0, GuildID: "guild1", Position: 3}, // keeper: has message history
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

	guild := result.Guilds["guild1"]
	if len(guild.DuplicateGroups) != 1 {
		t.Fatalf("expected 1 duplicate group, got %d", len(guild.DuplicateGroups))
	}

	dupGroup := guild.DuplicateGroups[0]
	kept := channelIDsAt(dupGroup, dupGroup.KeepIndices)
	if len(kept) != 1 || kept[0] != "ch3" {
		t.Errorf("expected to keep only channel ch3 (has message history), got kept=%v", kept)
	}
	if dupGroup.NeedsManualReview {
		t.Error("expected NeedsManualReview=false (only one channel has history)")
	}

	if guild.ChannelsDeleted != 2 {
		t.Errorf("expected 2 channels deleted, got %d", guild.ChannelsDeleted)
	}

	if deleteCount != 2 {
		t.Errorf("expected 2 DELETE requests, got %d", deleteCount)
	}
}

// TestAnalyzeAndDeduplicate_AllEmptyFallsBackToOldestID tests that when every channel in
// a duplicate group is empty, keep-selection falls back to the lowest (earliest-created)
// snowflake ID, since there's no history-based signal to use and nothing is lost either way.
func TestAnalyzeAndDeduplicate_AllEmptyFallsBackToOldestID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
		switch r.URL.Path {
		case "/users/@me/guilds":
			guilds := []Guild{{ID: "guild1", Name: "Test Guild"}}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(guilds)
		case "/guilds/guild1/channels":
			channels := []Channel{
				{ID: "100", Name: "general", Type: 0, GuildID: "guild1"},
				{ID: "300", Name: "general", Type: 0, GuildID: "guild1"},
				{ID: "200", Name: "general", Type: 0, GuildID: "guild1"},
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

	guild := result.Guilds["guild1"]
	dupGroup := guild.DuplicateGroups[0]
	kept := channelIDsAt(dupGroup, dupGroup.KeepIndices)
	if len(kept) != 1 || kept[0] != "100" {
		t.Errorf("expected to keep channel 100 (lowest numeric ID), got kept=%v", kept)
	}
}

// TestAnalyzeAndDeduplicate_ManualReviewWhenMultipleHaveHistory tests that a duplicate
// group with more than one channel showing real message history is flagged for manual
// review rather than auto-picking one to keep - only the genuinely empty channels (if any)
// are auto-deleted.
func TestAnalyzeAndDeduplicate_ManualReviewWhenMultipleHaveHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/channels/ch1/messages" || r.URL.Path == "/channels/ch2/messages" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]string{{"id": "msg1"}})
			return
		}
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
		switch r.URL.Path {
		case "/users/@me/guilds":
			guilds := []Guild{{ID: "guild1", Name: "Test Guild"}}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(guilds)
		case "/guilds/guild1/channels":
			channels := []Channel{
				{ID: "ch1", Name: "general", Type: 0, GuildID: "guild1"}, // has history
				{ID: "ch2", Name: "general", Type: 0, GuildID: "guild1"}, // has history
				{ID: "ch3", Name: "general", Type: 0, GuildID: "guild1"}, // empty
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

	guild := result.Guilds["guild1"]
	dupGroup := guild.DuplicateGroups[0]
	if !dupGroup.NeedsManualReview {
		t.Error("expected NeedsManualReview=true (two channels have message history)")
	}
	kept := channelIDsAt(dupGroup, dupGroup.KeepIndices)
	deleted := channelIDsAt(dupGroup, dupGroup.DeleteIndices)
	if len(kept) != 2 {
		t.Errorf("expected 2 channels kept (both with history), got %v", kept)
	}
	if len(deleted) != 1 || deleted[0] != "ch3" {
		t.Errorf("expected only the empty channel ch3 marked for deletion, got %v", deleted)
	}
}

// TestAnalyzeAndDeduplicate_APIError tests handling of Discord API errors.
func TestAnalyzeAndDeduplicate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
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
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
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
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
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
		if respondEmptyForAncillaryEndpoints(w, r) {
			return
		}
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
