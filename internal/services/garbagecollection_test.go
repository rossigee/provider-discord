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
	"github.com/rossigee/provider-discord/apis/channel/v1alpha1"
	"github.com/rossigee/provider-discord/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"net/http/httptest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sync"
	"testing"
)

// newGCTestScheme builds a runtime.Scheme with the channel v1alpha1 types registered.
func newGCTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := channelv1alpha1.SchemeBuilder.AddToScheme(s); err != nil {
		t.Fatalf("failed to add channel scheme: %v", err)
	}
	return s
}

// boolPtr is a helper to get a pointer to a bool literal.
func boolPtr(b bool) *bool { return &b }

// discordChannelFixture builds a minimal Discord channel JSON object.
func discordChannelFixture(id, name string, chType, position int) Channel {
	return Channel{ID: id, Name: name, Type: chType, Position: position}
}

// makeCrossplaneChannel builds a Crossplane Channel CR with the given external-name and guildID.
func makeCrossplaneChannel(name, externalName, guildID string) *channelv1alpha1.Channel {
	return &channelv1alpha1.Channel{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"crossplane.io/external-name": externalName,
			},
		},
		Spec: channelv1alpha1.ChannelSpec{
			ForProvider: channelv1alpha1.ChannelParameters{
				Name:    name,
				GuildID: guildID,
				Type:    0,
			},
		},
	}
}

// --- deleteUnmanagedChannels ---

// TestDeleteUnmanagedChannels_SafetyGuard_NoManagedChannels verifies that when NO
// Crossplane Channel CRs exist for a guild, the GC does NOT delete anything — even
// if the guild has many channels in Discord.
func TestDeleteUnmanagedChannels_SafetyGuard_NoManagedChannels(t *testing.T) {
	var deleted []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/guilds/guild1/channels":
			channels := []Channel{
				discordChannelFixture("ch1", "general", 0, 0),
				discordChannelFixture("ch2", "random", 0, 1),
				discordChannelFixture("ch3", "announcements", 0, 2),
			}
			_ = json.NewEncoder(w).Encode(channels)
		default:
			if r.Method == http.MethodDelete {
				mu.Lock()
				deleted = append(deleted, r.URL.Path)
				mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			}
		}
	}))
	defer server.Close()

	// No Channel CRs in the cluster
	k8s := fake.NewClientBuilder().WithScheme(newGCTestScheme(t)).Build()
	svc := &GarbageCollectionService{
		botToken:   "fake-token",
		baseURL:    server.URL,
		httpClient: server.Client(),
		k8sClient:  k8s,
	}

	n, err := svc.deleteUnmanagedChannels(context.Background(), "guild1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 deletions (safety guard), got %d", n)
	}
	if len(deleted) != 0 {
		t.Errorf("expected no Discord DELETE calls, got: %v", deleted)
	}
}

// TestDeleteUnmanagedChannels_DeletesUnmanaged verifies that channels without a
// Crossplane CR are deleted when at least one managed channel exists in the guild.
func TestDeleteUnmanagedChannels_DeletesUnmanaged(t *testing.T) {
	var deletedIDs []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/guilds/guild1/channels":
			channels := []Channel{
				discordChannelFixture("ch-managed", "general", 0, 0),
				discordChannelFixture("ch-unmanaged1", "random", 0, 1),
				discordChannelFixture("ch-unmanaged2", "off-topic", 0, 2),
			}
			_ = json.NewEncoder(w).Encode(channels)
		default:
			if r.Method == http.MethodDelete {
				mu.Lock()
				deletedIDs = append(deletedIDs, r.URL.Path)
				mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			}
		}
	}))
	defer server.Close()

	managed := makeCrossplaneChannel("general", "ch-managed", "guild1")
	k8s := fake.NewClientBuilder().WithScheme(newGCTestScheme(t)).WithObjects(managed).Build()
	svc := &GarbageCollectionService{
		botToken:   "fake-token",
		baseURL:    server.URL,
		httpClient: server.Client(),
		k8sClient:  k8s,
	}

	n, err := svc.deleteUnmanagedChannels(context.Background(), "guild1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 unmanaged deletions, got %d", n)
	}
	if len(deletedIDs) != 2 {
		t.Errorf("expected 2 Discord DELETE calls, got %d: %v", len(deletedIDs), deletedIDs)
	}
}

// TestDeleteUnmanagedChannels_SkipsCategories verifies that category channels (type=4)
// are never deleted by the GC even if unmanaged, to preserve server structure.
func TestDeleteUnmanagedChannels_SkipsCategories(t *testing.T) {
	var deletedIDs []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/guilds/guild1/channels":
			channels := []Channel{
				discordChannelFixture("ch-managed", "general", 0, 0),
				discordChannelFixture("cat-unmanaged", "Old Category", 4, 0), // category
				discordChannelFixture("ch-unmanaged", "orphan-text", 0, 1),
			}
			_ = json.NewEncoder(w).Encode(channels)
		default:
			if r.Method == http.MethodDelete {
				mu.Lock()
				deletedIDs = append(deletedIDs, r.URL.Path)
				mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			}
		}
	}))
	defer server.Close()

	managed := makeCrossplaneChannel("general", "ch-managed", "guild1")
	k8s := fake.NewClientBuilder().WithScheme(newGCTestScheme(t)).WithObjects(managed).Build()
	svc := &GarbageCollectionService{
		botToken:   "fake-token",
		baseURL:    server.URL,
		httpClient: server.Client(),
		k8sClient:  k8s,
	}

	n, err := svc.deleteUnmanagedChannels(context.Background(), "guild1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 deletion (category skipped), got %d", n)
	}
	for _, id := range deletedIDs {
		if id == "/channels/cat-unmanaged" {
			t.Errorf("category channel was incorrectly deleted")
		}
	}
}

