---
status: resolved
file: web/src/systems/session/components/message-markdown.tsx
line: 81
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57IvNy,comment:PRRC_kwDOR5y4QM63_Pzt
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_

## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Missing error handling for clipboard API.**

If `navigator.clipboard.writeText` fails (e.g., permissions denied, non-HTTPS), the UI will still show the "copied" state. Consider handling the promise rejection to avoid misleading feedback.

<details>
<summary>🛡️ Proposed fix</summary>

```diff
  const handleCopy = useCallback(() => {
-   void navigator.clipboard.writeText(code);
-   setCopied(true);
-   clearTimeout(timerRef.current);
-   timerRef.current = setTimeout(() => setCopied(false), COPY_RESET_MS);
+   navigator.clipboard.writeText(code).then(() => {
+     setCopied(true);
+     clearTimeout(timerRef.current);
+     timerRef.current = setTimeout(() => setCopied(false), COPY_RESET_MS);
+   }).catch(() => {
+     // Silently fail or optionally show error feedback
+   });
  }, [code]);
```

</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true);
      clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => setCopied(false), COPY_RESET_MS);
    }).catch(() => {
      // Silently fail or optionally show error feedback
    });
  }, [code]);
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/components/message-markdown.tsx` around lines 76 -
81, The handleCopy callback currently calls navigator.clipboard.writeText
without handling rejections, causing setCopied(true) to run even when copying
fails; update handleCopy to await or attach a .then/.catch to
navigator.clipboard.writeText(code) and only call setCopied(true) and
start/assign the timerRef timeout (COPY_RESET_MS) on success, while on failure
ensure the copied state is not set (or set to false) and optionally log or
surface the error; keep timerRef clearing logic but only start the reset timeout
after a successful write.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `CodeCopyButton` mirrors the same optimistic clipboard logic as `MessageCopyButton`, so rejected clipboard writes still show the copied state and schedule the reset timer.
- Fix approach: Reuse the shared clipboard copy button so code blocks only show success after `navigator.clipboard.writeText(code)` resolves and failures are handled explicitly.
- Resolution: Code blocks now use the shared async `CopyButton`, which keeps the button idle on clipboard failure and logs the error instead of reporting a false success state.
