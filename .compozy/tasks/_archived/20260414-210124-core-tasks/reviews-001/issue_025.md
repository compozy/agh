---
status: resolved
file: internal/network/tasks_integration_test.go
line: 315
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562aon,comment:PRRC_kwDOR5y4QM63mgR9
---

# Issue 025: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Fail if a request writes more than one audit row.**

This helper returns `entries[0]` on any non-empty match, so duplicate audit writes for the same `MessageID` still make the test pass.


<details>
<summary>Suggested fix</summary>

```diff
 func findNetworkAuditByMessageID(t *testing.T, db *globaldb.GlobalDB, messageID string) store.NetworkAuditEntry {
 	t.Helper()

 	entries, err := db.ListNetworkAudit(testutil.Context(t), store.NetworkAuditQuery{
 		MessageID: messageID,
 		Limit:     10,
 	})
 	if err != nil {
 		t.Fatalf("ListNetworkAudit(%q) error = %v", messageID, err)
 	}
-	if len(entries) == 0 {
-		t.Fatalf("ListNetworkAudit(%q) = no entries", messageID)
+	if len(entries) != 1 {
+		t.Fatalf("ListNetworkAudit(%q) returned %d entries, want exactly 1", messageID, len(entries))
 	}
 	return entries[0]
 }
```
</details>
As per coding guidelines, `**/*_test.go`: Ensure tests verify behavior outcomes, not just function calls.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func findNetworkAuditByMessageID(t *testing.T, db *globaldb.GlobalDB, messageID string) store.NetworkAuditEntry {
	t.Helper()

	entries, err := db.ListNetworkAudit(testutil.Context(t), store.NetworkAuditQuery{
		MessageID: messageID,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListNetworkAudit(%q) error = %v", messageID, err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListNetworkAudit(%q) returned %d entries, want exactly 1", messageID, len(entries))
	}
	return entries[0]
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/tasks_integration_test.go` around lines 302 - 315, The
helper findNetworkAuditByMessageID currently returns entries[0] for any
non-empty result; change it to assert exactly one matching audit row to catch
duplicate writes: after calling db.ListNetworkAudit in
findNetworkAuditByMessageID, fail the test if len(entries) == 0 or len(entries)
> 1 (use t.Fatalf with context including messageID and len(entries)), and only
return entries[0] when len(entries) == 1 so tests detect duplicate audit rows;
reference ListNetworkAudit, NetworkAuditQuery, and NetworkAuditEntry when making
the check.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  The helper currently treats any non-empty result set as success and returns the first audit row. That can hide duplicate writes for a single request id and let the test pass even when the system emits multiple audit entries.
  I will tighten the helper to require exactly one matching row so duplicate audit writes fail the integration test instead of being silently accepted.
  Resolution: Updated `findNetworkAuditByMessageID` to require exactly one matching audit row and fail on duplicates as well as missing entries.
