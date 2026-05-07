---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/acp/terminal.go
line: 113
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYRr3,comment:PRRC_kwDOR5y4QM6-67Dz
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Remove external ownership records when the terminal is gone.**

`recordTerminalOwnership` only adds entries. This file never deletes them on registration rollback, kill, wait completion, or release, so long-lived agents will leak `p.terminalOwnership` and keep authorizing dead terminal IDs until restart.


<details>
<summary>Possible fix</summary>

```diff
+func (p *AgentProcess) deleteTerminalOwnership(id string) {
+	if strings.TrimSpace(id) == "" {
+		return
+	}
+	p.terminalOwnershipMu.Lock()
+	defer p.terminalOwnershipMu.Unlock()
+	delete(p.terminalOwnership, id)
+}
+
 func (p *AgentProcess) handleCreateTerminal(
 	ctx context.Context,
 	request acpsdk.CreateTerminalRequest,
 ) (acpsdk.CreateTerminalResponse, error) {
@@
 	p.recordTerminalOwnership(response.TerminalId, ownership)
 	if err := p.registerExternalTerminalProcess(ctx, host, response.TerminalId, request, ownership); err != nil {
+		p.deleteTerminalOwnership(response.TerminalId)
 		if killErr := host.KillTerminal(response.TerminalId); killErr != nil {
@@
 	p.completeExternalTerminalProcess(
 		context.Background(),
 		request.TerminalId,
 		toolruntime.ProcessCompletion{Err: errors.New("terminal killed")},
 	)
+	p.deleteTerminalOwnership(request.TerminalId)
 	return acpsdk.KillTerminalCommandResponse{}, nil
 }
```
</details>


Also applies to: 117-128, 229-233, 289-293, 308-312

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/acp/terminal.go` around lines 103 - 113, recordTerminalOwnership
only adds entries and leaks terminal IDs; add removal logic to clean up
ownership when registration fails, on kill/cleanup, after wait completion, and
on release. Implement and use a helper (e.g., removeTerminalOwnership or
p.removeTerminalOwnership(terminalID)) and call it in the error branch after
p.registerExternalTerminalProcess fails (the block that calls host.KillTerminal
and returns), in the kill/cleanup path, in the wait-completion path, and in the
release handler so p.terminalOwnership no longer retains dead terminal IDs;
ensure the same helper is used wherever recordTerminalOwnership was previously
added.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  External-host network terminal ownership is recorded but never removed on create rollback, kill, or release. That leaks authorization entries and stale IDs. The fix needs an ownership-delete helper and explicit cleanup on teardown paths. Cleanup must not happen on `wait`, because the protocol still needs post-exit `terminal_output` and `release` access.
