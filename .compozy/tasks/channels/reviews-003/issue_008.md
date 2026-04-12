---
status: resolved
file: internal/daemon/daemon_test.go
line: 3119
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tkbn,comment:PRRC_kwDOR5y4QM624L_W
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`shutdown` currently truncates previously recorded marker lines.**

The shutdown branch writes `"shutdown"` with `os.WriteFile`, which overwrites the marker file and drops earlier initialize/delivery JSONL records appended in this same helper.

<details>
<summary>Proposed fix</summary>

```diff
-		if strings.TrimSpace(h.marker) != "" {
-			if err := os.WriteFile(h.marker, []byte("shutdown"), 0o600); err != nil {
-				return false, err
-			}
-		}
+		if strings.TrimSpace(h.marker) != "" {
+			if err := appendMarkerLine(h.marker, "shutdown"); err != nil {
+				return false, err
+			}
+		}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_test.go` around lines 3117 - 3119, The helper
currently overwrites the marker file via os.WriteFile(h.marker,
[]byte("shutdown"), 0o600) which truncates earlier JSONL records; change it to
open the file with os.OpenFile(h.marker, os.O_APPEND|os.O_WRONLY|os.O_CREATE,
0o600) and write the shutdown line (include a newline if other records are
newline-delimited), then close the file and return any write/close error. Update
the block referencing h.marker so it appends instead of replacing the file.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `Valid`
- Notes:
  The shutdown branch writes the marker file with `os.WriteFile`, which truncates any initialize/delivery records already appended for the same helper process. That breaks the JSONL-style marker log used by the surrounding daemon tests.
  Resolved in `internal/daemon/daemon_test.go` by appending the shutdown marker with `appendMarkerLine` and by adding a dedicated regression test for the helper behavior. A minimal out-of-scope assertion update in `internal/daemon/daemon_integration_test.go` was also required because the integration test was still asserting the old truncating payload. Verified with `go test ./internal/daemon -count=1`, `go test -tags integration ./internal/daemon -count=1`, and the final `make verify` pass.
