---
status: resolved
file: internal/task/manager_test.go
line: 197
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58pG6k,comment:PRRC_kwDOR5y4QM66Aom7
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Triage state cleanup is incomplete due to wrong map key deletion.**

Line 197 deletes by raw `taskID`, but `s.triageStates` keys are composite (`taskID|actorKind|actorRef`). This leaves stale triage state after task deletion.


<details>
<summary>Suggested fix</summary>

```diff
-	delete(s.triageStates, taskID)
+	triagePrefix := taskID + "|"
+	for key := range s.triageStates {
+		if strings.HasPrefix(key, triagePrefix) {
+			delete(s.triageStates, key)
+		}
+	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	triagePrefix := taskID + "|"
	for key := range s.triageStates {
		if strings.HasPrefix(key, triagePrefix) {
			delete(s.triageStates, key)
		}
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/manager_test.go` at line 197, The test deletes triage state
using the raw taskID but s.triageStates uses composite keys; update the cleanup
to delete the exact composite key constructed from taskID, actorKind and
actorRef (the same key format used when inserting into s.triageStates) — e.g.,
build the composite key for the entry under test and call delete(s.triageStates,
compositeKey) so the triage state is fully removed.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The in-memory task store used by tests removes triage state by raw task ID even though triage records are keyed by the composite `taskID|actorKind|actorRef` format. That leaves stale triage entries after delete. I will remove all triage entries for the deleted task by key prefix and cover the behavior with delete-path tests in `internal/task/manager_test.go`.
