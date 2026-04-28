---
status: resolved
file: internal/cli/install_test.go
line: 249
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4188693296,nitpick_hash:95f5b525c802
review_hash: 95f5b525c802
source_review_id: "4188693296"
source_review_submitted_at: "2026-04-28T12:24:35Z"
---

# Issue 010: Consider using the returned model from Update() for idiomatic Bubble Tea usage.
## Review Comment

While the current code works because `installWizardModel` uses pointer receivers and mutates in place, idiomatic Bubble Tea code typically uses the returned model value. This makes the test more resilient to future refactoring.

## Triage

- Decision: `VALID`
- Notes: `TestInstallWizardModelTransitions` calls `Update` and discards the returned Bubble Tea model in several places. It passes today because the model mutates in place, but the test would mask a future value-style refactor. Fix by assigning the returned model through a typed test helper.

## Resolution

- Added a typed `updateInstallWizardModel` helper and assigned the returned model for each wizard transition.
- Removed underscore-discarded command returns and asserted command behavior where relevant.
- Verified through targeted CLI tests and `make verify`.
