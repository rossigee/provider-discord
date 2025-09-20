package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rossigee/provider-discord/internal/metrics"
)

func main() {
	// Create metrics recorder
	recorder := metrics.NewMetricsRecorder()

	// Record some test metrics
	recorder.RecordAPIOperation("channel", "create", "success", 150*time.Millisecond)
	recorder.UpdateRateLimitStatus("channel", "/channels", 45, time.Now().Add(1*time.Hour))
	recorder.RecordAPIError("guild", "429", "rate_limited")

	// Start metrics server
	fmt.Println("Starting metrics server on :8080/metrics")
	fmt.Println("Expected metric names:")
	fmt.Println("- discord_api_operations_total")
	fmt.Println("- discord_api_operation_duration_seconds")
	fmt.Println("- discord_rate_limit_remaining")
	fmt.Println("- discord_rate_limit_reset_timestamp_seconds")
	fmt.Println("- discord_api_errors_total")

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8080", nil)
}
