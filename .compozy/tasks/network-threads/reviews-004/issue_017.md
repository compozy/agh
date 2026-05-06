---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/extension/manager.go
line: 1646
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_0DsE,comment:PRRC_kwDOR5y4QM6-RRZK
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Deep-clone the new nested matcher pointers too.**

These new `NetworkMatcher` / `CompactionMatcher` pointers are now part of every built hook, but `cloneHookDecl` later in this file still only deep-copies `ToolReadOnly`. Any caller mutating a cloned `HookDecl` can now mutate manager-owned matcher state through shared pointers.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/extension/manager.go` around lines 1636 - 1646, cloneHookDecl
currently only deep-clones ToolReadOnly causing shared pointer mutation for the
newly added matcher fields; update cloneHookDecl (the function that clones
HookDecl) to also deep-copy the NetworkMatcher and CompactionMatcher nested
pointers (and any other matcher fields added to the HookDecl struct) so each
clone gets its own instances rather than sharing the manager-owned pointers set
on matcher (the variable assigning NetworkMatcher and CompactionMatcher).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `internal/extension/manager.go:2385-2395` clones hook declarations but only deep-copies `ToolReadOnly`. The new embedded `NetworkMatcher` and `CompactionMatcher` pointers remain aliased to manager-owned state. Deep-copy those nested matchers and cover the behavior in an extension-package test.
