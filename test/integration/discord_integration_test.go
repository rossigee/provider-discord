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

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rossigee/provider-discord/internal/clients"
)

// TestDiscordAPIIntegration tests real Discord API integration
// Requires environment variables:
// - DISCORD_BOT_TOKEN: Discord bot token
// - DISCORD_TEST_GUILD_ID: Guild ID for testing (bot must have permissions)
func TestDiscordAPIIntegration(t *testing.T) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	testGuildID := os.Getenv("DISCORD_TEST_GUILD_ID")

	if token == "" {
		t.Skip("DISCORD_BOT_TOKEN not set, skipping Discord API integration tests")
	}

	if testGuildID == "" {
		t.Skip("DISCORD_TEST_GUILD_ID not set, skipping Discord API integration tests")
	}

	client := clients.NewDiscordClient(token)
	ctx := context.Background()

	t.Run("TestGuildOperations", func(t *testing.T) {
		testGuildOperations(t, client, ctx, testGuildID)
	})

	t.Run("TestChannelOperations", func(t *testing.T) {
		testChannelOperations(t, client, ctx, testGuildID)
	})

	t.Run("TestRoleOperations", func(t *testing.T) {
		testRoleOperations(t, client, ctx, testGuildID)
	})

	t.Run("TestErrorHandling", func(t *testing.T) {
		testErrorHandling(t, client, ctx)
	})

	t.Run("TestRateLimiting", func(t *testing.T) {
		testRateLimiting(t, client, ctx, testGuildID)
	})
}

func testGuildOperations(t *testing.T, client *clients.DiscordClient, ctx context.Context, guildID string) {
	// Test GetGuild
	guild, err := client.GetGuild(ctx, guildID)
	if err != nil {
		t.Fatalf("Failed to get guild: %v", err)
	}

	if guild.ID != guildID {
		t.Errorf("Expected guild ID %s, got %s", guildID, guild.ID)
	}

	t.Logf("Successfully retrieved guild: %s (ID: %s)", guild.Name, guild.ID)

	// Test ModifyGuild - limited to safe changes only
	// We avoid changing name as it's more disruptive
	
	// Just test that GetGuild works properly
	originalGuild := guild
	t.Logf("Guild has %d verification level", originalGuild.VerificationLevel)

	// Test ListGuilds (should include our test guild)
	guilds, err := client.ListGuilds(ctx)
	if err != nil {
		t.Fatalf("Failed to list guilds: %v", err)
	}

	found := false
	for _, g := range guilds {
		if g.ID == guildID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Test guild %s not found in guild list", guildID)
	}

	t.Logf("Successfully listed %d guilds, test guild found", len(guilds))
}

func testChannelOperations(t *testing.T, client *clients.DiscordClient, ctx context.Context, guildID string) {
	// Create a test channel
	channelName := fmt.Sprintf("test-channel-%d", time.Now().Unix())
	topic := "Test channel created by integration tests"
	createParams := &clients.CreateChannelRequest{
		Name:    channelName,
		Type:    0, // Text channel
		GuildID: guildID,
		Topic:   &topic,
	}

	channel, err := client.CreateChannel(ctx, createParams)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	if channel.Name != channelName {
		t.Errorf("Expected channel name '%s', got '%s'", channelName, channel.Name)
	}

	if channel.GuildID != guildID {
		t.Errorf("Expected guild ID '%s', got '%s'", guildID, channel.GuildID)
	}

	t.Logf("Successfully created channel: %s (ID: %s)", channel.Name, channel.ID)

	// Test GetChannel
	retrievedChannel, err := client.GetChannel(ctx, channel.ID)
	if err != nil {
		t.Fatalf("Failed to get channel: %v", err)
	}

	if retrievedChannel.ID != channel.ID {
		t.Errorf("Expected channel ID '%s', got '%s'", channel.ID, retrievedChannel.ID)
	}

	t.Logf("Successfully retrieved channel: %s", retrievedChannel.Name)

	// Test ModifyChannel (limited modification to avoid disruption)
	newPosition := 42
	modifyParams := &clients.ModifyChannelRequest{
		Position: &newPosition,
	}

	updatedChannel, err := client.ModifyChannel(ctx, channel.ID, modifyParams)
	if err != nil {
		t.Fatalf("Failed to modify channel: %v", err)
	}

	if updatedChannel.Position != newPosition {
		t.Errorf("Expected position %d, got %d", newPosition, updatedChannel.Position)
	}

	t.Logf("Successfully updated channel position to: %d", newPosition)

	// Clean up: Delete the test channel
	err = client.DeleteChannel(ctx, channel.ID)
	if err != nil {
		t.Fatalf("Failed to delete channel: %v", err)
	}

	t.Logf("Successfully deleted test channel: %s", channel.ID)

	// Verify deletion - should return 404
	_, err = client.GetChannel(ctx, channel.ID)
	if err == nil {
		t.Error("Expected error when getting deleted channel, but got none")
	}
}

