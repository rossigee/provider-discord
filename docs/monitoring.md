# Monitoring and Observability Guide

Complete guide for monitoring provider-discord in production environments with Prometheus, Grafana, and OpenTelemetry.

## Overview

Provider-discord includes comprehensive observability features:
- **Prometheus Metrics**: 10+ custom metrics for operations and performance
- **Health Endpoints**: Kubernetes-native health checking
- **OpenTelemetry Tracing**: Distributed tracing with correlation IDs
- **Structured Logging**: JSON logging with contextual information

## Metrics Reference

### Core Provider Metrics

#### API Operations
```
provider_discord_discord_api_operations_total{resource_type, operation, status}
```
- **Type**: Counter
- **Description**: Total Discord API operations
- **Labels**:
  - `resource_type`: guild, channel, role
  - `operation`: create, update, delete, observe
  - `status`: success, error, rate_limited

#### API Operation Duration
```
provider_discord_discord_api_operation_duration_seconds{resource_type, operation}
```
- **Type**: Histogram
- **Description**: Duration of Discord API operations
- **Buckets**: 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0

#### Rate Limiting
```
provider_discord_discord_rate_limits_total{resource_type, endpoint}
provider_discord_discord_rate_limit_remaining{resource_type, endpoint}
provider_discord_discord_rate_limit_reset_time{resource_type, endpoint}
```
- **Type**: Counter, Gauge, Gauge
- **Description**: Discord API rate limit tracking
- **Usage**: Monitor rate limit consumption and reset times

#### Managed Resources
```
provider_discord_managed_resources{resource_type, status}
```
- **Type**: Gauge
- **Description**: Number of managed Discord resources
- **Labels**:
  - `resource_type`: guild, channel, role
  - `status`: ready, creating, updating, deleting, error

#### Reconciliation Metrics
```
provider_discord_resource_reconciliations_total{resource_type, status}
provider_discord_resource_reconciliation_duration_seconds{resource_type}
```
- **Type**: Counter, Histogram
- **Description**: Resource reconciliation operations and timing

#### Error Tracking
```
provider_discord_discord_api_errors_total{resource_type, status_code, error_type}
```
- **Type**: Counter
- **Description**: Discord API errors by type and status code
- **Labels**:
  - `error_type`: rate_limit, auth_error, not_found, server_error, network_error

#### Health Metrics
```
provider_discord_health_check_requests_total{endpoint, status}
provider_discord_health_check_duration_seconds{endpoint}
provider_discord_discord_api_health{component}
provider_discord_provider_health{component}
```
- **Type**: Counter, Histogram, Gauge, Gauge
- **Description**: Health endpoint metrics and component status

## Prometheus Configuration

### Scrape Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "provider-discord-alerts.yml"

scrape_configs:
- job_name: 'provider-discord'
  kubernetes_sd_configs:
  - role: endpoints
    namespaces:
      names:
      - crossplane-system
  relabel_configs:
  - source_labels: [__meta_kubernetes_service_name]
    action: keep
    regex: provider-discord-metrics
  - source_labels: [__meta_kubernetes_endpoint_port_name]
    action: keep
    regex: metrics
  - source_labels: [__meta_kubernetes_namespace]
    target_label: kubernetes_namespace
  - source_labels: [__meta_kubernetes_service_name]
    target_label: kubernetes_name
  - source_labels: [__meta_kubernetes_pod_name]
    target_label: kubernetes_pod_name
```

### ServiceMonitor (Prometheus Operator)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: provider-discord
  namespace: crossplane-system
  labels:
    app.kubernetes.io/name: provider-discord
    app.kubernetes.io/component: metrics
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: provider-discord
      app.kubernetes.io/component: metrics
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
    honorLabels: true
    scrapeTimeout: 10s
    metricRelabelings:
    - sourceLabels: [__name__]
      regex: 'go_.*|process_.*|promhttp_.*'
      action: drop
  - port: health
    interval: 30s
    path: /metrics
    honorLabels: true
    scrapeTimeout: 5s
```

## Alerting Rules

### Critical Alerts

