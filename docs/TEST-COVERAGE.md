# Test Coverage Analysis & Improvement Plan

## Current Status

### Controllers WITH Tests (4/13)
| Controller | Coverage | Tests File | Notes |
|------------|----------|-----------|-------|
| channel | 49.3% | ✅ channel_test.go | Moderate - Observe/Create/Delete tested |
| guild | 67.8% | ✅ guild_test.go | Best coverage - type assertions, isUpToDate |
| role | 47.2% | ✅ role_test.go | Basic - Observe/Create/Update/Delete |
| deduplication | 60.5% | ✅ deduplication_test.go | Good - complex workflow tested |

### Controllers WITHOUT Tests (9/13)
| Controller | Coverage | Status |
|------------|----------|--------|
| application | 0.0% | ❌ No tests |
| integration | 0.0% | ❌ No tests |
| invite | 0.0% | ❌ No tests |
| member | 0.0% | ❌ No tests |
| user | 0.0% | ❌ No tests |
| webhook | 0.0% | ❌ No tests |
| garbagecollection | 0.0% | ❌ No tests |
| providerconfig | 0.0% | ❌ No tests |
| (root controller) | 0.0% | ❌ No tests |

## Test Coverage by Feature

### ✅ NEWLY TESTED (This Session)

#### `internal/controller/cache.go`
- **TestShouldSkipObserve** - 6 test cases
  - nil timestamp (first observe)
  - recently synced (within 5min)
  - just synced (current time)
  - expired cache (6min old)
  - at cache boundary (exactly 5min)
  - near boundary (4:59)
  - **Verdict:** Full coverage with edge cases

- **TestUpdateSyncTime** - 3 test cases
  - nil pointer update
  - old timestamp replacement
  - recent timestamp update
  - **Verdict:** Full coverage

- **TestCacheIntegration** - Full lifecycle
  - Observe → Update → Skip workflow
  - **Verdict:** Integration verified

- **Benchmarks**
  - BenchmarkShouldSkipObserve
  - BenchmarkUpdateSyncTime
  - **Verdict:** Performance characterized

#### Permission Error Handling
- **TestPermissionErrorDetection** - 5 test cases
  - 403 Forbidden detection
  - 401 Unauthorized detection
  - Status code differentiation
  - Nil error safety
  - **Verdict:** Detection robust

- **TestPermissionErrorConditions** - 2 test cases
  - Condition message appropriateness
  - Status field correctness
  - **Verdict:** Condition setting validated

- **TestPermissionErrorDoesNotRetry** - 4 test cases
  - Observe, Create, Update, Delete
  - Confirms nil return on permission errors
  - **Verdict:** Non-retry behavior documented

### ⚠️  PARTIAL COVERAGE (Existing Tests)

**channel_test.go**
- ✅ Type assertion checks
- ✅ Basic Observe logic
- ❌ Permission error scenarios
- ❌ Cache integration
- ❌ Update/position logic
- ❌ Permission overwrite handling

**guild_test.go**
- ✅ Type assertions
- ✅ isUpToDate comparisons
- ❌ Permission error scenarios
- ❌ Cache logic
- ❌ All create/update fields

**role_test.go**
- ✅ Observe/Create/Update/Delete
- ❌ Permission error scenarios
- ❌ Cache logic

**deduplication_test.go**
- ✅ Complex deduplication workflow
- ✅ Guild analysis
- ❌ Permission handling (service-level)

## Coverage Gaps by Priority

### 🔴 CRITICAL (Breaking Changes)
- [ ] Permission error handling in channels, guild, role (recently added)
- [ ] Cache helper integration with existing controllers (foundation laid)
- [ ] Clients package: Discord API request testing

### 🟠 HIGH (Correctness)
- [ ] 9 controllers with 0% test coverage
- [ ] Permission handling in 6 untested controllers
- [ ] Cache scenarios in all Observe methods

### 🟡 MEDIUM (Maintenance)
- [ ] Error recovery paths in tested controllers
- [ ] Rate-limiting behavior
- [ ] Edge cases in permission overwrites

### 🟢 LOW (Polish)
- [ ] Benchmarks for non-cache paths
- [ ] Integration tests (multi-controller scenarios)

