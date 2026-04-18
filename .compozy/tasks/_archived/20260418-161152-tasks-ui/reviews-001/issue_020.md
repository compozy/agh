---
status: resolved
file: internal/observe/tasks.go
line: 725
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lb4,comment:PRRC_kwDOR5y4QM65B8fX
---

# Issue 020: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`DependencyBlockedTasks` is undercounted for dual-blocked tasks.**

This derives dependency blocking as `BlockedTasks - AwaitingApprovalTasks`, which only works if those buckets are mutually exclusive. A manual-approval task can also have unresolved dependencies, so those tasks disappear from the dependency counter. Count dependency-blocked tasks explicitly from the snapshot instead of subtracting aggregates.

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: Confirmed. `DependencyBlockedTasks` is currently derived as `BlockedTasks - AwaitingApprovalTasks`, which assumes those sets are mutually exclusive. A task can be blocked by unresolved dependencies and also be awaiting approval, so subtraction undercounts real dependency pressure. I’ll count dependency-blocked tasks directly from the task snapshot.