```yaml
# provider-discord-alerts.yml
groups:
- name: provider-discord.critical
  rules:
  - alert: ProviderDiscordDown
    expr: up{job="provider-discord"} == 0
    for: 2m
    labels:
      severity: critical
      component: provider-discord
    annotations:
      summary: "Provider Discord is down"
      description: "Provider Discord has been down for more than 2 minutes. No Discord resources can be managed."
      runbook_url: "https://github.com/rossigee/provider-discord/blob/master/docs/troubleshooting.md#provider-down"

  - alert: DiscordAPIUnreachable
    expr: provider_discord_discord_api_health{component="discord_api"} == 0
    for: 5m
    labels:
      severity: critical
      component: discord-api
    annotations:
      summary: "Discord API unreachable"
      description: "Discord API has been unreachable for {{ $labels.component }} for more than 5 minutes."
      runbook_url: "https://github.com/rossigee/provider-discord/blob/master/docs/troubleshooting.md#discord-api-unreachable"

- name: provider-discord.warning
  rules:
  - alert: DiscordAPIHighErrorRate
    expr: |
      (
        rate(provider_discord_discord_api_errors_total[5m]) 
        / 
        rate(provider_discord_discord_api_operations_total[5m])
      ) > 0.1
    for: 3m
    labels:
      severity: warning
      component: discord-api
    annotations:
      summary: "High Discord API error rate"
      description: "Discord API error rate is {{ $value | humanizePercentage }} for the last 5 minutes."

  - alert: DiscordRateLimitHighUsage
    expr: rate(provider_discord_discord_rate_limits_total[5m]) > 0.05
    for: 2m
    labels:
      severity: warning
      component: discord-api
    annotations:
      summary: "High Discord rate limit usage"
      description: "Discord rate limits are being hit {{ $value | humanize }} times per second."

  - alert: ProviderDiscordHighMemoryUsage
    expr: |
      (
        container_memory_working_set_bytes{pod=~"provider-discord-.*",container="package-runtime"} 
        / 
        container_spec_memory_limit_bytes{pod=~"provider-discord-.*",container="package-runtime"}
      ) > 0.8
    for: 5m
    labels:
      severity: warning
      component: provider-discord
    annotations:
      summary: "Provider Discord high memory usage"
      description: "Provider Discord memory usage is {{ $value | humanizePercentage }} of limit."

  - alert: ProviderDiscordSlowReconciliation
    expr: |
      histogram_quantile(0.95, 
        rate(provider_discord_resource_reconciliation_duration_seconds_bucket[5m])
      ) > 30
    for: 5m
    labels:
      severity: warning
      component: provider-discord
    annotations:
      summary: "Slow resource reconciliation"
      description: "95th percentile reconciliation time is {{ $value | humanizeDuration }}."

- name: provider-discord.info
  rules:
  - alert: DiscordResourceCount
    expr: sum(provider_discord_managed_resources) > 100
    for: 0m
    labels:
      severity: info
      component: provider-discord
    annotations:
      summary: "High number of managed Discord resources"
      description: "Managing {{ $value }} Discord resources across all types."
```

### Alertmanager Configuration

