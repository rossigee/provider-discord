# Discord Deduplication Testing Guide

Comprehensive guide for testing the channel deduplication feature.

## Unit Tests

### Running Unit Tests


```bash

# Run all deduplication tests
make test

# Run only deduplication service tests
go test -v ./internal/services -run TestAnalyzeAndDeduplicate

# Run only deduplication controller tests
go test -v ./internal/controller/deduplication -run TestDeduplication

# Run with coverage
go test -v -cover ./internal/services ./internal/controller/deduplication

```


### Test Coverage

The test suite covers:

#### Service Tests (`internal/services/deduplication_test.go`)

1. **TestAnalyzeAndDeduplicate_NoDuplicates**
   - Verifies service handles guilds with no duplicates
   - Confirms summary statistics are correct

2. **TestAnalyzeAndDeduplicate_WithDuplicates**
   - Tests detection of duplicate channels
   - Verifies grouping by exact name match
   - Confirms summary aggregation

3. **TestAnalyzeAndDeduplicate_ActionMode_DeletesDuplicates**
   - Tests channel deletion in action mode
   - Verifies Discord API DELETE requests
   - Confirms deletion count tracking

4. **TestAnalyzeAndDeduplicate_MultipleGuilds**
   - Tests analysis across multiple Discord servers
   - Verifies per-guild aggregation
   - Tests summary across all guilds

5. **TestAnalyzeAndDeduplicate_TargetGuilds**
   - Tests guild filtering
   - Verifies only targeted guilds are processed
   - Confirms other guilds are skipped

6. **TestAnalyzeAndDeduplicate_KeepsOldestChannel**
   - Tests position-based channel selection
   - Verifies lowest position (oldest) is kept
   - Confirms other channels are marked for deletion

7. **TestAnalyzeAndDeduplicate_APIError**
   - Tests handling of Discord API errors
   - Verifies error propagation
   - Confirms error state tracking

8. **TestAnalyzeAndDeduplicate_DeleteError**
   - Tests handling of deletion failures
   - Verifies partial failure resilience
   - Confirms error logging per resource

9. **TestDuplicateGroup_FindsCorrectKeepIndex**
   - Unit test for keep index selection logic
   - Tests multiple position scenarios

10. **TestEmptyGuild**
    - Tests handling of guilds with no channels
    - Verifies no duplicates are reported

11. **TestNoGuilds**
    - Tests handling when bot is in no guilds
    - Confirms graceful handling of empty state

#### Controller Tests (`internal/controller/deduplication/deduplication_test.go`)

1. **TestDeduplicationAnnotationPredicate**
   - Tests predicate filters correct annotations
   - Verifies report/action modes are recognized
   - Tests rejection of invalid modes

2. **TestExtractCredentials_ValidSecret**
   - Tests credential extraction from Kubernetes secrets
   - Verifies token extraction
   - Confirms base URL handling

3. **TestExtractCredentials_CustomBaseURL**
   - Tests custom Discord API endpoint support

4. **TestLastProcessedAnnotation**
   - Tests idempotency mechanism
   - Verifies mode transitions are allowed
   - Confirms duplicate runs are prevented

5. **TestDeduplicationNameGeneration**
   - Tests unique resource naming

6. **TestProviderConfigWithDeduplicationSpec**
   - Tests spec deserialization
   - Verifies all dedup spec fields

7. **TestDeduplicationModeValidation**
   - Tests mode string validation
   - Verifies only "report" and "action" accepted

8. **TestAnnotationUpdate**
   - Tests annotation updates for mode transitions

9. **TestEventGeneration**
   - Tests Kubernetes event creation
   - Verifies event types and messages

### Running Individual Tests


```bash

# Run specific test
go test -v ./internal/services -run TestAnalyzeAndDeduplicate_WithDuplicates

# Run with verbose output
go test -vv ./internal/services

# Run with race detector
go test -race ./internal/services ./internal/controller/deduplication

# Run with benchmarks
go test -bench=. ./internal/services

```


## Integration Tests

### Prerequisites

- Kind cluster or test Kubernetes cluster
- Discord bot token with proper permissions
- Test Discord server with duplicate channels

### Manual Integration Test


