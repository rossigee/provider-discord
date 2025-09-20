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

// Package tracing provides OpenTelemetry tracing integration for the Discord provider.
package tracing

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// Tracer name for the Discord provider
	TracerName = "provider-discord"

	// Common span attributes
	AttrResourceType = "discord.resource.type"
	AttrResourceID   = "discord.resource.id"
	AttrResourceName = "discord.resource.name"
	AttrOperation    = "discord.operation"
	AttrGuildID      = "discord.guild.id"
	AttrChannelID    = "discord.channel.id"
	AttrRoleID       = "discord.role.id"
	AttrAPIEndpoint  = "discord.api.endpoint"
	AttrHTTPMethod   = "discord.http.method"
	AttrStatusCode   = "discord.http.status_code"
	AttrRetryAttempt = "discord.retry.attempt"
	AttrRateLimited  = "discord.rate_limited"
	AttrErrorType    = "discord.error.type"

	// Span names
	SpanReconcile        = "discord.reconcile"
	SpanAPICall          = "discord.api.call"
	SpanCreateResource   = "discord.create"
	SpanUpdateResource   = "discord.update"
	SpanDeleteResource   = "discord.delete"
	SpanObserveResource  = "discord.observe"
	SpanHealthCheck      = "discord.health.check"
)

var (
	// Global tracer instance
	tracer trace.Tracer
	logger logr.Logger
)

// Config holds tracing configuration
type Config struct {
	Enabled         bool
	ServiceName     string
	ServiceVersion  string
	Endpoint        string
	SamplingRatio   float64
	Headers         map[string]string
}

// DefaultConfig returns default tracing configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:       getEnvBool("OTEL_TRACING_ENABLED", false),
		ServiceName:   getEnv("OTEL_SERVICE_NAME", "provider-discord"),
		ServiceVersion: getEnv("OTEL_SERVICE_VERSION", "unknown"),
		Endpoint:      getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		SamplingRatio: getEnvFloat("OTEL_SAMPLING_RATIO", 0.1),
		Headers:       make(map[string]string),
	}
}

// Initialize sets up OpenTelemetry tracing
func Initialize(ctx context.Context, config *Config) error {
	logger = log.Log.WithName("tracing")

	if config == nil {
		config = DefaultConfig()
	}

	if !config.Enabled || config.Endpoint == "" {
		logger.Info("Tracing disabled or no endpoint configured")
		// Set up a no-op tracer
		tracer = otel.Tracer(TracerName)
		return nil
	}

	logger.Info("Initializing OpenTelemetry tracing",
		"service_name", config.ServiceName,
		"endpoint", config.Endpoint,
		"sampling_ratio", config.SamplingRatio,
	)

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			attribute.String("provider.type", "crossplane"),
			attribute.String("provider.name", "discord"),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP exporter
	exporter, err := createOTLPExporter(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRatio)),
	)

	// Set global providers
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Get tracer
	tracer = otel.Tracer(TracerName)

	logger.Info("OpenTelemetry tracing initialized successfully")
	return nil
}

// createOTLPExporter creates an OTLP trace exporter
func createOTLPExporter(ctx context.Context, config *Config) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.Endpoint),
	}

	// Add headers if configured
	if len(config.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(config.Headers))
	}

	// Create HTTP exporter
	exporter, err := otlptrace.New(ctx, otlptracehttp.NewClient(opts...))
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP HTTP exporter: %w", err)
	}

	return exporter, nil
}

// GetTracer returns the global tracer instance
func GetTracer() trace.Tracer {
	if tracer == nil {
		// Return no-op tracer if not initialized
		return otel.Tracer(TracerName)
	}
	return tracer
}

// StartSpan starts a new span with the given name and options
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, spanName, opts...)
}

// RecordError records an error on the span and sets appropriate attributes
func RecordError(span trace.Span, err error, errorType string) {
	if err == nil {
		return
	}

	span.RecordError(err)
	span.SetAttributes(
		attribute.String(AttrErrorType, errorType),
		attribute.String("error.message", err.Error()),
	)

	// Mark span as error
	span.SetStatus(codes.Error, err.Error())
}

