---
status: resolved
file: internal/api/contract/tasks.go
line: 25
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562anN,comment:PRRC_kwDOR5y4QM63mgQP
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# Find and examine the tasks.go file
fd -type f -name 'tasks.go' | head -20
```

Repository: compozy/agh

Length of output: 225

---

🏁 Script executed:

```shell
# Read the specific file mentioned in the review
head -100 internal/api/contract/tasks.go
```

Repository: compozy/agh

Length of output: 4790

---

🏁 Script executed:

```shell
# Check lines around the mentioned line numbers
sed -n '20,50p' internal/api/contract/tasks.go
sed -n '65,75p' internal/api/contract/tasks.go
```

Repository: compozy/agh

Length of output: 2265

---

🏁 Script executed:

```shell
# Search for any custom JSON marshaling in the file
rg -n 'MarshalJSON|UnmarshalJSON' internal/api/contract/
```

Repository: compozy/agh

Length of output: 37

---



</details>

**Use `*time.Time` or custom marshaling for optional timestamps.**

Go's `encoding/json` does not treat zero-valued `time.Time` as empty for `omitempty`, so unset fields like `closed_at`, `claimed_at`, `started_at`, and `ended_at` will serialize as `0001-01-01T00:00:00Z`. This breaks the API contract by leaking sentinel values to clients instead of truly omitting optional fields.

Apply to: `TaskSummaryPayload.ClosedAt` (line 25), `TaskPayload.ClosedAt` (line 44), and `TaskRunPayload.ClaimedAt`, `StartedAt`, `EndedAt` (lines 68-70).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/contract/tasks.go` at line 25, Task timestamp fields are
currently time.Time so omitempty won't work; change TaskSummaryPayload.ClosedAt
and TaskPayload.ClosedAt, and TaskRunPayload.ClaimedAt, StartedAt, EndedAt to
use *time.Time (or implement custom JSON marshaling) so unset values serialize
as omitted. Update any constructors/assignment sites that populate these fields
to take pointers (use &t for existing time values and nil for absent values) and
ensure JSON consumers/validators expect nullable timestamps.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TaskSummaryPayload`, `TaskPayload`, and `TaskRunPayload` model optional timestamps as value `time.Time`, so `omitempty` still serializes zero values as `0001-01-01T00:00:00Z`.
- Fix approach: change optional contract timestamp fields to `*time.Time`, map zero-valued task/task-run timestamps to `nil` in the API conversion helpers, and add JSON-shape regression coverage for omitted timestamps.

## Resolution

- Changed the shared optional task and task-run timestamp payload fields to pointers, mapped zero values to `nil` in the core and extension converters, and added JSON-shape regression coverage for omitted timestamps.
- Applied the minimal required out-of-scope follow-up in `internal/cli/task.go` so the CLI renders the shared pointer-based timestamps correctly after the contract change.
- Verified in the final `make verify` run.
