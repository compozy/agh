---
status: resolved
file: internal/api/contract/tasks.go
line: 72
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lbt,comment:PRRC_kwDOR5y4QM65B8fK
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Add `draft` to `TaskPayload`.**

`CreateTask`, `UpdateTask`, `PublishTask`, `CancelTask`, `ApproveTask`, and `RejectTask` all return `TaskResponse`, but this payload still drops the draft flag even though the new request/list/detail shapes carry it. That means the UI needs an extra GET just to know whether the returned task is still a draft.

<details>
<summary>Suggested contract fix</summary>

```diff
 type TaskPayload struct {
 	ID             string                 `json:"id"`
 	Identifier     string                 `json:"identifier,omitempty"`
 	Scope          taskpkg.Scope          `json:"scope"`
 	WorkspaceID    string                 `json:"workspace_id,omitempty"`
 	ParentTaskID   string                 `json:"parent_task_id,omitempty"`
 	NetworkChannel string                 `json:"network_channel,omitempty"`
 	Title          string                 `json:"title"`
 	Description    string                 `json:"description,omitempty"`
 	Priority       taskpkg.Priority       `json:"priority,omitempty"`
 	MaxAttempts    int                    `json:"max_attempts,omitempty"`
 	Status         taskpkg.Status         `json:"status"`
 	ApprovalPolicy taskpkg.ApprovalPolicy `json:"approval_policy,omitempty"`
 	ApprovalState  taskpkg.ApprovalState  `json:"approval_state,omitempty"`
+	Draft          bool                   `json:"draft,omitempty"`
 	Owner          *taskpkg.Ownership     `json:"owner,omitempty"`
 	CreatedBy      taskpkg.ActorIdentity  `json:"created_by"`
 	Origin         taskpkg.Origin         `json:"origin"`
 	CreatedAt      time.Time              `json:"created_at"`
 	UpdatedAt      time.Time              `json:"updated_at"`
 	ClosedAt       *time.Time             `json:"closed_at,omitempty"`
 	Metadata       json.RawMessage        `json:"metadata,omitempty"`
 }
```
</details>

Also mirror this in `internal/api/core/tasks.go:TaskPayloadFromTask`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/contract/tasks.go` around lines 51 - 72, The TaskPayload struct
is missing the draft flag so responses returned by
CreateTask/UpdateTask/PublishTask/CancelTask/ApproveTask/RejectTask lose draft
state; add a Draft bool `json:"draft,omitempty"` field to
internal/api/contract/tasks.go:TaskPayload and update
internal/api/core/tasks.go:TaskPayloadFromTask to set TaskPayload.Draft =
task.Draft (or equivalent) when converting from the domain Task so all
TaskResponse payloads include the draft value.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `contract.TaskPayload` currently omits `draft`, and `internal/api/core/tasks.go:TaskPayloadFromTask` therefore cannot propagate it. Mutation responses that return `TaskResponse` lose draft state even though list/detail payloads already expose it. Fix requires a minimal out-of-scope mapper edit in `internal/api/core/tasks.go` because that converter is the production call site.
