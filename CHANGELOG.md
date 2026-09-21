# Changelog

## v0.16.1 (2026-09-19)

### Security

- Stop logging Discord request bodies at Info level. OAuth2 invite tokens and
  similar secrets are now logged only at `V(1)` debug
  (`internal/clients/discord.go`).
- Block `http.Client` redirects from forwarding the `Authorization` (bot token)
  header to a different host.
- Scope the CI workflow permissions to `contents: read` and `actions: read`,
  removing the workflow-wide `packages: write` and `id-token: write` grants.
- Gate the release workflow on a trusted actor (repository owner) and validate
  the version tag against `^v[0-9]+\.[0-9]+\.[0-9]+$` before publishing.

### Fixes

- Return an error when `Delete` receives a 403/401 from the Discord API instead
  of reporting success, so managed resources stay in `Deleting` and retry rather
  than orphaning the Discord resource (guild, webhook, channel, role, member,
  integration, invite).
- Honour server `Retry-After` values even when fractional/sub-second, and only
  engage the global pause on `X-RateLimit-Global: true`.
- Clamp exponential backoff to `maxBackoff` and apply symmetric jitter.
- Cap error response body reads at 64 KiB.
- Trim whitespace from bot tokens extracted from Kubernetes secrets.
- Nil-guard `ProviderConfigReference` in the application connector and
  `SearchGuildMembers` request construction; encode list query parameters with
  `url.Values`.

### Build & Release

- The release workflow embeds the published
  `ghcr.io/rossigee/provider-discord` controller image in the xpkg instead of an
  unpushable per-host build ref, keeps the embedded package metadata (both the
  `image:` field and the install snippet in the embedded README) in sync with
  the released image, and fixes the uploaded xpkg artifact path.
- Resolve a `structured-merge-diff` v6/v7 mismatch by dropping the extraneous
  `k8s.io/kube-openapi` dependency.
- Refresh documentation, examples (add `member`, `user`, `application`,
  `integration`), and version references to `v0.16.1`.

---

## Release summary: v0.13.0 – v0.16.0 (collated)

These releases were published without changelog entries; the summary below is
collated from the merged pull requests and release tags in that range.

### Features

- Deduplication: message-history-based keep selection, webhook and role
  deduplication, and a deduplication action/report workflow.
- Global rate limit monitoring and status reporting.
- Multi-arch Docker image and Crossplane v2 XPKG release workflow.

### Fixes

- Deduplication safety fix (only deduplicate when channel type and parent
  category match) and improved deletion logging/validation for action mode.
- Increase the default poll interval from 1m to 5m and reduce the reconcile rate
  to 1 req/s to avoid Discord global rate limits.
- Graceful handling of RBAC errors; remove an invalid CEL validation rule from
  the ProviderConfig CRD; use `POD_SERVICE_ACCOUNT` for the RBAC binding.
- Drop the redundant v1beta1 controllers for 9 resource types and remove
  misleading v1alpha1 import aliases.
- Improve webhook, invite, and user controller test coverage.

---

## v0.12.1 (2026-08-09)

### Fixes

- Add global rate limiter (5 req/s with burst 5) to prevent exhaustion of Discord's API rate
  limits. All DiscordClient instances now share a package-level token bucket limiter, ensuring
  aggregate request rate stays well below Discord's global 50 req/s limit and per-route
  constraints.
- Implement `Retry-After` header parsing and exponential backoff on HTTP 429 (Too Many Requests)
  responses. When Discord or Cloudflare returns a 429, the client extracts the retry delay from
  the response body or headers, waits out the indicated duration, and retries up to 3 times
  before surfacing the error to the reconciler. This prevents the self-sustaining Cloudflare IP
  ban loops that occurred when reconcilers retried immediately without respecting rate-limit
  signals.

---

## v0.10.0 (2026-06-07)

### Features

- Add namespace-scoped resource support for all Discord managed resources
  (User, Role, Member, Channel, Category, RoleAssignment).
  Previously only cluster-scoped ClusterRoles were supported. Now users can
  create Discord resources within a specific namespace using Namespaced CRDs
  and a namespaced ProviderConfig.
- Upgrade Discord API to v11.

### Fixes

- Use `github.token` instead of `PAT_TOKEN` for GHCR authentication in build
  pipeline.

---

## v0.9.1 (2026-06-07)

### Fixes

- Fix `GetProviderConfigReference()` type-switch in `config.go` and all
  controllers (`user`, `role`, `member`, `integration`) to use
  `*xpv1.ProviderConfigReference` instead of the removed `*xpv1.Reference`.
  All `Connect()` calls were silently falling through to
  `errGetProviderConfig`, breaking reconciliation for every resource type
  after the v0.9.0 crossplane-runtime v2 upgrade.
- Fix `writeConnectionSecretsToRef` (plural) typo in `invite` and `webhook`
  examples — corrected to `writeConnectionSecretToRef`. The misspelled field
  was silently ignored, so connection secrets were never written.
- Add missing `kind: ClusterProviderConfig` to `providerConfigRef` in all
  example manifests. The field is now required by the updated CRD schema;
  without it `kubectl apply` returns a validation error.
- Remove `crd:allowDangerousTypes=true` from `apis/generate.go` (no float
  types exist in the API; the flag was unnecessarily disabling a controller-gen
  safety guard).
- Remove duplicate `controller-gen` `//go:generate` directives from
  `apis/v1alpha1/register.go` — fully covered by the top-level `apis/generate.go`.
- Switch golangci-lint pre-commit hook to `language: system` to prevent build
  failures when the pre-commit Go environment lags behind the project's Go
  version requirement.

### Migration Notes (crossplane-runtime v2 / v0.9.0 upgrade)

The v0.9.0 release upgraded to crossplane-runtime v2, which removed two
fields from the managed resource schema. Existing resources stored in etcd
before upgrading the CRDs will be affected:

**`deletionPolicy` removed**

The `deletionPolicy` field has been removed from all managed resource specs.
Resources that previously had `deletionPolicy: Orphan` set to prevent
deletion of the external Discord resource (guild, channel, role, etc.) will
silently lose that protection after the CRD upgrade.

Mitigation: before upgrading, identify any resources with
`deletionPolicy: Orphan` and add `managementPolicies: ["Observe"]` as the
equivalent replacement.

**`writeConnectionSecretToRef.namespace` removed**

The `namespace` field has been removed from `writeConnectionSecretToRef`.
Resources that previously specified a target namespace for their connection
secret will have the secret written to the provider's own namespace instead,
with no error raised.

Mitigation: before upgrading, identify any resources with a non-default
`writeConnectionSecretToRef.namespace` and ensure the provider has write
access to the namespace where it will now write (its own namespace), and
update any consumers of those secrets accordingly.

---

## v0.9.0 (2026-05-25)

- Upgrade to Go 1.26.3, golangci-lint 2.12.2, crossplane-runtime v2.3.1.

## v0.8.9 and earlier

See [GitHub releases](https://github.com/rossigee/provider-discord/releases).
