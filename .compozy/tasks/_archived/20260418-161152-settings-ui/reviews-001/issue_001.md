---
status: resolved
file: internal/api/core/errors.go
line: 133
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kRS,comment:PRRC_kwDOR5y4QM65B6zv
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Drop the message-based status heuristics.**

These `strings.Contains` fallbacks make transport behavior depend on English error text, so harmless wording changes can flip a response from 500 to 400/403/404/409. Please keep status mapping on typed sentinels only and propagate those from the service layer instead. As per coding guidelines, "Use errors.Is() and errors.As() for error matching — never compare error strings".




Also applies to: 282-312

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/errors.go` around lines 121 - 133, Replace the current
message-based heuristics in the HTTP status-mapping logic with typed error
matching: remove the calls to
settingsMessageLooksForbidden/NotFound/Conflict/Validation and instead check for
sentinel or concrete error types using errors.Is() or errors.As() (e.g., match
against exported sentinel errors from the service layer) inside the same
function that contains the switch, returning
http.StatusForbidden/NotFound/Conflict/BadRequest only when errors.Is/As detects
the corresponding typed error, otherwise return http.StatusInternalServerError;
apply the same change to the other mapping block referenced (around lines
282-312) so all mappings rely on typed sentinels rather than string contains.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in `internal/api/core/errors.go`: `StatusForSettingsError` still falls back to `strings.Contains` heuristics after the typed sentinel checks, so English wording can change transport status. I will remove the heuristic branch and rely on typed errors only, which also requires updating the existing core settings tests that currently codify heuristic behavior.
