# ADR-002: Provider Model Config Hard Cut

## Status

Accepted

## Context

`ProviderConfig` currently stores model selection as flat provider fields:

- `default_model`
- `supported_models`
- `supports_reasoning_effort`

Those fields conflate four different concepts:

- default model requested during session creation;
- curated model choices shown in UI;
- account/provider availability discovered at runtime;
- per-model reasoning capability and effort levels.

AGH is greenfield alpha. Keeping aliases or dual fields would encode an obsolete shape into every config parser, API payload, UI form, generated type, and test fixture.

## Decision

AGH will remove the flat provider model fields and replace them with a nested provider `models` block:

```toml
[providers.codex.models]
default = "gpt-5.4"

[[providers.codex.models.curated]]
id = "gpt-5.4"
display_name = "GPT-5.4"
supports_tools = true
supports_reasoning = true
reasoning_efforts = ["minimal", "low", "medium", "high", "xhigh"]
default_reasoning_effort = "medium"
context_window = 256000
max_output_tokens = 32000
```

`models.curated` is a curated selectable list with metadata. It is not an allowlist. Manual model IDs remain valid even when they are absent from config or catalog rows.

## Consequences

- Config containing `default_model`, `supported_models`, or `supports_reasoning_effort` fails validation with a deterministic hard-cut error.
- There are no aliases, fallback reads, compatibility renderers, or dual payload fields.
- Built-in provider definitions move their current defaults into `models.default` and `models.curated`.
- Web settings, generated OpenAPI types, and CLI docs must be updated in the same implementation pass.

## Delete Targets

- `internal/config.ProviderConfig.DefaultModel`
- `internal/config.ProviderConfig.SupportedModels`
- `internal/config.ProviderConfig.SupportsReasoningEffort`
- `internal/config.ProviderConfig.EffectiveSupportedModels`
- `internal/config.ProviderConfig.EffectiveSupportsReasoningEffort`
- TOML keys `providers.<id>.default_model`, `providers.<id>.supported_models`, `providers.<id>.supports_reasoning_effort`
- API fields `default_model`, `supported_models`, `supports_reasoning_effort` on provider settings and session provider option payloads
- Web provider settings controls that directly edit the old flat fields

## References

- `internal/config/provider.go`
- `internal/config/merge.go`
- `internal/api/contract/contract.go`
- `internal/api/core/conversions.go`
- `web/src/routes/_app/settings/providers.tsx`
- `web/src/systems/session/hooks/use-session-create-dialog.ts`