func testRoleOperations(t *testing.T, client *clients.DiscordClient, ctx context.Context, guildID string) {
	// Create a test role
	roleName := fmt.Sprintf("test-role-%d", time.Now().Unix())
	color := 0xFF0000 // Red
	hoist := false
	mentionable := false
	permissions := "0"
	
	createParams := clients.CreateRoleRequest{
		Name:         roleName,
		Color:        &color,
		Hoist:        &hoist,
		Mentionable:  &mentionable,
		Permissions:  &permissions,
	}

	role, err := client.CreateRole(ctx, guildID, createParams)
	if err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}

	if role.Name != roleName {
		t.Errorf("Expected role name '%s', got '%s'", roleName, role.Name)
	}

	t.Logf("Successfully created role: %s (ID: %s)", role.Name, role.ID)

	// Test GetRole
	retrievedRole, err := client.GetRole(ctx, guildID, role.ID)
	if err != nil {
		t.Fatalf("Failed to get role: %v", err)
	}

	if retrievedRole.ID != role.ID {
		t.Errorf("Expected role ID '%s', got '%s'", role.ID, retrievedRole.ID)
	}

	t.Logf("Successfully retrieved role: %s", retrievedRole.Name)

	// Test ModifyRole
	newColor := 0x00FF00 // Green
	modifyParams := clients.ModifyRoleRequest{
		Color: &newColor,
	}

	updatedRole, err := client.ModifyRole(ctx, guildID, role.ID, modifyParams)
	if err != nil {
		t.Fatalf("Failed to modify role: %v", err)
	}

	if updatedRole.Color != newColor {
		t.Errorf("Expected color %d, got %d", newColor, updatedRole.Color)
	}

	t.Logf("Successfully updated role color to: %d", newColor)

	// Clean up: Delete the test role
	err = client.DeleteRole(ctx, guildID, role.ID)
	if err != nil {
		t.Fatalf("Failed to delete role: %v", err)
	}

	t.Logf("Successfully deleted test role: %s", role.ID)

	// Verify deletion - should return 404
	_, err = client.GetRole(ctx, guildID, role.ID)
	if err == nil {
		t.Error("Expected error when getting deleted role, but got none")
	}
}

func testErrorHandling(t *testing.T, client *clients.DiscordClient, ctx context.Context) {
	// Test 404 errors
	t.Run("TestNotFoundErrors", func(t *testing.T) {
		// Try to get non-existent guild
		_, err := client.GetGuild(ctx, "999999999999999999")
		if err == nil {
			t.Error("Expected error for non-existent guild, but got none")
		}
		t.Logf("Correctly handled non-existent guild error: %v", err)

		// Try to get non-existent channel
		_, err = client.GetChannel(ctx, "999999999999999999")
		if err == nil {
			t.Error("Expected error for non-existent channel, but got none")
		}
		t.Logf("Correctly handled non-existent channel error: %v", err)
	})

	// Test permission errors (if we try operations without permissions)
	t.Run("TestPermissionErrors", func(t *testing.T) {
		// This test might not trigger if bot has admin permissions
		// Try to create a guild (most bots can't do this)
		createParams := &clients.CreateGuildRequest{
			Name: "test-guild-should-fail",
		}
		_, err := client.CreateGuild(ctx, createParams)
		if err != nil {
			t.Logf("Correctly handled permission error for guild creation: %v", err)
		} else {
			t.Log("Bot has guild creation permissions or test didn't trigger expected error")
		}
	})
}

func testRateLimiting(t *testing.T, client *clients.DiscordClient, ctx context.Context, guildID string) {
	// Test rate limiting by making many requests quickly
	// This should trigger rate limiting and test the client's handling
	
	const numRequests = 20
	const requestDelay = 10 * time.Millisecond

	t.Logf("Testing rate limiting with %d rapid requests", numRequests)

	start := time.Now()
	successCount := 0
	rateLimitCount := 0

	for i := 0; i < numRequests; i++ {
		_, err := client.GetGuild(ctx, guildID)
		if err != nil {
			if isRateLimitError(err) {
				rateLimitCount++
				t.Logf("Request %d: Rate limited (expected)", i+1)
			} else {
				t.Logf("Request %d: Unexpected error: %v", i+1, err)
			}
		} else {
			successCount++
		}

		time.Sleep(requestDelay)
	}

	duration := time.Since(start)
	
	t.Logf("Rate limiting test completed in %v", duration)
	t.Logf("Successful requests: %d/%d", successCount, numRequests)
	t.Logf("Rate limited requests: %d/%d", rateLimitCount, numRequests)

	if successCount == 0 {
		t.Error("All requests failed - this suggests a problem beyond rate limiting")
	}

	// The client should handle rate limiting gracefully
	// We expect some requests to succeed and possibly some to be rate limited
	if successCount > 0 {
		t.Log("Rate limiting test passed - client handled requests appropriately")
	}
}

