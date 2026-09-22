# Test Coverage Analysis

Snapshot: **v0.16.1 (2026-09-19)**, measured with `go test ./... -cover`.
Overall statement coverage is **40.7%**. Coverage is reported in CI but **not
enforced** (no minimum threshold); `go test ./...` failing is the only gate.

## Coverage by package

### Strong (>= 70%)

| Package | Coverage |
|---------|----------|
| `internal/metrics` | 100.0% |
| `internal/resilience` | 96.1% |
| `internal/controller/guild` | 78.4% |
| `internal/health` | 73.9% |
| `internal/controller/invite` | 71.7% |
| `internal/controller/webhook` | 70.3% |

### Moderate (40% – 69%)

| Package | Coverage |
|---------|----------|
| `internal/controller/role` | 67.1% |
| `internal/controller/application` | 62.5% |
| `internal/controller/deduplication` | 60.5% |
| `internal/controller/channel` | 60.0% |
| `internal/controller/user` | 58.2% |
| `internal/services` | 55.4% |
| `apis/role/v1beta1` | 53.5% |
| `internal/controller/integration` | 52.5% |
| `apis/guild/v1beta1` | 48.0% |
| `apis/channel/v1beta1` | 45.6% |
| `internal/clients` | 43.4% |

### None / very low (< 40%)

| Package | Coverage | Notes |
|---------|----------|-------|
| `internal/controller/member` | 7.6% | Controller has tests but most branches unexercised |
| `internal/controller` (root) | 5.4% | Cache/permission helpers only |
| `internal/controller/garbagecollection` | 0.0% | No tests |
| `internal/controller/providerconfig` | 0.0% | No tests |
| `internal/tracing` | 0.0% | No tests |
| `cmd/provider` | 0.0% | No tests (main wiring) |
| `apis/{application,deduplication,integration,invite,member,user,webhook}/v1beta1` | 0.0% | No per-type tests |
| `apis/v1beta1`, `apis` (root) | 0.0% | No tests |

## What the client tests cover

`internal/clients` (43.4%) exercises request construction, error mapping, the
429 retry/backoff path, the global rate-limit window, `Retry-After` parsing, and:

- `TestCheckRedirectDoesNotForwardAuthorizationCrossHost` — cross-host redirects
  return `http.ErrUseLastResponse` so the bot token is not forwarded.
- `TestGlobalRateLimitNotUpdatedOnErrorResponse` — the global pause is not
  engaged from error (`>= 400`) responses.
- `TestUpdateGlobalRateLimitFromHeaders` — only an explicit
  `X-RateLimit-Global: true` with a parseable reset > 100ms engages the pause;
  per-route `X-RateLimit-Reset-After` alone is ignored.

## Behavior note (permission errors)

As of v0.16.1, `Delete` returns an **error** on a Discord 403/401 so the
resource stays in `Deleting` and retries instead of being reported deleted while
the Discord object is orphaned. Tests for this live in each controller's
`*_test.go` (`TestDeletePermissionDenied`) for guild, webhook, channel, role,
integration, invite, and member. Older revisions of this document stated the
opposite (nil return); that is no longer accurate.

## Priorities

1. **`internal/controller/member`** — raise from 7.6% toward the other
   controllers (~60%+).
2. **`internal/clients`** — cover the create/update/delete request builders and
   the remaining error branches.
3. **Untested controllers** — `garbagecollection`, `providerconfig`.
4. **API type packages** — add `types_test.go` (deepcopy/JSON) for application,
   integration, invite, member, user, webhook like the channel/guild/role ones.
5. **Enforce coverage in CI** — optional minimum threshold so coverage does not
   regress.

## CI integration

- `go test ./...` runs on every PR (`unit-tests` job) and fails on test errors.
- `coverage.out` is uploaded to Codecov.
- There is **no** coverage gate today; adding one would require a threshold in
  the workflow or a Codecov project/patch target.
