---
status: resolved
file: internal/daemon/automation_resources.go
line: 26
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:f1fd9bf84f21
review_hash: f1fd9bf84f21
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 039: Avoid any(...) for this type assertion
## Review Comment

You can assert directly from `runtime` without widening to `any`.

As per coding guidelines: `Never use interface{}/any when a concrete type is known`.

## Triage

- Decision: `VALID`
- Notes: `automationResourceTarget` still widens `runtime` to `any` before asserting it to `automationResourceProjectorTarget`, even though Go supports the interface-to-interface assertion directly. This is a small cleanup, but it is a concrete code-style issue against the project rule to avoid `any` when the concrete interface type is already known. The behavior check lives in a minimal out-of-scope unit test file, `internal/daemon/automation_resources_test.go`, because the scoped daemon test file is integration-tagged.