```yaml
# alertmanager.yml
global:
  slack_api_url: 'YOUR_SLACK_WEBHOOK_URL'

route:
  group_by: ['alertname', 'component']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'
  routes:
  - match:
      severity: critical
    receiver: 'critical-alerts'
  - match:
      component: provider-discord
    receiver: 'discord-alerts'

receivers:
- name: 'default'
  slack_configs:
  - channel: '#alerts'
    title: 'Alert: {{ .GroupLabels.alertname }}'
    text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

- name: 'critical-alerts'
  slack_configs:
  - channel: '#critical-alerts'
    title: '🚨 CRITICAL: {{ .GroupLabels.alertname }}'
    text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
    actions:
    - type: button
      text: 'Runbook'
      url: '{{ (index .Alerts 0).Annotations.runbook_url }}'

- name: 'discord-alerts'
  slack_configs:
  - channel: '#discord-provider'
    title: 'Discord Provider: {{ .GroupLabels.alertname }}'
    text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

## Grafana Dashboards

### Main Dashboard JSON

```json
{
  "dashboard": {
    "id": null,
    "title": "Provider Discord - Overview",
    "tags": ["crossplane", "discord", "provider"],
    "timezone": "browser",
    "refresh": "30s",
    "time": {
      "from": "now-1h",
      "to": "now"
    },
    "panels": [
      {
        "title": "Provider Status",
        "type": "stat",
        "targets": [
          {
            "expr": "up{job=\"provider-discord\"}",
            "legendFormat": "Provider Up"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "thresholds"
            },
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "green", "value": 1}
              ]
            }
          }
        }
      },
      {
        "title": "API Operations Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(provider_discord_discord_api_operations_total[5m])",
            "legendFormat": "{{operation}} {{status}}"
          }
        ],
        "yAxes": [
          {
            "label": "Operations/sec",
            "min": 0
          }
        ]
      },
      {
        "title": "API Response Times",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, rate(provider_discord_discord_api_operation_duration_seconds_bucket[5m]))",
            "legendFormat": "50th percentile"
          },
          {
            "expr": "histogram_quantile(0.95, rate(provider_discord_discord_api_operation_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.99, rate(provider_discord_discord_api_operation_duration_seconds_bucket[5m]))",
            "legendFormat": "99th percentile"
          }
        ],
        "yAxes": [
          {
            "label": "Duration (seconds)",
            "min": 0
          }
        ]
      },
      {
        "title": "Managed Resources",
        "type": "stat",
        "targets": [
          {
            "expr": "sum by (resource_type) (provider_discord_managed_resources)",
            "legendFormat": "{{resource_type}}"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(provider_discord_discord_api_errors_total[5m])",
            "legendFormat": "{{error_type}} ({{status_code}})"
          }
        ],
        "yAxes": [
          {
            "label": "Errors/sec",
            "min": 0
          }
        ]
      },
      {
        "title": "Rate Limit Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "provider_discord_discord_rate_limit_remaining",
            "legendFormat": "{{endpoint}} remaining"
          }
        ],
        "yAxes": [
          {
            "label": "Requests remaining",
            "min": 0
          }
        ]
      }
    ]
  }
}
```

### Resource-Specific Dashboards

Create separate dashboards for detailed resource monitoring:
- **Guilds Dashboard**: Guild creation, modification, member counts
- **Channels Dashboard**: Channel operations, message rates, permissions
- **Roles Dashboard**: Role assignments, permission changes, hierarchy
- **Performance Dashboard**: Detailed timing, memory usage, reconciliation

## OpenTelemetry Tracing

### Trace Collection

Provider-discord emits OpenTelemetry traces for:
- Resource reconciliation operations
- Discord API calls with timing
- Error scenarios with context
- Rate limiting and retry logic

### Trace Attributes

#### Standard Attributes
- `service.name`: provider-discord
- `service.version`: provider version
- `resource.type`: guild, channel, role
- `operation.name`: create, update, delete, observe

#### Discord-Specific Attributes
- `discord.resource.id`: Discord resource ID
- `discord.resource.name`: Human-readable name
- `discord.guild.id`: Guild ID for context
- `discord.api.endpoint`: API endpoint called
- `discord.http.method`: HTTP method
- `discord.http.status_code`: Response status
- `discord.rate_limited`: Whether rate limited
- `discord.retry.attempt`: Retry attempt number

### Jaeger Configuration

```yaml
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: provider-discord-jaeger
  namespace: jaeger-system
spec:
  strategy: production
  collector:
    config: |
      receivers:
        otlp:
          protocols:
            grpc:
              endpoint: 0.0.0.0:14250
            http:
              endpoint: 0.0.0.0:14268
      processors:
        batch:
          timeout: 1s
          send_batch_size: 1024
        attributes:
          actions:
          - key: environment
            value: production
            action: upsert
      exporters:
        jaeger:
          endpoint: jaeger-collector:14250
          tls:
            insecure: true
      service:
        pipelines:
          traces:
            receivers: [otlp]
            processors: [batch, attributes]
            exporters: [jaeger]
```

### Trace Analysis Queries

#### Common Queries
```
# All Discord operations
service:"provider-discord"

# Failed operations
service:"provider-discord" AND error:true

# Slow operations (>1s)
service:"provider-discord" AND duration:>1s

# Rate limited operations
service:"provider-discord" AND discord.rate_limited:true

# Guild operations
service:"provider-discord" AND resource.type:"guild"

