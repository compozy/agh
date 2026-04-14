---
status: resolved
file: internal/cli/task_test.go
line: 611
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564Lfi,comment:PRRC_kwDOR5y4QM63o2PP
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`relates` is a valid dependency kind, so this negative test is inverted.**

`parseOptionalTaskDependencyKind` explicitly accepts `blocks`, `blocked_by`, and `relates`. This assertion currently expects an error for a supported value, so it will fail once exercised.


<details>
<summary>💡 Suggested fix</summary>

```diff
-	if _, err := parseOptionalTaskDependencyKind("relates"); err == nil || !strings.Contains(err.Error(), "unsupported value") {
-		t.Fatalf("parseOptionalTaskDependencyKind(relates) error = %v, want unsupported value validation", err)
-	}
+	if kind, err := parseOptionalTaskDependencyKind("relates"); err != nil || kind != taskpkg.DependencyKindRelates {
+		t.Fatalf("parseOptionalTaskDependencyKind(relates) = (%q, %v), want relates", kind, err)
+	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if kind, err := parseOptionalTaskDependencyKind("blocks"); err != nil || kind != taskpkg.DependencyKindBlocks {
		t.Fatalf("parseOptionalTaskDependencyKind(blocks) = (%q, %v), want blocks", kind, err)
	}
	if kind, err := parseOptionalTaskDependencyKind("relates"); err != nil || kind != taskpkg.DependencyKindRelates {
		t.Fatalf("parseOptionalTaskDependencyKind(relates) = (%q, %v), want relates", kind, err)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/task_test.go` around lines 607 - 611, The test is inverted:
parseOptionalTaskDependencyKind accepts "relates" so the negative assertion
should be changed to expect success; update the second assertion in the test to
call parseOptionalTaskDependencyKind("relates") and assert err == nil and kind
== taskpkg.DependencyKindRelates (or the appropriate constant), instead of
checking for an "unsupported value" error, so both supported values ("blocks"
and "relates") are validated as successful parses.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  This review comment is stale relative to the current task domain. `parseOptionalTaskDependencyKind` delegates to `taskpkg.DependencyKind.Validate`, and the current enum only supports `blocks`.
  Evidence:
  `internal/task/types.go` defines only `DependencyKindBlocks`.
  `internal/task/validate.go` accepts only `DependencyKindBlocks` and returns `unsupported value` for everything else.
  `internal/cli/task_test.go` is therefore correct to keep `"relates"` as the negative regression case.
  No production or test change is required for this issue.

## Resolution

- No code change was made. The current dependency-kind enum only supports `blocks`, so keeping `"relates"` as the unsupported-value regression case is correct.
