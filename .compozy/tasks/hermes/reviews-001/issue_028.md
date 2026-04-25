---
status: resolved
file: internal/session/prompt_activity.go
line: 21
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:b37417ef54f2
review_hash: b37417ef54f2
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 028: Avoid hardcoded stop timeout in runtime supervision path.
## Review Comment

Line 21 introduces a fixed `5s` deadline that overrides runtime configuration when stopping timed-out sessions. This makes stop behavior non-configurable and inconsistent with the rest of supervision settings.

As per coding guidelines, "Never hardcode configuration — use TOML config or functional options."

Also applies to: 268-269

## Triage

- Decision: `valid`
- Root cause: `prompt_activity.go` uses the hardcoded `runtimeTimeoutStopDeadline = 5 * time.Second` for the forced stop after runtime inactivity timeout. Operators can configure the cancel grace, but cannot influence this stop deadline, so timeout supervision has a fixed runtime behavior outside the TOML-backed supervision settings.
- Fix approach: remove the fixed deadline and derive the forced-stop context deadline from the runtime supervision configuration, falling back to the default supervision config only when a test or caller constructs an invalid zero value.
