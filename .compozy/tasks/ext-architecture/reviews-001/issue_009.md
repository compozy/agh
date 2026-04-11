---
status: resolved
file: internal/cli/cli_integration_test.go
line: 510
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAaL,comment:PRRC_kwDOR5y4QM62zlsW
---

# Issue 009: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Don't hardcode `DaemonRunning` to `true` in the integration extension service.**

`Status()` always calls `DescribeExtension(..., true, ...)`, so this harness can report a healthy daemon-backed extension even when the manager failed to start or the runtime is actually down. That weakens the extension lifecycle assertions in these CLI integration tests.


As per coding guidelines, "MUST test meaningful business logic, not trivial operations".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/cli_integration_test.go` around lines 499 - 510, The Status
method currently passes a hardcoded true for the daemon-running flag to
DescribeExtension, which can falsely report extensions as healthy; update
integrationExtensionService.Status to compute the daemon/running boolean from
real state (e.g., whether s.manager successfully started/connected or a
runtime/health check on the manager/runtime succeeds) and pass that boolean into
extensionpkg.DescribeExtension(ext, daemonRunning, time.Now().UTC()) instead of
true; reference the Status method and extensionpkg.DescribeExtension to locate
the change and ensure any manager errors or runtime-down conditions flip
daemonRunning to false so tests observe real lifecycle state.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes: In this integration harness, `DaemonRunning` is modeling whether the daemon-backed extension API is available, not whether a specific extension process is healthy. The harness creates `integrationExtensionService` only after `extManager.Start()` succeeds, so the reported "manager failed to start but daemon_running stays true" state cannot occur. Extension liveness is already represented through `State` and `Health`; changing `DaemonRunning` here would make the harness diverge from the real daemon service, which treats daemon availability as `runtime != nil`.
