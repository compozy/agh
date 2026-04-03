---
status: resolved
file: internal/drivers/pi/pi.go
line: 313
severity: medium
author: claude-reviewer
---

# Issue 148: Pi TypeScript extension uses synchronous execSync blocking the Node.js event loop



## Review Comment

The Pi driver generates a TypeScript extension that uses `execSync` to forward hook events:

```typescript
export default function (pi) {
  const { execSync } = require("child_process");
  const forward = async (event) => {
    execSync("agh hook-event --agent %s", {
      input: JSON.stringify(event),
    });
  };

  pi.on("tool_execution_start", async (event) => {
    await forward(event);
  });
  ...
}
```

There are two problems:

1. `execSync` blocks the Node.js event loop. If the `agh hook-event` command takes time (network latency, slow processing), it will freeze Pi's TUI and all pending I/O. This could degrade the user experience significantly.

2. The `forward` function is declared `async` but uses synchronous `execSync` inside it. The `async` keyword is misleading and provides no benefit - `execSync` will still block.

**Suggested fix**: Use `execFile` (async, no shell) instead of `execSync` (sync, with shell). This also eliminates the command injection risk from issue 140:

```typescript
const { execFile } = require("child_process");
const forward = (event) => {
  const child = execFile("agh", ["hook-event", "--agent", agentName]);
  child.stdin.write(JSON.stringify(event));
  child.stdin.end();
};
```

## Triage

- Decision: `valid`
- Notes:
  - The generated Pi extension currently blocks on `execFileSync`, so every forwarded hook event stalls the Node event loop until `agh hook-event` completes.
  - This is a concrete runtime defect, not a style preference: slow hook ingestion can freeze the Pi UI and delay subsequent extension callbacks.
  - The fix should make forwarding asynchronous and avoid shell/path ambiguity while preserving the existing hook payload contract.
  - Resolution: updated the generated Pi extension to use non-blocking `execFile(...)` with a resolved hook binary path and added regression coverage in `internal/drivers/pi/pi_test.go`.
