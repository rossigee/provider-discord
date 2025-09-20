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

package tracing

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	// Clear environment variables
	if err := os.Unsetenv("OTEL_TRACING_ENABLED"); err != nil {
		t.Errorf("Failed to unset OTEL_TRACING_ENABLED: %v", err)
	}
	if err := os.Unsetenv("OTEL_SERVICE_NAME"); err != nil {
		t.Errorf("Failed to unset OTEL_SERVICE_NAME: %v", err)
	}
	if err := os.Unsetenv("OTEL_SERVICE_VERSION"); err != nil {
		t.Errorf("Failed to unset OTEL_SERVICE_VERSION: %v", err)
	}
	if err := os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT"); err != nil {
		t.Errorf("Failed to unset OTEL_EXPORTER_OTLP_ENDPOINT: %v", err)
	}
	if err := os.Unsetenv("OTEL_SAMPLING_RATIO"); err != nil {
		t.Errorf("Failed to unset OTEL_SAMPLING_RATIO: %v", err)
	}

	config := DefaultConfig()

	assert.False(t, config.Enabled)
	assert.Equal(t, "provider-discord", config.ServiceName)
	assert.Equal(t, "unknown", config.ServiceVersion)
	assert.Equal(t, "", config.Endpoint)
	assert.Equal(t, 0.1, config.SamplingRatio)
	assert.NotNil(t, config.Headers)
}

func TestDefaultConfigWithEnvVars(t *testing.T) {
	// Set environment variables
	if err := os.Setenv("OTEL_TRACING_ENABLED", "false"); err != nil {
		t.Errorf("Failed to set OTEL_TRACING_ENABLED: %v", err)
	}
	if err := os.Setenv("OTEL_SERVICE_NAME", "test-service"); err != nil {
		t.Errorf("Failed to set OTEL_SERVICE_NAME: %v", err)
	}
	if err := os.Setenv("OTEL_SERVICE_VERSION", "v1.0.0"); err != nil {
		t.Errorf("Failed to set OTEL_SERVICE_VERSION: %v", err)
	}
	if err := os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"); err != nil {
		t.Errorf("Failed to set OTEL_EXPORTER_OTLP_ENDPOINT: %v", err)
	}
	if err := os.Setenv("OTEL_SAMPLING_RATIO", "0.5"); err != nil {
		t.Errorf("Failed to set OTEL_SAMPLING_RATIO: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("OTEL_TRACING_ENABLED"); err != nil {
			t.Errorf("Failed to unset OTEL_TRACING_ENABLED: %v", err)
		}
		_ = os.Unsetenv("OTEL_SERVICE_NAME")
		_ = os.Unsetenv("OTEL_SERVICE_VERSION")
		_ = os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		_ = os.Unsetenv("OTEL_SAMPLING_RATIO")
	}()

	config := DefaultConfig()

	assert.False(t, config.Enabled)
	assert.Equal(t, "test-service", config.ServiceName)
	assert.Equal(t, "v1.0.0", config.ServiceVersion)
	assert.Equal(t, "http://localhost:4318", config.Endpoint)
	assert.Equal(t, 0.5, config.SamplingRatio)
}

func TestInitializeDisabled(t *testing.T) {
	config := &Config{
		Enabled: false,
	}

	err := Initialize(context.Background(), config)
	assert.NoError(t, err)

	tracer := GetTracer()
	assert.NotNil(t, tracer)
}

func TestInitializeNoEndpoint(t *testing.T) {
	config := &Config{
		Enabled:  true,
		Endpoint: "",
	}

	err := Initialize(context.Background(), config)
	assert.NoError(t, err)

	tracer := GetTracer()
	assert.NotNil(t, tracer)
}

func TestGetTracerBeforeInit(t *testing.T) {
	// Reset global tracer
	tracer = nil

	tr := GetTracer()
	assert.NotNil(t, tr)
}

func TestStartSpan(t *testing.T) {
	ctx := context.Background()

	_, span := StartSpan(ctx, "test-span")
	assert.NotNil(t, span)

	span.End()
}

func TestRecordError(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-span")

	testErr := errors.New("test error")
	RecordError(span, testErr, "test_error")

	span.End()

	// Test with nil error
	RecordError(span, nil, "no_error")
}

func TestRecordSuccess(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-span")

	RecordSuccess(span)

	span.End()
}

func TestTraceReconciliation(t *testing.T) {
	ctx := context.Background()

	_, span := TraceReconciliation(ctx, "guild", "test-guild", "create")
	assert.NotNil(t, span)

	span.End()
}

