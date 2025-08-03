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

// Package resilience provides error handling, retry logic, and circuit breaker patterns
// for the Discord provider to handle rate limiting and API failures gracefully.
package resilience

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/rossigee/provider-discord/internal/metrics"
)

const (
	// Default retry configuration
	DefaultMaxRetries     = 3
	DefaultBaseDelay      = 100 * time.Millisecond
	DefaultMaxDelay       = 30 * time.Second
	DefaultJitterFactor   = 0.1
	
	// Circuit breaker defaults
	DefaultFailureThreshold = 5
	DefaultRecoveryTimeout  = 60 * time.Second
	DefaultSuccessThreshold = 3
	
	// Discord-specific constants
	DiscordRateLimitHeader  = "X-RateLimit-Remaining"
	DiscordRateLimitReset   = "X-RateLimit-Reset-After"
	DiscordRetryAfterHeader = "Retry-After"
)

// RetryConfig defines configuration for retry logic
type RetryConfig struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	JitterFactor  float64
	Multiplier    float64
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:   DefaultMaxRetries,
		BaseDelay:    DefaultBaseDelay,
		MaxDelay:     DefaultMaxDelay,
		JitterFactor: DefaultJitterFactor,
		Multiplier:   2.0,
	}
}

// CircuitBreakerConfig defines configuration for circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold int
	RecoveryTimeout  time.Duration
	SuccessThreshold int
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold: DefaultFailureThreshold,
		RecoveryTimeout:  DefaultRecoveryTimeout,
		SuccessThreshold: DefaultSuccessThreshold,
	}
}

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeRateLimit     ErrorType = "rate_limit"
	ErrorTypeTemporary     ErrorType = "temporary"
	ErrorTypePermanent     ErrorType = "permanent"
	ErrorTypeUnknown       ErrorType = "unknown"
	ErrorTypeNetwork       ErrorType = "network"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypePermission    ErrorType = "permission"
	ErrorTypeNotFound      ErrorType = "not_found"
)

// DiscordError represents a Discord API error with retry information
type DiscordError struct {
	StatusCode   int
	Message      string
	ErrorType    ErrorType
	RetryAfter   time.Duration
	RateLimited  bool
	Retryable    bool
	ResourceType string
	Operation    string
}

func (e *DiscordError) Error() string {
	return fmt.Sprintf("Discord API error [%d]: %s (type: %s, retryable: %v)", 
		e.StatusCode, e.Message, e.ErrorType, e.Retryable)
}

// IsRetryable returns whether the error should be retried
func (e *DiscordError) IsRetryable() bool {
	return e.Retryable
}

// GetRetryAfter returns the duration to wait before retrying
func (e *DiscordError) GetRetryAfter() time.Duration {
	return e.RetryAfter
}

// CircuitState represents the state of a circuit breaker
type CircuitState string

const (
	StateClosed   CircuitState = "closed"
	StateOpen     CircuitState = "open"
	StateHalfOpen CircuitState = "half_open"
)

