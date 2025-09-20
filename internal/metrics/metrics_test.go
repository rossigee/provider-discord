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

package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetricsRecorder_RecordAPIOperation(t *testing.T) {
	recorder := NewMetricsRecorder()

	// Clear metrics before test
	discordAPIOperations.Reset()
	discordAPIOperationDuration.Reset()

	// Record an operation
	duration := 100 * time.Millisecond
	recorder.RecordAPIOperation(ResourceGuild, OpCreate, StatusSuccess, duration)

	// Verify counter was incremented
	counter, err := discordAPIOperations.GetMetricWithLabelValues(ResourceGuild, OpCreate, StatusSuccess)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), testutil.ToFloat64(counter))
}

func TestMetricsRecorder_RecordRateLimit(t *testing.T) {
	recorder := NewMetricsRecorder()

	// Clear metrics before test
	discordRateLimits.Reset()
	discordRateLimitRemaining.Reset()
	discordRateLimitResetTime.Reset()

	// Record rate limit
	resetTime := time.Now().Add(1 * time.Hour)
	recorder.RecordRateLimit(ResourceGuild, "/guilds", 10, resetTime)

	// Verify metrics were set
	rateLimitCounter, err := discordRateLimits.GetMetricWithLabelValues(ResourceGuild, "/guilds")
	assert.NoError(t, err)
	assert.Equal(t, float64(1), testutil.ToFloat64(rateLimitCounter))

	remainingGauge, err := discordRateLimitRemaining.GetMetricWithLabelValues(ResourceGuild, "/guilds")
	assert.NoError(t, err)
	assert.Equal(t, float64(10), testutil.ToFloat64(remainingGauge))

	resetTimeGauge, err := discordRateLimitResetTime.GetMetricWithLabelValues(ResourceGuild, "/guilds")
	assert.NoError(t, err)
	assert.Equal(t, float64(resetTime.Unix()), testutil.ToFloat64(resetTimeGauge))
}

func TestMetricsRecorder_UpdateRateLimitStatus(t *testing.T) {
	recorder := NewMetricsRecorder()

	// Clear metrics before test
	discordRateLimitRemaining.Reset()
	discordRateLimitResetTime.Reset()

	// Update rate limit status
	resetTime := time.Now().Add(30 * time.Minute)
	recorder.UpdateRateLimitStatus(ResourceChannel, "/channels", 25, resetTime)

	// Verify metrics were updated
	remainingGauge, err := discordRateLimitRemaining.GetMetricWithLabelValues(ResourceChannel, "/channels")
	assert.NoError(t, err)
	assert.Equal(t, float64(25), testutil.ToFloat64(remainingGauge))

	resetTimeGauge, err := discordRateLimitResetTime.GetMetricWithLabelValues(ResourceChannel, "/channels")
	assert.NoError(t, err)
	assert.Equal(t, float64(resetTime.Unix()), testutil.ToFloat64(resetTimeGauge))
}

func TestMetricsRecorder_RecordManagedResource(t *testing.T) {
	recorder := NewMetricsRecorder()

	// Clear metrics before test
	managedResources.Reset()

	// Record managed resource changes
	recorder.RecordManagedResource(ResourceGuild, "ready", 1)
	recorder.RecordManagedResource(ResourceGuild, "ready", 2)
	recorder.RecordManagedResource(ResourceGuild, "ready", -1)

	// Verify gauge was updated correctly
	gauge, err := managedResources.GetMetricWithLabelValues(ResourceGuild, "ready")
	assert.NoError(t, err)
	assert.Equal(t, float64(2), testutil.ToFloat64(gauge))
}

func TestMetricsRecorder_RecordReconciliation(t *testing.T) {
	recorder := NewMetricsRecorder()

	// Clear metrics before test
	resourceReconciliations.Reset()
	resourceReconciliationDuration.Reset()

	// Record reconciliation
	duration := 250 * time.Millisecond
	recorder.RecordReconciliation(ResourceRole, "success", duration)

	// Verify metrics were recorded
	counter, err := resourceReconciliations.GetMetricWithLabelValues(ResourceRole, "success")
	assert.NoError(t, err)
	assert.Equal(t, float64(1), testutil.ToFloat64(counter))
}

