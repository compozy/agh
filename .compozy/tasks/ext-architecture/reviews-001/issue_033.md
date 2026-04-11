---
status: resolved
file: internal/skills/loader.go
line: 52
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:56e2c9de28d8
review_hash: 56e2c9de28d8
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 033: Add context.Context to the new disk-backed parser entrypoint.
## Review Comment

Line 52 introduces a new file-reading API without cancellation support. Please make this `ParseSkillFileWithSource(ctx context.Context, path string, source SkillSource)` and thread `ctx` through the read/parse path to keep runtime-boundary calls cancelable.

As per coding guidelines, `Pass context.Context as first argument to functions crossing runtime boundaries — avoid context.Background() outside main and focused tests`.

## Triage

- Decision: `invalid`
- Notes:
  `ParseSkillFileWithSource` is a synchronous parsing helper, not the runtime boundary itself, and its only new I/O is `os.ReadFile`, which is not context-cancelable. Threading `context.Context` through this helper would add signature churn across callers without materially improving cancellation behavior.
  The higher-level manager and registry flows already carry context at the orchestration boundary where cancellation decisions belong.
