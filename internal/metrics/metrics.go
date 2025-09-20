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

// Package metrics provides Discord provider metrics collection and reporting.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	// Metric namespaces
	ProviderNamespace = "discord"

	// Resource types
	ResourceGuild   = "guild"
	ResourceChannel = "channel"
	ResourceRole    = "role"

	// Operation types
	OpCreate = "create"
	OpUpdate = "update"
	OpDelete = "delete"
	OpObserve = "observe"

	// Status types
	StatusSuccess = "success"
	StatusError   = "error"
	StatusRateLimited = "rate_limited"
)

var (
	// API operation metrics
	discordAPIOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ProviderNamespace,
			Name:      "discord_api_operations_total",
			Help:      "Total number of Discord API operations",
		},
		[]string{"resource_type", "operation", "status"},
	)

	discordAPIOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ProviderNamespace,
			Name:      "discord_api_operation_duration_seconds",
			Help:      "Duration of Discord API operations in seconds",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"resource_type", "operation"},
	)

	// Rate limiting metrics
	discordRateLimits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ProviderNamespace,
			Name:      "discord_rate_limits_total",
			Help:      "Total number of Discord API rate limit hits",
		},
		[]string{"resource_type", "endpoint"},
	)

	discordRateLimitRemaining = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: ProviderNamespace,
			Name:      "discord_rate_limit_remaining",
			Help:      "Remaining Discord API rate limit calls",
		},
		[]string{"resource_type", "endpoint"},
	)

	discordRateLimitResetTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: ProviderNamespace,
			Name:      "discord_rate_limit_reset_timestamp_seconds",
			Help:      "Unix timestamp when Discord API rate limit resets",
		},
		[]string{"resource_type", "endpoint"},
	)

	// Resource management metrics
	managedResources = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: ProviderNamespace,
			Name:      "managed_resources",
			Help:      "Number of resources currently managed by the provider",
		},
		[]string{"resource_type", "status"},
	)

	resourceReconciliations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ProviderNamespace,
			Name:      "resource_reconciliations_total",
			Help:      "Total number of resource reconciliations",
		},
		[]string{"resource_type", "result"},
	)

	resourceReconciliationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ProviderNamespace,
			Name:      "resource_reconciliation_duration_seconds",
			Help:      "Duration of resource reconciliations in seconds",
			Buckets:   []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0},
		},
		[]string{"resource_type"},
	)

	// Error metrics
	discordAPIErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ProviderNamespace,
			Name:      "discord_api_errors_total",
			Help:      "Total number of Discord API errors",
		},
		[]string{"resource_type", "error_code", "error_type"},
	)

	// Provider health metrics
	providerHealth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: ProviderNamespace,
			Name:      "provider_health",
			Help:      "Provider health status (1 = healthy, 0 = unhealthy)",
		},
		[]string{"component"},
	)
)

func init() {
	// Register all metrics with controller-runtime's metrics registry
	metrics.Registry.MustRegister(
		discordAPIOperations,
		discordAPIOperationDuration,
		discordRateLimits,
		discordRateLimitRemaining,
		discordRateLimitResetTime,
		managedResources,
		resourceReconciliations,
		resourceReconciliationDuration,
		discordAPIErrors,
		providerHealth,
	)
}

// MetricsRecorder provides methods for recording various metrics
type MetricsRecorder struct {
	logger logr.Logger
}

// NewMetricsRecorder creates a new metrics recorder
func NewMetricsRecorder() *MetricsRecorder {
	return &MetricsRecorder{
		logger: log.Log.WithName("metrics"),
	}
}

// RecordAPIOperation records a Discord API operation
func (m *MetricsRecorder) RecordAPIOperation(resourceType, operation, status string, duration time.Duration) {
	discordAPIOperations.WithLabelValues(resourceType, operation, status).Inc()
	discordAPIOperationDuration.WithLabelValues(resourceType, operation).Observe(duration.Seconds())

	m.logger.V(1).Info("Recorded API operation",
		"resource_type", resourceType,
		"operation", operation,
		"status", status,
		"duration", duration,
	)
}

// RecordRateLimit records Discord API rate limit information
func (m *MetricsRecorder) RecordRateLimit(resourceType, endpoint string, remaining int, resetTime time.Time) {
	discordRateLimits.WithLabelValues(resourceType, endpoint).Inc()
	discordRateLimitRemaining.WithLabelValues(resourceType, endpoint).Set(float64(remaining))
	discordRateLimitResetTime.WithLabelValues(resourceType, endpoint).Set(float64(resetTime.Unix()))

	m.logger.Info("Recorded rate limit hit",
		"resource_type", resourceType,
		"endpoint", endpoint,
		"remaining", remaining,
		"reset_time", resetTime,
	)
}

// UpdateRateLimitStatus updates current rate limit status without recording a hit
func (m *MetricsRecorder) UpdateRateLimitStatus(resourceType, endpoint string, remaining int, resetTime time.Time) {
	discordRateLimitRemaining.WithLabelValues(resourceType, endpoint).Set(float64(remaining))
	discordRateLimitResetTime.WithLabelValues(resourceType, endpoint).Set(float64(resetTime.Unix()))
}

// RecordManagedResource updates the count of managed resources
func (m *MetricsRecorder) RecordManagedResource(resourceType, status string, delta int) {
	managedResources.WithLabelValues(resourceType, status).Add(float64(delta))

	m.logger.V(1).Info("Updated managed resource count",
		"resource_type", resourceType,
		"status", status,
		"delta", delta,
	)
}

// RecordReconciliation records a resource reconciliation
func (m *MetricsRecorder) RecordReconciliation(resourceType, result string, duration time.Duration) {
	resourceReconciliations.WithLabelValues(resourceType, result).Inc()
	resourceReconciliationDuration.WithLabelValues(resourceType).Observe(duration.Seconds())

	m.logger.V(1).Info("Recorded reconciliation",
		"resource_type", resourceType,
		"result", result,
		"duration", duration,
	)
}

// RecordAPIError records a Discord API error
func (m *MetricsRecorder) RecordAPIError(resourceType, errorCode, errorType string) {
	discordAPIErrors.WithLabelValues(resourceType, errorCode, errorType).Inc()

	m.logger.Info("Recorded API error",
		"resource_type", resourceType,
		"error_code", errorCode,
		"error_type", errorType,
	)
}

// SetProviderHealth sets the provider health status
func (m *MetricsRecorder) SetProviderHealth(component string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	providerHealth.WithLabelValues(component).Set(value)

	m.logger.Info("Updated provider health",
		"component", component,
		"healthy", healthy,
	)
}

// RecordOperationWithTimer records an operation with automatic timing
func (m *MetricsRecorder) RecordOperationWithTimer(resourceType, operation string) func(string) {
	start := time.Now()
	return func(status string) {
		duration := time.Since(start)
		m.RecordAPIOperation(resourceType, operation, status, duration)
	}
}

// RecordReconciliationWithTimer records a reconciliation with automatic timing
func (m *MetricsRecorder) RecordReconciliationWithTimer(resourceType string) func(string) {
	start := time.Now()
	return func(result string) {
		duration := time.Since(start)
		m.RecordReconciliation(resourceType, result, duration)
	}
}

// GetMetricsRecorder returns the global metrics recorder instance
var globalRecorder = NewMetricsRecorder()

func GetMetricsRecorder() *MetricsRecorder {
	return globalRecorder
}
