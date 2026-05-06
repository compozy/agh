---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/cli/memory.go
line: 552
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Isv,comment:PRRC_kwDOR5y4QM6-UFV4
---

# Issue 010: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Unused variable `includeSystem` assigned but never used**

The `includeSystem` flag is captured on line 536 but then assigned to blank identifier on line 552 (`_ = includeSystem`). This suggests the feature is not yet implemented. Consider either implementing the functionality or removing the flag to avoid confusion.




<details>
<summary>🔧 Either implement or remove the unused flag</summary>

If the feature is not yet available, consider removing the flag registration:
```diff
-cmd.Flags().BoolVar(&includeSystem, "include-system", false, "Include _system memory state")
```

Or if it should be passed to the API:
```diff
-_ = includeSystem
 response, err := client.ResetMemory(cmd.Context(), MemoryResetRequest{
 	Scope:       selector.Scope,
 	WorkspaceID: selector.WorkspaceID,
 	AgentName:   selector.AgentName,
 	AgentTier:   selector.AgentTier,
+	IncludeSystem: includeSystem,
 	DerivedOnly: !includeDaily,
 	Confirm:     !dryRun,
 })
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/cli/memory.go` at line 552, The includeSystem boolean flag is
captured but then discarded with `_ = includeSystem`; either remove the flag
registration entirely or actually use includeSystem when querying/filtering
memories. Locate the flag variable `includeSystem` and either delete its
declaration and any flag registration code, or thread it into the memory
listing/query call (e.g., pass `includeSystem` into the function that fetches
memories such as the memory query/list function) and ensure the query logic
respects the flag to include or exclude system memories.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `newMemoryResetCommand` still registers `--include-system`, stores it in `includeSystem`, and then discards it with `_ = includeSystem`.
  - The current daemon request surface (`contract.MemoryResetRequest`) has no `include_system` field, so the flag is not "temporarily unused" by design; it is dead CLI surface that cannot affect behavior.
  - Fix approach: remove the unused flag and dead local variable. This likely requires a minimal command-test update outside the scoped file list if existing CLI tests cover the flag inventory.
