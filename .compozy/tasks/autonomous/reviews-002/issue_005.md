---
status: resolved
file: internal/api/core/conversions_parsers_test.go
line: 327
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59q6tc,comment:PRRC_kwDOR5y4QM67YhqA
---

# Issue 005: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert schedule deep-copy semantics explicitly.**

Line 325 verifies task/owner pointers are not reused, but schedule pointer reuse is not asserted. A regression in `Schedule` copy behavior would currently pass.


<details>
<summary>Suggested test hardening</summary>

```diff
 		if payload.Schedule == nil || payload.Schedule.Interval != "10m" {
 			t.Fatalf("schedule payload = %#v", payload.Schedule)
 		}
+		if payload.Schedule == &schedule {
+			t.Fatal("JobPayloadFromJob reused schedule input pointer")
+		}
 		if payload.Task == nil || payload.Task.Owner == nil || payload.Task.Owner.Ref != "triage" {
 			t.Fatalf("task payload = %#v", payload.Task)
 		}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		if payload.Schedule == nil || payload.Schedule.Interval != "10m" {
			t.Fatalf("schedule payload = %#v", payload.Schedule)
		}
		if payload.Schedule == &schedule {
			t.Fatal("JobPayloadFromJob reused schedule input pointer")
		}
		if payload.Task == nil || payload.Task.Owner == nil || payload.Task.Owner.Ref != "triage" {
			t.Fatalf("task payload = %#v", payload.Task)
		}
		if payload.Task == &jobTask || payload.Task.Owner == &owner {
			t.Fatal("JobPayloadFromJob reused nested input pointers")
		}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/conversions_parsers_test.go` around lines 319 - 327, The
test fails to assert that Schedule was deep-copied; add an explicit
pointer-inequality check for payload.Schedule to ensure JobPayloadFromJob
doesn't reuse the input Schedule pointer (e.g., assert payload.Schedule !=
&job.Schedule or compare to the original schedule variable), and also optionally
verify key fields (like Interval) still equal to confirm content was copied;
update the test near the existing task/owner pointer checks (which reference
payload.Task, jobTask, and owner) to include this Schedule pointer check.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestJobPayloadFromJobCopiesNestedOptionalFields` verifies the copied schedule content but not pointer independence. A regression that reuses the input `Schedule` pointer would pass while task/owner pointer reuse is already covered. The fix is to add the missing schedule pointer-inequality assertion.
- Resolution: Added the missing schedule pointer-inequality assertion and verified via focused tests plus full `make verify`.