```bash

# Step 1: Build provider image
make build

# Step 2: Deploy to test cluster
kind load docker-image provider-discord:latest --name test-cluster
kubectl apply -f config/crd/
kubectl apply -f config/provider/

# Step 3: Create ProviderConfig with test credentials
cat <<EOF | kubectl apply -f -
apiVersion: discord.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: test
spec:
  credentials:
    source: Secret
    secretRef:
      name: discord-token
      namespace: crossplane-system
EOF

# Step 4: Create secret with bot token
kubectl create secret generic discord-token \
  --from-literal=token='YOUR_BOT_TOKEN' \
  -n crossplane-system

# Step 5: Trigger report mode
kubectl annotate providerconfig test \
  discord.crossplane.io/deduplication=report --overwrite

# Step 6: Monitor results
kubectl get deduplication -w
kubectl describe deduplication <name>

# Step 7: Verify in Discord
# Log in and check server for reported duplicates

# Step 8: Trigger action mode (if results are correct)
kubectl annotate providerconfig test \
  discord.crossplane.io/deduplication=action --overwrite

# Step 9: Verify deletion
kubectl describe deduplication <name>
# Check Discord for deleted channels

```


## Performance Tests

### Load Testing

Test with guilds containing many channels:


```bash

# Run service analysis on guild with 1000 channels (mocked)
go test -v -bench=BenchmarkAnalyzeGuildLarge ./internal/services

```


### Rate Limiting Tests

Discord has rate limits. Test behavior under rate limiting:


```bash

# Simulate rate limit responses in test server
# Verify exponential backoff
# Confirm retry logic

```


## Test Coverage Analysis

### Check Coverage by Component


```bash

# Service coverage
go test -cover ./internal/services

# Controller coverage
go test -cover ./internal/controller/deduplication

# Combined coverage
go test -cover ./internal/services ./internal/controller/deduplication

# Generate coverage report
go test -coverprofile=coverage.out ./internal/services ./internal/controller/deduplication
go tool cover -html=coverage.out

```


### Coverage Goals

- **Target**: >80% coverage
- **Critical paths**: >95% coverage
- **Error handling**: >90% coverage

## Continuous Integration

### CI/CD Pipeline Tests

The provider includes GitHub Actions workflow that runs:


```bash

# Lint
make lint

# Build
make build

# Test with coverage
make test

# Security scan
govulncheck ./...

```


### Run CI Checks Locally


```bash

# Run all checks that CI runs
make reviewable

# This includes:
# - Code generation
# - Linting
# - Testing
# - Security checks

```


## End-to-End Test Scenarios

### Scenario 1: No Duplicates

**Setup**: Guild with 10 channels, all unique names
**Expected**: Report finds no duplicates, action mode has nothing to delete


```bash

kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite
kubectl get deduplication -o yaml | grep totalDuplicateChannelsFound
# Should be: 0

```


### Scenario 2: Simple Duplicates

**Setup**: Guild with 5 channels including 2 named "general"
**Expected**: Report identifies 1 duplicate, action deletes 1 channel


```bash

kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite
kubectl describe deduplication <name> | grep -A5 "Duplicate Groups"
# Should show: Count: 2, ChannelsDeleted: 1

kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=action --overwrite
kubectl describe deduplication <name> | grep channelsDeleted
# Should show: 1

```


### Scenario 3: Complex Duplicates

**Setup**: Multiple guilds with overlapping duplicate names
**Expected**: Correct per-guild analysis and aggregation


```bash

kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite
kubectl get deduplication -o yaml | jq '.status.summary'
# Verify:
# - totalGuildsAnalyzed: correct count
# - totalDuplicateChannelsFound: sum of all duplicates
# - duplicateGroupsFound: sum of all groups

```


### Scenario 4: Mode Transition

**Setup**: Run report, then transition to action
**Expected**: Annotation prevents re-running report, action proceeds


```bash

# Run report
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite
sleep 5
kubectl get deduplication | wc -l  # Should have one entry

# Try applying report again
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite --force
sleep 2
kubectl get deduplication | wc -l  # Should still have one entry (not re-run)

# Transition to action
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=action --overwrite
sleep 5
kubectl get deduplication | wc -l  # Should have new entry (action mode)

```


### Scenario 5: Error Handling

**Setup**: Guild where bot lacks permissions
**Expected**: Error recorded, operation continues for other guilds


```bash

kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite
kubectl describe deduplication <name> | grep -i error
# Should show permission error for affected guild

```


