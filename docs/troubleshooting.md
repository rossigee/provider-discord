# Troubleshooting Guide

Comprehensive troubleshooting guide for provider-discord covering common issues, diagnostic procedures, and resolution steps.

## Quick Diagnostic Commands

### Provider Status
```bash
# Check provider installation
kubectl get providers

# Check provider logs
kubectl logs -n crossplane-system deployment/provider-discord -f

# Check provider configuration
kubectl get providerconfigs
kubectl describe providerconfig default

# Check managed resources
kubectl get guilds,channels,roles -A
```

### Health Checks
```bash
# Provider health endpoints
kubectl port-forward -n crossplane-system deployment/provider-discord 8081:8081
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz

# Metrics endpoint
kubectl port-forward -n crossplane-system deployment/provider-discord 8080:8080
curl http://localhost:8080/metrics
```

## Common Issues

### 1. Provider Not Starting

#### Symptoms
- Provider pod in `CrashLoopBackOff` state
- Provider not listed in `kubectl get providers`
- No provider logs available

#### Diagnostic Steps
```bash
# Check provider pod status
kubectl get pods -n crossplane-system -l pkg.crossplane.io/provider=provider-discord

# Check pod events
kubectl describe pod -n crossplane-system -l pkg.crossplane.io/provider=provider-discord

# Check provider package status
kubectl describe provider provider-discord

# Check Crossplane core logs
kubectl logs -n crossplane-system deployment/crossplane -c crossplane
```

#### Common Causes & Solutions

**Missing RBAC Permissions**
```bash
# Check RBAC
kubectl auth can-i create guilds --as=system:serviceaccount:crossplane-system:provider-discord

# Apply missing RBAC (if needed)
kubectl apply -f https://raw.githubusercontent.com/rossigee/provider-discord/master/cluster/rbac.yaml
```

**Image Pull Issues**
```yaml
# Check image pull policy and registry access
spec:
  package: ghcr.io/rossigee/provider-discord:latest
  packagePullPolicy: IfNotPresent
```

**Resource Constraints**
```yaml
# Check and adjust resource limits
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

### 2. Authentication Failures

#### Symptoms
- "Invalid token" errors in logs
- "Unauthorized" HTTP 401 responses
- Discord API requests failing

#### Diagnostic Steps
```bash
# Check secret exists and has correct format
kubectl get secret discord-creds -n crossplane-system -o yaml

# Verify secret key matches providerconfig
kubectl get providerconfig default -o yaml

# Test token manually (remove bot prefix if present)
curl -H "Authorization: Bot YOUR_TOKEN_HERE" https://discord.com/api/v10/users/@me
```

#### Solutions

**Invalid Token Format**
```bash
# Ensure token starts with bot ID (not "Bot " prefix)
# Correct format: MTAxNTYyNjA4MDQwNzU5OTIzNQ.GkQTzg.example
# Wrong format: Bot MTAxNTYyNjA4MDQwNzU5OTIzNQ.GkQTzg.example

# Update secret with correct token
kubectl create secret generic discord-creds \
  -n crossplane-system \
  --from-literal=token=YOUR_ACTUAL_TOKEN \
  --dry-run=client -o yaml | kubectl apply -f -
```

**Token Regenerated**
```bash
# Get new token from Discord Developer Portal
# Update secret immediately
kubectl patch secret discord-creds -n crossplane-system \
  -p '{"data":{"token":"'$(echo -n "NEW_TOKEN" | base64)'"}}'
```

**Wrong Secret Key**
```yaml
# Ensure providerconfig references correct key
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: discord-creds
      key: token  # Must match secret key
```

### 3. Permission Denied Errors

#### Symptoms
- "Missing Permissions" errors
- "Forbidden" HTTP 403 responses
- Cannot create/modify Discord resources

#### Diagnostic Steps
```bash
# Check bot permissions in Discord server
# - Go to Discord server settings
# - Check bot role permissions
# - Verify role hierarchy

