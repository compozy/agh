---
provider: coderabbit
pr: "120"
round: 2
round_created_at: 2026-05-07T19:41:55.305082Z
status: resolved
file: internal/agentidentity/errors.go
line: 112
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AbUsC,comment:PRRC_kwDOR5y4QM6-_G2n
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**JSONL helper does not emit a newline-delimited frame.**

`MarshalErrorJSONL` returns a JSON object, but not a JSONL line (missing trailing `\n`). When multiple frames are concatenated, parsers expecting one-object-per-line can break.
 
<details>
<summary>Suggested fix</summary>

```diff
 func MarshalErrorJSONL(err error) ([]byte, error) {
-	return json.Marshal(struct {
+	frame, marshalErr := json.Marshal(struct {
 		Type  string       `json:"type"`
 		Error ErrorPayload `json:"error"`
 	}{
 		Type:  "error",
 		Error: ErrorPayloadFor(err),
 	})
+	if marshalErr != nil {
+		return nil, marshalErr
+	}
+	return append(frame, '\n'), nil
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/agentidentity/errors.go` around lines 104 - 112, The
MarshalErrorJSONL function currently returns a JSON object but does not
terminate it with a newline, breaking JSONL consumers; update MarshalErrorJSONL
to append a trailing '\n' to the marshalled bytes (i.e., call json.Marshal on
the struct as now, then ensure the returned []byte ends with a single newline)
so that the output is a valid JSONL frame; reference MarshalErrorJSONL and
ErrorPayloadFor when making the change.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/agentidentity/errors.go:104-111` currently returns raw `json.Marshal(...)` output without the newline required for a JSONL frame.
  - The package test at `internal/agentidentity/identity_test.go:203-223` currently asserts the opposite behavior, so the fix requires a minimal out-of-scope test update in that file.
  - Fix plan: append exactly one trailing `\n` in `MarshalErrorJSONL` and update the JSONL test to assert newline-delimited framing while still decoding the trimmed JSON object.
  - Resolved: `MarshalErrorJSONL` now appends a trailing newline and the package test asserts/decodes the JSONL frame correctly.
