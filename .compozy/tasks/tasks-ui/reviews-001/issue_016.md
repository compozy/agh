---
status: resolved
file: internal/extension/host_api_integration_test.go
line: 667
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lby,comment:PRRC_kwDOR5y4QM65B8fP
---

# Issue 016: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Inbox assertion is order-dependent and can be flaky.**

The check assumes the target task is always at `groups[0].items[0]`. If ordering changes, this test can fail even when the task is present.

<details>
<summary>✅ More stable assertion</summary>

```diff
-	if got, want := inbox.Groups[0].Items[0].Task.ID, approvalTask.ID; got != want {
-		t.Fatalf("tasks/inbox.groups[0].items[0].task.id = %q, want %q", got, want)
-	}
+	foundApprovalTask := false
+	for _, group := range inbox.Groups {
+		for _, item := range group.Items {
+			if item.Task.ID == approvalTask.ID {
+				foundApprovalTask = true
+				break
+			}
+		}
+		if foundApprovalTask {
+			break
+		}
+	}
+	if !foundApprovalTask {
+		t.Fatalf("tasks/inbox groups = %#v, want task %q", inbox.Groups, approvalTask.ID)
+	}
```

</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	foundApprovalTask := false
	for _, group := range inbox.Groups {
		for _, item := range group.Items {
			if item.Task.ID == approvalTask.ID {
				foundApprovalTask = true
				break
			}
		}
		if foundApprovalTask {
			break
		}
	}
	if !foundApprovalTask {
		t.Fatalf("tasks/inbox groups = %#v, want task %q", inbox.Groups, approvalTask.ID)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_integration_test.go` around lines 665 - 667, The
current assertion assumes the approval task is exactly at
inbox.Groups[0].Items[0] which is order-dependent; instead search through
inbox.Groups and their Items for an item whose Task.ID equals approvalTask.ID
(or use a helper like findTaskInInbox) and assert that such an item exists;
update the test around inbox.Groups/Items/Task.ID to perform a loop or use a
predicate that returns true if any group's items contain the expected
approvalTask.ID, and fail the test only if no match is found.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: Confirmed. The inbox assertion assumes the approval task will always be the first item of the first group. That is unnecessarily order-dependent and can fail if grouping or sorting changes while the payload remains correct. I’ll change the test to search for the expected task ID across all groups.
