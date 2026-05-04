---
provider: coderabbit
pr: "85"
round: 2
round_created_at: 2026-04-30T19:49:37.693355Z
status: resolved
file: internal/api/core/tools_test.go
line: 251
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-0Ill,comment:PRRC_kwDOR5y4QM687orR
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Add explicit response-body assertions on invoke success paths.**

At Line 245 and Line 297, status and side effects are asserted, but the response payload contract is not directly validated. Add minimal body assertions (e.g., `tool_id`, `status`) to keep HTTP tests contract-focused.

<details>
<summary>Suggested assertions</summary>

```diff
@@
 		invokeResp := performRequest(t, engine, http.MethodPost, "/tools/ext__ask_tool/invoke", body)
 		if invokeResp.Code != http.StatusOK {
 			t.Fatalf("invoke status = %d, want %d; body=%s", invokeResp.Code, http.StatusOK, invokeResp.Body.String())
 		}
+		var invokePayload contract.ToolInvokeResponse
+		decodeToolJSON(t, invokeResp.Body.Bytes(), &invokePayload)
+		if invokePayload.ToolID != "ext__ask_tool" || invokePayload.Status != "completed" {
+			t.Fatalf("invoke payload = %#v, want completed ext__ask_tool", invokePayload)
+		}
 		if registry.callCount("ext__ask_tool") != 1 {
 			t.Fatalf("registry call count = %d, want 1", registry.callCount("ext__ask_tool"))
 		}
@@
 		if resp.Code != http.StatusOK {
 			t.Fatalf("invoke status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
 		}
+		var invokePayload contract.ToolInvokeResponse
+		decodeToolJSON(t, resp.Body.Bytes(), &invokePayload)
+		if invokePayload.ToolID != toolspkg.ToolIDSkillView || invokePayload.Status != "completed" {
+			t.Fatalf("invoke payload = %#v, want completed skill_view", invokePayload)
+		}
 		scope, call := registry.lastCall()
```
</details>

 

As per coding guidelines, "Always assert both status code AND response body in HTTP tests in Go — status-code-only assertions are insufficient."


Also applies to: 297-313

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tools_test.go` around lines 245 - 251, The test currently
checks invokeResp.Code and registry.callCount("ext__ask_tool") but omits
validating the response payload; update the test that calls performRequest(t,
engine, http.MethodPost, "/tools/ext__ask_tool/invoke", body) to decode
invokeResp.Body (e.g., into a map or struct) and assert minimal fields such as
"tool_id" equals "ext__ask_tool" and "status" equals the expected value (e.g.,
"ok" or "success"); do the same for the other invoke path around lines where
performRequest is used (also asserting those response-body fields) so both
status code and response body contracts are verified.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  Two successful invoke paths assert status and side effects but do not assert
  the returned `ToolInvokeResponse` contract. Add minimal payload assertions for
  `tool_id` and `status` so the tests prove the HTTP response shape as well as
  the registry side effect.
