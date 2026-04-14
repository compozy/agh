---
status: resolved
file: internal/automation/model/validate.go
line: 351
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107787502,nitpick_hash:05ee35958032
review_hash: 05ee35958032
source_review_id: "4107787502"
source_review_submitted_at: "2026-04-14T17:02:50Z"
---

# Issue 006: Reuse the shared channel validator here.
## Review Comment

`JobTaskConfig` now hardcodes its own regex, but task transport validation goes through `internal/network.ValidateChannel` in `internal/api/core/tasks.go`. If those rules drift, automation config can pass model validation and still fail when the delegated task is materialized. Prefer calling the shared validator instead of maintaining a second pattern.

## Triage

- Decision: `valid`
- Notes:
  `JobTaskConfig.Validate` currently maintains its own channel regex while task transport validation uses `internal/network.ValidateChannel`, so the two validation paths can drift. I will replace the local regex check with the shared network validator and preserve the existing field-path context in the returned error.
  Resolution: Removed the duplicated regex and moved the channel grammar into a shared leaf helper under `internal/network/rules`, so both network validation and automation-model validation use the same rule without introducing a package cycle.
