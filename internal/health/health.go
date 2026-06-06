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
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

)

const (
	// Health check paths
	HealthzPath = "/healthz"
	ReadyzPath  = "/readyz"
)

var (
	// Health check metrics
	healthCheckRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "provider_discord",
			Name:      "health_check_requests_total",
			Help:      "Total number of health check requests",
		},
		[]string{"endpoint", "status"},
	)

	healthCheckDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "provider_discord",
			Name:      "health_check_duration_seconds",
			Help:      "Duration of health check requests in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
		},
		[]string{"endpoint"},
	)

	discordAPIHealth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "provider_discord",
			Name:      "discord_api_health",
			Help:      "Discord API health status (1 = healthy, 0 = unhealthy)",
		},
		[]string{"component"},
	)
)

func init() {
	metrics.Registry.MustRegister(healthCheckRequests, healthCheckDuration, discordAPIHealth)
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status      string            `json:"status"`
	Message     string            `json:"message,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Details     map[string]string `json:"details,omitempty"`
	Duration    string            `json:"duration,omitempty"`
}

// HealthChecker provides health checking functionality
type HealthChecker struct {
	kubeClient   client.Client
	discordCheck func(context.Context) error
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(kubeClient client.Client, discordCheckFunc func(context.Context) error) *HealthChecker {
	return &HealthChecker{
		kubeClient:   kubeClient,
		discordCheck: discordCheckFunc,
	}
}

// ServeHealthz handles liveness probe requests
func (h *HealthChecker) ServeHealthz(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		healthCheckDuration.WithLabelValues("healthz").Observe(time.Since(start).Seconds())
	}()

	logger := log.FromContext(r.Context()).WithValues("endpoint", "healthz")

	// Simple liveness check - just ensure the process is running
	status := HealthStatus{
		Status:    "healthy",
		Message:   "Discord provider is running",
		Timestamp: time.Now(),
		Duration:  time.Since(start).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	healthCheckRequests.WithLabelValues("healthz", "success").Inc()

	if err := json.NewEncoder(w).Encode(status); err != nil {
		logger.Error(err, "failed to encode health status")
	}
}

// ServeReadyz handles readiness probe requests
func (h *HealthChecker) ServeReadyz(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		healthCheckDuration.WithLabelValues("readyz").Observe(time.Since(start).Seconds())
	}()

	logger := log.FromContext(r.Context()).WithValues("endpoint", "readyz")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	details := make(map[string]string)
	allHealthy := true

	// Check Kubernetes connectivity
	if h.kubeClient != nil {
		if err := h.checkKubernetes(ctx); err != nil {
			details["kubernetes"] = fmt.Sprintf("unhealthy: %s", err.Error())
			allHealthy = false
			logger.Info("Kubernetes connectivity check failed", "error", err)
			discordAPIHealth.WithLabelValues("kubernetes").Set(0)
		} else {
			details["kubernetes"] = "healthy"
			discordAPIHealth.WithLabelValues("kubernetes").Set(1)
		}
	}

	// Check Discord API connectivity (if available)
	if h.discordCheck != nil {
		if err := h.discordCheck(ctx); err != nil {
			details["discord_api"] = fmt.Sprintf("unhealthy: %s", err.Error())
			allHealthy = false
			logger.Info("Discord API connectivity check failed", "error", err)
			discordAPIHealth.WithLabelValues("discord_api").Set(0)
		} else {
			details["discord_api"] = "healthy"
			discordAPIHealth.WithLabelValues("discord_api").Set(1)
		}
	}

	status := HealthStatus{
		Timestamp: time.Now(),
		Details:   details,
		Duration:  time.Since(start).String(),
	}

	w.Header().Set("Content-Type", "application/json")

	if allHealthy {
		status.Status = "ready"
		status.Message = "All components are healthy"
		w.WriteHeader(http.StatusOK)
		healthCheckRequests.WithLabelValues("readyz", "success").Inc()
	} else {
		status.Status = "not_ready"
		status.Message = "Some components are unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
		healthCheckRequests.WithLabelValues("readyz", "failure").Inc()
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		logger.Error(err, "failed to encode readiness status")
	}
}

// checkKubernetes verifies Kubernetes API connectivity
func (h *HealthChecker) checkKubernetes(ctx context.Context) error {
	// Try to get API resources as a basic connectivity test
	_, err := h.kubeClient.RESTMapper().RESTMappings(schema.GroupKind{Group: "", Kind: "Namespace"})
	if err != nil {
		return fmt.Errorf("kubernetes API not accessible: %w", err)
	}
	return nil
}

// CreateDiscordHealthCheck creates a health check function for Discord API
// If DISCORD_BOT_TOKEN environment variable is set, it will attempt to verify
// connectivity to the Discord API
func CreateDiscordHealthCheck() func(context.Context) error {
	return func(ctx context.Context) error {
		token := os.Getenv("DISCORD_BOT_TOKEN")
		if token == "" {
			// No token configured - skip Discord API check
			// This is expected in development or when credentials aren't available
			return nil
		}

		// Make a lightweight API call to verify connectivity
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequestWithContext(ctx, "GET",
			"https://discord.com/api/v11/users/@me", nil)
		if err != nil {
			return fmt.Errorf("failed to create Discord API request: %w", err)
		}
		req.Header.Set("Authorization", "Bot "+token)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("discord API unreachable: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode >= 500 {
			return fmt.Errorf("discord API returned server error: %d", resp.StatusCode)
		}
		if resp.StatusCode == 401 {
			return fmt.Errorf("discord API token is invalid")
		}

		return nil
	}
}

// SetupHealthChecks sets up health check endpoints
func SetupHealthChecks(mux *http.ServeMux, checker *HealthChecker) {
	mux.HandleFunc(HealthzPath, checker.ServeHealthz)
	mux.HandleFunc(ReadyzPath, checker.ServeReadyz)
}
