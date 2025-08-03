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

package performance

import (
	"context"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/rossigee/provider-discord/internal/clients"
)

// PerformanceConfig defines configuration for performance tests
type PerformanceConfig struct {
	Concurrency       int
	RequestsPerWorker int
	TestDuration      time.Duration
	RampUpDuration    time.Duration
}

// PerformanceResult holds results from performance tests
type PerformanceResult struct {
	TotalRequests     int
	SuccessfulRequests int
	FailedRequests    int
	TotalDuration     time.Duration
	AverageLatency    time.Duration
	MinLatency        time.Duration
	MaxLatency        time.Duration
	RequestsPerSecond float64
	Errors            []error
}

// TestDiscordAPIPerformance runs comprehensive performance tests against Discord API
func TestDiscordAPIPerformance(t *testing.T) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	testGuildID := os.Getenv("DISCORD_TEST_GUILD_ID")

	if token == "" {
		t.Skip("DISCORD_BOT_TOKEN not set, skipping performance tests")
	}

	if testGuildID == "" {
		t.Skip("DISCORD_TEST_GUILD_ID not set, skipping performance tests")
	}

	client := clients.NewDiscordClient(token)

	// Test different load scenarios
	scenarios := []struct {
		name   string
		config PerformanceConfig
	}{
		{
			name: "Light Load",
			config: PerformanceConfig{
				Concurrency:       5,
				RequestsPerWorker: 20,
				TestDuration:      30 * time.Second,
				RampUpDuration:    5 * time.Second,
			},
		},
		{
			name: "Medium Load",
			config: PerformanceConfig{
				Concurrency:       10,
				RequestsPerWorker: 50,
				TestDuration:      60 * time.Second,
				RampUpDuration:    10 * time.Second,
			},
		},
		{
			name: "Heavy Load",
			config: PerformanceConfig{
				Concurrency:       20,
				RequestsPerWorker: 25,
				TestDuration:      120 * time.Second,
				RampUpDuration:    20 * time.Second,
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			result := runPerformanceTest(t, client, testGuildID, scenario.config)
			validatePerformanceResult(t, result, scenario.config)
			logPerformanceResults(t, scenario.name, result)
		})
	}
}

// TestConcurrentResourceOperations tests concurrent operations on different resource types
func TestConcurrentResourceOperations(t *testing.T) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	testGuildID := os.Getenv("DISCORD_TEST_GUILD_ID")

	if token == "" || testGuildID == "" {
		t.Skip("Required environment variables not set, skipping concurrent tests")
	}

	client := clients.NewDiscordClient(token)
	ctx := context.Background()

	// Test concurrent operations
	concurrency := 10
	operationsPerWorker := 5

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*operationsPerWorker)
	durations := make(chan time.Duration, concurrency*operationsPerWorker)

	startTime := time.Now()

	// Start concurrent workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runConcurrentOperations(ctx, client, testGuildID, workerID, operationsPerWorker, errors, durations)
		}(i)
	}

	wg.Wait()
	close(errors)
	close(durations)

	totalDuration := time.Since(startTime)

	// Collect results
	var errorList []error
	var durationList []time.Duration

	for err := range errors {
		if err != nil {
			errorList = append(errorList, err)
		}
	}

	for duration := range durations {
		durationList = append(durationList, duration)
	}

	// Analyze results
	totalOps := concurrency * operationsPerWorker
	successOps := totalOps - len(errorList)
	avgDuration := calculateAverageDuration(durationList)

	t.Logf("Concurrent Operations Results:")
	t.Logf("  Total Operations: %d", totalOps)
	t.Logf("  Successful Operations: %d", successOps)
	t.Logf("  Failed Operations: %d", len(errorList))
	t.Logf("  Success Rate: %.2f%%", float64(successOps)/float64(totalOps)*100)
	t.Logf("  Total Duration: %v", totalDuration)
	t.Logf("  Average Operation Duration: %v", avgDuration)
	t.Logf("  Operations per Second: %.2f", float64(totalOps)/totalDuration.Seconds())

	// Validate results
	if len(errorList) > totalOps/10 { // Allow up to 10% failures
		t.Errorf("Too many failures: %d/%d", len(errorList), totalOps)
	}

	if avgDuration > 2*time.Second {
		t.Errorf("Average duration too high: %v", avgDuration)
	}
}

