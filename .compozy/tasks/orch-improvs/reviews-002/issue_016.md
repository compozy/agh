---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/store/globaldb/global_db_notification_cursor.go
line: 235
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-j,comment:PRRC_kwDOR5y4QM6-VcDD
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Make `RecordCursorError` atomic with its readback.**

This writes through `ExecContext(...)` and then reloads the row in a separate statement on `g.db`. A concurrent `AdvanceCursor` or `ResetCursor` can interleave between those calls, so the returned cursor may no longer reflect the error this method just recorded. Using the same single-connection transaction pattern as the other cursor mutations, or returning the row from the write itself, would close that gap.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/store/globaldb/global_db_notification_cursor.go` around lines 205 -
235, RecordCursorError currently does an ExecContext to write the error and then
calls loadNotificationCursor in a separate statement, allowing
AdvanceCursor/ResetCursor to interleave; make the write-and-read atomic by
performing both operations on the same DB transaction/connection. Modify
RecordCursorError to begin a transaction or acquire a single connection (using
BeginTx/Conn) and run the INSERT ... ON CONFLICT and the subsequent
loadNotificationCursor logic within that transaction (or use INSERT ...
RETURNING to read and return the row directly), ensuring you call
loadNotificationCursor against the same tx/conn and commit before returning;
reference RecordCursorError, loadNotificationCursor, AdvanceCursor, and
ResetCursor in the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `RecordCursorError` writes on `g.db` and then reloads the row in a separate statement, so concurrent cursor mutations can interleave and change the returned row.
- Fix approach: Make the write/read path atomic on a single immediate transaction/connection and add regression coverage in `internal/store/globaldb/global_db_notification_cursor_test.go`.
