---
status: resolved
file: internal/api/core/bundles.go
line: 329
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__Be,comment:PRRC_kwDOR5y4QM63zbx4
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Default bundle errors should map to 500, not 400.**

The default branch turns store/loader/syncer failures into `400 Bad Request`. Those are server-side faults, so this will misclassify operational failures and break retry/alerting behavior. Keep explicit 4xx mappings for known validation/sentinel cases, but fall back to `http.StatusInternalServerError` here.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/bundles.go` around lines 311 - 329, The default branch of
StatusForBundleError currently returns http.StatusBadRequest which misclassifies
server-side failures; change the default return value in the
StatusForBundleError function to http.StatusInternalServerError so
unknown/store/loader/syncer errors map to 500 while keeping the existing
explicit 4xx mappings (e.g., ErrActivationNotFound, ErrBundleNotFound,
ErrDefaultChannelBusy, ErrWebhookUnsupported) and the workspace-specific
delegation to StatusForWorkspaceError unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `StatusForBundleError` still maps the default branch to `400 Bad Request`, which misclassifies loader/store/reconcile failures as client errors even though they are server-side operational faults.
- Fix plan: keep the explicit 4xx sentinel mappings intact and change the fallback branch to `500 Internal Server Error`.
- Resolution: changed the default bundle-error mapping to `http.StatusInternalServerError` while preserving the explicit 4xx/special-case branches.
- Verification: added coverage in `internal/api/core/network_test.go` and passed `go test ./internal/api/core` plus `make verify`.
