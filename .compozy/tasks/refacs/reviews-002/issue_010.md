---
provider: coderabbit
pr: "120"
round: 2
round_created_at: 2026-05-07T19:41:55.305082Z
status: resolved
file: internal/bridges/resource_projection.go
line: 327
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AbUs8,comment:PRRC_kwDOR5y4QM6-_G32
---

# Issue 010: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Unreachable code after successful `extra` decode.**

If `decoder.Decode(&extra)` succeeds (no EOF error) and `extra` is not `nil`, the error at line 324 is returned. However, if `extra` is `nil` (which would be unusual for successful decode), line 326 is returned. But the condition at line 323 will almost never be false after a successful decode - if `Decode` succeeds, `extra` will contain the decoded value.

More critically: if `Decode` returns `nil` error and `extra != nil`, both line 324 and 326 return identical errors, making the `if extra != nil` check redundant.




<details>
<summary>🐛 Simplify the multiple-value detection logic</summary>

```diff
 func decodeSemanticJSON(data []byte) (any, error) {
 	decoder := json.NewDecoder(bytes.NewReader(data))
 	decoder.UseNumber()
 	var value any
 	if err := decoder.Decode(&value); err != nil {
 		return nil, err
 	}
-	var extra json.RawMessage
-	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
+	var extra any
+	if err := decoder.Decode(&extra); err != nil {
+		if errors.Is(err, io.EOF) {
+			return value, nil
+		}
+		return nil, err
+	}
-		return value, nil
-	} else if err != nil {
-		return nil, err
-	}
-	if extra != nil {
-		return nil, errors.New("bridges: JSON payload contains multiple values")
-	}
 	return nil, errors.New("bridges: JSON payload contains multiple values")
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/bridges/resource_projection.go` around lines 310 - 327, The
decodeSemanticJSON function has redundant/unreachable branches after the second
decoder.Decode call; simplify by treating decoder.Decode(&extra) as: if it
returns io.EOF then return value, nil; if it returns a non-nil error return that
error; otherwise (successful decode means there is an extra value) return the
"bridges: JSON payload contains multiple values" error. Update
decodeSemanticJSON to remove the extra json.RawMessage nil check and return the
multiple-values error whenever the second Decode succeeds, keeping
errors.Is(err, io.EOF) to detect the single-value case.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/bridges/resource_projection.go:317-326` returns the same multiple-values error regardless of whether `extra` is nil, so the branch on `extra != nil` is redundant after a successful second decode.
  - The behavior should be simplified to the three actual outcomes: EOF => single value, non-EOF error => decode error, success => multiple values.
  - Fix plan: collapse the redundant branch and add regression coverage in the already-scoped `internal/bridges/json_equal_test.go`.
  - Resolved: the redundant branch was removed and bridge tests now cover the multiple-values path through `semanticJSONEqual`.
