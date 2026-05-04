---
status: resolved
file: internal/api/core/agent_contracts_test.go
line: 40
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsC,comment:PRRC_kwDOR5y4QM67YHCb
---

# Issue 007: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Use named subtests for coordinator config scenarios.**

This single compound assertion makes failures opaque and leaves the new mapper lightly covered. Please convert it to table-driven `t.Run("Should...")` cases so trimming, disabled configs, and empty workspace handling are asserted independently.



<details>
<summary>Example scaffold</summary>

```diff
 func TestCoordinatorConfigPayloadFromConfig(t *testing.T) {
 	t.Parallel()
 
-	payload := core.CoordinatorConfigPayloadFromConfig(
-		aghconfig.CoordinatorConfig{ ... },
-		contract.CoordinatorConfigSourceWorkspace,
-		" ws-1 ",
-	)
-
-	if !payload.Enabled || ... {
-		t.Fatalf("CoordinatorConfigPayloadFromConfig() = %#v", payload)
-	}
+	testCases := []struct {
+		name        string
+		cfg         aghconfig.CoordinatorConfig
+		source      contract.CoordinatorConfigSource
+		workspaceID string
+		assert      func(*testing.T, contract.CoordinatorConfigPayload)
+	}{
+		{
+			name: "Should trim workspace config fields",
+			// ...
+		},
+	}
+
+	for _, tc := range testCases {
+		tc := tc
+		t.Run(tc.name, func(t *testing.T) {
+			t.Parallel()
+			payload := core.CoordinatorConfigPayloadFromConfig(tc.cfg, tc.source, tc.workspaceID)
+			tc.assert(t, payload)
+		})
+	}
 }
```
</details>

As per coding guidelines, `Use table-driven tests with subtests (t.Run) as default pattern`, `MUST use t.Run("Should...") pattern for ALL test cases`, and `Use t.Parallel() for independent subtests in Go tests`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/agent_contracts_test.go` around lines 12 - 40, Replace the
single compound assertion in TestCoordinatorConfigPayloadFromConfig with a
table-driven set of t.Run subtests that each assert one behavior of
CoordinatorConfigPayloadFromConfig: trimming of strings
(AgentName/Provider/Model), conversion of DefaultTTL to DefaultTTLSeconds,
handling of Enabled=false (disabled configs), MaxChildren/MaxActivePerWorkspace
mapping, Source/WorkspaceID handling including empty workspace trimming, etc.;
for each table case call t.Run("Should ...") and run t.Parallel() inside each
subtest, construct inputs using aghconfig.CoordinatorConfig and call
core.CoordinatorConfigPayloadFromConfig, then assert only the one expected
property per subtest to make failures clear.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestCoordinatorConfigPayloadFromConfig` asserts trimming, TTL conversion, limits, source, and workspace handling as one compound check. This makes mapper regressions hard to isolate and does not follow the repository's table/subtest pattern. Fix by converting it to independent `t.Run("Should ...")` cases.
