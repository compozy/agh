---
status: pending
file: internal/cli/session_test.go
line: 354
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-O8E5,comment:PRRC_kwDOR5y4QM68JGPi
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wrap this test case in a `t.Run("Should...")` subtest.**

The current test is parallelized, but it misses the required subtest wrapper pattern used by your Go test guidelines.

<details>
<summary>🔧 Suggested update</summary>

```diff
 func TestSessionRepairPassesFlagsAndRendersJSON(t *testing.T) {
 	t.Parallel()
 
-	var seenQuery SessionRepairQuery
-	var seenID string
-	deps := newTestDeps(t, &stubClient{
-		repairSessionFn: func(_ context.Context, id string, query SessionRepairQuery) (SessionRepairRecord, error) {
-			seenID = id
-			seenQuery = query
-			return SessionRepairRecord{
-				SessionID: id,
-				Issues: []SessionRepairIssueRecord{{
-					Code:     session.RepairIssueStopReasonRequiresForce,
-					Severity: session.RepairSeverityError,
-					TurnID:   "turn-1",
-				}},
-				Actions: []SessionRepairActionRecord{{
-					Code:      session.RepairActionAppendTerminalError,
-					TurnID:    "turn-1",
-					Persisted: false,
-				}},
-			}, nil
-		},
-	})
-
-	stdout, _, err := executeRootCommand(
-		t,
-		deps,
-		"session",
-		"repair",
-		"sess-1",
-		"--dry-run",
-		"--force",
-		"-o",
-		"json",
-	)
-	if err != nil {
-		t.Fatalf("executeRootCommand(session repair) error = %v", err)
-	}
-	if seenID != "sess-1" || !seenQuery.DryRun || !seenQuery.Force {
-		t.Fatalf("repair call = id %q query %#v, want dry-run force for sess-1", seenID, seenQuery)
-	}
-
-	var decoded SessionRepairRecord
-	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
-		t.Fatalf("json.Unmarshal(session repair) error = %v", err)
-	}
-	if decoded.SessionID != "sess-1" || len(decoded.Issues) != 1 || len(decoded.Actions) != 1 {
-		t.Fatalf("decoded repair = %#v, want one issue and one action for sess-1", decoded)
-	}
+	t.Run("ShouldPassFlagsAndRenderJSON", func(t *testing.T) {
+		t.Parallel()
+
+		var seenQuery SessionRepairQuery
+		var seenID string
+		deps := newTestDeps(t, &stubClient{
+			repairSessionFn: func(_ context.Context, id string, query SessionRepairQuery) (SessionRepairRecord, error) {
+				seenID = id
+				seenQuery = query
+				return SessionRepairRecord{
+					SessionID: id,
+					Issues: []SessionRepairIssueRecord{{
+						Code:     session.RepairIssueStopReasonRequiresForce,
+						Severity: session.RepairSeverityError,
+						TurnID:   "turn-1",
+					}},
+					Actions: []SessionRepairActionRecord{{
+						Code:      session.RepairActionAppendTerminalError,
+						TurnID:    "turn-1",
+						Persisted: false,
+					}},
+				}, nil
+			},
+		})
+
+		stdout, _, err := executeRootCommand(
+			t,
+			deps,
+			"session",
+			"repair",
+			"sess-1",
+			"--dry-run",
+			"--force",
+			"-o",
+			"json",
+		)
+		if err != nil {
+			t.Fatalf("executeRootCommand(session repair) error = %v", err)
+		}
+		if seenID != "sess-1" || !seenQuery.DryRun || !seenQuery.Force {
+			t.Fatalf("repair call = id %q query %#v, want dry-run force for sess-1", seenID, seenQuery)
+		}
+
+		var decoded SessionRepairRecord
+		if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
+			t.Fatalf("json.Unmarshal(session repair) error = %v", err)
+		}
+		if decoded.SessionID != "sess-1" || len(decoded.Issues) != 1 || len(decoded.Actions) != 1 {
+			t.Fatalf("decoded repair = %#v, want one issue and one action for sess-1", decoded)
+		}
+	})
 }
```
</details>

As per coding guidelines: "MUST use t.Run("Should...") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/session_test.go` around lines 304 - 354, Wrap the existing test
body of TestSessionRepairPassesFlagsAndRendersJSON in a t.Run subtest (e.g.
t.Run("Should pass flags and render JSON", func(t *testing.T) { ... })) and move
the t.Parallel() call inside that subtest; remove the top-level t.Parallel() so
the test follows the required subtest pattern. Ensure the existing variables and
calls (seenQuery, seenID, newTestDeps, stubClient.repairSessionFn,
executeRootCommand, json.Unmarshal, and assertions) remain inside the subtest
closure so behavior and scoping are preserved.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
