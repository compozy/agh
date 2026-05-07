---
provider: coderabbit
pr: "120"
round: 2
round_created_at: 2026-05-07T19:41:55.305082Z
status: resolved
file: internal/automation/model/template_test.go
line: 118
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4247165327,nitpick_hash:31ef1bb04f95
review_hash: 31ef1bb04f95
source_review_id: "4247165327"
source_review_submitted_at: "2026-05-07T19:37:05Z"
---

# Issue 007: Use errors.Is to assert the sentinel instead of substring matching.
## Review Comment

`ParseTriggerPromptTemplate` returns `errTriggerPromptTemplateRequired` unwrapped (line 16 of `template.go`), and the test file is in `package model`, so it can reference the unexported sentinel directly. A substring match on `"required"` would silently pass if the sentinel message is renamed or if a completely different error happens to contain `"required"`.

As per coding guidelines, "Use `errors.Is` and `errors.As` only for error type checking in Go — do not use other error assertion patterns."

## Triage

- Decision: `valid`
- Notes:
  - `template_test.go` currently checks the required-input path via substring matching even though the test is in `package model` and can assert the sentinel directly.
  - `ValidateTriggerPromptTemplate` wraps `errTriggerPromptTemplateRequired`, so `errors.Is` is the correct and stable assertion.
  - Fix plan: switch the required-input test coverage to use `errors.Is` for the sentinel and keep substring checks only where they are actually needed.
  - Resolved: required-input coverage now uses `errors.Is` against `errTriggerPromptTemplateRequired`.
