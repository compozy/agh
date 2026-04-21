---
status: resolved
file: internal/transcript/transcript.go
line: 331
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dNA,comment:PRRC_kwDOR5y4QM65IPEV
---

# Issue 030: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep `ToolCallID` as the primary lifecycle key.**

When a call/result pair shares `ToolCallID` but only one side carries `TurnID`, this now generates different keys (`turn:tool` vs `tool`) and the transcript stops merging them. `ToolCallID` is already the stable cross-event correlation key; only scope by `TurnID` when you have to fall back to `parsed.ID`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/transcript/transcript.go` around lines 318 - 331, The current
toolLifecycleKey function uses TurnID whenever present, causing events sharing
ToolCallID but missing TurnID on one side to get different keys; change it so
ToolCallID (parsed.ToolCallID) is the primary key: if parsed.ToolCallID
(trimmed) is non-empty return that (ignore TurnID), otherwise fall back to
parsed.ID and, only in that fallback case, prepend TurnID if present (i.e.,
return turn + ":" + id when turn exists, else id); update function
toolLifecycleKey and references to parsed.ToolCallID/parsed.ID accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `toolLifecycleKey` still prepends `TurnID` whenever present, even when `ToolCallID` is available. That splits a tool call/result pair across two lifecycle keys when only one side carries `TurnID`, so the transcript can no longer correlate the pair on the stable `ToolCallID`.
- Fix approach: make trimmed `ToolCallID` the primary lifecycle key, and only scope by `TurnID` when falling back to `parsed.ID` because no `ToolCallID` exists.
