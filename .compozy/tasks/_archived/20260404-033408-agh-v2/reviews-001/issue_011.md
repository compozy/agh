---
status: resolved
file: internal/acp/handlers.go
line: 352
severity: medium
author: claude-code
provider_ref:
---

# Issue 011: Terminal commands from agent executed without sandboxing

## Review Comment

`handleCreateTerminal` (line 286) takes `request.Command` and `request.Args` from the ACP agent and passes them directly to `exec.CommandContext` at line 352. Although terminal creation is behind a permission check (`permissionCreateTerminal`), when `approve-all` mode is set, the agent can execute *any* command with the daemon's full privileges. The terminal's Cwd is resolved via `resolvePath` (sandbox to workspace), but the command itself has no restrictions -- an agent could run `rm -rf /` or exfiltrate data via network commands.

**Suggested fix:** Document this as an explicit security boundary in the permission model. Consider adding command allowlists for non-`approve-all` modes, or at minimum log all terminal commands at `warn` level for audit.

## Triage

- Decision: `invalid`
- Notes: This is an explicit trust boundary in the current ACP terminal permission model, not a hidden implementation bug. `handleCreateTerminal()` already requires `permissionCreateTerminal`, and `approve-all` intentionally delegates command execution authority to the agent. Logging or documenting that boundary could be useful, but the review does not identify an incorrect behavior in the scoped code.
