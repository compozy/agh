---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/bridges/task_notifier.go
line: 597
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-L,comment:PRRC_kwDOR5y4QM6-VcCk
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**UTF-8 byte truncation may produce invalid strings.**

Truncating at a fixed byte offset can split a multi-byte UTF-8 character, yielding an invalid string. Consider using `utf8.RuneCountInString` or iterating runes to find a safe cut point.




<details>
<summary>🐛 Proposed fix using rune-safe truncation</summary>

```diff
+import "unicode/utf8"
+
 func truncateTerminalTaskCursorError(value string) string {
     trimmed := strings.TrimSpace(value)
     if len(trimmed) <= maxTerminalTaskCursorErrorBytes {
         return trimmed
     }
-    return trimmed[:maxTerminalTaskCursorErrorBytes]
+    // Find the last valid rune boundary at or before the limit
+    for i := maxTerminalTaskCursorErrorBytes; i > 0; {
+        r, size := utf8.DecodeLastRuneInString(trimmed[:i])
+        if r != utf8.RuneError || size == 1 {
+            return trimmed[:i]
+        }
+        i -= size
+    }
+    return ""
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/bridges/task_notifier.go` around lines 591 - 597, The current
truncateTerminalTaskCursorError truncates by byte index and can split multi-byte
UTF-8 runes; update truncateTerminalTaskCursorError to perform rune-safe
truncation by iterating the string with a for-range to get rune start byte
indices and cut at the last rune boundary that does not exceed
maxTerminalTaskCursorErrorBytes (use the range index as the slice boundary).
Ensure you still TrimSpace at the start and return the full trimmed string if
its byte length is within the limit, otherwise return trimmed[:safeCutIndex].
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `truncateTerminalTaskCursorError` slices by byte count and can cut through a multi-byte rune, producing invalid UTF-8 in persisted cursor diagnostics.
- Fix approach: Make truncation rune-safe at the byte limit and add regression coverage in `internal/bridges/task_notifier_test.go`.
