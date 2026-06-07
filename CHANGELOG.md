# Changelog

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
