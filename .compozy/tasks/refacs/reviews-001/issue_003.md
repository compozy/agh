---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/acp/terminal.go
line: 153
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYRr9,comment:PRRC_kwDOR5y4QM6-67D5
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cd internal/acp && wc -l terminal.go
```

Repository: compozy/agh

Length of output: 71

---

🏁 Script executed:

```shell
cd internal/acp && sed -n '140,160p' terminal.go | cat -n
```

Repository: compozy/agh

Length of output: 744

---

🏁 Script executed:

```shell
cd internal/acp && sed -n '400,465p' terminal.go | cat -n
```

Repository: compozy/agh

Length of output: 2554

---

🏁 Script executed:

```shell
cd internal/acp && sed -n '1,50p' terminal.go | cat -n
```

Repository: compozy/agh

Length of output: 1240

---



</details>

**Detach registry writes from request cancellation.**

Both registration paths run after the terminal/process is created. If the RPC context is cancelled here, `Register` can fail while the terminal persists, leaving an orphaned process with no registry record. Use `context.WithoutCancel(ctx)` for registry calls and re-apply any deadline you need.

Also applies to: lines 408–409, 456–459.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/acp/terminal.go` around lines 144 - 153, The registry Register call
is using the incoming RPC ctx which can be cancelled and leave an orphaned
process; change the registration context creation to detach cancellation by
using context.WithoutCancel(ctx) (or equivalent) to create registerCtx, then
re-apply any deadline from the original ctx if present before calling
p.processRegistry.Register (the registerCtx used for
toolruntime.RegisterConfig). Apply the same fix for the other registry calls
shown (the other p.processRegistry.Register usages near the terminal/process
creation paths) so registry writes are not aborted by request cancellation.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  Both external and managed terminal registration happen after the process exists. Using the request context means a canceled RPC can abort registry persistence and orphan a live process. The registration calls should detach cancellation while preserving any original deadline.
