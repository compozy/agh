---
status: resolved
file: internal/drivers/codex/codex.go
line: 306
severity: low
author: claude-reviewer
---

# Issue 157: Codex driver's hookEndpoint parameter is silently ignored



## Review Comment

The `BuildHookConfig` method accepts a `hookEndpoint` parameter but explicitly discards it:

```go
func (d *CodexDriver) BuildHookConfig(agentName string, hookEndpoint string) (*kernel.HookConfig, error) {
    name := strings.TrimSpace(agentName)
    if name == "" {
        return nil, errors.New("codex: agent name is required")
    }
    _ = hookEndpoint
    ...
}
```

The `_ = hookEndpoint` blank identifier assignment indicates the parameter is intentionally unused. The same pattern exists in the Pi driver at `pi.go:319` and the Claude driver at `claude.go:369`.

While the `AgentDriver` interface requires this parameter, silently ignoring it without documentation is misleading. If the hook endpoint is provided to enable HTTP-based hook forwarding as an alternative to CLI-based forwarding, the driver should either use it or document why it's not applicable.

**Suggested fix**: Add a comment explaining why the hookEndpoint is unused for this driver (e.g., "Codex hooks are forwarded via CLI command execution, not HTTP endpoints"). If the hookEndpoint could be used to construct the hook command URL, consider supporting it.

## Triage

- Decision: `invalid`
- Notes:
  - The unused `hookEndpoint` parameter is a consequence of the shared `AgentDriver` interface, not a runtime bug in the Codex driver.
  - Codex hook delivery here is file-based and CLI-forwarded; the endpoint value is not part of the current transport path, and ignoring it does not break hook execution.
  - Adding comments or alternate endpoint support would be documentation/interface cleanup, not a scoped correctness fix.
  - Resolution: closed as interface/documentation feedback rather than a production defect.
