---
provider: coderabbit
pr: "120"
round: 2
round_created_at: 2026-05-07T19:41:55.305082Z
status: resolved
file: internal/api/testutil/sse.go
line: 20
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AbUsH,comment:PRRC_kwDOR5y4QM6-_G2u
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Avoid printing raw SSE payloads in test failures.**

Line [20] includes full event payload in `t.Fatalf`. That can leak sensitive fields into CI logs when decoding fails.

<details>
<summary>Suggested fix</summary>

```diff
-		t.Fatalf("json.Unmarshal(sse data) error = %v; data=%s", err, string(record.Data))
+		t.Fatalf("json.Unmarshal(sse data) error = %v; data_len=%d", err, len(record.Data))
```
</details>

   
As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings MUST NEVER appear in logs...".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if err := json.Unmarshal(record.Data, dest); err != nil {
		t.Fatalf("json.Unmarshal(sse data) error = %v; data_len=%d", err, len(record.Data))
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/testutil/sse.go` around lines 19 - 20, The test currently logs
raw SSE payload by passing record.Data into t.Fatalf which may leak secrets;
change the failure message in the SSE test helper so it does NOT include the raw
payload — e.g., on json.Unmarshal error for record.Data, log the error plus safe
metadata only (length, first N bytes, or a redacted JSON with sensitive keys
removed) instead of string(record.Data); implement redaction by unmarshaling
into a generic map, removing known sensitive keys like "agh_claim_*",
"claim_token", "mcp_auth_token", OAuth/PKCE keys, and secret bindings, then
include the redacted representation or just non-sensitive diagnostics in the
t.Fatalf message where json.Unmarshal(record.Data, dest) is handled.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/api/testutil/sse.go:19-20` currently prints the full raw SSE payload when JSON decoding fails.
  - Even though this is test helper code, the helper can surface real runtime payloads in CI logs, so the leak concern is real.
  - Fix plan: replace raw payload logging with safe metadata only so failures remain diagnosable without dumping event contents.
  - Resolved: the SSE decode helper now reports only safe metadata (`data_len`) on failure.
