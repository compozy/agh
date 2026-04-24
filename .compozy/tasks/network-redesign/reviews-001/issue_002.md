---
status: resolved
file: internal/acp/types.go
line: 396
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIeF,comment:PRRC_kwDOR5y4QM66CAkm
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Bound deferred tool-result buffering.**

`pendingToolResults` now grows until a matching tool call arrives or the turn finishes. Because ACP session updates are agent-originated and flow straight into `emitPromptEvent`, a noisy or misordered agent can keep sending unique `ToolCallID`s here and retain them for the full prompt. Please cap or dedupe this buffer by `ToolCallID` so one bad session cannot grow memory without bound.



Also applies to: 587-599

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/types.go` around lines 395 - 396, pendingToolResults can grow
unbounded because unique ToolCallID values are appended until a matching tool
call arrives; update the buffering logic around pendingToolResults and
seenToolCalls (and the code paths that push into pendingToolResults, e.g.,
emitPromptEvent and the related handler at the other location referenced) to
dedupe by ToolCallID using seenToolCalls (skip adding if already seen) and
enforce a hard cap (e.g., maxPendingToolResults constant) so when the cap is
reached you drop oldest entries or refuse new ones; ensure the dedupe check uses
the AgentEvent.ToolCallID (or equivalent field) and that seenToolCalls is kept
in sync when removing entries.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `activePromptState.deferToolResultLocked()` appends every unseen deferred tool result and never bounds `pendingToolResults`. A misordered or noisy agent can therefore retain unbounded `ToolCallID` state until the prompt ends.
- Fix plan: dedupe deferred tool results per `ToolCallID`, add a hard cap with oldest-entry eviction, keep the deferred-ID bookkeeping in sync when entries are flushed or dropped, and add ACP tests that cover duplicate buffering and cap behavior.
- Resolution: added deferred `ToolCallID` dedupe, a hard `maxPendingToolResults` cap with oldest-entry eviction, synchronized pending-ID bookkeeping across flush and drop paths, and ACP regression coverage for duplicate and capped buffering.
- Verification: `go test ./internal/acp` and `make verify`
