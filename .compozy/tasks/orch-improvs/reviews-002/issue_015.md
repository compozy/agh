---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/store/globaldb/global_db_notification_cursor.go
line: 116
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-i,comment:PRRC_kwDOR5y4QM6-VcDB
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cat -n internal/store/globaldb/global_db_notification_cursor.go
```

Repository: compozy/agh

Length of output: 17625

---

🏁 Script executed:

```shell
rg -n "func rollbackImmediate" --type go
```

Repository: compozy/agh

Length of output: 281

---

🏁 Script executed:

```shell
sed -n '1,50p' internal/store/globaldb/tx_helpers.go
```

Repository: compozy/agh

Length of output: 1120

---

🏁 Script executed:

```shell
rg -n "context.WithoutCancel" --type go -A 3 -B 1 | head -80
```

Repository: compozy/agh

Length of output: 4834

---



</details>

**Reattach deadline to rollback context after stripping cancellation.**

`context.WithoutCancel(ctx)` strips the deadline. If the rollback connection wedges, `conn.ExecContext(rollbackCtx, "ROLLBACK")` will block indefinitely without a timeout. Reattach a deadline: use `context.WithTimeout(context.WithoutCancel(ctx), timeout)` before passing to `rollbackImmediate(...)`, matching the pattern used elsewhere in the codebase (e.g., `workspace/resolver.go:238`, `cli_integration_test.go:2850`).

Also applies to: 170-178

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/store/globaldb/global_db_notification_cursor.go` around lines 108 -
116, The rollbackCtx created with context.WithoutCancel(ctx) strips the deadline
so rollbackImmediate/conn.ExecContext can block indefinitely; change the code
that sets rollbackCtx in the notification cursor advance path (currently using
rollbackCtx, joinCleanupError and calling rollbackImmediate for "notification
cursor advance") to reattach a deadline by wrapping WithoutCancel with a timeout
(e.g. context.WithTimeout(context.WithoutCancel(ctx), timeout)) and pass that
bounded context into rollbackImmediate/conn.ExecContext; apply the same fix to
the other occurrence around lines 170-178 so all rollbackImmediate calls use a
deadline-bound context.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: the rollback contexts in `AdvanceCursor` and `ResetCursor` are detached with `context.WithoutCancel` but never get a replacement timeout, so rollback can block indefinitely.
- Fix approach: Reattach a bounded rollback timeout in both cursor mutation paths and cover the behavior in `internal/store/globaldb/global_db_notification_cursor_test.go`.
