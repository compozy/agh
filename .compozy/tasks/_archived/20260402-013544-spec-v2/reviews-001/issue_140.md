---
status: resolved
file: internal/drivers/pi/pi.go
line: 313
severity: critical
author: claude-reviewer
---

# Issue 140: Command injection via unsanitized agent name in hook config generation (pi and codex drivers)



## Review Comment

**Partially fixed:** The Claude driver's `BuildHookConfig` (claude.go:365) now calls `kernel.ValidateAgentName(name)` which enforces the regex `^[A-Za-z0-9_-]+$`, preventing injection. However, the Pi and Codex drivers have **not** been updated with the same validation.

The Pi driver's `BuildHookConfig` (pi.go:313) only checks that the name is non-empty before injecting it directly into a JavaScript/TypeScript template string using `fmt.Sprintf`:

```go
func (d *PiDriver) BuildHookConfig(agentName string, hookEndpoint string) (*kernel.HookConfig, error) {
    name := strings.TrimSpace(agentName)
    if name == "" {
        return nil, errors.New("pi: agent name is required")
    }

    _ = hookEndpoint

    script := fmt.Sprintf(`export default function (pi) {
  const { execSync } = require("child_process");
  const forward = async (event) => {
    execSync("agh hook-event --agent %s", {
      input: JSON.stringify(event),
    });
  };
```

The Codex driver's `BuildHookConfig` (codex.go:300) has the same issue -- it only checks for non-empty before interpolating the name into a shell command string in hooks.json:

```go
func (d *CodexDriver) BuildHookConfig(agentName string, hookEndpoint string) (*kernel.HookConfig, error) {
    name := strings.TrimSpace(agentName)
    if name == "" {
        return nil, errors.New("codex: agent name is required")
    }
    ...
    "command": fmt.Sprintf("agh hook-event --agent %s", name),
```

While `kernel.NewStartOpts` (called in `Start`) does validate the name via `validateAgentName` with the strict regex, `BuildHookConfig` can be called independently of `Start`. If called directly with an unsanitized name containing shell metacharacters, it would produce injectable output.

**Suggested fix**: Replace the non-empty check in Pi and Codex `BuildHookConfig` with `kernel.ValidateAgentName(name)`, consistent with the Claude driver. Additionally, the Pi driver should use array-form `execFileSync` instead of string-form `execSync` to avoid shell interpretation entirely.

## Triage

- Decision: `valid`
- Notes: Confirmed for the scoped `pi` driver: `BuildHookConfig` only checks for a non-empty agent name and then interpolates it into a shell command string inside the generated TypeScript extension. `kernel.NewStartOpts` validates names during `Start`, but `BuildHookConfig` is callable independently and should enforce the same validation itself. The review also mentions Codex, but that file is outside this batch scope.
