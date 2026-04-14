---
status: resolved
file: internal/cli/task.go
line: 780
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564Lfw,comment:PRRC_kwDOR5y4QM63o2Pi
---

# Issue 008: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Reject half-specified owner filters in `task list`.**

`parseTaskListFilters()` lets `--owner-kind` or `--owner-ref` through on their own, so the CLI pushes an obviously invalid filter to the API instead of failing fast with a local usage error.

<details>
<summary>Suggested fix</summary>

```diff
 func parseTaskListFilters(scopeRaw string, workspaceRef string, statusRaw string, ownerKindRaw string, ownerRef string, parentTaskID string, channelRaw string, last int) (TaskListQuery, error) {
 	scope, workspace, err := resolveTaskScopeWorkspace(scopeRaw, workspaceRef, false)
 	if err != nil {
 		return TaskListQuery{}, err
 	}
 	status, err := parseOptionalTaskStatus(statusRaw)
 	if err != nil {
 		return TaskListQuery{}, err
 	}
 	ownerKind, err := parseOptionalTaskOwnerKind(ownerKindRaw)
 	if err != nil {
 		return TaskListQuery{}, err
 	}
+	trimmedOwnerRef := strings.TrimSpace(ownerRef)
+	if (ownerKind != "" && trimmedOwnerRef == "") || (ownerKind == "" && trimmedOwnerRef != "") {
+		return TaskListQuery{}, errors.New("cli: --owner-kind and --owner-ref must be provided together")
+	}
 	if err := validateTaskChannelFlag("channel", channelRaw); err != nil {
 		return TaskListQuery{}, err
 	}
 	if err := validateTaskLast(last); err != nil {
 		return TaskListQuery{}, err
 	}

 	return TaskListQuery{
 		Scope:          scope,
 		Workspace:      workspace,
 		Status:         status,
 		OwnerKind:      ownerKind,
-		OwnerRef:       strings.TrimSpace(ownerRef),
+		OwnerRef:       trimmedOwnerRef,
 		ParentTaskID:   strings.TrimSpace(parentTaskID),
 		NetworkChannel: strings.TrimSpace(channelRaw),
 		Limit:          last,
 	}, nil
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func parseTaskListFilters(scopeRaw string, workspaceRef string, statusRaw string, ownerKindRaw string, ownerRef string, parentTaskID string, channelRaw string, last int) (TaskListQuery, error) {
	scope, workspace, err := resolveTaskScopeWorkspace(scopeRaw, workspaceRef, false)
	if err != nil {
		return TaskListQuery{}, err
	}
	status, err := parseOptionalTaskStatus(statusRaw)
	if err != nil {
		return TaskListQuery{}, err
	}
	ownerKind, err := parseOptionalTaskOwnerKind(ownerKindRaw)
	if err != nil {
		return TaskListQuery{}, err
	}
	trimmedOwnerRef := strings.TrimSpace(ownerRef)
	if (ownerKind != "" && trimmedOwnerRef == "") || (ownerKind == "" && trimmedOwnerRef != "") {
		return TaskListQuery{}, errors.New("cli: --owner-kind and --owner-ref must be provided together")
	}
	if err := validateTaskChannelFlag("channel", channelRaw); err != nil {
		return TaskListQuery{}, err
	}
	if err := validateTaskLast(last); err != nil {
		return TaskListQuery{}, err
	}

	return TaskListQuery{
		Scope:          scope,
		Workspace:      workspace,
		Status:         status,
		OwnerKind:      ownerKind,
		OwnerRef:       trimmedOwnerRef,
		ParentTaskID:   strings.TrimSpace(parentTaskID),
		NetworkChannel: strings.TrimSpace(channelRaw),
		Limit:          last,
	}, nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/task.go` around lines 751 - 780, parseTaskListFilters currently
allows a half-specified owner filter (only owner-kind or only owner-ref) which
results in invalid API requests; update parseTaskListFilters to validate that
ownerKindRaw and ownerRef are provided together: trim ownerRef
(strings.TrimSpace(ownerRef)) and if one is non-empty while the other is empty
return a usage error (e.g., fmt.Errorf("both --owner-kind and --owner-ref must
be specified together")) before calling parseOptionalTaskOwnerKind and before
building TaskListQuery; ensure the error is returned early so TaskListQuery
never contains a mismatched owner filter.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `parseTaskListFilters()` trims and forwards `owner_kind` and `owner_ref` independently, so the CLI can send a half-specified owner filter that the server will reject later.
  Root cause: unlike create/update paths, the task-list filter parser never enforced the paired owner flag invariant locally.
  Planned fix: require `--owner-kind` and `--owner-ref` together in `parseTaskListFilters()` and add unit coverage for the usage error.

## Resolution

- Tightened `parseTaskListFilters()` in `internal/cli/task.go` to reject half-specified owner filters locally and reuse the trimmed owner ref in the emitted query.
- Added `TestParseTaskListFiltersRejectsHalfSpecifiedOwnerFilter` in `internal/cli/task_test.go` to lock in the CLI usage error.
