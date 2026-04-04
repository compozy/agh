---
status: resolved
file: internal/session/manager.go
line: 521
severity: medium
author: claude-code
provider_ref:
---

# Issue 008: TOCTOU race between state check and driver.Prompt call

## Review Comment

In `Manager.Prompt` (line 521), the code checks `session.Info().State != StateActive` and then `session.processHandle()`, each under separate lock acquisitions. Between these checks and the actual `m.driver.Prompt(ctx, proc, ...)` call, the session could transition to stopping/stopped and the process could be cleared by a concurrent `finalizeStopped`. This means a prompt could be sent to a process that is in the middle of shutting down.

**Suggested fix:** Perform the state check, process retrieval, and marking as "prompting" in a single lock scope, or use an atomic flag that prevents concurrent stop while a prompt is inflight.

## Triage

- Decision: `valid`
- Notes: `Manager.Prompt()` checks `session.Info().State` and `session.processHandle()` in separate unsynchronized steps before calling `driver.Prompt()`. A concurrent `Stop()` can move the session into shutdown after those checks but before prompt startup, so a new prompt can still be initiated against a session that is already stopping.
