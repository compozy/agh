---
status: resolved
file: internal/drivers/opencode/opencode.go
line: 1062
severity: medium
author: claude-reviewer
---

# Issue 159: OpenCode defaultHookForwarder executes external command without path validation



## Review Comment

The `defaultHookForwarder` function executes `agh hook-event` as an external command:

```go
func defaultHookForwarder(ctx context.Context, agentName string, env []string, rawPayload []byte) error {
    ...
    command := exec.CommandContext(ctx, "agh", "hook-event", "--agent", agentName)
    command.Env = append([]string(nil), env...)
    command.Stdin = bytes.NewReader(rawPayload)
    output, err := command.CombinedOutput()
    ...
}
```

This uses a relative binary name `"agh"` which is resolved via PATH lookup. The command inherits the custom environment (`env`), which could potentially include a modified PATH. If an attacker can control environment variables passed to the agent (via `EnvVars` in `StartOpts`), they could redirect `agh` to a malicious binary.

Unlike the Pi driver's `execSync` (issue 140), this code does use the array form of `exec.CommandContext` which avoids shell interpretation. However, the agent name is still passed as a raw argument, and the environment inheritance is a concern.

**Suggested fix**: Resolve the `agh` binary path once at driver construction time using `exec.LookPath` and store the absolute path. This prevents PATH manipulation attacks via the custom environment.

## Triage

- Decision: `valid`
- Notes:
  - `defaultHookForwarder` currently resolves `agh` through `PATH` while also accepting caller-provided environment overrides, including `PATH` itself.
  - That means a manipulated agent environment can redirect hook forwarding to the wrong executable, which is a real integrity/security defect.
  - The fix should resolve the hook command path up front and invoke the absolute binary path during forwarding.
  - Resolution: OpenCode now resolves the hook binary path at driver construction time and invokes that absolute path during forwarding, with regression coverage in `internal/drivers/opencode/opencode_test.go`.
