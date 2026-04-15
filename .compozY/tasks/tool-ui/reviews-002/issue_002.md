---
status: resolved
file: web/src/systems/session/components/copy-button.test.tsx
line: 11
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4114092850,nitpick_hash:0604616589d9
review_hash: 0604616589d9
source_review_id: "4114092850"
source_review_submitted_at: "2026-04-15T13:47:00Z"
---

# Issue 002: Consider restoring navigator.clipboard after each test for stronger isolation.

## Review Comment

The current overrides work, but restoring the original descriptor reduces cross-test leakage risk.

Also applies to: 22-25, 50-53

## Triage

- Decision: `valid`
- Root cause: The test file overwrites `navigator.clipboard` with an own property in each test and only resets Vitest mocks/timers in `afterEach`, so the mutated global descriptor can leak into later tests in this file or other files.
- Fix approach: Capture the original clipboard property descriptor before each test, restore or delete the test override in `afterEach`, and keep the current success/failure assertions intact.
- Resolution: Stored the pre-test clipboard descriptor in `beforeEach` and restored it in `afterEach` so the test suite leaves `navigator.clipboard` in its original state. Verified with the focused component tests, `make web-lint`, `make web-typecheck`, and `make verify`.
