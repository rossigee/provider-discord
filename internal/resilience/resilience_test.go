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

package resilience

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, DefaultMaxRetries, config.MaxRetries)
	assert.Equal(t, DefaultBaseDelay, config.BaseDelay)
	assert.Equal(t, DefaultMaxDelay, config.MaxDelay)
	assert.Equal(t, DefaultJitterFactor, config.JitterFactor)
	assert.Equal(t, 2.0, config.Multiplier)
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig()

	assert.Equal(t, DefaultFailureThreshold, config.FailureThreshold)
	assert.Equal(t, DefaultRecoveryTimeout, config.RecoveryTimeout)
	assert.Equal(t, DefaultSuccessThreshold, config.SuccessThreshold)
}

func TestDiscordError(t *testing.T) {
	err := &DiscordError{
		StatusCode:   429,
		Message:      "Rate limited",
		ErrorType:    ErrorTypeRateLimit,
		RetryAfter:   30 * time.Second,
		RateLimited:  true,
		Retryable:    true,
		ResourceType: "guild",
		Operation:    "create",
	}

	assert.True(t, err.IsRetryable())
	assert.Equal(t, 30*time.Second, err.GetRetryAfter())
	assert.Contains(t, err.Error(), "Discord API error [429]")
	assert.Contains(t, err.Error(), "Rate limited")
	assert.Contains(t, err.Error(), "rate_limit")
	assert.Contains(t, err.Error(), "retryable: true")
}

func TestCircuitBreaker_NewCircuitBreaker(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config, "test")

	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, config, cb.config)
	assert.Equal(t, "test", cb.resourceType)
	assert.Equal(t, 0, cb.failures)
	assert.Equal(t, 0, cb.successes)
}