// TestRateLimitHandling tests how the client handles Discord rate limits
func TestRateLimitHandling(t *testing.T) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	testGuildID := os.Getenv("DISCORD_TEST_GUILD_ID")

	if token == "" || testGuildID == "" {
		t.Skip("Required environment variables not set, skipping rate limit tests")
	}

	client := clients.NewDiscordClient(token)
	ctx := context.Background()

	// Rapid fire requests to trigger rate limiting
	requestCount := 100
	interval := 10 * time.Millisecond

	var successCount, rateLimitCount, errorCount int
	durations := make([]time.Duration, 0, requestCount)

	t.Logf("Sending %d rapid requests (interval: %v)", requestCount, interval)

	for i := 0; i < requestCount; i++ {
		start := time.Now()
		_, err := client.GetGuild(ctx, testGuildID)
		duration := time.Since(start)
		durations = append(durations, duration)

		if err != nil {
			if isRateLimitError(err) {
				rateLimitCount++
			} else {
				errorCount++
			}
		} else {
			successCount++
		}

		time.Sleep(interval)
	}

	avgDuration := calculateAverageDuration(durations)
	maxDuration := findMaxDuration(durations)

	t.Logf("Rate Limit Test Results:")
	t.Logf("  Successful Requests: %d", successCount)
	t.Logf("  Rate Limited Requests: %d", rateLimitCount)
	t.Logf("  Other Errors: %d", errorCount)
	t.Logf("  Average Duration: %v", avgDuration)
	t.Logf("  Max Duration: %v", maxDuration)

	// Validate that rate limiting is handled gracefully
	if errorCount > requestCount/10 { // Allow up to 10% non-rate-limit errors
		t.Errorf("Too many non-rate-limit errors: %d", errorCount)
	}

	// Check that rate limited requests don't cause excessive delays
	if maxDuration > 30*time.Second {
		t.Errorf("Rate limit backoff too aggressive: %v", maxDuration)
	}
}

// TestMemoryUsageUnderLoad monitors memory usage during load testing
func TestMemoryUsageUnderLoad(t *testing.T) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	testGuildID := os.Getenv("DISCORD_TEST_GUILD_ID")

	if token == "" || testGuildID == "" {
		t.Skip("Required environment variables not set, skipping memory tests")
	}

	client := clients.NewDiscordClient(token)

	// Baseline memory measurement
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Run sustained load
	config := PerformanceConfig{
		Concurrency:       15,
		RequestsPerWorker: 100,
		TestDuration:      180 * time.Second,
		RampUpDuration:    30 * time.Second,
	}

	result := runPerformanceTest(t, client, testGuildID, config)

	// Post-test memory measurement
	runtime.GC()
	runtime.ReadMemStats(&m2)

	memoryIncrease := m2.Alloc - m1.Alloc
	heapIncrease := m2.HeapAlloc - m1.HeapAlloc

	t.Logf("Memory Usage Results:")
	t.Logf("  Requests Processed: %d", result.TotalRequests)
	t.Logf("  Memory Increase: %d bytes (%.2f MB)", memoryIncrease, float64(memoryIncrease)/(1024*1024))
	t.Logf("  Heap Increase: %d bytes (%.2f MB)", heapIncrease, float64(heapIncrease)/(1024*1024))
	t.Logf("  Memory per Request: %.2f bytes", float64(memoryIncrease)/float64(result.TotalRequests))

	// Validate memory usage is reasonable
	memoryPerRequest := float64(memoryIncrease) / float64(result.TotalRequests)
	if memoryPerRequest > 1024 { // More than 1KB per request indicates potential leak
		t.Errorf("Memory usage per request too high: %.2f bytes", memoryPerRequest)
	}

	maxMemoryIncrease := 100 * 1024 * 1024 // 100MB
	if memoryIncrease > uint64(maxMemoryIncrease) {
		t.Errorf("Total memory increase too high: %d bytes", memoryIncrease)
	}
}

// BenchmarkDiscordOperations provides benchmark tests for different operations
func BenchmarkDiscordOperations(b *testing.B) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	testGuildID := os.Getenv("DISCORD_TEST_GUILD_ID")

	if token == "" || testGuildID == "" {
		b.Skip("Required environment variables not set, skipping benchmarks")
	}

	client := clients.NewDiscordClient(token)
	ctx := context.Background()

	b.Run("GetGuild", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.GetGuild(ctx, testGuildID)
			if err != nil {
				b.Fatalf("GetGuild failed: %v", err)
			}
		}
	})

	b.Run("ListGuilds", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.ListGuilds(ctx)
			if err != nil {
				b.Fatalf("ListGuilds failed: %v", err)
			}
		}
	})
}

// Helper functions

