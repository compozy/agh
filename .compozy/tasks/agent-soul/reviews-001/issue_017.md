---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: pending
file: internal/cli/authored_context.go
line: 382
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_Irda,comment:PRRC_kwDOR5y4QM69XbzZ
---

# Issue 017: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Rename Heartbeat CAS flags to `--expected-digest`.**

These commands still publish `--if-match`, while the authored-context contract and the new Soul/session commands have already standardized on `expected_digest`. Shipping both terms in the new CLI surface bakes in an avoidable alias and makes the heartbeat workflow look transport-specific again.

Please rename the heartbeat flags/help/examples/tests to `--expected-digest` in this PR.  

As per coding guidelines, "Renames must update code, storage, APIs, CLI, extensions, specs, RFCs, and `.compozy/tasks/*` artifacts in a single change; no aliases or dual fields."


Also applies to: 385-418, 458-519

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/authored_context.go` around lines 340 - 382, The CLI flag and
usages for Heartbeat CAS should be renamed from "--if-match" to
"--expected-digest": update the Cobra command in newAgentHeartbeatWriteCommand
to change the Example string, the flag registration cmd.Flags().StringVar(...)
(currently using "if-match") to "expected-digest", and any reads of that flag
(the call to optionalStringFlag(cmd, "if-match", expectedDigest)) to use
"expected-digest"; also update the flag name and help text variable
expectedDigest and any related commands/handlers in the same file (the other
heartbeat commands referenced around the 385-418 and 458-519 ranges) as well as
tests/examples to use --expected-digest consistently (no alias).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
