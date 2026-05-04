---
status: resolved
file: internal/api/core/session_workspace_internal_test.go
line: 187
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:722c7bc52f5e
review_hash: 722c7bc52f5e
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 001: Hardcoded expected provider list may become fragile.
## Review Comment

The expected list on line 198 includes built-in providers (`codex`, `copilot`, `cursor`, etc.) alongside the config-defined `alpha` and `claude`. If built-in providers are added or removed in the future, this test will break silently or require manual updates.

Consider deriving the expected list programmatically or extracting the built-in provider names from a shared constant to reduce maintenance burden.

## Triage

- Decision: `valid`
- Root cause: `TestSessionProviderOptionPayloadsFromConfig` hardcodes the current built-in provider names instead of deriving them from `config.BuiltinProviders()`, so any registry change creates avoidable test drift.
- Fix plan: build the expected provider-name set from the built-in registry plus config-defined providers, then compare the normalized payload names against that derived expectation.