func TestTraceAPICall(t *testing.T) {
	ctx := context.Background()

	_, span := TraceAPICall(ctx, "POST", "/guilds")
	assert.NotNil(t, span)

	span.End()
}

func TestTraceResourceOperation(t *testing.T) {
	ctx := context.Background()

	_, span := TraceResourceOperation(ctx, "channel", "update", "123456789")
	assert.NotNil(t, span)

	span.End()
}

func TestSetDiscordAttributes(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-span")

	attrs := map[string]interface{}{
		"string_attr":  "test_value",
		"int_attr":     42,
		"int64_attr":   int64(9876543210),
		"bool_attr":    true,
		"float64_attr": 3.14,
		"other_attr":   struct{ Name string }{Name: "test"},
	}

	SetDiscordAttributes(span, attrs)

	span.End()
}

func TestAddGuildContext(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-span")

	AddGuildContext(span, "123456789", "Test Guild")

	span.End()
}

func TestAddChannelContext(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-span")

	AddChannelContext(span, "987654321", "general", 0)

	span.End()
}

func TestAddRoleContext(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-span")

	AddRoleContext(span, "555666777", "Admin")

	span.End()
}

func TestRecordAPIResponse(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-span")

	// Test successful response
	RecordAPIResponse(span, 200, false)

	// Test error response
	RecordAPIResponse(span, 429, true)

	span.End()
}

func TestRecordRetryAttempt(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-span")

	RecordRetryAttempt(span, 2, "500ms")

	span.End()
}

func TestConstants(t *testing.T) {
	// Verify important constants are defined
	assert.Equal(t, "provider-discord", TracerName)
	assert.Equal(t, "discord.resource.type", AttrResourceType)
	assert.Equal(t, "discord.resource.id", AttrResourceID)
	assert.Equal(t, "discord.operation", AttrOperation)
	assert.Equal(t, "discord.reconcile", SpanReconcile)
	assert.Equal(t, "discord.api.call", SpanAPICall)
}

func TestGetEnvHelpers(t *testing.T) {
	// Test getEnv
	if err := os.Setenv("TEST_STRING", "test_value"); err != nil {
		t.Errorf("Failed to set TEST_STRING: %v", err)
	}
	assert.Equal(t, "test_value", getEnv("TEST_STRING", "default"))
	assert.Equal(t, "default", getEnv("NON_EXISTENT", "default"))
	_ = os.Unsetenv("TEST_STRING")

	// Test getEnvBool
	if err := os.Setenv("TEST_BOOL_TRUE", "true"); err != nil {
		t.Errorf("Failed to set TEST_BOOL_TRUE: %v", err)
	}
	if err := os.Setenv("TEST_BOOL_1", "1"); err != nil {
		t.Errorf("Failed to set TEST_BOOL_1: %v", err)
	}
	if err := os.Setenv("TEST_BOOL_FALSE", "false"); err != nil {
		t.Errorf("Failed to set TEST_BOOL_FALSE: %v", err)
	}
	assert.True(t, getEnvBool("TEST_BOOL_TRUE", false))
	assert.True(t, getEnvBool("TEST_BOOL_1", false))
	assert.False(t, getEnvBool("TEST_BOOL_FALSE", true))
	assert.True(t, getEnvBool("NON_EXISTENT", true))
	_ = os.Unsetenv("TEST_BOOL_TRUE")
	_ = os.Unsetenv("TEST_BOOL_1")
	_ = os.Unsetenv("TEST_BOOL_FALSE")

	// Test getEnvFloat
	if err := os.Setenv("TEST_FLOAT", "0.5"); err != nil {
		t.Errorf("Failed to set TEST_FLOAT: %v", err)
	}
	assert.Equal(t, 0.5, getEnvFloat("TEST_FLOAT", 0.0))
	assert.Equal(t, 0.1, getEnvFloat("NON_EXISTENT", 0.1))
	_ = os.Unsetenv("TEST_FLOAT")
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		hasError bool
	}{
		{"0.0", 0.0, false},
		{"0", 0.0, false},
		{"0.1", 0.1, false},
		{"0.5", 0.5, false},
		{"1.0", 1.0, false},
		{"1", 1.0, false},
		{"invalid", 0.0, true},
		{"2.5", 0.0, true}, // Not supported in simple implementation
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseFloat(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestShutdown(t *testing.T) {
	// Test shutdown with no provider
	err := Shutdown(context.Background())
	assert.NoError(t, err)
}
