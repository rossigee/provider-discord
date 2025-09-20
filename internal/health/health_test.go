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

package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

)

func TestHealthChecker_ServeHealthz(t *testing.T) {
	checker := NewHealthChecker(nil, nil)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	checker.ServeHealthz(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var status HealthStatus
	err := json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.Equal(t, "healthy", status.Status)
	assert.Equal(t, "Discord provider is running", status.Message)
	assert.NotZero(t, status.Timestamp)
	assert.NotEmpty(t, status.Duration)
}

func TestHealthChecker_ServeReadyz_AllHealthy(t *testing.T) {
	// Mock Discord check that succeeds
	discordCheck := func(ctx context.Context) error {
		return nil
	}

	// Don't use kubeClient to avoid fake client limitations
	checker := NewHealthChecker(nil, discordCheck)

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	checker.ServeReadyz(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var status HealthStatus
	err := json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.Equal(t, "ready", status.Status)
	assert.Equal(t, "All components are healthy", status.Message)
	assert.NotZero(t, status.Timestamp)
	assert.NotEmpty(t, status.Duration)
	assert.Equal(t, "healthy", status.Details["discord_api"])
}

func TestHealthChecker_ServeReadyz_DiscordUnhealthy(t *testing.T) {
	// Mock Discord check that fails
	discordCheck := func(ctx context.Context) error {
		return errors.New("Discord API unreachable")
	}

	// Don't use kubeClient to avoid fake client limitations
	checker := NewHealthChecker(nil, discordCheck)

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	checker.ServeReadyz(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var status HealthStatus
	err := json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.Equal(t, "not_ready", status.Status)
	assert.Equal(t, "Some components are unhealthy", status.Message)
	assert.NotZero(t, status.Timestamp)
	assert.NotEmpty(t, status.Duration)
	assert.Contains(t, status.Details["discord_api"], "unhealthy: Discord API unreachable")
}

func TestHealthChecker_ServeReadyz_NoChecks(t *testing.T) {
	checker := NewHealthChecker(nil, nil)

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	checker.ServeReadyz(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status HealthStatus
	err := json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.Equal(t, "ready", status.Status)
	assert.Equal(t, "All components are healthy", status.Message)
	assert.Empty(t, status.Details)
}

func TestCreateDiscordHealthCheck(t *testing.T) {
	check := CreateDiscordHealthCheck()
	assert.NotNil(t, check)

	// Test that the health check function works
	ctx := context.Background()
	err := check(ctx)
	assert.NoError(t, err) // Should succeed with placeholder implementation
}

func TestSetupHealthChecks(t *testing.T) {
	mux := http.NewServeMux()
	checker := NewHealthChecker(nil, nil)

	SetupHealthChecks(mux, checker)

	// Test that health endpoints are registered
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest("GET", "/readyz", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthStatus_JSONMarshal(t *testing.T) {
	status := HealthStatus{
		Status:    "healthy",
		Message:   "test message",
		Timestamp: testTime,
		Details:   map[string]string{"component": "healthy"},
		Duration:  "10ms",
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)

	var unmarshaled HealthStatus
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, status.Status, unmarshaled.Status)
	assert.Equal(t, status.Message, unmarshaled.Message)
	assert.Equal(t, status.Details, unmarshaled.Details)
	assert.Equal(t, status.Duration, unmarshaled.Duration)
}

// Helper variable for consistent testing
var testTime = mustParseTime("2023-01-01T00:00:00Z")

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