## Debugging Failed Tests

### Enable Debug Logging


```bash

# Set debug log level for provider
kubectl set env deployment/provider-discord \
  -n crossplane-system \
  DEBUG=true

# Tail logs
kubectl logs -n crossplane-system -l app=provider-discord -f

```


### Inspect Deduplication CRD Details


```bash

# View full CRD status
kubectl get deduplication <name> -o yaml

# Watch for changes
kubectl get deduplication <name> -w

# Check events
kubectl describe deduplication <name>

```


### Test Discord API Connectivity


```bash

# Verify bot token works
curl -H "Authorization: Bot $DISCORD_BOT_TOKEN" \
  https://discord.com/api/v10/users/@me

# List guilds
curl -H "Authorization: Bot $DISCORD_BOT_TOKEN" \
  https://discord.com/api/v10/users/@me/guilds

# List channels for guild
curl -H "Authorization: Bot $DISCORD_BOT_TOKEN" \
  https://discord.com/api/v10/guilds/{guild_id}/channels

```


## Test Cleanup

### Clean Test Resources


```bash

# Delete all deduplication records
kubectl delete deduplication --all

# Remove ProviderConfig annotations
kubectl patch providerconfig default \
  -p '{"metadata":{"annotations":{"discord.crossplane.io/deduplication":null,"discord.crossplane.io/deduplication-processed":null}}}'

# Clean up test Discord server
# Manually restore deleted channels or re-create test server

```


### Reset Test Environment


```bash

# Delete provider
helm uninstall crossplane-provider-discord -n crossplane-system

# Delete CRDs
kubectl delete crd \
  providerconfigs.discord.crossplane.io \
  deduplicatons.deduplication.discord.crossplane.io

# Restart cluster
kind delete cluster --name test-cluster
kind create cluster --name test-cluster

```


## Performance Benchmarks

### Expected Performance

| Operation | Time | Guilds | Channels |
|-----------|------|--------|----------|
| Analyze | <5s | 1 | 100 |
| Analyze | <30s | 5 | 500 |
| Delete single | <1s | 1 | 1 |
| Batch delete | <5s | 1 | 10 |

### Run Benchmarks


```bash

# Simple benchmark
go test -bench=. -benchtime=10s ./internal/services

# Profile memory
go test -memprofile=mem.prof ./internal/services
go tool pprof mem.prof

# Profile CPU
go test -cpuprofile=cpu.prof ./internal/services
go tool pprof cpu.prof

```


## Troubleshooting Tests

### Tests Hang


```bash

# Kill hanging tests
pkill -f "go test"

# Run with timeout
go test -timeout 30s ./internal/services

```


### Tests Fail Intermittently

Possible causes:
- Timing issues (increase test timeout)
- Race conditions (run with `-race`)
- Flaky mock server (check httptest implementation)


```bash

# Run with race detection
go test -race ./internal/services

# Run multiple times to catch flakiness
for i in {1..10}; do go test ./internal/services || break; done

```


### Integration Tests Fail

1. **Check Discord API connectivity**
   ```bash
   curl -H "Authorization: Bot $DISCORD_BOT_TOKEN" \
     https://discord.com/api/v10/users/@me/guilds
   ```

2. **Verify bot permissions**
   - Check bot has "View Channels" permission
   - Check bot has "Manage Channels" for action mode

3. **Check cluster connectivity**
   ```bash
   kubectl cluster-info
   kubectl get nodes
   ```

4. **Verify secret exists**
   ```bash
   kubectl get secret discord-token -n crossplane-system
   ```

## Best Practices for Testing

1. **Isolate Tests**: Use separate Discord test servers
2. **Mock External APIs**: Use httptest for Discord API
3. **Clean Up Resources**: Always delete test resources
4. **Test Error Cases**: Include negative test scenarios
5. **Document Assumptions**: Note any test-specific setup
6. **Use Test Fixtures**: Pre-defined test data for consistency
7. **Automate Tests**: Run in CI/CD pipeline
8. **Monitor Coverage**: Track and maintain test coverage

## Related Resources

- [Deduplication Feature Documentation](./deduplication.md)
- [Integration Tests Guide](../../../CONTRIBUTING.md)
- [Discord API Testing Best Practices](https://discord.com/developers/docs)