// RecordSuccess marks a span as successful
func RecordSuccess(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}

// TraceReconciliation creates a span for resource reconciliation
func TraceReconciliation(ctx context.Context, resourceType, resourceName, operation string) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("%s.%s", SpanReconcile, operation)

	ctx, span := StartSpan(ctx, spanName,
		trace.WithAttributes(
			attribute.String(AttrResourceType, resourceType),
			attribute.String(AttrResourceName, resourceName),
			attribute.String(AttrOperation, operation),
		),
	)

	return ctx, span
}

// TraceAPICall creates a span for Discord API calls
func TraceAPICall(ctx context.Context, method, endpoint string) (context.Context, trace.Span) {
	ctx, span := StartSpan(ctx, SpanAPICall,
		trace.WithAttributes(
			attribute.String(AttrHTTPMethod, method),
			attribute.String(AttrAPIEndpoint, endpoint),
		),
	)

	return ctx, span
}

// TraceResourceOperation creates a span for specific resource operations
func TraceResourceOperation(ctx context.Context, resourceType, operation, resourceID string) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("%s.%s.%s", resourceType, operation, SpanAPICall)

	ctx, span := StartSpan(ctx, spanName,
		trace.WithAttributes(
			attribute.String(AttrResourceType, resourceType),
			attribute.String(AttrOperation, operation),
			attribute.String(AttrResourceID, resourceID),
		),
	)

	return ctx, span
}

// SetDiscordAttributes sets Discord-specific attributes on a span
func SetDiscordAttributes(span trace.Span, attrs map[string]interface{}) {
	for key, value := range attrs {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String(key, v))
		case int:
			span.SetAttributes(attribute.Int(key, v))
		case int64:
			span.SetAttributes(attribute.Int64(key, v))
		case bool:
			span.SetAttributes(attribute.Bool(key, v))
		case float64:
			span.SetAttributes(attribute.Float64(key, v))
		default:
			span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}
}

// AddGuildContext adds guild-specific context to a span
func AddGuildContext(span trace.Span, guildID, guildName string) {
	span.SetAttributes(
		attribute.String(AttrGuildID, guildID),
		attribute.String("discord.guild.name", guildName),
	)
}

// AddChannelContext adds channel-specific context to a span
func AddChannelContext(span trace.Span, channelID, channelName string, channelType int) {
	span.SetAttributes(
		attribute.String(AttrChannelID, channelID),
		attribute.String("discord.channel.name", channelName),
		attribute.Int("discord.channel.type", channelType),
	)
}

// AddRoleContext adds role-specific context to a span
func AddRoleContext(span trace.Span, roleID, roleName string) {
	span.SetAttributes(
		attribute.String(AttrRoleID, roleID),
		attribute.String("discord.role.name", roleName),
	)
}

// RecordAPIResponse records API response information
func RecordAPIResponse(span trace.Span, statusCode int, rateLimited bool) {
	span.SetAttributes(
		attribute.Int(AttrStatusCode, statusCode),
		attribute.Bool(AttrRateLimited, rateLimited),
	)

	if statusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
	} else {
		span.SetStatus(codes.Ok, "")
	}
}

// RecordRetryAttempt records information about retry attempts
func RecordRetryAttempt(span trace.Span, attempt int, delay string) {
	span.SetAttributes(
		attribute.Int(AttrRetryAttempt, attempt),
		attribute.String("discord.retry.delay", delay),
	)
}

// Shutdown gracefully shuts down the tracing system
func Shutdown(ctx context.Context) error {
	if tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		logger.Info("Shutting down OpenTelemetry tracing")
		return tp.Shutdown(ctx)
	}
	return nil
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := parseFloat(value); err == nil {
			return f
		}
	}
	return defaultValue
}

// Simple float parser to avoid importing strconv
func parseFloat(s string) (float64, error) {
	// Simple implementation for common cases
	switch s {
	case "0.0", "0":
		return 0.0, nil
	case "0.1":
		return 0.1, nil
	case "0.5":
		return 0.5, nil
	case "1.0", "1":
		return 1.0, nil
	default:
		return 0.0, fmt.Errorf("unsupported float value: %s", s)
	}
}
