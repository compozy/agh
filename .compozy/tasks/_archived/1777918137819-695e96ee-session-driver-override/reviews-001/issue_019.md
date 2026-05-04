---
status: resolved
file: web/src/systems/session/components/chat-header.test.tsx
line: 16
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:ecb9bcb2f7c5
review_hash: ecb9bcb2f7c5
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 019: Add a direct assertion for provider UI behavior.
## Review Comment

The fixture now carries `provider`, but the suite still doesn’t explicitly verify provider rendering (badge/label/icon). A focused assertion would prevent silent regressions.

## Triage

- Decision: `valid`
- Root cause: the `ChatHeader` fixture now includes `provider`, but the suite never directly asserts that the provider badge renders correctly, so a provider-display regression could slip through.
- Fix plan: add a focused assertion for rendered provider UI and pair it with the whitespace-guard case from issue 020.
