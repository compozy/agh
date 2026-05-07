---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/acp/terminal.go
line: 396
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYRsD,comment:PRRC_kwDOR5y4QM6-67ED
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Honor `OutputByteLimit` for local terminals.**

The request carries `OutputByteLimit` into `create`, but `appendOutput` always trims against the hardcoded 64 KiB default. Local terminals therefore ignore the caller's requested retention window and can return/truncate the wrong amount of output.


<details>
<summary>Possible fix</summary>

```diff
 type managedTerminal struct {
 	id string
 
 	cmd           *exec.Cmd
 	processRecord *toolruntime.Handle
+	outputLimit   int
@@
 func (m *terminalManager) create(
 	ctx context.Context,
 	cwd string,
 	request acpsdk.CreateTerminalRequest,
 	ownership terminalOwnership,
 ) (acpsdk.CreateTerminalResponse, error) {
@@
+	outputLimit := defaultTerminalOutputLimit
+	if request.OutputByteLimit != nil {
+		outputLimit = *request.OutputByteLimit
+		if outputLimit < 0 {
+			outputLimit = 0
+		}
+	}
+
 	term := &managedTerminal{
 		id:             fmt.Sprintf("term-%d", m.nextID.Add(1)),
 		cmd:            cmd,
+		outputLimit:    outputLimit,
 		networkOwned:   ownership.networkOwned,
@@
 func (t *managedTerminal) appendOutput(p []byte) {
 	t.mu.Lock()
 	defer t.mu.Unlock()
 	var truncated bool
-	t.output, truncated = appendTerminalOutputWindow(t.output, p, defaultTerminalOutputLimit)
+	t.output, truncated = appendTerminalOutputWindow(t.output, p, t.outputLimit)
 	if truncated {
 		t.truncated = true
 	}
 }
```
</details>


Also applies to: 589-596

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/acp/terminal.go` around lines 380 - 396, Local terminals ignore the
caller's request.OutputByteLimit because appendOutput uses a hardcoded 64 KiB;
modify the terminal creation to pass the request.OutputByteLimit into the
terminal/writer (managedTerminal and terminalOutputWriter) and update
appendOutput to trim using that configured limit instead of the constant; locate
where managedTerminal is constructed (the block creating term with id, cmd,
networkOwned, ownerSessionID/ownerTurnID and done) and add a field for
outputByteLimit sourced from request.OutputByteLimit, wire that through to
terminalOutputWriter, and change appendOutput to reference term.outputByteLimit
(or writer.outputByteLimit) so both local terminal creation sites use the
caller's limit.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  Local managed terminals currently always trim against `defaultTerminalOutputLimit`, even when the caller supplies `OutputByteLimit`. That ignores the request contract for local terminals and needs a per-terminal configured output limit.
