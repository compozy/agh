---
provider: coderabbit
pr: "106"
round: 3
round_created_at: 2026-05-06T06:28:14.497092Z
status: resolved
file: internal/bridges/task_notifier.go
line: 23
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4233729248,nitpick_hash:e4d6a0aa8c4c
review_hash: e4d6a0aa8c4c
source_review_id: "4233729248"
source_review_submitted_at: "2026-05-06T06:27:47Z"
---

# Issue 001: Move the default sweep limit out of code.
## Review Comment

`defaultTerminalTaskNotifierLimit` hardcodes an operational tuning knob into the binary. This notifier is new infrastructure, so the default should come from config/env and `EventLimit` should only override it when explicitly supplied.

As per coding guidelines, Never hardcode configuration values in Go code; always read from `config.toml` or environment variables

Also applies to: 80-83

## Triage

- Decision: `invalid`
- Root cause analysis: `TerminalTaskNotifier` already exposes the sweep bound through `TerminalTaskNotifierConfig.EventLimit`; the constant at line 23 is only the constructor fallback when the caller leaves that field unset.
- Why this is invalid: the review asks for a new repo-wide config/env surface, but the current code is already overrideable via the constructor and there is no correctness or safety defect in the scoped implementation itself. In AGH, built-in defaults commonly live in Go code until a config lifecycle change is explicitly designed and wired end-to-end.
- Scope note: introducing a new `task.orchestration` config key would require non-scoped config/docs/test updates outside this batch and would be a product-surface expansion rather than a localized review remediation.

## Resolution

- Analysis completed. No code change was made for this item.

## Verification

- Fresh full gate after the batch fixes: `make verify` exited `0` in this session.