func TestMetricsRecorder_RecordAPIError(t *testing.T) {
	recorder := NewMetricsRecorder()

	// Clear metrics before test
	discordAPIErrors.Reset()

	// Record API error
	recorder.RecordAPIError(ResourceGuild, "404", "not_found")

	// Verify counter was incremented
	counter, err := discordAPIErrors.GetMetricWithLabelValues(ResourceGuild, "404", "not_found")
	assert.NoError(t, err)
	assert.Equal(t, float64(1), testutil.ToFloat64(counter))
}

func TestMetricsRecorder_SetProviderHealth(t *testing.T) {
	recorder := NewMetricsRecorder()

	// Clear metrics before test
	providerHealth.Reset()

	// Set health status
	recorder.SetProviderHealth("discord_api", true)
	recorder.SetProviderHealth("kubernetes", false)

	// Verify gauges were set correctly
	healthyGauge, err := providerHealth.GetMetricWithLabelValues("discord_api")
	assert.NoError(t, err)
	assert.Equal(t, float64(1), testutil.ToFloat64(healthyGauge))

	unhealthyGauge, err := providerHealth.GetMetricWithLabelValues("kubernetes")
	assert.NoError(t, err)
	assert.Equal(t, float64(0), testutil.ToFloat64(unhealthyGauge))
}

func TestMetricsRecorder_RecordOperationWithTimer(t *testing.T) {
	recorder := NewMetricsRecorder()

	// Clear metrics before test
	discordAPIOperations.Reset()
	discordAPIOperationDuration.Reset()

	// Use timer function
	finishFunc := recorder.RecordOperationWithTimer(ResourceChannel, OpUpdate)

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	// Finish the operation
	finishFunc(StatusSuccess)

	// Verify metrics were recorded
	counter, err := discordAPIOperations.GetMetricWithLabelValues(ResourceChannel, OpUpdate, StatusSuccess)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), testutil.ToFloat64(counter))
}

func TestMetricsRecorder_RecordReconciliationWithTimer(t *testing.T) {
	recorder := NewMetricsRecorder()

	// Clear metrics before test
	resourceReconciliations.Reset()
	resourceReconciliationDuration.Reset()

	// Use timer function
	finishFunc := recorder.RecordReconciliationWithTimer(ResourceRole)

	// Simulate some work
	time.Sleep(5 * time.Millisecond)

	// Finish the reconciliation
	finishFunc("success")

	// Verify metrics were recorded
	counter, err := resourceReconciliations.GetMetricWithLabelValues(ResourceRole, "success")
	assert.NoError(t, err)
	assert.Equal(t, float64(1), testutil.ToFloat64(counter))
}

func TestGetMetricsRecorder(t *testing.T) {
	recorder1 := GetMetricsRecorder()
	recorder2 := GetMetricsRecorder()

	// Should return the same instance
	assert.Same(t, recorder1, recorder2)
	assert.NotNil(t, recorder1)
}

func TestNewMetricsRecorder(t *testing.T) {
	recorder := NewMetricsRecorder()
	assert.NotNil(t, recorder)
	assert.NotNil(t, recorder.logger)
}

func TestConstants(t *testing.T) {
	// Verify metric constants are defined
	assert.Equal(t, "discord", ProviderNamespace)
	assert.Equal(t, "guild", ResourceGuild)
	assert.Equal(t, "channel", ResourceChannel)
	assert.Equal(t, "role", ResourceRole)
	assert.Equal(t, "create", OpCreate)
	assert.Equal(t, "update", OpUpdate)
	assert.Equal(t, "delete", OpDelete)
	assert.Equal(t, "observe", OpObserve)
	assert.Equal(t, "success", StatusSuccess)
	assert.Equal(t, "error", StatusError)
	assert.Equal(t, "rate_limited", StatusRateLimited)
}
