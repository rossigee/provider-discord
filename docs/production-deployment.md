# Production Deployment Guide

This guide covers deploying provider-discord in production environments with enterprise-grade configurations.

## Prerequisites

- Kubernetes cluster (v1.27+)
- Crossplane installed (v1.19+)
- Helm 3.x
- kubectl configured
- Discord bot token (see [Discord Setup Guide](discord-setup.md))

## Deployment Architecture

### Recommended Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                      │
├─────────────────────────────────────────────────────────────┤
│  Namespace: crossplane-system                              │
│  ┌─────────────────┐  ┌─────────────────┐                │
│  │   Crossplane    │  │ Provider Discord│                │
│  │   Core          │  │                 │                │
│  │                 │  │ • Health Checks │                │
│  │ • CRDs          │  │ • Metrics       │                │
│  │ • Controllers   │  │ • Tracing       │                │
│  │ • Webhooks      │  │ • Rate Limiting │                │
│  └─────────────────┘  └─────────────────┘                │
├─────────────────────────────────────────────────────────────┤
│  Namespace: monitoring                                     │
│  ┌─────────────────┐  ┌─────────────────┐                │
│  │   Prometheus    │  │    Grafana      │                │
│  │                 │  │                 │                │
│  │ • Metrics       │  │ • Dashboards    │                │
│  │ • Alerting      │  │ • Visualization │                │
│  └─────────────────┘  └─────────────────┘                │
├─────────────────────────────────────────────────────────────┤
│  Namespace: jaeger-system                                  │
│  ┌─────────────────┐  ┌─────────────────┐                │
│  │     Jaeger      │  │  OpenTelemetry  │                │
│  │                 │  │   Collector     │                │
│  │ • Tracing       │  │                 │                │
│  │ • Performance   │  │ • Trace Export  │                │
│  └─────────────────┘  └─────────────────┘                │
└─────────────────────────────────────────────────────────────┘
```

## Production Installation

### Method 1: Helm Chart (Recommended)

```bash
# Add Crossplane Helm repository
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update

# Install Crossplane
helm install crossplane crossplane-stable/crossplane \
  --namespace crossplane-system \
  --create-namespace \
  --wait

# Install provider-discord with production configuration
kubectl apply -f https://raw.githubusercontent.com/rossigee/provider-discord/master/examples/provider-config.yaml
```

### Method 2: Direct Kubernetes Manifests

```bash
# Install provider
kubectl apply -f https://github.com/rossigee/provider-discord/releases/latest/download/provider.yaml

# Apply production configuration
kubectl apply -f - <<EOF
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-discord
spec:
  package: ghcr.io/rossigee/provider-discord:latest
  packagePullPolicy: IfNotPresent
  revisionActivationPolicy: Automatic
  revisionHistoryLimit: 3
  runtimeConfig:
    name: provider-discord-config
---
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: provider-discord-config
spec:
  deploymentTemplate:
    spec:
      replicas: 2
      selector:
        matchLabels:
          pkg.crossplane.io/provider: provider-discord
      template:
        metadata:
          labels:
            pkg.crossplane.io/provider: provider-discord
        spec:
          securityContext:
            runAsNonRoot: true
            runAsUser: 65532
            runAsGroup: 65532
            fsGroup: 65532
            seccompProfile:
              type: RuntimeDefault
          containers:
          - name: package-runtime
            securityContext:
              runAsNonRoot: true
              runAsUser: 65532
              runAsGroup: 65532
              allowPrivilegeEscalation: false
              readOnlyRootFilesystem: true
              capabilities:
                drop:
                - ALL
            resources:
              limits:
                cpu: 500m
                memory: 512Mi
              requests:
                cpu: 100m
                memory: 128Mi
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
            env:
            # Enterprise Features
            - name: DISCORD_METRICS_ENABLED
              value: "true"
            - name: DISCORD_CIRCUIT_BREAKER_ENABLED
              value: "true"
            - name: DISCORD_HEALTH_CHECK_INTERVAL
              value: "30s"
            # Observability
            - name: OTEL_TRACING_ENABLED
              value: "true"
            - name: OTEL_EXPORTER_OTLP_ENDPOINT
              value: "http://jaeger-collector.jaeger-system:14268/api/traces"
            - name: OTEL_SAMPLING_RATIO
              value: "0.1"
            # Performance
            - name: DISCORD_RATE_LIMIT_BACKOFF_MAX
              value: "30s"
            - name: DISCORD_RETRY_MAX_ATTEMPTS
              value: "3"
            ports:
            - name: metrics
              containerPort: 8080
            - name: health
              containerPort: 8081
