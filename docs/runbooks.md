# Operational Runbooks

## Overview

This document contains operational runbooks for managing and troubleshooting the Crossplane Provider Discord in production environments.

## Table of Contents

- [Provider Down](#provider-down)
- [High Error Rate](#high-error-rate)
- [API Health Check Failing](#api-health-check-failing)
- [High Latency](#high-latency)
- [Circuit Breaker Open](#circuit-breaker-open)
- [Reconciliation Stuck](#reconciliation-stuck)
- [High Memory Usage](#high-memory-usage)
- [Rate Limit Approaching](#rate-limit-approaching)
- [Security Incident Response](#security-incident-response)
- [Disaster Recovery](#disaster-recovery)

---

## Provider Down

### Symptoms
- AlertManager: `ProviderDiscordDown`
- No metrics from provider
- Discord resources not reconciling
- Provider pod not responding

### Investigation Steps

1. **Check Pod Status**
   ```bash
   kubectl get pods -n crossplane-system -l app=provider-discord
   kubectl describe pod -n crossplane-system <provider-pod-name>
   ```

2. **Check Pod Logs**
   ```bash
   kubectl logs -n crossplane-system <provider-pod-name> --previous
   kubectl logs -n crossplane-system <provider-pod-name> -f
   ```

3. **Check Resource Usage**
   ```bash
   kubectl top pod -n crossplane-system <provider-pod-name>
   ```

4. **Check Node Resources**
   ```bash
   kubectl describe node <node-name>
   kubectl get events --sort-by=.lastTimestamp
   ```

### Resolution Steps

1. **If Pod is OOMKilled**
   ```bash
   # Increase memory limits
   kubectl patch deployment -n crossplane-system provider-discord \
     -p '{"spec":{"template":{"spec":{"containers":[{"name":"provider","resources":{"limits":{"memory":"512Mi"}}}]}}}}'
   ```

2. **If Pod is CrashLooping**
   ```bash
   # Check configuration
   kubectl get providerconfig
   kubectl describe providerconfig <config-name>

   # Check secrets
   kubectl get secret -n crossplane-system discord-credentials
   kubectl describe secret -n crossplane-system discord-credentials
   ```

3. **If Image Pull Issues**
   ```bash
   # Check image availability
   docker pull ghcr.io/rossigee/provider-discord:v0.2.0

   # Update deployment if needed
   kubectl set image deployment/provider-discord -n crossplane-system \
     provider=ghcr.io/rossigee/provider-discord:v0.2.0
   ```

4. **Force Pod Restart**
   ```bash
   kubectl delete pod -n crossplane-system <provider-pod-name>
   ```

### Escalation
If provider continues to fail after above steps:
- Contact: Platform Team
- Escalate to: SRE Team after 15 minutes
- Emergency contact: On-call engineer

---

## High Error Rate

### Symptoms
- AlertManager: `ProviderDiscordHighErrorRate`
- Discord API returning errors
- Resources failing to reconcile
- Increased retry attempts

### Investigation Steps

1. **Check Error Metrics**
   ```bash
   # Using Prometheus/Grafana
   sum(rate(provider_discord_api_operations_total{status!="success"}[5m])) by (status, resource_type)
   ```

2. **Check Recent Errors in Logs**
   ```bash
   kubectl logs -n crossplane-system deployment/provider-discord | grep -i error | tail -20
   ```

3. **Check Discord API Status**
   ```bash
   curl -H "Authorization: Bot $DISCORD_BOT_TOKEN" https://discord.com/api/v10/users/@me
   ```

4. **Check Resource States**
   ```bash
   kubectl get guilds,channels,roles -A -o wide
   ```

### Resolution Steps

1. **If Authentication Errors**
   ```bash
   # Verify bot token
   kubectl get secret -n crossplane-system discord-credentials -o yaml

   # Test token validity
   curl -H "Authorization: Bot $DISCORD_BOT_TOKEN" \
        https://discord.com/api/v10/users/@me

   # Update token if needed
   kubectl create secret generic discord-credentials \
     --from-literal=token=$NEW_BOT_TOKEN \
     --dry-run=client -o yaml | kubectl apply -f -
   ```

2. **If Rate Limiting Errors**
   ```bash
   # Check current request rate
   sum(rate(provider_discord_api_operations_total[1m])) * 60

   # Implement backoff (restart provider to reset state)
   kubectl rollout restart deployment/provider-discord -n crossplane-system
   ```

3. **If Permission Errors**
   ```bash
   # Check bot permissions in Discord
   # Navigate to Discord Developer Portal
   # Verify bot has required permissions:
   # - Manage Channels
   # - Manage Roles
   # - View Channels
   ```

4. **If Network Errors**
   ```bash
   # Test connectivity from pod
   kubectl exec -n crossplane-system deployment/provider-discord -- \
     curl -s https://discord.com/api/v10

   # Check network policies
   kubectl get networkpolicy -n crossplane-system
   ```

### Escalation
- High error rate > 20%: Immediate escalation to Platform Team
- Authentication failures: Contact Security Team
- Sustained errors > 30 minutes: Escalate to on-call

---

## API Health Check Failing

### Symptoms
- AlertManager: `ProviderDiscordAPIUnhealthy`
- Health endpoint returning 503
- Discord API connectivity issues

### Investigation Steps

1. **Check Health Endpoint**
   ```bash
   kubectl port-forward -n crossplane-system deployment/provider-discord 8080:8080
   curl http://localhost:8080/healthz
   curl http://localhost:8080/readyz
   ```

2. **Check Discord API Connectivity**
   ```bash
   kubectl exec -n crossplane-system deployment/provider-discord -- \
     curl -v https://discord.com/api/v10
   ```

3. **Check Network Configuration**
   ```bash
   kubectl get networkpolicy -n crossplane-system
   kubectl describe pod -n crossplane-system deployment/provider-discord
   ```

### Resolution Steps

1. **If DNS Issues**
   ```bash
   kubectl exec -n crossplane-system deployment/provider-discord -- nslookup discord.com

   # Check CoreDNS
   kubectl get pods -n kube-system -l k8s-app=kube-dns
   ```

2. **If Network Policy Blocking**
   ```bash
   # Temporarily remove network policies for testing
   kubectl delete networkpolicy -n crossplane-system provider-discord-netpol

   # Test connectivity
   curl http://localhost:8080/healthz

   # Restore network policy with correct rules
   ```

3. **If Certificate Issues**
   ```bash
   # Check TLS connectivity
   kubectl exec -n crossplane-system deployment/provider-discord -- \
     openssl s_client -connect discord.com:443 -verify_return_error
   ```

### Escalation
- DNS/Network issues: Platform Team
- Certificate issues: Security Team
- Persistent failures > 10 minutes: On-call engineer

---

## High Latency

### Symptoms
- AlertManager: `ProviderDiscordHighLatency`
- Slow API responses
- Timeouts in logs
- Poor user experience

### Investigation Steps

1. **Check Latency Metrics**
   ```bash
   # 95th percentile latency
   histogram_quantile(0.95, rate(provider_discord_api_duration_seconds_bucket[5m]))

   # By operation type
   histogram_quantile(0.95, rate(provider_discord_api_duration_seconds_bucket[5m])) by (operation)
   ```

2. **Check Resource Usage**
   ```bash
   kubectl top pod -n crossplane-system
   kubectl describe node
   ```

3. **Network Connectivity Test**
   ```bash
   kubectl exec -n crossplane-system deployment/provider-discord -- \
     time curl -s https://discord.com/api/v10
   ```

### Resolution Steps

1. **If High CPU Usage**
   ```bash
   # Scale up resources
   kubectl patch deployment -n crossplane-system provider-discord \
     -p '{"spec":{"template":{"spec":{"containers":[{"name":"provider","resources":{"limits":{"cpu":"500m"}}}]}}}}'
   ```

2. **If Network Latency**
   ```bash
   # Check if Discord API is slow
   curl -w "@curl-format.txt" -s https://discord.com/api/v10

   # Consider regional Discord API endpoints if available
   ```

3. **If Memory Pressure**
   ```bash
   # Increase memory limits
   kubectl patch deployment -n crossplane-system provider-discord \
     -p '{"spec":{"template":{"spec":{"containers":[{"name":"provider","resources":{"limits":{"memory":"512Mi"}}}]}}}}'
   ```

### Escalation
- Latency > 10 seconds: Immediate escalation
- Performance degradation > 1 hour: Platform Team

---

## Circuit Breaker Open

### Symptoms
- AlertManager: `ProviderDiscordCircuitBreakerOpen`
- Resources not reconciling
- Repeated failures triggering circuit breaker

### Investigation Steps

1. **Check Circuit Breaker State**
   ```bash
   # Check which resource type has open circuit breaker
   provider_discord_circuit_breaker_state{state="open"}
   ```

2. **Check Recent Failures**
   ```bash
   kubectl logs -n crossplane-system deployment/provider-discord | grep -i "circuit.*open"
   ```

3. **Check Underlying Issues**
   ```bash
   # Check error patterns
   sum(rate(provider_discord_api_operations_total{status!="success"}[5m])) by (resource_type, status)
   ```

### Resolution Steps

1. **Identify Root Cause**
   - Check if underlying issue (auth, network, Discord API) is resolved
   - Verify Discord service status
   - Check provider logs for error patterns

2. **Reset Circuit Breaker**
   ```bash
   # Circuit breaker will auto-reset after configured time
   # Or restart provider to reset state
   kubectl rollout restart deployment/provider-discord -n crossplane-system
   ```

3. **Monitor Recovery**
   ```bash
   # Watch circuit breaker state
   watch kubectl logs -n crossplane-system deployment/provider-discord --tail=10
   ```

### Escalation
- Multiple circuit breakers open: Platform Team
- Circuit breaker not recovering: SRE Team

---

## Reconciliation Stuck

### Symptoms
- AlertManager: `ProviderDiscordReconciliationStuck`
- Resources in pending state
- No reconciliation activity

### Investigation Steps

1. **Check Resource States**
   ```bash
   kubectl get guilds,channels,roles -A -o wide
   kubectl describe guild <stuck-resource>
   ```

2. **Check Controller Logs**
   ```bash
   kubectl logs -n crossplane-system deployment/provider-discord | grep -i reconcile
   ```

3. **Check Resource Events**
   ```bash
   kubectl get events --sort-by=.lastTimestamp -A | grep -i discord
   ```

### Resolution Steps

1. **Force Reconciliation**
   ```bash
   # Add annotation to trigger reconciliation
   kubectl annotate guild <resource-name> reconcile.crossplane.io/trigger="$(date)"
   ```

2. **Check Resource Dependencies**
   ```bash
   # Verify ProviderConfig exists and is ready
   kubectl get providerconfig
   kubectl describe providerconfig <config-name>
   ```

3. **Restart Controller**
   ```bash
   kubectl rollout restart deployment/provider-discord -n crossplane-system
   ```

### Escalation
- Stuck resources > 30 minutes: Platform Team
- Critical resources stuck: Immediate escalation

---

## High Memory Usage

### Symptoms
- AlertManager: `ProviderDiscordMemoryUsageHigh`
- Pod OOMKilled events
- Slow performance

### Investigation Steps

1. **Check Memory Usage**
   ```bash
   kubectl top pod -n crossplane-system
   process_resident_memory_bytes{job="provider-discord"}
   ```

2. **Check Memory Limits**
   ```bash
   kubectl describe deployment -n crossplane-system provider-discord
   ```

3. **Check for Memory Leaks**
   ```bash
   # Monitor memory usage over time
   kubectl logs -n crossplane-system deployment/provider-discord | grep -i memory
   ```

### Resolution Steps

1. **Increase Memory Limits**
   ```bash
   kubectl patch deployment -n crossplane-system provider-discord \
     -p '{"spec":{"template":{"spec":{"containers":[{"name":"provider","resources":{"limits":{"memory":"1Gi"}}}]}}}}'
   ```

2. **Restart Provider**
   ```bash
   kubectl rollout restart deployment/provider-discord -n crossplane-system
   ```

3. **Monitor Resource Count**
   ```bash
   # Check if high resource count is causing memory pressure
   provider_discord_resources_total
   ```

### Escalation
- Memory usage > 1GB: Platform Team
- Repeated OOMKilled: Development Team for memory leak investigation

---

## Rate Limit Approaching

### Symptoms
- AlertManager: `ProviderDiscordRateLimitApproaching`
- 429 responses from Discord API
- Increased latency

### Investigation Steps

1. **Check Request Rate**
   ```bash
   sum(rate(provider_discord_api_operations_total[1m])) * 60
   ```

2. **Check Discord Rate Limits**
   ```bash
   # Discord global rate limit is typically 50 requests per second
   # Check specific endpoint limits in Discord documentation
   ```

3. **Check Resource Activity**
   ```bash
   # See which resources are generating most requests
   sum(rate(provider_discord_api_operations_total[5m])) by (resource_type, operation)
   ```

### Resolution Steps

1. **Implement Backoff**
   ```bash
   # Provider should automatically implement exponential backoff
   # Check if retry configuration needs adjustment
   kubectl logs -n crossplane-system deployment/provider-discord | grep -i "rate.*limit"
   ```

2. **Reduce Reconciliation Frequency**
   ```bash
   # Consider increasing reconciliation intervals for non-critical resources
   # This may require code changes
   ```

3. **Scale Resources If Needed**
   ```bash
   # Consider if resource count can be reduced
   kubectl get guilds,channels,roles -A --no-headers | wc -l
   ```

### Escalation
- Rate limits being hit consistently: Development Team
- Service degradation due to rate limits: Platform Team

---

## Security Incident Response

### Symptoms
- Unauthorized Discord changes
- Suspicious authentication attempts
- Security alerts from monitoring

### Investigation Steps

1. **Check Audit Logs**
   ```bash
   kubectl logs -n crossplane-system deployment/provider-discord | grep -i security
   ```

2. **Verify Bot Token Security**
   ```bash
   # Check if token has been compromised
   kubectl get secret -n crossplane-system discord-credentials
   ```

3. **Check Resource Changes**
   ```bash
   kubectl get events --sort-by=.lastTimestamp -A | grep -i discord
   ```

### Resolution Steps

1. **Rotate Credentials**
   ```bash
   # Generate new bot token in Discord Developer Portal
   # Update Kubernetes secret
   kubectl create secret generic discord-credentials \
     --from-literal=token=$NEW_BOT_TOKEN \
     --dry-run=client -o yaml | kubectl apply -f -
   ```

2. **Review Access**
   ```bash
   # Check who has access to Discord credentials
   kubectl get rolebinding,clusterrolebinding -A -o wide | grep discord
   ```

3. **Monitor for Anomalies**
   ```bash
   # Increase monitoring sensitivity temporarily
   # Review all Discord changes in the last 24 hours
   ```

### Escalation
- Suspected credential compromise: Immediate Security Team escalation
- Unauthorized resource changes: Platform Team + Security Team
- Active attack: Emergency response team

---

## Disaster Recovery

### Backup Procedures

1. **Export Resource Configurations**
   ```bash
   # Backup all Discord resources
   kubectl get guilds,channels,roles -A -o yaml > discord-resources-backup.yaml

   # Backup provider configuration
   kubectl get providerconfig -o yaml > provider-config-backup.yaml

   # Backup secrets (encrypted)
   kubectl get secret discord-credentials -n crossplane-system -o yaml > discord-secrets-backup.yaml
   ```

2. **Document Discord State**
   ```bash
   # Export Discord server state using Discord API
   # This requires custom tooling but provides backup of actual Discord configuration
   ```

### Recovery Procedures

1. **Restore Provider**
   ```bash
   # Reinstall provider
   helm install provider-discord ./chart/provider-discord

   # Restore configuration
   kubectl apply -f provider-config-backup.yaml
   kubectl apply -f discord-secrets-backup.yaml
   ```

2. **Restore Resources**
   ```bash
   # Restore Discord resources
   kubectl apply -f discord-resources-backup.yaml

   # Monitor reconciliation
   kubectl get guilds,channels,roles -A -w
   ```

3. **Verify Recovery**
   ```bash
   # Check all resources are ready
   kubectl get guilds,channels,roles -A

   # Verify Discord server state matches expectations
   # Manual verification in Discord UI may be required
   ```

### Recovery Time Objectives (RTO)

- Provider restoration: 15 minutes
- Resource reconciliation: 30 minutes
- Full service restoration: 45 minutes

### Recovery Point Objectives (RPO)

- Configuration backup: 24 hours
- Resource state: 1 hour (via monitoring)

---

## Contact Information

### Primary Contacts
- **Platform Team**: platform@company.com
- **Security Team**: security@company.com
- **Development Team**: dev@company.com

### Escalation Matrix

| Severity | Initial Response | Escalation (30 min) | Escalation (1 hour) |
|----------|------------------|---------------------|---------------------|
| Critical | On-call Engineer | Platform Team Lead | CTO |
| High     | Platform Team   | SRE Team           | Engineering Manager |
| Medium   | Development Team| Platform Team      | Team Lead |
| Low      | Development Team| -                  | - |

### Emergency Contacts
- **24/7 On-call**: +1-XXX-XXX-XXXX
- **Security Hotline**: +1-XXX-XXX-XXXX
- **Executive Escalation**: +1-XXX-XXX-XXXX

---

## Runbook Maintenance

- **Review Frequency**: Monthly
- **Update Triggers**: Incident lessons learned, system changes
- **Owner**: Platform Team
- **Approvers**: SRE Team, Security Team

Last Updated: 2025-01-03
Version: 1.0
