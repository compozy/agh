---
status: resolved
file: internal/cli/state.go
line: 183
severity: medium
author: claude-reviewer
---

# Issue 103: agent-status command ambiguously interprets single positional argument



## Review Comment

The `newAgentStatusCommand` at line 183 uses a single positional argument `<task-or-agent>` and tries to determine whether the user wants to update their own status or read another agent's status. The logic at lines 221-222 is:

```go
target, targetErr := findSessionAgent(detail.Agents, args[0])
readMode := targetErr == nil && target.ID != caller.ID
```

This means if the caller passes a task description that happens to match another agent's name or ID, the CLI will silently interpret it as a "read" command instead of an "update" command. For example, if the user runs `agh agent-status exec-1` intending to update their own task to "exec-1", but there is an agent named "exec-1" in the session, it will instead read that agent's status.

Note that the `findSessionAgent` function now accepts variadic `refs` (`findSessionAgent(agents []kernel.SessionAgentResponse, refs ...string)`), and the caller lookup at line 216 passes multiple identifiers (`callerEnv.AgentID, callerEnv.AgentName, callerRef`), but the target lookup at line 221 still passes only `args[0]`.

This is a design ambiguity that could cause subtle, surprising behavior. Consider adding an explicit `--read <agent>` flag or using subcommands (`agent-status update <task>` vs `agent-status read <agent>`) to disambiguate.

## Triage

- Decision: `valid`
- Notes: Confirmed in `newAgentStatusCommand`: with caller context present, a single positional argument is treated as a read if it matches another agent in the session, otherwise it updates the caller's task. That makes the behavior dependent on ambient session state and can silently reinterpret a task string as an agent lookup. This needs explicit read-mode selection to remove the ambiguity.