# Check specific resource permissions
kubectl describe guild my-guild
kubectl get events --field-selector involvedObject.name=my-guild
```

#### Solutions

**Insufficient Bot Permissions**
1. **Discord Server Settings**:
   - Server Settings → Members → Bot Role
   - Ensure "Manage Server", "Manage Channels", "Manage Roles"
   - Check bot role position in hierarchy

2. **Re-invite Bot with Permissions**:
   ```
   https://discord.com/api/oauth2/authorize?client_id=YOUR_BOT_ID&permissions=268435472&scope=bot
   ```

**Role Hierarchy Issues**
- Bot role must be higher than roles it manages
- Cannot manage roles with higher position
- Move bot role up in server role hierarchy

### 4. Rate Limiting Issues

#### Symptoms
- "Rate limited" errors in logs
- Slow resource operations
- HTTP 429 responses

#### Diagnostic Steps
```bash
# Check rate limit metrics
curl -s http://localhost:8080/metrics | grep rate_limit

# Check circuit breaker status
kubectl logs -n crossplane-system deployment/provider-discord | grep "circuit breaker"

# Monitor rate limit headers in traces
# Access Jaeger UI to see detailed timing
```

#### Solutions

**High Rate Limit Usage**
```yaml
# Adjust rate limiting configuration
env:
- name: DISCORD_RATE_LIMIT_BACKOFF_MAX
  value: "60s"  # Increase max backoff
- name: DISCORD_CIRCUIT_BREAKER_ENABLED
  value: "true"  # Enable circuit breaker
```

**Too Many Concurrent Operations**
```yaml
# Reduce controller concurrency
args:
- --max-reconcile-rate=5
```

**Bot Shared Across Multiple Providers**
- Use separate bots for different environments
- Implement request queuing
- Monitor rate limit distribution

### 5. Resource Synchronization Issues

#### Symptoms
- Resources stuck in "Creating" state
- Manual Discord changes not reflected
- Resource drift not corrected

#### Diagnostic Steps
```bash
# Check resource status
kubectl describe guild my-guild

# Check reconciliation timing
kubectl get guild my-guild -o yaml | grep -A 5 -B 5 observedGeneration

# Check provider reconciliation metrics
curl -s http://localhost:8080/metrics | grep reconciliation
```

#### Solutions

**Stuck Resources**
```bash
# Force reconciliation
kubectl annotate guild my-guild crossplane.io/external-name=GUILD_ID

# Check for finalizers blocking deletion
kubectl get guild my-guild -o yaml | grep finalizers

# Remove stuck finalizers (last resort)
kubectl patch guild my-guild --type='merge' -p '{"metadata":{"finalizers":[]}}'
```

**Resource Drift**
```yaml
# Enable drift detection
spec:
  managementPolicy: Observe  # For read-only monitoring
  # or
  managementPolicy: ObserveCreateUpdate  # Skip deletion
```

**Long Reconciliation Times**
```yaml
# Adjust controller settings
env:
- name: RECONCILE_TIMEOUT
  value: "10m"
```

### 6. Network Connectivity Issues

#### Symptoms
- "Connection refused" errors
- "DNS resolution failed"
- Timeouts connecting to Discord API

#### Diagnostic Steps
```bash
# Test network connectivity from pod
kubectl exec -n crossplane-system deployment/provider-discord -- \
  curl -v https://discord.com/api/v10/gateway

# Check DNS resolution
kubectl exec -n crossplane-system deployment/provider-discord -- \
  nslookup discord.com

# Check network policies
kubectl get networkpolicy -n crossplane-system
```

#### Solutions

**DNS Issues**
```yaml
# Add DNS configuration
spec:
  template:
    spec:
      dnsPolicy: ClusterFirst
      dnsConfig:
        nameservers:
        - 8.8.8.8
```

**Network Policy Blocking**
```yaml
# Allow Discord API access
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: provider-discord-egress
spec:
  podSelector:
    matchLabels:
      pkg.crossplane.io/provider: provider-discord
  policyTypes:
  - Egress
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443
```

**Proxy Configuration**
```yaml
# Configure HTTP proxy
env:
- name: HTTP_PROXY
  value: "http://proxy.company.com:8080"
- name: HTTPS_PROXY
  value: "http://proxy.company.com:8080"
- name: NO_PROXY
  value: "kubernetes.default.svc,10.0.0.0/8"
```

### 7. Performance Issues

#### Symptoms
- Slow resource operations
- High memory/CPU usage
- Controller falling behind

#### Diagnostic Steps
```bash
# Check resource usage
kubectl top pod -n crossplane-system -l pkg.crossplane.io/provider=provider-discord

