---
status: pending
file: internal/session/repair.go
line: 173
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-O8E_,comment:PRRC_kwDOR5y4QM68JGPp
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't skip dangling `tool_result` repair just because the turn is already terminal.**

This early return suppresses `RepairActionAppendInterruptedToolResult` too. If a session managed to persist `done`/`error` but crashed before one or more matching `tool_result` rows were written, this path can never close those tool calls.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/repair.go` around lines 166 - 173, The early return inside
the block that checks analysis.turn.terminal (which currently appends a
RepairIssue with Code RepairIssueTerminalEventAlreadyExists) prevents subsequent
dangling tool_result repair actions (e.g.,
RepairActionAppendInterruptedToolResult) from running; remove the return so the
function still records the TerminalEventAlreadyExists issue but continues
executing the rest of the repair logic that scans for and appends interrupted
tool results (look for logic that generates
RepairActionAppendInterruptedToolResult and ensure it runs even when
analysis.turn.terminal is true). Ensure tests cover a case where terminal event
exists but matching tool_result rows are missing so the
append-interrupted-tool-result path executes.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
