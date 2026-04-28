---
status: resolved
file: internal/api/core/tasks_terminal_integration_test.go
line: 112
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-IGMQ,comment:PRRC_kwDOR5y4QM67_zdD
---

# Issue 005: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Compare JSON payloads semantically, not as raw strings.**

These assertions lock the test to a specific marshal order for `metadata`/`result`. A harmless re-encode of the same JSON object will start failing this suite even when the handler behavior is still correct.



Also applies to: 157-158, 210-215

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_terminal_integration_test.go` around lines 111 - 112,
The test is comparing JSON payloads as raw strings which is brittle; instead,
parse capture.failure.Metadata into a generic JSON structure (e.g.,
map[string]interface{} or a struct) and compare it semantically to the expected
JSON (e.g., expected :=
map[string]interface{}{"step":"claim","mode":"historical-http"}) using deep
equality (reflect.DeepEqual or cmp.Equal) and only call t.Fatalf with the
marshaled/pretty-printed values when they differ; apply the same change for the
other raw-string comparisons mentioned around the capture variables at the other
locations (lines referenced: the blocks containing capture.failure.Metadata
checks and the similar comparisons at 157-158 and 210-215).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The integration test compares JSON metadata and result payloads via raw string equality. That makes the test sensitive to object key order rather than behavior. Fix by decoding `json.RawMessage` values and expected JSON strings into generic values and comparing them semantically.

## Resolution

- Replaced raw JSON string equality with semantic JSON decoding/comparison in the task terminal integration test.
- Verified with the integration test target and `make verify`.