## Recommended Testing Strategy

### Phase 1: Foundation ✅ COMPLETE
- [x] Cache helper (`cache_test.go`) - 11 tests, 100% coverage
- [x] Permission error detection (`permission_test.go`) - 11 tests, 100% coverage

### Phase 2: Permission Error Propagation (NEXT)
Add tests to existing test files:
- [ ] `channel_test.go` → 403/401 scenarios in Observe/Create/Update/Delete
- [ ] `guild_test.go` → 403/401 scenarios
- [ ] `role_test.go` → 403/401 scenarios
- [ ] `deduplication_test.go` → service-level permission handling
- **Expected:** 20-30 new tests

### Phase 3: Cache Integration (AFTER)
Add to tested controllers:
- [ ] `channel_test.go` → ShouldSkipObserve integration, LastSyncTime updates
- [ ] `guild_test.go` → Cache boundary cases
- [ ] `role_test.go` → Cache with concurrent updates
- **Expected:** 15-20 new tests

### Phase 4: Coverage for Untested Controllers (LATER)
Create `*_test.go` files for:
- [ ] `application_test.go` - Target 50%+ coverage
- [ ] `webhook_test.go` - Target 50%+ coverage
- [ ] `member_test.go` - Target 50%+ coverage
- [ ] `user_test.go` - Target 50%+ coverage
- [ ] `invite_test.go` - Target 50%+ coverage
- [ ] `integration_test.go` - Target 50%+ coverage
- **Expected:** 100-150 new tests

## Testing Best Practices (This Codebase)

### Pattern 1: Error Scenario Testing
```go
// Test that permission errors set correct conditions
func TestObservePermissionDenied(t *testing.T) {
  // Mock Discord client to return 403 error
  // Call Observe() 
  // Verify: cr.Status has Unavailable condition
  // Verify: error returned is nil (no retry)
}
```

### Pattern 2: Cache Lifecycle Testing
```go
// Test full lifecycle: never-synced → recent → expired
func TestObserveCacheBehavior(t *testing.T) {
  // First observe: no cache, make API call
  // Verify API called, sync time set
  // Second observe: skip API call
  // Verify API not called, returns cached state
  // Wait 6min: expired cache, make API call again
}
```

### Pattern 3: Field Update Testing
```go
// Test that Observe correctly updates all status fields
func TestObserveUpdateStatus(t *testing.T) {
  // Create resource with partial status
  // Call Observe()
  // Verify all fields updated: ID, Name, Position, etc.
  // Verify LastSyncTime set
}
```

## CI/CD Integration

### Current
- ✅ `go test ./...` runs all tests
- ✅ Coverage reported but not enforced
- ❌ No coverage gate (minimum % required)

### Recommended
```yaml
# Add to CI
- name: Test Coverage
  run: |
    go test ./... -coverprofile=coverage.out
    go tool cover -func=coverage.out | grep total
    # Fail if coverage < 50%
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print int($3)}')
    if [ "$COVERAGE" -lt 50 ]; then
      echo "Coverage $COVERAGE% below threshold of 50%"
      exit 1
    fi
```

## Maintenance Notes

### Test Stability
- Cache tests use time-based comparisons (may flake if timing is tight)
  - **Mitigation:** Use 1-second deltas for boundaries
- Permission tests mock error strings (fragile if message format changes)
  - **Mitigation:** Use error type assertions (if we refactor to custom errors)

### Future Improvements
1. **Custom Error Types** - Replace string matching with type assertions
   ```go
   type PermissionError struct { StatusCode int }
   type UnauthorizedError struct { Reason string }
   ```
2. **Mock Interfaces** - Extract DiscordClient to interface for better testing
3. **Table-Driven Tests** - Systematize Observe/Create/Update/Delete patterns

## Reporting

To check current coverage:
```bash
go test ./... -cover
go tool cover -html=coverage.out  # Visual report
```

Target for next release: **50% overall coverage** (from current ~26%)
- Phase 1: ✅ Done (cache + permission helpers)
- Phase 2: +15-20% (permission handling in existing tests)
- Phase 3: +10-15% (cache integration)
- Phase 4: +5-10% (untested controllers)

**Estimated effort:** 2-3 days for full Phase 1-3 completion
