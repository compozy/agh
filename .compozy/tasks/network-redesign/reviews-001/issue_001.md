---
status: resolved
file: extensions/bridges/whatsapp/provider_test.go
line: 1804
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151559901,nitpick_hash:dc73e086f285
review_hash: dc73e086f285
source_review_id: "4151559901"
source_review_submitted_at: "2026-04-22T01:22:21Z"
---

# Issue 001: Keep the longer timeout local to the flaky path.
## Review Comment

`waitForCondition` is shared across this file, so Line 1804 makes every broken condition take up to 5s to fail. If only one scenario needed extra slack, prefer a per-call timeout (or a dedicated helper) so unrelated regressions still fail fast.

## Triage

- Decision: `invalid`
- Reasoning: `waitForCondition()` is scoped to this integration-only test file, and every current caller waits on subprocess startup, HTTP ingress, file markers, or batched runtime state. The 5s budget is intentional for this whole suite, not just a single call site, so splitting timeouts per helper call would add churn without addressing a demonstrated regression.
- Resolution: no code change. The shared 5s timeout remains intentional for this integration-only WhatsApp provider suite.
- Verification: `make verify`
