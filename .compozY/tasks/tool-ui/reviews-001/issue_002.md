---
status: resolved
file: web/src/systems/session/components/message-bubble.tsx
line: 100
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57IvNt,comment:PRRC_kwDOR5y4QM63_Pzm
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_

## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Copy button renders even when user message content is empty.**

The copy button row (lines 93-100) renders outside the `message.content` conditional block (lines 83-92). If `message.content` is an empty string or falsy, the button will still appear and copy empty text. Consider wrapping this row in the same conditional or guard the button itself.

<details>
<summary>🛡️ Proposed fix</summary>

```diff
-           <div className="mt-1.5 flex items-center justify-end gap-2">
-             <MessageCopyButton text={message.content} />
-             {timestamp ? (
-               <span className="text-[10px] text-[color:var(--color-text-tertiary)]/50">
-                 {timestamp}
-               </span>
-             ) : null}
-           </div>
+           {(message.content || timestamp) && (
+             <div className="mt-1.5 flex items-center justify-end gap-2">
+               {message.content && <MessageCopyButton text={message.content} />}
+               {timestamp ? (
+                 <span className="text-[10px] text-[color:var(--color-text-tertiary)]/50">
+                   {timestamp}
+                 </span>
+               ) : null}
+             </div>
+           )}
```

</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/components/message-bubble.tsx` around lines 93 - 100,
The MessageCopyButton is rendered even when message.content is empty; update the
render so the copy button (MessageCopyButton) and its enclosing div (currently
the div with className "mt-1.5 flex items-center justify-end gap-2") are only
output when message.content is non-empty (or alternatively guard
MessageCopyButton with a truthy check on message.content), and keep timestamp
rendering as before; locate the JSX around message.content and the div/timestamp
block in message-bubble.tsx and wrap or conditionally render the
div/MessageCopyButton based on message.content.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: The user-message footer renders unconditionally, while the message body is guarded by `message.content`; that leaves a copy button visible for empty user messages and allows copying an empty string.
- Fix approach: Render the copy control only when `message.content` is non-empty, while keeping the timestamp visible when present.
- Resolution: User-message footers now render the copy control only when content exists, and a regression test covers the empty-message case.