# Check performance metrics
curl -s http://localhost:8080/metrics | grep duration

# Check reconciliation queue depth
kubectl logs -n crossplane-system deployment/provider-discord | grep "queue depth"
```

#### Solutions

**Resource Optimization**
```yaml
# Increase resource limits
resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi
```

**Controller Tuning**
```yaml
# Adjust controller concurrency
args:
- --max-reconcile-rate=10
- --poll-interval=1m
- --sync-period=10m
```

**Enable Profiling**
```yaml
# Add profiling endpoint
args:
- --debug
ports:
- name: profiling
  containerPort: 6060
```

### 8. Monitoring and Observability Issues

#### Symptoms
- Missing metrics in Prometheus
- No traces in Jaeger
- Health checks failing

#### Diagnostic Steps
```bash
# Check metrics endpoint
curl http://localhost:8080/metrics

# Check health endpoints
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz

# Check service discovery
kubectl get servicemonitor -n crossplane-system
```

#### Solutions

**Metrics Not Scraped**
```yaml
# Ensure ServiceMonitor exists
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: provider-discord
spec:
  selector:
    matchLabels:
      app: provider-discord
  endpoints:
  - port: metrics
```

**Tracing Not Working**
```yaml
# Enable tracing
env:
- name: OTEL_TRACING_ENABLED
  value: "true"
- name: OTEL_EXPORTER_OTLP_ENDPOINT
  value: "http://jaeger-collector:14268/api/traces"
```

**Health Checks Failing**
```bash
# Check component health
curl -s http://localhost:8081/readyz | jq .

# Fix common health issues
kubectl rollout restart deployment/provider-discord -n crossplane-system
```

## Advanced Troubleshooting

### Debug Mode

Enable debug logging:
```yaml
env:
- name: LOG_LEVEL
  value: "debug"
- name: CONTROLLER_LOG_LEVEL
  value: "debug"
```

### Memory Profiling

```bash
# Enable pprof endpoint
kubectl port-forward -n crossplane-system deployment/provider-discord 6060:6060

# Get memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile
```

### Distributed Tracing Analysis

```bash
# Find slow operations
# In Jaeger UI: service:"provider-discord" AND duration:>5s

# Find error patterns
# In Jaeger UI: service:"provider-discord" AND error:true

# Analyze retry patterns
# Look for spans with discord.retry.attempt attribute
```

### Log Analysis

```bash
# Parse structured logs
kubectl logs -n crossplane-system deployment/provider-discord | jq 'select(.level=="error")'

# Filter by correlation ID
kubectl logs -n crossplane-system deployment/provider-discord | jq 'select(.correlation_id=="abc123")'

# Count error types
kubectl logs -n crossplane-system deployment/provider-discord | \
  jq -r 'select(.level=="error") | .msg' | sort | uniq -c
```

## Prevention Strategies

### Proactive Monitoring

1. **Set up alerts** for critical metrics
2. **Monitor trends** in performance data
3. **Regular health checks** automated testing
4. **Capacity planning** based on growth patterns

### Configuration Best Practices

1. **Resource limits** appropriate for workload
2. **Health checks** properly configured
3. **Secrets rotation** automated
4. **Network policies** restrictive but functional

### Operational Procedures

1. **Regular updates** of provider and dependencies
2. **Backup strategies** for configurations
3. **Incident response** procedures documented
4. **Performance baselines** established and monitored

## Getting Help

### Community Support
- **GitHub Issues**: https://github.com/rossigee/provider-discord/issues
- **Crossplane Slack**: `#providers` channel
- **Discussions**: GitHub repository discussions

### Enterprise Support
Contact enterprise support for:
- Production environment issues
- Performance optimization
- Custom integrations
- Training and consultation

### Reporting Bugs

Include the following information:
1. Provider version
2. Kubernetes version
3. Crossplane version
4. Complete error logs
5. Resource manifests (sanitized)
6. Steps to reproduce

### Performance Issues

For performance problems, provide:
1. Resource utilization metrics
2. Performance profile data
3. Workload characteristics
4. Scale and timing information
5. Environment specifications

This guide covers the most common issues. For specific problems not covered here, consult the community resources or file an issue with detailed information.