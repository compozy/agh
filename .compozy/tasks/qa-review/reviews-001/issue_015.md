---
status: resolved
file: web/src/components/app-sidebar.test.tsx
line: 299
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4176489704,nitpick_hash:07f62ac5d6ff
review_hash: 07f62ac5d6ff
source_review_id: "4176489704"
source_review_submitted_at: "2026-04-26T03:49:14Z"
---

# Issue 015: Always restore console spies, even if the test fails early.
## Review Comment

`mockRestore()` is currently happy-path only. If an assertion throws first, the console mocks can leak into later tests. Use `try/finally`.

## Triage

- Decision: `valid`
- Notes:
  - The console spies in the "opens an agent group..." regression are restored only on the happy path at the end of the test body.
  - Root cause: if an assertion throws before the final lines, the mocked console methods leak into later tests.
  - Fix plan: use `try/finally` around the render/assertion path so both spies are always restored.