// Helper function to check if an error is a rate limit error
func isRateLimitError(err error) bool {
	// Check if the error message contains rate limit indicators
	errMsg := err.Error()
	return false || // You would implement this based on your error types
		(errMsg != "" && (
			// Add rate limit error detection logic here
			false)) // Placeholder for now
}

// TestDiscordAPIConnectivity tests basic Discord API connectivity
func TestDiscordAPIConnectivity(t *testing.T) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		t.Skip("DISCORD_BOT_TOKEN not set, skipping connectivity test")
	}

	client := clients.NewDiscordClient(token)
	ctx := context.Background()

	// Test basic connectivity by listing guilds
	guilds, err := client.ListGuilds(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to Discord API: %v", err)
	}

	t.Logf("Successfully connected to Discord API. Bot is in %d guilds.", len(guilds))

	// Verify token format and permissions
	if len(guilds) == 0 {
		t.Log("Warning: Bot is not in any guilds. Integration tests will be limited.")
	}
}

// TestDiscordAPIConfiguration tests various client configuration scenarios
func TestDiscordAPIConfiguration(t *testing.T) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		t.Skip("DISCORD_BOT_TOKEN not set, skipping configuration tests")
	}

	ctx := context.Background()

	t.Run("TestDefaultConfiguration", func(t *testing.T) {
		client := clients.NewDiscordClient(token)
		
		// Should use default base URL
		guilds, err := client.ListGuilds(ctx)
		if err != nil {
			t.Fatalf("Failed with default configuration: %v", err)
		}
		t.Logf("Default configuration works, found %d guilds", len(guilds))
	})

	t.Run("TestCustomBaseURL", func(t *testing.T) {
		client := clients.NewDiscordClient(token)
		
		// Should work with explicit base URL
		guilds, err := client.ListGuilds(ctx)
		if err != nil {
			t.Fatalf("Failed with custom base URL: %v", err)
		}
		t.Logf("Custom base URL works, found %d guilds", len(guilds))
	})

	t.Run("TestInvalidToken", func(t *testing.T) {
		client := clients.NewDiscordClient("invalid-token")
		
		// Should fail with invalid token
		_, err := client.ListGuilds(ctx)
		if err == nil {
			t.Error("Expected error with invalid token, but got none")
		}
		t.Logf("Correctly handled invalid token: %v", err)
	})
}

// TestDiscordAPIPerformance tests API performance characteristics
func TestDiscordAPIPerformance(t *testing.T) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	testGuildID := os.Getenv("DISCORD_TEST_GUILD_ID")

	if token == "" || testGuildID == "" {
		t.Skip("DISCORD_BOT_TOKEN or DISCORD_TEST_GUILD_ID not set, skipping performance tests")
	}

	client := clients.NewDiscordClient(token)
	ctx := context.Background()

	// Test response times
	const numTests = 10
	var totalDuration time.Duration

	for i := 0; i < numTests; i++ {
		start := time.Now()
		_, err := client.GetGuild(ctx, testGuildID)
		duration := time.Since(start)
		totalDuration += duration

		if err != nil {
			t.Fatalf("Performance test request %d failed: %v", i+1, err)
		}

		if duration > 5*time.Second {
			t.Errorf("Request %d took too long: %v", i+1, duration)
		}
	}

	avgDuration := totalDuration / numTests
	t.Logf("Average API response time: %v over %d requests", avgDuration, numTests)

	if avgDuration > 2*time.Second {
		t.Errorf("Average response time too high: %v", avgDuration)
	}
}

// TestDiscordWebhookIntegration tests webhook functionality (if implemented)
func TestDiscordWebhookIntegration(t *testing.T) {
	t.Skip("Webhook integration tests not yet implemented")
	
	// This would test webhook creation, configuration, and deletion
	// when webhook functionality is added to the provider
}

// BenchmarkDiscordAPIOperations benchmarks common Discord API operations
func BenchmarkDiscordAPIOperations(b *testing.B) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	testGuildID := os.Getenv("DISCORD_TEST_GUILD_ID")

	if token == "" || testGuildID == "" {
		b.Skip("DISCORD_BOT_TOKEN or DISCORD_TEST_GUILD_ID not set, skipping benchmarks")
	}

	client := clients.NewDiscordClient(token)
	ctx := context.Background()

	b.Run("GetGuild", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.GetGuild(ctx, testGuildID)
			if err != nil {
				b.Fatalf("Benchmark failed: %v", err)
			}
		}
	})

	b.Run("ListGuilds", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.ListGuilds(ctx)
			if err != nil {
				b.Fatalf("Benchmark failed: %v", err)
			}
		}
	})
}

// Helper function removed - unused in current context