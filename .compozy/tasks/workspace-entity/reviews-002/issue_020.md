---
status: resolved
file: internal/session/transcript_test.go
line: 257
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCh,comment:PRRC_kwDOR5y4QM61T6H8
---

# Issue 020: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cd /tmp && find . -type f -name "transcript_test.go" 2>/dev/null | head -5
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Find the file in the repository
fd -t f "transcript_test.go"
```

Repository: compozy/agh

Length of output: 90

---

🏁 Script executed:

```shell
# Read the specific test function
if [ -f "internal/session/transcript_test.go" ]; then
  sed -n '217,257p' internal/session/transcript_test.go
fi
```

Repository: compozy/agh

Length of output: 1411

---

🏁 Script executed:

```shell
rg -n "firstNonEmptyRaw|firstNonNil" internal/session/transcript_test.go
```

Repository: compozy/agh

Length of output: 376

---

🏁 Script executed:

```shell
# Check if there are separate tests for these helpers
rg -n "func Test.*[Ff]irst" internal/session/transcript_test.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check if there are other related tests in the file
rg -n "func Test" internal/session/transcript_test.go | head -20
```

Repository: compozy/agh

Length of output: 280

---

🏁 Script executed:

```shell
# Verify the coding guidelines apply to this file
ls -la internal/session/transcript_test.go
```

Repository: compozy/agh

Length of output: 139

---



</details>

**Refactor this test to use table-driven subtests with `t.Run`**

This test mixes multiple independent test cases (tool result parsing and two helper function assertions) into a single linear flow. Refactor to table-driven subtests so each behavior is isolated and independently parallelizable, following the guideline: "Use table-driven tests with subtests (`t.Run`) as default in Go tests".

<details>
<summary>Suggested structure</summary>

```diff
 func TestParseLooseTranscriptEventBuildsToolResultFromLoosePayload(t *testing.T) {
 	t.Parallel()
-
-	event := parseLooseTranscriptEvent(transcriptEvent{Type: acp.EventTypeToolResult}, map[string]any{
-		"type":         acp.EventTypeToolResult,
-		"tool_call_id": "call-loose",
-		"title":        "Bash",
-		"rawInput": map[string]any{
-			"command": "pwd",
-		},
-		"rawOutput": map[string]any{
-			"stdout": "workspace\n",
-		},
-	})
-
-	if got := event.ToolCallID; got != "call-loose" {
-		t.Fatalf("ToolCallID = %q, want %q", got, "call-loose")
-	}
-	if got := event.ToolName; got != "Bash" {
-		t.Fatalf("ToolName = %q, want %q", got, "Bash")
-	}
-	if got := string(event.ToolInput); got != `{"command":"pwd"}` {
-		t.Fatalf("ToolInput = %s, want JSON command payload", got)
-	}
-	if event.ToolResult == nil {
-		t.Fatal("ToolResult = nil, want populated result")
-	}
-	if got := event.ToolResult.Stdout; got != "workspace\n" {
-		t.Fatalf("ToolResult.Stdout = %q, want %q", got, "workspace\n")
-	}
-	if event.ToolError {
-		t.Fatal("ToolError = true, want false")
-	}
-
-	if got := string(firstNonEmptyRaw(nil, json.RawMessage(`{"ok":true}`))); got != `{"ok":true}` {
-		t.Fatalf("firstNonEmptyRaw() = %s, want non-empty raw payload", got)
-	}
-	if got := firstNonNil(nil, "", "value"); got != "" {
-		t.Fatalf("firstNonNil(nil, \"\", \"value\") = %#v, want empty string first", got)
-	}
+	tests := []struct {
+		name string
+		run  func(t *testing.T)
+	}{
+		{
+			name: "builds tool result from loose payload",
+			run: func(t *testing.T) {
+				t.Parallel()
+				event := parseLooseTranscriptEvent(transcriptEvent{Type: acp.EventTypeToolResult}, map[string]any{
+					"type":         acp.EventTypeToolResult,
+					"tool_call_id": "call-loose",
+					"title":        "Bash",
+					"rawInput":     map[string]any{"command": "pwd"},
+					"rawOutput":    map[string]any{"stdout": "workspace\n"},
+				})
+				// existing assertions...
+			},
+		},
+		{
+			name: "firstNonEmptyRaw returns first non-empty raw",
+			run: func(t *testing.T) {
+				t.Parallel()
+				if got := string(firstNonEmptyRaw(nil, json.RawMessage(`{"ok":true}`))); got != `{"ok":true}` {
+					t.Fatalf("firstNonEmptyRaw() = %s, want non-empty raw payload", got)
+				}
+			},
+		},
+		{
+			name: "firstNonNil returns first non-nil value",
+			run: func(t *testing.T) {
+				t.Parallel()
+				if got := firstNonNil(nil, "", "value"); got != "" {
+					t.Fatalf("firstNonNil(nil, \"\", \"value\") = %#v, want empty string first", got)
+				}
+			},
+		},
+	}
+
+	for _, tc := range tests {
+		tc := tc
+		t.Run(tc.name, tc.run)
+	}
 }
```

</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/transcript_test.go` around lines 217 - 257, Split
TestParseLooseTranscriptEventBuildsToolResultFromLoosePayload into table-driven
subtests using t.Run: create a slice of test cases (name, setup, expected)
covering the main behavior of parseLooseTranscriptEvent (validate ToolCallID,
ToolName, ToolInput, ToolResult.Stdout, ToolError) as one subtest, and separate
subtests for firstNonEmptyRaw and firstNonNil helper assertions; inside each
subtest call t.Parallel() and assert only that case’s expectations, invoking
parseLooseTranscriptEvent, firstNonEmptyRaw, or firstNonNil as needed and
comparing results to expected values so failures are isolated and parallelizable
while keeping the overall test name
TestParseLooseTranscriptEventBuildsToolResultFromLoosePayload.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  - This is another structural refactor suggestion rather than a defect.
  - The existing test already runs in parallel, exercises the loose tool-result parsing path directly, and includes focused assertions for the two helper functions.
  - Rewriting it into table-driven subtests would add ceremony without increasing correctness or coverage in a meaningful way.
