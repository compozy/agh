---
status: resolved
file: web/src/systems/session/components/message-bubble.tsx
line: 45
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57IvNq,comment:PRRC_kwDOR5y4QM63_Pzi
---

# Issue 001: _⚠️ Potential issue_ | _🟡 Minor_

## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Same clipboard error handling concern as `CodeCopyButton`.**

The voided promise pattern doesn't handle clipboard failures. When extracting a shared `CopyButton` component, include proper error handling.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/components/message-bubble.tsx` around lines 40 - 45,
The handleCopy implementation currently voids navigator.clipboard.writeText and
doesn't handle failures; update handleCopy (used in message-bubble.tsx and in
the shared CopyButton/CodeCopyButton) to await or attach a .then/.catch to
navigator.clipboard.writeText(text), only call setCopied(true) and start/clear
timerRef on success, and in the catch branch setCopied(false) and surface or log
the error (e.g., processLogger/console.error or a UI fallback) so clipboard
permission or write failures are handled gracefully; keep useCallback and
dependencies (text, timerRef, COPY_RESET_MS) consistent when you refactor into
the shared CopyButton.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `MessageCopyButton` sets the optimistic copied state immediately after calling `navigator.clipboard.writeText(text)` and ignores promise rejection with `void`, so the UI reports success even when the Clipboard API fails.
- Fix approach: Move the shared clipboard interaction into a reusable session copy button, await the clipboard write, only flip to the copied state after success, and log failures without starting the reset timer.
- Resolution: Introduced a shared session `CopyButton` that awaits clipboard writes, logs failures, and only starts the reset timer after success. Added direct regression tests for success and failure paths.
