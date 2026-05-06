---
provider: coderabbit
pr: "113"
round: 3
round_created_at: 2026-05-06T21:26:19.336933Z
status: resolved
file: web/src/systems/agent/components/stories/agent-command-select.stories.tsx
line: 29
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4239683335,nitpick_hash:d03f90234852
review_hash: d03f90234852
source_review_id: "4239683335"
source_review_submitted_at: "2026-05-06T21:25:45Z"
---

# Issue 001: Use camelCase for the local variable and map to backend category_path explicitly.
## Review Comment

On line 29, `category_path` is a local variable that should follow camelCase naming. When returning the object, explicitly map the renamed variable to the backend field `category_path`.

## Triage

- Decision: `valid`
- Notes:
  - The story helper currently uses a snake_case local variable (`category_path`) inside TypeScript code, which violates the web package's camelCase naming convention for local identifiers.
  - The underlying backend field name must remain `category_path` on `AgentPayload`, so the fix is to rename only the local variable and map it back explicitly in the returned object.
  - No dedicated test file change was warranted because this is a story-only naming refactor with no behavior change; fresh full-repo verification passed with `make verify`.
