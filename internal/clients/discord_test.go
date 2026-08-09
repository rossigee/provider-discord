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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestNewDiscordClient(t *testing.T) {
	token := "test-token"
	client := NewDiscordClient(token)

	if client.token != token {
		t.Errorf("Expected token %s, got %s", token, client.token)
	}

	if client.baseURL != DiscordAPIBaseURL {
		t.Errorf("Expected baseURL %s, got %s", DiscordAPIBaseURL, client.baseURL)
	}

	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
}

func TestGetGuild(t *testing.T) {
	mockGuild := Guild{
		ID:                          "123456789",
		Name:                        "Test Guild",
		OwnerID:                     "987654321",
		VerificationLevel:           1,
		DefaultMessageNotifications: 0,
		ExplicitContentFilter:       1,
		Features:                    []string{"COMMUNITY"},
		AFKTimeout:                  300,
		SystemChannelFlags:          0,
		ApproximateMemberCount:      func() *int { v := 100; return &v }(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/guilds/123456789" {
			t.Errorf("Expected path /guilds/123456789, got %s", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "Bot test-token" {
			t.Errorf("Expected Authorization header 'Bot test-token', got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockGuild); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	guild, err := client.GetGuild(context.Background(), "123456789")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if diff := cmp.Diff(&mockGuild, guild); diff != "" {
		t.Errorf("Guild mismatch (-want +got):\n%s", diff)
	}
}

func TestGetGuildNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(`{"message": "Unknown Guild", "code": 10004}`)); err != nil {
			t.Errorf("Failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	_, err := client.GetGuild(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent guild, got nil")
	}
}

func TestCreateGuild(t *testing.T) {
	mockRequest := CreateGuildRequest{
		Name:                        "New Test Guild",
		VerificationLevel:           func() *int { v := 1; return &v }(),
		DefaultMessageNotifications: func() *int { v := 0; return &v }(),
	}

	mockResponse := Guild{
		ID:                          "987654321",
		Name:                        "New Test Guild",
		OwnerID:                     "123456789",
		VerificationLevel:           1,
		DefaultMessageNotifications: 0,
		ExplicitContentFilter:       0,
		Features:                    []string{},
		AFKTimeout:                  300,
		SystemChannelFlags:          0,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/guilds" {
			t.Errorf("Expected path /guilds, got %s", r.URL.Path)
		}

		var receivedRequest CreateGuildRequest
		if err := json.NewDecoder(r.Body).Decode(&receivedRequest); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if diff := cmp.Diff(mockRequest, receivedRequest); diff != "" {
			t.Errorf("Request mismatch (-want +got):\n%s", diff)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	guild, err := client.CreateGuild(context.Background(), &mockRequest)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if diff := cmp.Diff(&mockResponse, guild); diff != "" {
		t.Errorf("Guild mismatch (-want +got):\n%s", diff)
	}
}

func TestModifyGuild(t *testing.T) {
	mockRequest := ModifyGuildRequest{
		Name:              func() *string { v := "Updated Guild Name"; return &v }(),
		VerificationLevel: func() *int { v := 2; return &v }(),
	}

	mockResponse := Guild{
		ID:                          "123456789",
		Name:                        "Updated Guild Name",
		OwnerID:                     "987654321",
		VerificationLevel:           2,
		DefaultMessageNotifications: 0,
		ExplicitContentFilter:       1,
		Features:                    []string{"COMMUNITY"},
		AFKTimeout:                  300,
		SystemChannelFlags:          0,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}

		if r.URL.Path != "/guilds/123456789" {
			t.Errorf("Expected path /guilds/123456789, got %s", r.URL.Path)
		}

		var receivedRequest ModifyGuildRequest
		if err := json.NewDecoder(r.Body).Decode(&receivedRequest); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if diff := cmp.Diff(mockRequest, receivedRequest); diff != "" {
			t.Errorf("Request mismatch (-want +got):\n%s", diff)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	guild, err := client.ModifyGuild(context.Background(), "123456789", &mockRequest)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if diff := cmp.Diff(&mockResponse, guild); diff != "" {
		t.Errorf("Guild mismatch (-want +got):\n%s", diff)
	}
}

func TestDeleteGuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}

		if r.URL.Path != "/guilds/123456789" {
			t.Errorf("Expected path /guilds/123456789, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	err := client.DeleteGuild(context.Background(), "123456789")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestListGuilds(t *testing.T) {
	mockGuilds := []Guild{
		{
			ID:      "123456789",
			Name:    "Guild 1",
			OwnerID: "987654321",
		},
		{
			ID:      "987654321",
			Name:    "Guild 2",
			OwnerID: "123456789",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/users/@me/guilds" {
			t.Errorf("Expected path /users/@me/guilds, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockGuilds); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	guilds, err := client.ListGuilds(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if diff := cmp.Diff(mockGuilds, guilds); diff != "" {
		t.Errorf("Guilds mismatch (-want +got):\n%s", diff)
	}
}

func TestGetChannel(t *testing.T) {
	mockChannel := Channel{
		ID:      "123456789",
		Name:    "test-channel",
		Type:    0,
		GuildID: "987654321",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/channels/123456789" {
			t.Errorf("Expected path /channels/123456789, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockChannel); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	channel, err := client.GetChannel(context.Background(), "123456789")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if diff := cmp.Diff(&mockChannel, channel); diff != "" {
		t.Errorf("Channel mismatch (-want +got):\n%s", diff)
	}
}

func TestCreateChannel(t *testing.T) {
	mockRequest := CreateChannelRequest{
		Name:    "new-channel",
		Type:    0,
		GuildID: "987654321",
	}

	mockResponse := Channel{
		ID:      "123456789",
		Name:    mockRequest.Name,
		Type:    mockRequest.Type,
		GuildID: mockRequest.GuildID,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/guilds/987654321/channels" {
			t.Errorf("Expected path /guilds/987654321/channels, got %s", r.URL.Path)
		}

		var receivedRequest CreateChannelRequest
		if err := json.NewDecoder(r.Body).Decode(&receivedRequest); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// GuildID is not sent in body, it's in URL
		expectedRequest := mockRequest
		expectedRequest.GuildID = ""

		if diff := cmp.Diff(expectedRequest, receivedRequest); diff != "" {
			t.Errorf("Request mismatch (-want +got):\n%s", diff)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	channel, err := client.CreateChannel(context.Background(), &mockRequest)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if diff := cmp.Diff(&mockResponse, channel); diff != "" {
		t.Errorf("Channel mismatch (-want +got):\n%s", diff)
	}
}

func TestModifyChannel(t *testing.T) {
	newName := "updated-channel"
	mockRequest := ModifyChannelRequest{
		Name: &newName,
	}

	mockResponse := Channel{
		ID:      "123456789",
		Name:    newName,
		Type:    0,
		GuildID: "987654321",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}

		if r.URL.Path != "/channels/123456789" {
			t.Errorf("Expected path /channels/123456789, got %s", r.URL.Path)
		}

		var receivedRequest ModifyChannelRequest
		if err := json.NewDecoder(r.Body).Decode(&receivedRequest); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if diff := cmp.Diff(mockRequest, receivedRequest); diff != "" {
			t.Errorf("Request mismatch (-want +got):\n%s", diff)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	channel, err := client.ModifyChannel(context.Background(), "123456789", &mockRequest)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if diff := cmp.Diff(&mockResponse, channel); diff != "" {
		t.Errorf("Channel mismatch (-want +got):\n%s", diff)
	}
}

func TestDeleteChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}

		if r.URL.Path != "/channels/123456789" {
			t.Errorf("Expected path /channels/123456789, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(Channel{ID: "123456789"}); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	err := client.DeleteChannel(context.Background(), "123456789")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestMakeRequestHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"message": "Internal Server Error", "code": 0}`)); err != nil {
			t.Errorf("Failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	resp, err := client.makeRequest(context.Background(), "GET", "/test", nil)
	if resp != nil {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}
	if err == nil {
		t.Error("Expected error for HTTP 500, got nil")
	}
}

func TestMakeRequestInvalidURL(t *testing.T) {
	client := NewDiscordClient("test-token")
	client.baseURL = "://invalid-url"

	resp, err := client.makeRequest(context.Background(), "GET", "/test", nil)
	if resp != nil {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestMakeRequestNetworkError(t *testing.T) {
	client := NewDiscordClient("test-token")
	client.baseURL = "http://localhost:99999" // Non-existent port

	resp, err := client.makeRequest(context.Background(), "GET", "/test", nil)
	if resp != nil {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}
	if err == nil {
		t.Error("Expected network error, got nil")
	}
}

func TestMakeRequestInvalidBodyJSON(t *testing.T) {
	client := NewDiscordClient("test-token")
	client.baseURL = "http://example.com"

	// Create an invalid body that can't be marshaled to JSON
	invalidBody := make(chan int)
	resp, err := client.makeRequest(context.Background(), "POST", "/test", invalidBody)
	if resp != nil {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}
	if err == nil {
		t.Error("Expected error for invalid JSON body, got nil")
	}
}

func TestCreateGuildError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"message": "Name already taken", "code": 50035}`)); err != nil {
			t.Errorf("Failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	req := &CreateGuildRequest{Name: "Test Guild"}
	_, err := client.CreateGuild(context.Background(), req)
	if err == nil {
		t.Error("Expected error for guild creation failure, got nil")
	}
}

func TestModifyGuildError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		if _, err := w.Write([]byte(`{"message": "Missing Permissions", "code": 50013}`)); err != nil {
			t.Errorf("Failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	req := &ModifyGuildRequest{Name: func() *string { v := "New Name"; return &v }()}
	_, err := client.ModifyGuild(context.Background(), "123456789", req)
	if err == nil {
		t.Error("Expected error for guild modification failure, got nil")
	}
}

func TestDeleteGuildError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		if _, err := w.Write([]byte(`{"message": "Missing Permissions", "code": 50013}`)); err != nil {
			t.Errorf("Failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	err := client.DeleteGuild(context.Background(), "123456789")
	if err == nil {
		t.Error("Expected error for guild deletion failure, got nil")
	}
}

func TestCreateChannelError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"message": "Invalid Form Body", "code": 50035}`)); err != nil {
			t.Errorf("Failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	req := &CreateChannelRequest{Name: "test-channel", GuildID: "123456789"}
	_, err := client.CreateChannel(context.Background(), req)
	if err == nil {
		t.Error("Expected error for channel creation failure, got nil")
	}
}

func TestModifyChannelError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(`{"message": "Unknown Channel", "code": 10003}`)); err != nil {
			t.Errorf("Failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	req := &ModifyChannelRequest{Name: func() *string { v := "new-name"; return &v }()}
	_, err := client.ModifyChannel(context.Background(), "123456789", req)
	if err == nil {
		t.Error("Expected error for channel modification failure, got nil")
	}
}

func TestDeleteChannelError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(`{"message": "Unknown Channel", "code": 10003}`)); err != nil {
			t.Errorf("Failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	err := client.DeleteChannel(context.Background(), "123456789")
	if err == nil {
		t.Error("Expected error for channel deletion failure, got nil")
	}
}

func TestListGuildsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`{"message": "401: Unauthorized", "code": 0}`)); err != nil {
			t.Errorf("Failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	_, err := client.ListGuilds(context.Background())
	if err == nil {
		t.Error("Expected error for unauthorized guild listing, got nil")
	}
}

func TestGetChannelError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(`{"message": "Unknown Channel", "code": 10003}`)); err != nil {
			t.Errorf("Failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	_, err := client.GetChannel(context.Background(), "123456789")
	if err == nil {
		t.Error("Expected error for unknown channel, got nil")
	}
}

func TestMakeRequest429WithRetryAfter(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: 429 with retry_after in JSON body
			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"retry_after": 0.05, "message": "Rate limited"}`)); err != nil {
				t.Errorf("Failed to write error response: %v", err)
			}
		} else {
			// Second call (after retry): success
			w.Header().Set("Content-Type", "application/json")
			mockGuild := Guild{
				ID:                          "123456789",
				Name:                        "Test Guild",
				OwnerID:                     "987654321",
				VerificationLevel:           1,
				DefaultMessageNotifications: 0,
				ExplicitContentFilter:       1,
				Features:                    []string{"COMMUNITY"},
				AFKTimeout:                  300,
				SystemChannelFlags:          0,
			}
			if err := json.NewEncoder(w).Encode(mockGuild); err != nil {
				t.Errorf("Failed to encode mock response: %v", err)
			}
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	guild, err := client.GetGuild(context.Background(), "123456789")
	if err != nil {
		t.Fatalf("Expected successful retry after 429, got error: %v", err)
	}

	if guild.ID != "123456789" {
		t.Errorf("Expected guild ID 123456789, got %s", guild.ID)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 calls (1 429 + 1 retry), got %d", callCount)
	}
}

func TestMakeRequest429ExhaustedRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return 429
		w.WriteHeader(http.StatusTooManyRequests)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"retry_after": 0.01, "message": "Rate limited"}`)); err != nil {
			t.Errorf("Failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	_, err := client.GetGuild(context.Background(), "123456789")
	if err == nil {
		t.Error("Expected error after exhausting retries, got nil")
	}

	if !strings.Contains(err.Error(), "Discord API error: 429") {
		t.Errorf("Expected 429 error, got: %v", err)
	}
}

func TestRateLimiterThrottles(t *testing.T) {
	start := time.Now()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		mockGuild := Guild{
			ID:                          fmt.Sprintf("guild%d", callCount),
			Name:                        "Test Guild",
			OwnerID:                     "987654321",
			VerificationLevel:           1,
			DefaultMessageNotifications: 0,
			ExplicitContentFilter:       1,
			Features:                    []string{},
			AFKTimeout:                  300,
			SystemChannelFlags:          0,
		}
		if err := json.NewEncoder(w).Encode(mockGuild); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	// Make 11 sequential requests (should hit rate limit with burst of 5, require waits)
	for i := 0; i < 11; i++ {
		_, err := client.GetGuild(context.Background(), "123456789")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
	}

	elapsed := time.Since(start)

	// With rate limiter at 5 req/s and burst 5, first 5 should be instant,
	// next 6 should take roughly 6/5 = ~1.2 seconds of wait time
	// We check that it took at least 1 second (accounting for timing variance and test overhead)
	if elapsed < 800*time.Millisecond {
		t.Errorf("Rate limiter did not throttle; expected >= 800ms, got %v", elapsed)
	}
}

func TestMakeRequest429WithRetryAfterHeader(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: 429 with Retry-After header (seconds)
			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "0.05")
			if _, err := w.Write([]byte(`{"message": "Rate limited"}`)); err != nil {
				t.Errorf("Failed to write error response: %v", err)
			}
		} else {
			// Second call (after retry): success
			w.Header().Set("Content-Type", "application/json")
			mockGuild := Guild{
				ID:                          "123456789",
				Name:                        "Test Guild",
				OwnerID:                     "987654321",
				VerificationLevel:           1,
				DefaultMessageNotifications: 0,
				ExplicitContentFilter:       1,
				Features:                    []string{"COMMUNITY"},
				AFKTimeout:                  300,
				SystemChannelFlags:          0,
			}
			if err := json.NewEncoder(w).Encode(mockGuild); err != nil {
				t.Errorf("Failed to encode mock response: %v", err)
			}
		}
	}))
	defer server.Close()

	client := NewDiscordClient("test-token")
	client.baseURL = server.URL

	guild, err := client.GetGuild(context.Background(), "123456789")
	if err != nil {
		t.Fatalf("Expected successful retry after 429, got error: %v", err)
	}

	if guild.ID != "123456789" {
		t.Errorf("Expected guild ID 123456789, got %s", guild.ID)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 calls (1 429 + 1 retry), got %d", callCount)
	}
}

func TestExtractRetryAfterFromHeader(t *testing.T) {
	tests := []struct {
		name    string
		errMsg  string
		want    time.Duration
		minWait time.Duration
		maxWait time.Duration
	}{
		{
			name:    "Retry-After header with seconds",
			errMsg:  `Discord API error: 429 - {"message": "Rate limited"} | Retry-After: 0.5`,
			want:    500 * time.Millisecond,
			minWait: 400 * time.Millisecond,
			maxWait: 600 * time.Millisecond,
		},
		{
			name:    "Retry-After header with fractional seconds",
			errMsg:  `Discord API error: 429 - {"message": "Rate limited"} | Retry-After: 2.5`,
			want:    2500 * time.Millisecond,
			minWait: 2400 * time.Millisecond,
			maxWait: 2600 * time.Millisecond,
		},
		{
			name:    "Retry-After header with integer",
			errMsg:  `Discord API error: 429 - {"message": "Rate limited"} | Retry-After: 1`,
			want:    1000 * time.Millisecond,
			minWait: 900 * time.Millisecond,
			maxWait: 1100 * time.Millisecond,
		},
		{
			name:    "No Retry-After header, uses JSON body",
			errMsg:  `Discord API error: 429 - {"retry_after": 0.1, "message": "Rate limited"}`,
			want:    100 * time.Millisecond,
			minWait: 50 * time.Millisecond,
			maxWait: 200 * time.Millisecond,
		},
		{
			name:    "No Retry-After anywhere, uses default",
			errMsg:  `Discord API error: 429 - {"message": "Rate limited"}`,
			want:    1000 * time.Millisecond,
			minWait: 900 * time.Millisecond,
			maxWait: 1100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewDiscordClient("test-token")
			err := fmt.Errorf("%s", tt.errMsg)
			got := client.extractRetryAfter(err)

			if got < tt.minWait || got > tt.maxWait {
				t.Errorf("extractRetryAfter() = %v, want %v (range %v-%v)",
					got, tt.want, tt.minWait, tt.maxWait)
			}
		})
	}
}
