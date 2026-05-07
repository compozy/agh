# ADR-003: Extension Model Source Contract

## Status

Accepted

## Context

AGH's extension system already has manifest-provided capabilities and AGH -> extension service methods for provider-like surfaces. Model discovery needs the same extensibility path because built-in sources cannot cover every ACP wrapper, local runtime, gateway, or proprietary provider.

The extension contract must not let extensions own global catalog state. Extensions provide source rows; the daemon validates, stores, merges, and exposes the projection.

## Decision

AGH will add a manifest-provided capability named `model.source` and an AGH -> extension service method `models/list`.

Extension model sources will:

- Return rows scoped to provider IDs the extension declares.
- Include `source_id`, `provider_id`, `model_id`, source priority, freshness metadata, and optional model metadata.
- Run under daemon-enforced timeout, capability grants, and provider env/home policy.
- Fail closed by recording source status instead of blocking session creation.

The extension Host API will also gain read/refresh/status methods so extension authors and agents can inspect the daemon-owned projection.

## Consequences

- Extensions can enrich or provide model availability without bypassing catalog merge rules.
- Marketplace extensions cannot obtain unrestricted model refresh privileges by default; grants are explicit.
- AGH can document one provider-model surface for built-ins and extensions.

## References

- `internal/extension/protocol/host_api.go`
- `internal/extension/contract/host_api.go`
- `internal/extension/manager.go`
- `internal/extension/capability_test.go`
- `.resources/paperclip/adapter-plugin.md`
- `/Users/pedronauck/dev/compozy/compozy-code/providers/sdk/src/models/types.ts`
