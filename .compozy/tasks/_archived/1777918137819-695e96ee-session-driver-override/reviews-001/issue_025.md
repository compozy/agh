---
status: resolved
file: web/src/systems/session/hooks/use-sessions.test.tsx
line: 37
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:2366e3b98d81
review_hash: 2366e3b98d81
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 025: Consider asserting provider propagation in hook results.
## Review Comment

Mocks now include `provider`, but tests don’t verify it reaches hook consumers. Adding explicit assertions will lock in the new contract behavior.

Also applies to: 72-81

## Triage

- Decision: `valid`
- Notes:
  - The mocked session payloads in `use-sessions.test.tsx` now include `provider`, but the assertions only check IDs and adapter calls, so the new contract could regress without a failing test.
  - Root cause: the tests were updated with provider-bearing fixtures but never promoted that field into explicit expectations.
  - Fix approach: assert provider propagation in both the list and single-session hook results so consumers keep a hard guarantee around the session provider field.
  - Resolved: `use-sessions.test.tsx` now asserts provider propagation for both the sessions list hook and the single-session detail hook.
  - Verified: focused Vitest session tests passed, then `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify` all completed successfully.