EOF
```

## Security Configuration

### 1. Pod Security Standards

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: crossplane-system
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### 2. Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: provider-discord-netpol
  namespace: crossplane-system
spec:
  podSelector:
    matchLabels:
      pkg.crossplane.io/provider: provider-discord
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080
  - from: []
    ports:
    - protocol: TCP
      port: 8081
  egress:
  # Discord API
  - to: []
    ports:
    - protocol: TCP
      port: 443
  # Kubernetes API
  - to: []
    ports:
    - protocol: TCP
      port: 6443
  # DNS
  - to: []
    ports:
    - protocol: UDP
      port: 53
  # Jaeger (if tracing enabled)
  - to:
    - namespaceSelector:
        matchLabels:
          name: jaeger-system
    ports:
    - protocol: TCP
      port: 14268
```

### 3. RBAC Configuration

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: provider-discord-manager
rules:
# Core provider permissions
- apiGroups: [""]
  resources: ["secrets", "configmaps", "events"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "watch"]
# Discord resources
- apiGroups: ["discord.crossplane.io"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["guild.discord.crossplane.io"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["channel.discord.crossplane.io"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["role.discord.crossplane.io"]
  resources: ["*"]
  verbs: ["*"]
# Crossplane core
- apiGroups: ["pkg.crossplane.io"]
  resources: ["providerconfigs", "providerconfigusages"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: provider-discord-manager-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: provider-discord-manager
subjects:
- kind: ServiceAccount
  name: provider-discord
  namespace: crossplane-system
```

## Monitoring Setup

### 1. Prometheus ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: provider-discord
  namespace: crossplane-system
  labels:
    app: provider-discord
spec:
  selector:
    matchLabels:
      app: provider-discord
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
    honorLabels: true
---
apiVersion: v1
kind: Service
metadata:
  name: provider-discord-metrics
  namespace: crossplane-system
  labels:
    app: provider-discord
spec:
  selector:
    pkg.crossplane.io/provider: provider-discord
  ports:
  - name: metrics
    port: 8080
    targetPort: 8080
  - name: health
    port: 8081
    targetPort: 8081
```

### 2. AlertManager Rules

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: provider-discord-alerts
  namespace: crossplane-system
spec:
  groups:
  - name: provider-discord
    rules:
    - alert: ProviderDiscordDown
      expr: up{job="provider-discord-metrics"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Provider Discord is down"
        description: "Provider Discord has been down for more than 5 minutes"
    
    - alert: DiscordAPIHighErrorRate
      expr: rate(provider_discord_discord_api_errors_total[5m]) > 0.1
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: "High Discord API error rate"
        description: "Discord API error rate is {{ $value }} errors/second"
    
    - alert: DiscordRateLimitHit
      expr: rate(provider_discord_discord_rate_limits_total[5m]) > 0.05
      for: 1m
      labels:
        severity: warning
      annotations:
        summary: "Discord rate limits being hit"
        description: "Rate limit hits: {{ $value }}/second"
    
    - alert: ProviderDiscordHighMemory
      expr: container_memory_usage_bytes{pod=~"provider-discord-.*"} / container_spec_memory_limit_bytes > 0.8
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Provider Discord high memory usage"
        description: "Memory usage is above 80%"
```

## Observability Configuration

### 1. OpenTelemetry Collector

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-collector-config
  namespace: jaeger-system
data:
  otel-collector-config.yaml: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318
    
    processors:
      batch:
        timeout: 1s
        send_batch_size: 1024
        send_batch_max_size: 2048
      
      resource:
        attributes:
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
          processors: [batch, resource]
          exporters: [jaeger]
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otel-collector
  namespace: jaeger-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: otel-collector
  template:
    metadata:
      labels:
        app: otel-collector
    spec:
      containers:
      - name: otel-collector
        image: otel/opentelemetry-collector-contrib:latest
        args:
        - --config=/conf/otel-collector-config.yaml
        volumeMounts:
        - name: config
          mountPath: /conf
        ports:
        - containerPort: 4317
        - containerPort: 4318
        resources:
          limits:
            memory: 512Mi
            cpu: 500m
          requests:
            memory: 256Mi
            cpu: 100m
      volumes:
      - name: config
        configMap:
          name: otel-collector-config
```

### 2. Jaeger Installation

```bash
# Install Jaeger Operator
kubectl create namespace jaeger-system
kubectl apply -f https://github.com/jaegertracing/jaeger-operator/releases/latest/download/jaeger-operator.yaml -n jaeger-system

# Create Jaeger instance
kubectl apply -f - <<EOF
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: provider-discord-jaeger
  namespace: jaeger-system
spec:
  strategy: production
  storage:
    type: elasticsearch
    elasticsearch:
      nodeCount: 3
      redundancyPolicy: SingleRedundancy
      resources:
        requests:
          cpu: 200m
          memory: 1Gi
        limits:
          cpu: 1
          memory: 2Gi
  collector:
    replicas: 2
    resources:
      requests:
        cpu: 100m
        memory: 256Mi
      limits:
        cpu: 500m
        memory: 512Mi
  query:
    replicas: 2
    resources:
      requests:
        cpu: 100m
        memory: 256Mi
      limits:
        cpu: 500m
        memory: 512Mi
EOF
```

## High Availability Configuration

### 1. Multi-Replica Deployment

```yaml
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
```

### 2. Pod Disruption Budget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: provider-discord-pdb
  namespace: crossplane-system
spec:
  minAvailable: 2
  selector:
    matchLabels:
      pkg.crossplane.io/provider: provider-discord
```

### 3. Affinity Rules

```yaml
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  pkg.crossplane.io/provider: provider-discord
              topologyKey: kubernetes.io/hostname
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/control-plane
                operator: DoesNotExist
```

## Backup and Disaster Recovery

### 1. Configuration Backup

```bash
#!/bin/bash
# backup-discord-config.sh

BACKUP_DIR="/backups/discord-provider-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Backup provider configuration
kubectl get provider provider-discord -o yaml > "$BACKUP_DIR/provider.yaml"
kubectl get providerconfig -o yaml > "$BACKUP_DIR/providerconfigs.yaml"

# Backup Discord resources
kubectl get guilds -A -o yaml > "$BACKUP_DIR/guilds.yaml"
kubectl get channels -A -o yaml > "$BACKUP_DIR/channels.yaml"
kubectl get roles -A -o yaml > "$BACKUP_DIR/roles.yaml"

# Backup secrets (excluding sensitive data)
kubectl get secrets -n crossplane-system -l app=provider-discord -o yaml | \
  sed 's/data:.*/data: {}/' > "$BACKUP_DIR/secrets-metadata.yaml"

echo "Backup completed: $BACKUP_DIR"
```

### 2. Disaster Recovery Plan

1. **Preparation**
   - Regular automated backups
   - Documented recovery procedures
   - Tested restore processes

2. **Recovery Steps**
   - Restore Kubernetes cluster
   - Install Crossplane
   - Restore provider and configurations
   - Validate Discord connectivity
   - Reconcile managed resources

## Production Checklist

### Pre-Deployment
- [ ] Discord bot created and configured
- [ ] Kubernetes cluster prepared (RBAC, PSS, etc.)
- [ ] Monitoring stack deployed (Prometheus, Grafana)
- [ ] Tracing infrastructure ready (Jaeger, OTEL)
- [ ] Security policies applied
- [ ] Network policies configured
- [ ] Backup strategy implemented

### Post-Deployment
- [ ] Provider health checks passing
- [ ] Metrics being collected
- [ ] Alerts configured and tested
- [ ] Traces visible in Jaeger
- [ ] Discord API connectivity verified
- [ ] Test resource creation/deletion
- [ ] Performance baseline established
- [ ] Documentation updated

### Ongoing Operations
- [ ] Regular security updates
- [ ] Monitor resource utilization
- [ ] Review and rotate credentials
- [ ] Performance optimization
- [ ] Capacity planning
- [ ] Incident response testing

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md) for detailed problem resolution steps.

## Performance Tuning

### Resource Optimization
- CPU requests: 100m (minimum), 500m (recommended)
- Memory requests: 128Mi (minimum), 512Mi (recommended)
- Adjust based on Discord server size and activity

### Discord API Optimization
- Rate limiting: Built-in exponential backoff
- Circuit breakers: Automatic failure protection
- Connection pooling: HTTP/2 multiplexing

### Monitoring Metrics
- `provider_discord_discord_api_operations_total`
- `provider_discord_discord_rate_limits_total`
- `provider_discord_managed_resources`
- `provider_discord_health_check_requests_total`

## Compliance and Security

### Security Standards
- CIS Kubernetes Benchmark compliance
- Pod Security Standards (Restricted)
- Network segmentation
- Encryption in transit (TLS)
- Secrets management best practices

### Audit Logging
- Kubernetes audit logs enabled
- Discord API audit logs monitored
- Provider operation logs retained
- Access logs for compliance

For detailed compliance requirements, see [Security Policy](../SECURITY.md).