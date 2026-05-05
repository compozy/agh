---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/mcp_auth_test.go
line: 93
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulKg,comment:PRRC_kwDOR5y4QM680KJG
---

# Issue 021: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use `t.Run("Should ...")` subtests for each assertion group.**

Line 46 introduces a monolithic test body; per repo test policy, this should be split into `t.Run("Should ...")` subtests (with `t.Parallel()` in each where safe) for deterministic structure and better failure isolation.



<details>
<summary>Proposed refactor</summary>

```diff
 func TestMCPAuthStatusBundlesRenderHumanAndToon(t *testing.T) {
 	t.Parallel()
@@
-	bundle := mcpAuthStatusBundle(status)
-	human, err := bundle.human()
-	if err != nil {
-		t.Fatalf("mcpAuthStatusBundle.human() error = %v", err)
-	}
-	if !strings.Contains(human, "linear") || !strings.Contains(human, "Refreshable") {
-		t.Fatalf("mcp auth status human = %q, want status rows", human)
-	}
-	toon, err := bundle.toon()
-	if err != nil {
-		t.Fatalf("mcpAuthStatusBundle.toon() error = %v", err)
-	}
-	if !strings.Contains(toon, "mcp_auth") || !strings.Contains(toon, "read|write") {
-		t.Fatalf("mcp auth status toon = %q, want toon fields", toon)
-	}
-
-	listBundle := mcpAuthStatusListBundle([]mcpauth.Status{status})
-	listHuman, err := listBundle.human()
-	if err != nil {
-		t.Fatalf("mcpAuthStatusListBundle.human() error = %v", err)
-	}
-	if !strings.Contains(listHuman, "MCP Auth") || !strings.Contains(listHuman, "linear") {
-		t.Fatalf("mcp auth list human = %q, want status table", listHuman)
-	}
-	listToon, err := listBundle.toon()
-	if err != nil {
-		t.Fatalf("mcpAuthStatusListBundle.toon() error = %v", err)
-	}
-	if !strings.Contains(listToon, "mcp_auth[1]") {
-		t.Fatalf("mcp auth list toon = %q, want toon table", listToon)
-	}
+	t.Run("Should render single status in human format", func(t *testing.T) {
+		t.Parallel()
+		bundle := mcpAuthStatusBundle(status)
+		human, err := bundle.human()
+		if err != nil {
+			t.Fatalf("mcpAuthStatusBundle.human() error = %v", err)
+		}
+		if !strings.Contains(human, "linear") || !strings.Contains(human, "Refreshable") {
+			t.Fatalf("mcp auth status human = %q, want status rows", human)
+		}
+	})
+
+	t.Run("Should render single status in toon format", func(t *testing.T) {
+		t.Parallel()
+		bundle := mcpAuthStatusBundle(status)
+		toon, err := bundle.toon()
+		if err != nil {
+			t.Fatalf("mcpAuthStatusBundle.toon() error = %v", err)
+		}
+		if !strings.Contains(toon, "mcp_auth") || !strings.Contains(toon, "read|write") {
+			t.Fatalf("mcp auth status toon = %q, want toon fields", toon)
+		}
+	})
+
+	t.Run("Should render status list in human format", func(t *testing.T) {
+		t.Parallel()
+		listBundle := mcpAuthStatusListBundle([]mcpauth.Status{status})
+		listHuman, err := listBundle.human()
+		if err != nil {
+			t.Fatalf("mcpAuthStatusListBundle.human() error = %v", err)
+		}
+		if !strings.Contains(listHuman, "MCP Auth") || !strings.Contains(listHuman, "linear") {
+			t.Fatalf("mcp auth list human = %q, want status table", listHuman)
+		}
+	})
+
+	t.Run("Should render status list in toon format", func(t *testing.T) {
+		t.Parallel()
+		listBundle := mcpAuthStatusListBundle([]mcpauth.Status{status})
+		listToon, err := listBundle.toon()
+		if err != nil {
+			t.Fatalf("mcpAuthStatusListBundle.toon() error = %v", err)
+		}
+		if !strings.Contains(listToon, "mcp_auth[1]") {
+			t.Fatalf("mcp auth list toon = %q, want toon table", listToon)
+		}
+	})
 }
```
</details>

As per coding guidelines, `**/*_test.go`: “Use `t.Run("Should ...")` subtests with `t.Parallel` as default in Go tests” and “MUST use `t.Run("Should...")` pattern for ALL test cases.”

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/mcp_auth_test.go` around lines 46 - 93, Split the monolithic
TestMCPAuthStatusBundlesRenderHumanAndToon into multiple t.Run subtests (e.g.,
"Should render single status human", "Should render single status toon", "Should
render list human", "Should render list toon"), move t.Parallel() into each
subtest where safe, and group the related assertions inside their respective
subtest blocks; keep the shared setup (fixedTestNow, status creation) at the top
of TestMCPAuthStatusBundlesRenderHumanAndToon and call
mcpAuthStatusBundle(status) and
mcpAuthStatusListBundle([]mcpauth.Status{status}) inside the appropriate
subtests, then run bundle.human(), bundle.toon(), listBundle.human(), and
listBundle.toon() with their existing checks inside the corresponding t.Run
bodies so failures isolate to the specific behavior.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestMCPAuthStatusBundlesRenderHumanAndToon` currently has one monolithic body with multiple assertion groups. This violates the repository Go test convention that cases use `t.Run("Should ...")` subtests with `t.Parallel()` where safe. The fix is test-only: keep the shared `mcpauth.Status` fixture and split single-status/list human/toon assertions into independent `Should ...` subtests.
- Resolution: Split the assertion groups into `Should ...` subtests and verified with focused `internal/cli` tests plus `make verify`.
