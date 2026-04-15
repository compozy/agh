---
status: resolved
file: web/src/systems/session/components/message-markdown.tsx
line: 66
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4113957510,nitpick_hash:8f197ac7936c
review_hash: 8f197ac7936c
source_review_id: "4113957510"
source_review_submitted_at: "2026-04-15T13:28:12Z"
---

# Issue 003: Consider extracting shared copy button logic.

## Review Comment

`CodeCopyButton` here and `MessageCopyButton` in `message-bubble.tsx` share nearly identical logic (state, timer cleanup, icon swap). Consider extracting a reusable `CopyButton` component to reduce duplication.

## Triage

- Decision: `valid`
- Root cause: `MessageCopyButton` and `CodeCopyButton` duplicate the same clipboard state/timer logic, which is why the same clipboard failure bug exists in both files in this batch.
- Fix approach: Extract a shared session `CopyButton` component so the clipboard success/failure handling, timer cleanup, and icon state are implemented once. This requires one new component file outside the original six scoped code files; that is the minimum change that removes the duplicated root cause.
- Resolution: Extracted `web/src/systems/session/components/copy-button.tsx` and switched both message and code-copy surfaces to it so the clipboard behavior now lives in one tested implementation.
