---
status: resolved
file: web/src/systems/session/hooks/use-session-create-dialog.ts
line: 115
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:089f9d13107c
review_hash: 089f9d13107c
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 024: Consider refactoring to avoid useEffect for derived state.
## Review Comment

This `useEffect` sets `draft.provider` when `providerOptions` becomes available after the dialog opens—a form of derived state synchronization. The coding guidelines state: "`useEffect` is an escape hatch — only for external system sync; never for derived state."

An alternative approach would be to compute the provider eagerly during render or leverage the query's data arrival through a different pattern.

That said, this handles a legitimate async race condition where workspace provider data may arrive after `openForAgent` is called. If the current behavior is working correctly and the trade-off is acceptable, this can be deferred.

As per coding guidelines: "`useEffect` is an escape hatch — only for external system sync; never for derived state or event responses."

## Triage

- Decision: `valid`
- Notes:
  - The current `useEffect` exists only to backfill `draft.provider` after `providerOptions` arrive, which means the hook is mutating stored state to mirror a value that is derivable from `draft.agentName`, the loaded provider list, and any user-chosen override.
  - Root cause: the hook stores the effective provider directly instead of storing only the operator's explicit override and deriving the effective provider from current inputs.
  - Fix approach: refactor the hook to keep agent selection plus an optional explicit provider override in state, derive the effective provider during render, remove the synchronization effect, and add a dedicated hook regression test. This requires a minimal new file `web/src/systems/session/hooks/use-session-create-dialog.test.tsx` because the hook does not currently have test coverage.
  - Resolved: `use-session-create-dialog.ts` now stores `providerOverride` instead of derived provider state, computes `selectedProvider` from the current agent plus available providers, and removes the synchronization effect entirely.
  - Resolved: added `use-session-create-dialog.test.tsx` to cover delayed provider arrival and override-reset behavior when the agent changes.
  - Verified: focused Vitest session tests passed, then `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify` all completed successfully.