// CircuitBreaker implements the circuit breaker pattern for Discord API calls
type CircuitBreaker struct {
	config         *CircuitBreakerConfig
	state          CircuitState
	failures       int
	lastFailureTime time.Time
	successes      int
	logger         logr.Logger
	metrics        *metrics.MetricsRecorder
	resourceType   string
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig, resourceType string) *CircuitBreaker {
	return &CircuitBreaker{
		config:       config,
		state:        StateClosed,
		failures:     0,
		successes:    0,
		resourceType: resourceType,
		logger:       log.Log.WithName("circuit-breaker").WithValues("resource_type", resourceType),
		metrics:      metrics.GetMetricsRecorder(),
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, operation string, fn func() error) error {
	if !cb.canCall() {
		return &DiscordError{
			StatusCode:   503,
			Message:      "Circuit breaker is open",
			ErrorType:    ErrorTypeTemporary,
			Retryable:    false,
			ResourceType: cb.resourceType,
			Operation:    operation,
		}
	}

	err := fn()
	cb.recordResult(err)
	return err
}

// canCall checks if the circuit breaker allows the call
func (cb *CircuitBreaker) canCall() bool {
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) > cb.config.RecoveryTimeout {
			cb.setState(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult records the result of a call and updates circuit breaker state
func (cb *CircuitBreaker) recordResult(err error) {
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()
	cb.successes = 0

	if cb.state == StateHalfOpen {
		cb.setState(StateOpen)
	} else if cb.failures >= cb.config.FailureThreshold {
		cb.setState(StateOpen)
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	cb.successes++

	if cb.state == StateHalfOpen && cb.successes >= cb.config.SuccessThreshold {
		cb.setState(StateClosed)
		cb.failures = 0
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState CircuitState) {
	if cb.state != newState {
		cb.logger.Info("Circuit breaker state changed",
			"old_state", cb.state,
			"new_state", newState,
			"failures", cb.failures,
			"successes", cb.successes,
		)
		cb.state = newState
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	return cb.state
}

// ResilientClient wraps Discord API calls with retry logic and circuit breaking
type ResilientClient struct {
	retryConfig    *RetryConfig
	circuitBreaker *CircuitBreaker
	logger         logr.Logger
	metrics        *metrics.MetricsRecorder
	resourceType   string
}

// NewResilientClient creates a new resilient client
func NewResilientClient(resourceType string, retryConfig *RetryConfig, cbConfig *CircuitBreakerConfig) *ResilientClient {
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}
	if cbConfig == nil {
		cbConfig = DefaultCircuitBreakerConfig()
	}

	return &ResilientClient{
		retryConfig:    retryConfig,
		circuitBreaker: NewCircuitBreaker(cbConfig, resourceType),
		resourceType:   resourceType,
		logger:         log.Log.WithName("resilient-client").WithValues("resource_type", resourceType),
		metrics:        metrics.GetMetricsRecorder(),
	}
}

// Do executes a function with full resilience (retry + circuit breaking)
func (rc *ResilientClient) Do(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= rc.retryConfig.MaxRetries; attempt++ {
		err := rc.circuitBreaker.Call(ctx, operation, fn)
		if err == nil {
			if attempt > 0 {
				rc.logger.Info("Operation succeeded after retries",
					"operation", operation,
					"attempts", attempt+1,
				)
			}
			rc.metrics.RecordAPIOperation(rc.resourceType, operation, metrics.StatusSuccess, 0)
			return nil
		}

		lastErr = err
		discordErr := ParseDiscordError(err, rc.resourceType, operation)

		// Record the error
		rc.metrics.RecordAPIError(rc.resourceType, 
			strconv.Itoa(discordErr.StatusCode), 
			string(discordErr.ErrorType))

		if discordErr.RateLimited {
			rc.metrics.RecordAPIOperation(rc.resourceType, operation, metrics.StatusRateLimited, 0)
		} else {
			rc.metrics.RecordAPIOperation(rc.resourceType, operation, metrics.StatusError, 0)
		}

		// Don't retry if not retryable or if we've exhausted attempts
		if !discordErr.IsRetryable() || attempt >= rc.retryConfig.MaxRetries {
			break
		}

		// Calculate delay
		delay := rc.calculateDelay(attempt, discordErr.GetRetryAfter())
		
		rc.logger.Info("Retrying operation after error",
			"operation", operation,
			"attempt", attempt+1,
			"max_attempts", rc.retryConfig.MaxRetries+1,
			"delay", delay,
			"error", discordErr.Message,
		)

		// Wait for the calculated delay
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return lastErr
}

// calculateDelay calculates the delay for the next retry attempt
func (rc *ResilientClient) calculateDelay(attempt int, serverRetryAfter time.Duration) time.Duration {
	// If server specified retry-after, use that with some jitter
	if serverRetryAfter > 0 {
		jitter := time.Duration(float64(serverRetryAfter) * rc.retryConfig.JitterFactor * (rand.Float64() - 0.5))
		return serverRetryAfter + jitter
	}

	// Exponential backoff with jitter
	delay := float64(rc.retryConfig.BaseDelay) * math.Pow(rc.retryConfig.Multiplier, float64(attempt))
	
	// Add jitter (±10% by default)
	jitter := delay * rc.retryConfig.JitterFactor * (rand.Float64() - 0.5)
	delay += jitter

	// Cap at maximum delay
	if delay > float64(rc.retryConfig.MaxDelay) {
		delay = float64(rc.retryConfig.MaxDelay)
	}

	return time.Duration(delay)
}

// ParseDiscordError parses an error and extracts Discord-specific information
func ParseDiscordError(err error, resourceType, operation string) *DiscordError {
	if err == nil {
		return nil
	}

	// If it's already a DiscordError, return it
	if discordErr, ok := err.(*DiscordError); ok {
		return discordErr
	}

	// Try to extract information from HTTP responses
	discordErr := &DiscordError{
		StatusCode:   500,
		Message:      err.Error(),
		ErrorType:    ErrorTypeUnknown,
		RetryAfter:   0,
		RateLimited:  false,
		Retryable:    false,
		ResourceType: resourceType,
		Operation:    operation,
	}

	// Parse HTTP errors if available
	// This would typically involve checking response headers and status codes
	// Implementation depends on the specific HTTP client being used

	// Default retry logic based on error content
	errorStr := err.Error()
	
	switch {
	case containsAny(errorStr, []string{"rate limit", "too many requests"}):
		discordErr.ErrorType = ErrorTypeRateLimit
		discordErr.RateLimited = true
		discordErr.Retryable = true
		discordErr.StatusCode = 429
	case containsAny(errorStr, []string{"timeout", "connection", "network"}):
		discordErr.ErrorType = ErrorTypeNetwork
		discordErr.Retryable = true
		discordErr.StatusCode = 0
	case containsAny(errorStr, []string{"unauthorized", "invalid token"}):
		discordErr.ErrorType = ErrorTypeAuthentication
		discordErr.Retryable = false
		discordErr.StatusCode = 401
	case containsAny(errorStr, []string{"forbidden", "missing permissions"}):
		discordErr.ErrorType = ErrorTypePermission
		discordErr.Retryable = false
		discordErr.StatusCode = 403
	case containsAny(errorStr, []string{"not found"}):
		discordErr.ErrorType = ErrorTypeNotFound
		discordErr.Retryable = false
		discordErr.StatusCode = 404
	case containsAny(errorStr, []string{"internal server error", "bad gateway", "service unavailable"}):
		discordErr.ErrorType = ErrorTypeTemporary
		discordErr.Retryable = true
		discordErr.StatusCode = 500
	default:
		discordErr.ErrorType = ErrorTypeUnknown
		discordErr.Retryable = true // Be conservative and retry unknown errors
	}

	return discordErr
}

// ParseRateLimitHeaders extracts rate limit information from HTTP headers
func ParseRateLimitHeaders(headers http.Header) (remaining int, resetAfter time.Duration, err error) {
	// Parse remaining requests
	if remainingStr := headers.Get(DiscordRateLimitHeader); remainingStr != "" {
		if remaining, err = strconv.Atoi(remainingStr); err != nil {
			return 0, 0, fmt.Errorf("failed to parse rate limit remaining: %w", err)
		}
	}

	// Parse reset time
	if resetAfterStr := headers.Get(DiscordRateLimitReset); resetAfterStr != "" {
		if resetAfterSeconds, parseErr := strconv.ParseFloat(resetAfterStr, 64); parseErr == nil {
			resetAfter = time.Duration(resetAfterSeconds * float64(time.Second))
		}
	}

	// Check for retry-after header (used in 429 responses)
	if retryAfterStr := headers.Get(DiscordRetryAfterHeader); retryAfterStr != "" {
		if retryAfterSeconds, parseErr := strconv.ParseFloat(retryAfterStr, 64); parseErr == nil {
			resetAfter = time.Duration(retryAfterSeconds * float64(time.Second))
		}
	}

	return remaining, resetAfter, nil
}

// containsAny checks if a string contains any of the given substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		// Simple contains check (case-insensitive would be better)
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
	}
	return false
}