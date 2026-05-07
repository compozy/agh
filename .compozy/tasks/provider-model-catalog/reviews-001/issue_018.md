---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/modelcatalog/errors.go
line: 31
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6s1,comment:PRRC_kwDOR5y4QM6-6bsi
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Keep `StaleFallbackError.Error()` redacted.**

This currently embeds `e.Err` verbatim. Any generic `%v`/`err.Error()` path will bypass `sourceErrorText` and can leak provider discovery details or secret-bearing upstream messages into logs or API errors. Make `Error()` return only the stable stale-fallback context and leave redacted rendering to helper call sites.

 

<details>
<summary>Proposed fix</summary>

```diff
 func (e *StaleFallbackError) Error() string {
 	if e == nil {
 		return "model catalog: stale fallback"
 	}
-	if e.Err == nil {
-		return fmt.Sprintf("model catalog: source %q returned stale fallback", e.SourceID)
-	}
-	return fmt.Sprintf("model catalog: source %q returned stale fallback: %v", e.SourceID, e.Err)
+	return fmt.Sprintf("model catalog: source %q returned stale fallback", e.SourceID)
 }
```
</details>

As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings MUST NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (e *StaleFallbackError) Error() string {
	if e == nil {
		return "model catalog: stale fallback"
	}
	return fmt.Sprintf("model catalog: source %q returned stale fallback", e.SourceID)
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/modelcatalog/errors.go` around lines 23 - 31,
StaleFallbackError.Error() must not embed the underlying e.Err (which can leak
secrets); change the implementation of StaleFallbackError.Error() so it never
includes e.Err’s text — always return the stable stale-fallback context (e.g.
"model catalog: stale fallback" or "model catalog: source %q returned stale
fallback" using SourceID) and remove any fmt usage that formats e.Err; leave any
redacted rendering of e.Err to helper call sites such as sourceErrorText.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `StaleFallbackError.Error()` currently formats `e.Err` directly when present.
  - Any generic `err.Error()` logging or transport path would bypass `sourceErrorText(...)` and could leak upstream discovery details or secret-shaped tokens.
  - Fix: keep `Error()` stable and redacted by returning only the stale-fallback context, and leave detailed redacted rendering to helper call sites.
