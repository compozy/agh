---
status: resolved
file: internal/api/core/settings.go
line: 262
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kRT,comment:PRRC_kwDOR5y4QM65B6zw
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Restart-controller errors currently fall through to 500.**

`StatusForSettingsError` only recognizes the settings/workspace error set, so restart-specific failures like `daemon.ErrRestartOperationNotFound` end up as internal errors. That makes `GET /api/settings/actions/restart/:operation_id` return 500 instead of 404, and `TriggerSettingsRestart` has the same mapping gap for restart-specific failures.



Also applies to: 281-284

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/settings.go` around lines 259 - 262, The restart controller
currently maps restart-specific failures (e.g.,
daemon.ErrRestartOperationNotFound) to 500 because StatusForSettingsError
doesn't handle restart errors; update the error-to-status mapping for the
restart endpoints by either extending StatusForSettingsError to recognize
restart-specific errors or by adding a dedicated StatusForRestartError and
calling it before the generic mapping in both SettingsRestart.RequestRestart and
TriggerSettingsRestart handlers (i.e., check errors returned from
SettingsRestart.RequestRestart and the restart lookup paths for
daemon.ErrRestartOperationNotFound and other restart-related sentinel errors and
map them to 404/appropriate HTTP codes when calling h.respondError).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  The reported `GET /api/settings/actions/restart/:operation_id` gap is already fixed on the current branch. `internal/daemon/settings.go` wraps `daemon.ErrRestartOperationNotFound` with `core.NewSettingsNotFoundError`, so restart-status lookups map to 404 before they reach `StatusForSettingsError`. `TriggerSettingsRestart` does not currently expose a comparable restart-specific not-found class, so the cited failure mode is not reproducible from the scoped files.
