---
status: resolved
file: internal/cli/doc_test.go
line: 114
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hC_X,comment:PRRC_kwDOR5y4QM64gE3-
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Do not discard `filepath.Rel` errors in test walks.**

Ignoring these errors can hide real path issues and weakens test diagnostics.

<details>
<summary>Suggested patch</summary>

```diff
-		rel, _ := filepath.Rel(outputDir, p)
+		rel, relErr := filepath.Rel(outputDir, p)
+		if relErr != nil {
+			return relErr
+		}
 		if filepath.Dir(rel) == "." {
 			t.Errorf("doc command must not write root meta.json (hand-maintained)")
 			return nil
 		}
...
-			rel, _ := filepath.Rel(outputDir, p)
+			rel, relErr := filepath.Rel(outputDir, p)
+			if relErr != nil {
+				return relErr
+			}
 			t.Errorf("file %s contains absolute paths", rel)
```
</details>


As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification".


Also applies to: 143-143

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/doc_test.go` at line 114, The test currently ignores the error
returned by filepath.Rel (rel, _ := filepath.Rel(outputDir, p)); update the test
to check the error and fail the test if non-nil (e.g., use t.Fatalf or t.Errorf
with context), so replace the blank identifier with err and handle it; refer to
the variables and call filepath.Rel(outputDir, p) and ensure the error is
reported with the path context (also apply the same fix at the other occurrence
around p on the later line).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - Two `WalkDir` closures in `internal/cli/doc_test.go` discard the `filepath.Rel` error even though the test depends on that relative path for diagnostics.
  - Root cause: the walk helpers use `_` for `filepath.Rel` failures instead of surfacing path problems immediately.
  - Fix plan: handle `filepath.Rel` errors explicitly in both walks and fail the test with path context instead of silently continuing.
  - Resolution: both `filepath.Rel` calls now handle errors explicitly, and the generated-file walk also stops on `os.ReadFile` failures instead of swallowing them.
  - Verification: `go test ./internal/cli/...` passed.
