---
status: resolved
file: internal/daemon/coordinator_config_test.go
line: 70
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:0458891e0f6f
review_hash: 0458891e0f6f
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 034: Workspace config test copies the entire global config struct.
## Review Comment

Line 80 performs a shallow copy of the config struct. While this works for value types, if `aghconfig.Config` contains pointer or slice fields, modifications to `workspaceCfg` could affect `global`. This appears safe given the current usage, but the intent would be clearer with explicit field initialization.

## Triage

- Decision: `VALID`
- Notes: The workspace config test shallow-copies the global config struct before overriding autonomy fields. It is safe today, but the test intent is workspace override resolution and should not depend on unrelated pointer/slice fields staying harmless.
- Fix: Build the workspace config from `defaultCoordinatorResolverConfig(t)` explicitly and set only the fields needed for the workspace case.