func runPerformanceTest(t *testing.T, client *clients.DiscordClient, guildID string, config PerformanceConfig) PerformanceResult {
	ctx := context.Background()
	result := PerformanceResult{}

	var wg sync.WaitGroup
	errors := make(chan error, config.Concurrency*config.RequestsPerWorker)
	durations := make(chan time.Duration, config.Concurrency*config.RequestsPerWorker)

	startTime := time.Now()

	// Gradual ramp-up of workers
	workerInterval := config.RampUpDuration / time.Duration(config.Concurrency)

	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			// Stagger worker start times
			time.Sleep(time.Duration(workerID) * workerInterval)
			
			runWorker(ctx, client, guildID, config.RequestsPerWorker, errors, durations)
		}(i)
	}

	wg.Wait()
	close(errors)
	close(durations)

	endTime := time.Now()
	result.TotalDuration = endTime.Sub(startTime)

	// Collect results
	var errorList []error
	var durationList []time.Duration

	for err := range errors {
		result.TotalRequests++
		if err != nil {
			result.FailedRequests++
			errorList = append(errorList, err)
		} else {
			result.SuccessfulRequests++
		}
	}

	for duration := range durations {
		durationList = append(durationList, duration)
	}

	// Calculate statistics
	if len(durationList) > 0 {
		result.AverageLatency = calculateAverageDuration(durationList)
		result.MinLatency = findMinDuration(durationList)
		result.MaxLatency = findMaxDuration(durationList)
	}

	result.RequestsPerSecond = float64(result.TotalRequests) / result.TotalDuration.Seconds()
	result.Errors = errorList

	return result
}

func runWorker(ctx context.Context, client *clients.DiscordClient, guildID string, requests int, errors chan<- error, durations chan<- time.Duration) {
	for i := 0; i < requests; i++ {
		start := time.Now()
		_, err := client.GetGuild(ctx, guildID)
		duration := time.Since(start)

		errors <- err
		durations <- duration

		// Small delay to avoid overwhelming the API
		time.Sleep(50 * time.Millisecond)
	}
}

func runConcurrentOperations(ctx context.Context, client *clients.DiscordClient, guildID string, workerID, operations int, errors chan<- error, durations chan<- time.Duration) {
	for i := 0; i < operations; i++ {
		start := time.Now()
		
		// Vary operations to test different endpoints
		var err error
		switch i % 3 {
		case 0:
			_, err = client.GetGuild(ctx, guildID)
		case 1:
			_, err = client.ListGuilds(ctx)
		case 2:
			// Test channel listing if implemented
			_, err = client.GetGuild(ctx, guildID) // Fallback to guild get
		}
		
		duration := time.Since(start)
		errors <- err
		durations <- duration

		time.Sleep(100 * time.Millisecond)
	}
}

func validatePerformanceResult(t *testing.T, result PerformanceResult, config PerformanceConfig) {
	successRate := float64(result.SuccessfulRequests) / float64(result.TotalRequests) * 100

	// Performance thresholds
	minSuccessRate := 95.0
	maxAverageLatency := 3 * time.Second
	minRequestsPerSecond := 1.0

	if successRate < minSuccessRate {
		t.Errorf("Success rate too low: %.2f%% (expected >= %.2f%%)", successRate, minSuccessRate)
	}

	if result.AverageLatency > maxAverageLatency {
		t.Errorf("Average latency too high: %v (expected <= %v)", result.AverageLatency, maxAverageLatency)
	}

	if result.RequestsPerSecond < minRequestsPerSecond {
		t.Errorf("Requests per second too low: %.2f (expected >= %.2f)", result.RequestsPerSecond, minRequestsPerSecond)
	}

	// Log warnings for concerning metrics
	if successRate < 99.0 {
		t.Logf("WARNING: Success rate below 99%%: %.2f%%", successRate)
	}

	if result.AverageLatency > time.Second {
		t.Logf("WARNING: Average latency above 1s: %v", result.AverageLatency)
	}
}

func logPerformanceResults(t *testing.T, scenarioName string, result PerformanceResult) {
	t.Logf("%s Results:", scenarioName)
	t.Logf("  Total Requests: %d", result.TotalRequests)
	t.Logf("  Successful: %d", result.SuccessfulRequests)
	t.Logf("  Failed: %d", result.FailedRequests)
	t.Logf("  Success Rate: %.2f%%", float64(result.SuccessfulRequests)/float64(result.TotalRequests)*100)
	t.Logf("  Total Duration: %v", result.TotalDuration)
	t.Logf("  Average Latency: %v", result.AverageLatency)
	t.Logf("  Min Latency: %v", result.MinLatency)
	t.Logf("  Max Latency: %v", result.MaxLatency)
	t.Logf("  Requests per Second: %.2f", result.RequestsPerSecond)
	
	if len(result.Errors) > 0 {
		t.Logf("  Error Sample (first 5):")
		for i, err := range result.Errors {
			if i >= 5 {
				break
			}
			t.Logf("    - %v", err)
		}
	}
}

func calculateAverageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func findMinDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

func findMaxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}

func isRateLimitError(err error) bool {
	// Implement rate limit error detection based on your error types
	errMsg := err.Error()
	return contains(errMsg, "rate limit") || contains(errMsg, "429") || contains(errMsg, "too many requests")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr ||
		      func() bool {
		      	for i := 0; i <= len(s)-len(substr); i++ {
		      		if s[i:i+len(substr)] == substr {
		      			return true
		      		}
		      	}
		      	return false
		      }())))
}