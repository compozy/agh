---
status: resolved
file: internal/api/core/tasks_surface_test.go
line: 657
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:cd31cbfb58b4
review_hash: cd31cbfb58b4
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 014: Add a negative-path test for POST /tasks/{id}/start.
## Review Comment

This file adds happy-path coverage for the new start endpoint, but the error-path suite never exercises it. A regression in start-specific conflict/not-found mapping would currently slip through.

As per coding guidelines, "Focus on critical paths: workflow execution, state management, error handling".

Also applies to: 855-1058

## Triage

- Decision: `VALID`
- Notes: `TestBaseHandlersExpandedTaskEndpointErrorPaths` covers publish and approve conflicts but omits `/tasks/{id}/start`, even though start has its own handler and response mapping. Fix by wiring `StartTaskFn` to return a conflict and asserting `POST /tasks/task-1/start` maps it to HTTP 409.