// TestDeleteUnmanagedChannels_SkipsOtherGuilds verifies that Channel CRs for a
// different guild are not counted as managed for the target guild.
func TestDeleteUnmanagedChannels_SkipsOtherGuilds(t *testing.T) {
	var deletedIDs []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/guilds/guild1/channels":
			channels := []Channel{
				discordChannelFixture("ch1", "general", 0, 0),
			}
			_ = json.NewEncoder(w).Encode(channels)
		default:
			if r.Method == http.MethodDelete {
				mu.Lock()
				deletedIDs = append(deletedIDs, r.URL.Path)
				mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			}
		}
	}))
	defer server.Close()

	// CR exists but for a DIFFERENT guild
	managedOtherGuild := makeCrossplaneChannel("general", "ch1", "guild2")
	k8s := fake.NewClientBuilder().WithScheme(newGCTestScheme(t)).WithObjects(managedOtherGuild).Build()
	svc := &GarbageCollectionService{
		botToken:   "fake-token",
		baseURL:    server.URL,
		httpClient: server.Client(),
		k8sClient:  k8s,
	}

	// Safety guard should fire — no CRs for guild1
	n, err := svc.deleteUnmanagedChannels(context.Background(), "guild1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 deletions (safety guard for wrong guild), got %d", n)
	}
	if len(deletedIDs) != 0 {
		t.Errorf("expected no Discord DELETE calls, got: %v", deletedIDs)
	}
}

// TestDeleteUnmanagedChannels_NilK8sClient verifies graceful handling when no k8s client.
func TestDeleteUnmanagedChannels_NilK8sClient(t *testing.T) {
	svc := &GarbageCollectionService{
		botToken:   "fake-token",
		baseURL:    "http://localhost",
		httpClient: http.DefaultClient,
		k8sClient:  nil,
	}

	n, err := svc.deleteUnmanagedChannels(context.Background(), "guild1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 with nil k8sClient, got %d", n)
	}
}

// --- RunGarbageCollection integration ---

// TestRunGarbageCollection_DeleteUnmanaged_DisabledByDefault verifies that
// deleteUnmanagedChannels does NOT run unless explicitly enabled — even when
// there are unmanaged channels present.
func TestRunGarbageCollection_DeleteUnmanaged_DisabledByDefault(t *testing.T) {
	var deletedIDs []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/@me/guilds":
			_ = json.NewEncoder(w).Encode([]Guild{{ID: "guild1", Name: "Test Guild"}})
		case "/guilds/guild1/channels":
			_ = json.NewEncoder(w).Encode([]Channel{
				discordChannelFixture("ch-managed", "general", 0, 0),
				discordChannelFixture("ch-unmanaged", "orphan", 0, 1),
			})
		default:
			if r.Method == http.MethodDelete {
				mu.Lock()
				deletedIDs = append(deletedIDs, r.URL.Path)
				mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			}
		}
	}))
	defer server.Close()

	managed := makeCrossplaneChannel("general", "ch-managed", "guild1")
	k8s := fake.NewClientBuilder().WithScheme(newGCTestScheme(t)).WithObjects(managed).Build()
	svc := &GarbageCollectionService{
		botToken:   "fake-token",
		baseURL:    server.URL,
		httpClient: server.Client(),
		k8sClient:  k8s,
	}

	// Spec with deleteUnmanagedChannels NOT set (nil = disabled)
	spec := &discordv1alpha1.GarbageCollectionSpec{
		Enabled: true,
	}

	result, err := svc.RunGarbageCollection(context.Background(), spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UnmanagedChannelsDeleted != 0 {
		t.Errorf("expected 0 unmanaged deletions (disabled), got %d", result.UnmanagedChannelsDeleted)
	}
	if len(deletedIDs) != 0 {
		t.Errorf("expected no Discord DELETE calls when disabled, got: %v", deletedIDs)
	}
}

// TestRunGarbageCollection_DeleteUnmanaged_Enabled verifies end-to-end that
// enabling DeleteUnmanagedChannels causes unmanaged Discord channels to be deleted.
func TestRunGarbageCollection_DeleteUnmanaged_Enabled(t *testing.T) {
	var deletedIDs []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/@me/guilds":
			_ = json.NewEncoder(w).Encode([]Guild{{ID: "guild1", Name: "Test Guild"}})
		case "/guilds/guild1/channels":
			_ = json.NewEncoder(w).Encode([]Channel{
				discordChannelFixture("ch-managed", "general", 0, 0),
				discordChannelFixture("ch-unmanaged", "orphan", 0, 1),
			})
		default:
			if r.Method == http.MethodDelete {
				mu.Lock()
				deletedIDs = append(deletedIDs, r.URL.Path)
				mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			}
		}
	}))
	defer server.Close()

	managed := makeCrossplaneChannel("general", "ch-managed", "guild1")
	k8s := fake.NewClientBuilder().WithScheme(newGCTestScheme(t)).WithObjects(managed).Build()
	svc := &GarbageCollectionService{
		botToken:   "fake-token",
		baseURL:    server.URL,
		httpClient: server.Client(),
		k8sClient:  k8s,
	}

	spec := &discordv1alpha1.GarbageCollectionSpec{
		Enabled:                 true,
		DeleteUnmanagedChannels: boolPtr(true),
	}

	result, err := svc.RunGarbageCollection(context.Background(), spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UnmanagedChannelsDeleted != 1 {
		t.Errorf("expected 1 unmanaged deletion, got %d", result.UnmanagedChannelsDeleted)
	}
}
