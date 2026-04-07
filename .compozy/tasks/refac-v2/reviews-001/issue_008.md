---
status: resolved
file: magefile.go
line: 113
severity: medium
author: claude-code
provider_ref:
---

# Issue 008: Boundary check misses new api/* package rules

## Review Comment

The `Boundaries()` function checks that lower-level packages don't import `daemon/`, `api/httpapi/`, `api/udsapi/`, or `cli/`. But it does **not** enforce rules for the new packages introduced by refac-v2:

- `internal/api/core` must not import `daemon/`, `api/httpapi/`, `api/udsapi/`, or `cli/`
- `internal/api/httpapi` must not import `daemon/`, `cli/`, or `api/udsapi/`
- `internal/api/udsapi` must not import `daemon/`, `cli/`, or `api/httpapi/`
- `internal/api/contract` must not import any of the above

Without these rules, a future change could silently introduce import cycles or layer violations in the API subtree that refac-v2 just established.

**Fix:** Add the missing rows to the `forbidden` slice for `api/contract`, `api/core`, `api/httpapi`, and `api/udsapi`.

## Triage

- Decision: `valid`
- Root cause: `mage Boundaries` still enforces only the older top-level layering rules and does not cover the refac-v2 API subtree split. That leaves the new `api/contract`, `api/core`, `api/httpapi`, and `api/udsapi` packages unprotected against import-boundary regressions.
- Fix approach: Add the missing forbidden-import rows for the API packages and keep the rule set aligned with the current architecture comments.
- Resolution: Implemented and exercised by the passing `Boundaries` step in `make verify`.
