---
status: resolved
file: internal/hooks/async_clone.go
line: 260
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:8eb6febd8a21
review_hash: 8eb6febd8a21
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 010: cloneAnyValue still aliases typed nested containers.
## Review Comment

On these lines, the deep copy only recurses through `map[string]any` and `[]any`. Values such as `[]map[string]any`, `map[string][]string`, or any other concrete slice/map stored inside `AutomationTriggerPreFirePayload.Payload` fall into `default` and stay shared, so async hooks can still observe caller mutations.

## Triage

- Decision: `VALID`
- Notes:
  `cloneAnyValue` only recurses through `map[string]any` and `[]any`, so typed
  nested containers such as `[]map[string]any` or `map[string][]string` stay
  aliased inside async hook payload snapshots. Plan: implement deep cloning for
  arbitrary map/slice/array container values and add regression coverage for
  typed nested payloads.