# API errors by endpoint
service:"provider-discord" AND discord.http.status_code:>=400
```

## Health Monitoring

### Health Endpoints

#### /healthz (Liveness)
- **Purpose**: Basic liveness check
- **Response**: Simple health status
- **Use**: Kubernetes liveness probe

#### /readyz (Readiness)
- **Purpose**: Comprehensive readiness check
- **Checks**:
  - Discord API connectivity
  - Kubernetes API access
  - Internal component health
- **Use**: Kubernetes readiness probe

### Health Check Configuration

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8081
  initialDelaySeconds: 30
  periodSeconds: 30
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /readyz
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

## Log Analysis

### Structured Logging

Provider-discord uses structured JSON logging:

```json
{
  "level": "info",
  "ts": "2023-12-07T10:30:45.123Z",
  "logger": "controllers.guild",
  "msg": "reconciling guild",
  "guild": "my-server",
  "operation": "update",
  "correlation_id": "abc123",
  "trace_id": "def456"
}
```

### Log Aggregation

#### Fluent Bit Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
  namespace: logging
data:
  fluent-bit.conf: |
    [SERVICE]
        Flush         1
        Log_Level     info
        Daemon        off
        Parsers_File  parsers.conf

    [INPUT]
        Name              tail
        Path              /var/log/containers/provider-discord-*.log
        Parser            docker
        Tag               provider.discord.*
        Refresh_Interval  5

    [FILTER]
        Name   parser
        Match  provider.discord.*
        Key_Name log
        Parser json
        Reserve_Data On

    [OUTPUT]
        Name  es
        Match provider.discord.*
        Host  elasticsearch.logging.svc.cluster.local
        Port  9200
        Index provider-discord-logs
```

### Log Queries

#### Elasticsearch/Kibana Queries
```
# All provider logs
kubernetes.labels.app:"provider-discord"

# Error logs only
kubernetes.labels.app:"provider-discord" AND level:"error"

# Specific resource operations
kubernetes.labels.app:"provider-discord" AND guild:"my-server"

# Discord API errors
kubernetes.labels.app:"provider-discord" AND msg:"Discord API error"

# Rate limit events
kubernetes.labels.app:"provider-discord" AND msg:"rate limited"
```

## Performance Monitoring

### Key Performance Indicators (KPIs)

1. **Availability**: >99.9% uptime
2. **API Response Time**: <200ms average
3. **Reconciliation Time**: <30s for 95th percentile
4. **Error Rate**: <1% of operations
5. **Rate Limit Usage**: <80% of available requests

### Performance Dashboards

Create dedicated performance dashboards tracking:
- Resource utilization (CPU, memory, network)
- Operation latencies and throughput
- Queue depths and processing rates
- Discord API performance metrics
- Kubernetes cluster health impacts

### Capacity Planning

Monitor trends for:
- Resource growth rates
- Performance degradation patterns
- Rate limit consumption trends
- Error rate correlations
- Infrastructure scaling needs

## Troubleshooting with Monitoring

### Common Scenarios

1. **High Error Rate**
   - Check error type distribution
   - Correlate with Discord API status
   - Review trace details for patterns

2. **Slow Performance**
   - Analyze latency percentiles
   - Check resource utilization
   - Review Discord rate limiting

3. **Resource Drift**
   - Monitor reconciliation metrics
   - Check error patterns
   - Trace specific resource operations

For detailed troubleshooting procedures, see [Troubleshooting Guide](troubleshooting.md).

## Integration with External Systems

### Datadog Integration

```yaml
# DataDog annotations for auto-discovery
metadata:
  annotations:
    ad.datadoghq.com/provider-discord.check_names: '["prometheus"]'
    ad.datadoghq.com/provider-discord.init_configs: '[{}]'
    ad.datadoghq.com/provider-discord.instances: |
      [
        {
          "prometheus_url": "http://%%host%%:8080/metrics",
          "namespace": "provider_discord",
          "metrics": ["*"]
        }
      ]
```

### New Relic Integration

```yaml
# New Relic monitoring
env:
- name: NEW_RELIC_LICENSE_KEY
  valueFrom:
    secretKeyRef:
      name: newrelic-license
      key: license
- name: NEW_RELIC_APP_NAME
  value: "provider-discord"
```

This comprehensive monitoring setup provides complete visibility into provider-discord operations, enabling proactive issue detection and resolution.