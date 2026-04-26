---
status: resolved
file: internal/hooks/matcher_test.go
line: 458
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vM,comment:PRRC_kwDOR5y4QM67Z0NH
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Rename subtests to the required `Should...` pattern.**

The table-driven structure is good; please update case names (Lines 443-447) so each `t.Run` uses `Should...` wording.

<details>
<summary>Example rename</summary>

```diff
- {name: "session workspace root", event: HookSessionPostCreate, field: "workspace_root", want: true},
+ {name: "ShouldAllowWorkspaceRootForSessionHooks", event: HookSessionPostCreate, field: "workspace_root", want: true},
```
</details>


As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
        {name: "ShouldAllowWorkspaceRootForSessionPostCreateHook", event: HookSessionPostCreate, field: "workspace_root", want: true},
        {name: "ShouldAllowWorkspaceIdForTaskRunEnqueuedHook", event: HookTaskRunEnqueued, field: "workspace_id", want: true},
        {name: "ShouldDenyWorkspaceRootForTaskRunEnqueuedHook", event: HookTaskRunEnqueued, field: "workspace_root", want: false},
        {name: "ShouldDenyWorkspaceIdForMessageDeltaHook", event: HookMessageDelta, field: "workspace_id", want: false},
        {name: "ShouldDenyInvalidEvent", event: HookEvent("bad.event"), field: "workspace_id", want: false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            if got := MatcherFieldAllowedForEvent(tt.event, tt.field); got != tt.want {
                t.Fatalf("MatcherFieldAllowedForEvent(%q, %q) = %v, want %v", tt.event, tt.field, got, tt.want)
            }
        })
    }
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/hooks/matcher_test.go` around lines 443 - 458, Update the t.Run case
names in the table-driven test for MatcherFieldAllowedForEvent so they follow
the "Should..." pattern; specifically rename the entries currently named
"session workspace root", "task run workspace id", "task run workspace root",
"message workspace id", and "invalid event" to descriptive "Should..." strings
(e.g., "Should allow session workspace root", "Should allow task run workspace
id", "Should not allow task run workspace root", "Should not allow message
workspace id", "Should not allow invalid event") so t.Run uses the required
format while keeping the same event constants (HookSessionPostCreate,
HookTaskRunEnqueued, HookMessageDelta, HookEvent("bad.event")) and field values
("workspace_root", "workspace_id") used by MatcherFieldAllowedForEvent.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestMatcherFieldAllowedForEvent` is already table-driven, but its `name` values do not follow the mandatory `Should ...` convention. Fix by renaming the five cases only.
