---
status: resolved
file: internal/config/automation.go
line: 258
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4106878777,nitpick_hash:e6581020c011
review_hash: e6581020c011
source_review_id: "4106878777"
source_review_submitted_at: "2026-04-14T14:46:54Z"
---

# Issue 018: Avoid shallow-copying JobTaskConfig.
## Review Comment

`taskCfg := *j.Task` only copies the top-level struct, so nested pointers remain shared between the parsed config and the runtime job. A small `cloneJobTaskConfig` helper would prevent runtime mutations from leaking back into config state.

## Triage

- Decision: `valid`
- Root cause: `taskCfg := *j.Task` only clones the top-level `JobTaskConfig`; nested pointer fields like `Owner` remain shared between parsed config state and runtime job state.
- Fix approach: add a local deep-clone helper for parsed task config and use it in `toAutomationJob`.

## Resolution

- Added a local task-config clone helper that also deep-copies the nested owner pointer before wiring parsed automation jobs into runtime state.
- Verified in the final `make verify` run.
