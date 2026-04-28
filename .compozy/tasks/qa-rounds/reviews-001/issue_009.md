---
status: resolved
file: internal/cli/config_test.go
line: 242
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4188693296,nitpick_hash:bd52edee9423
review_hash: bd52edee9423
source_review_id: "4188693296"
source_review_submitted_at: "2026-04-28T12:24:35Z"
---

# Issue 009: Add a negative regression for legacy environments.* mutation paths.
## Review Comment

After the rename to `sandboxes.*`, add an explicit assertion that `config set environments.dev.backend local` fails. This prevents accidental alias/fallback reintroduction.

Based on learnings, "Renames must update code, storage, APIs, CLI, extensions, specs, RFCs, and .compozy/tasks/* artifacts all in a single change. Do not create aliases, dual fields, or schema fallback paths."

## Triage

- Decision: `VALID`
- Notes: The hard rename from environments to sandboxes needs a negative guard against alias reintroduction. `config set environments.dev.backend local` is currently unsupported through the classifier, but no regression pins that behavior. Add an explicit CLI failure assertion for the legacy path.

## Resolution

- Added negative CLI regressions for legacy `defaults.environment` and `environments.*` mutation paths.
- Verified through targeted CLI tests and `make verify`.