func TestCircuitBreaker_SuccessfulCalls(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  1 * time.Second,
		SuccessThreshold: 2,
	}
	cb := NewCircuitBreaker(config, "test")

	// Successful calls should keep circuit closed
	err := cb.Call(context.Background(), "test", func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())

	err = cb.Call(context.Background(), "test", func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_FailuresOpenCircuit(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  1 * time.Second,
		SuccessThreshold: 2,
	}
	cb := NewCircuitBreaker(config, "test")

	testErr := errors.New("test error")

	// First failure
	err := cb.Call(context.Background(), "test", func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateClosed, cb.GetState())

	// Second failure should open circuit
	err = cb.Call(context.Background(), "test", func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateOpen, cb.GetState())

	// Circuit should now reject calls
	err = cb.Call(context.Background(), "test", func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Circuit breaker is open")
}

func TestCircuitBreaker_RecoveryAfterTimeout(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 1,
		RecoveryTimeout:  10 * time.Millisecond, // Short timeout for testing
		SuccessThreshold: 1,
	}
	cb := NewCircuitBreaker(config, "test")

	// Cause failure to open circuit
	testErr := errors.New("test error")
	err := cb.Call(context.Background(), "test", func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for recovery timeout
	time.Sleep(15 * time.Millisecond)

	// Next call should transition to half-open
	err = cb.Call(context.Background(), "test", func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState()) // Should close after successful call
}

func TestResilientClient_SuccessfulOperation(t *testing.T) {
	retryConfig := &RetryConfig{
		MaxRetries:   2,
		BaseDelay:    1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		JitterFactor: 0.1,
		Multiplier:   2.0,
	}

	client := NewResilientClient("test", retryConfig, nil)

	callCount := 0
	err := client.Do(context.Background(), "test_op", func() error {
		callCount++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestResilientClient_RetryOnRetryableError(t *testing.T) {
	retryConfig := &RetryConfig{
		MaxRetries:   2,
		BaseDelay:    1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		JitterFactor: 0.1,
		Multiplier:   2.0,
	}

	client := NewResilientClient("test", retryConfig, nil)

	callCount := 0
	err := client.Do(context.Background(), "test_op", func() error {
		callCount++
		if callCount < 3 {
			return errors.New("rate limit exceeded") // Should trigger retry
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestResilientClient_NoRetryOnNonRetryableError(t *testing.T) {
	retryConfig := &RetryConfig{
		MaxRetries:   2,
		BaseDelay:    1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		JitterFactor: 0.1,
		Multiplier:   2.0,
	}

	client := NewResilientClient("test", retryConfig, nil)

	callCount := 0
	err := client.Do(context.Background(), "test_op", func() error {
		callCount++
		return errors.New("unauthorized") // Should not trigger retry
	})

	assert.Error(t, err)
	assert.Equal(t, 1, callCount)
}

func TestResilientClient_ExhaustRetries(t *testing.T) {
	retryConfig := &RetryConfig{
		MaxRetries:   2,
		BaseDelay:    1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		JitterFactor: 0.1,
		Multiplier:   2.0,
	}

	client := NewResilientClient("test", retryConfig, nil)

	callCount := 0
	testErr := errors.New("rate limit exceeded")
	err := client.Do(context.Background(), "test_op", func() error {
		callCount++
		return testErr
	})

	assert.Equal(t, testErr, err)
	assert.Equal(t, 3, callCount) // Initial call + 2 retries
}

func TestParseDiscordError_ExistingDiscordError(t *testing.T) {
	originalErr := &DiscordError{
		StatusCode:   429,
		Message:      "Rate limited",
		ErrorType:    ErrorTypeRateLimit,
		RetryAfter:   30 * time.Second,
		RateLimited:  true,
		Retryable:    true,
		ResourceType: "guild",
		Operation:    "create",
	}

	parsed := ParseDiscordError(originalErr, "test", "test_op")
	assert.Equal(t, originalErr, parsed)
}

func TestParseDiscordError_GenericError(t *testing.T) {
	tests := []struct {
		name          string
		errorMessage  string
		expectedType  ErrorType
		expectedRetryable bool
		expectedStatus int
	}{
		{
			name:          "rate limit error",
			errorMessage:  "rate limit exceeded",
			expectedType:  ErrorTypeRateLimit,
			expectedRetryable: true,
			expectedStatus: 429,
		},
		{
			name:          "network error",
			errorMessage:  "connection timeout",
			expectedType:  ErrorTypeNetwork,
			expectedRetryable: true,
			expectedStatus: 0,
		},
		{
			name:          "unauthorized error",
			errorMessage:  "unauthorized access",
			expectedType:  ErrorTypeAuthentication,
			expectedRetryable: false,
			expectedStatus: 401,
		},
		{
			name:          "forbidden error",
			errorMessage:  "forbidden operation",
			expectedType:  ErrorTypePermission,
			expectedRetryable: false,
			expectedStatus: 403,
		},
		{
			name:          "not found error",
			errorMessage:  "resource not found",
			expectedType:  ErrorTypeNotFound,
			expectedRetryable: false,
			expectedStatus: 404,
		},
		{
			name:          "server error",
			errorMessage:  "internal server error",
			expectedType:  ErrorTypeTemporary,
			expectedRetryable: true,
			expectedStatus: 500,
		},
		{
			name:          "unknown error",
			errorMessage:  "some random error",
			expectedType:  ErrorTypeUnknown,
			expectedRetryable: true,
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errorMessage)
			parsed := ParseDiscordError(err, "test", "test_op")

			assert.Equal(t, tt.expectedType, parsed.ErrorType)
			assert.Equal(t, tt.expectedRetryable, parsed.Retryable)
			assert.Equal(t, tt.expectedStatus, parsed.StatusCode)
			assert.Equal(t, "test", parsed.ResourceType)
			assert.Equal(t, "test_op", parsed.Operation)
		})
	}
}

func TestParseRateLimitHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set(DiscordRateLimitHeader, "5")
	headers.Set(DiscordRateLimitReset, "1.5")

	remaining, resetAfter, err := ParseRateLimitHeaders(headers)

	assert.NoError(t, err)
	assert.Equal(t, 5, remaining)
	assert.Equal(t, time.Duration(1.5*float64(time.Second)), resetAfter)
}

func TestParseRateLimitHeaders_RetryAfter(t *testing.T) {
	headers := http.Header{}
	headers.Set(DiscordRateLimitHeader, "0")
	headers.Set(DiscordRetryAfterHeader, "30.0")

	remaining, resetAfter, err := ParseRateLimitHeaders(headers)

	assert.NoError(t, err)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 30*time.Second, resetAfter)
}

func TestParseRateLimitHeaders_InvalidValues(t *testing.T) {
	headers := http.Header{}
	headers.Set(DiscordRateLimitHeader, "invalid")

	_, _, err := ParseRateLimitHeaders(headers)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse rate limit remaining")
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		substrings []string
		expected   bool
	}{
		{
			name:       "contains one substring",
			s:          "this is a rate limit error",
			substrings: []string{"rate limit", "timeout"},
			expected:   true,
		},
		{
			name:       "contains no substrings",
			s:          "this is a normal error",
			substrings: []string{"rate limit", "timeout"},
			expected:   false,
		},
		{
			name:       "contains multiple substrings",
			s:          "connection timeout during rate limit",
			substrings: []string{"rate limit", "timeout"},
			expected:   true,
		},
		{
			name:       "empty substrings",
			s:          "any string",
			substrings: []string{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrings)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResilientClient_CalculateDelay(t *testing.T) {
	retryConfig := &RetryConfig{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		JitterFactor: 0.1,
		Multiplier:   2.0,
	}

	client := NewResilientClient("test", retryConfig, nil)

	// Test with server retry-after
	delay := client.calculateDelay(0, 2*time.Second)
	assert.True(t, delay >= 1800*time.Millisecond) // Should be around 2s ± 10%
	assert.True(t, delay <= 2200*time.Millisecond)

	// Test exponential backoff
	delay1 := client.calculateDelay(0, 0)
	delay2 := client.calculateDelay(1, 0)

	// Second delay should be roughly double the first (with jitter)
	assert.True(t, delay2 > delay1)
	assert.True(t, delay1 >= 90*time.Millisecond)   // Should be around 100ms ± 10%
	assert.True(t, delay1 <= 110*time.Millisecond)
}

func TestNewResilientClient_WithDefaults(t *testing.T) {
	client := NewResilientClient("test", nil, nil)

	assert.NotNil(t, client.retryConfig)
	assert.NotNil(t, client.circuitBreaker)
	assert.Equal(t, DefaultMaxRetries, client.retryConfig.MaxRetries)
	assert.Equal(t, DefaultBaseDelay, client.retryConfig.BaseDelay)
}
