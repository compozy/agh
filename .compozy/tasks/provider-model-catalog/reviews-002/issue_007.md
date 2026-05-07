---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/api/testutil/model_catalog_parity_test.go
line: 85
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYapq,comment:PRRC_kwDOR5y4QM6-7HX9
---

# Issue 007: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Add the same OpenAI projection check for UDS.**

This utility only hits `/api/openai/v1/models` on `httpEngine`, so a UDS-only routing or payload regression would still pass. Mirror the request/assertions against `udsEngine` to keep the new public surface covered end-to-end.

 

Based on learnings: No partial-surface completions — any change touching a public surface closes the loop end-to-end in one pass: contract → HTTP handler → UDS handler → CLI client → CLI command → extension/config/docs surfaces → tests → docs.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/testutil/model_catalog_parity_test.go` around lines 62 - 85,
Duplicate the existing OpenAI projection test that calls performParityRequest
against httpEngine for the "/api/openai/v1/models?provider_id=codex" endpoint
and run the identical assertions against udsEngine: call performParityRequest(t,
udsEngine, http.MethodGet, "/api/openai/v1/models?provider_id=codex"), verify
the response code is http.StatusOK, unmarshal into
contract.OpenAIModelListResponse, assert len(Data) == 1, then compare the
returned openAIModel (the Data[0]) to the same nativeModel (cliRecord.Models[0])
using the same identity checks (ID == ModelID, OwnedBy == ProviderID,
AGH.ProviderID == ProviderID, AGH.ModelID == ModelID) so the UDS routing/payload
path is covered end-to-end.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - In this branch, UDS intentionally does not expose `/api/openai/v1/models`; `internal/api/udsapi/model_catalog_test.go` already asserts that the route returns `404`.
  - Adding parity coverage for an OpenAI projection on UDS would contradict the current transport contract instead of protecting it.
  - No code change is needed; the correct transport coverage is native-provider parity on UDS plus explicit OpenAI absence.
  - Resolved as invalid after branch inspection and full verification.
