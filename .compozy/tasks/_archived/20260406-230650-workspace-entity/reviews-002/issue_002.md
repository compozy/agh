---
status: resolved
file: internal/acp/client.go
line: 242
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoBv,comment:PRRC_kwDOR5y4QM61T6HA
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Same nil-safety concern for new session request.**

Same issue as above — `append([]string(nil), normalized.AdditionalDirs...)` converts `nil` to empty slice.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/client.go` around lines 238 - 242, The construction of
newWireRequest (type wireNewSessionRequest) currently uses append([]string(nil),
normalized.AdditionalDirs...) which always yields an empty slice instead of
preserving a nil value; update the AdditionalDirs assignment so it preserves nil
when normalized.AdditionalDirs is nil and otherwise makes a copy (e.g., if
normalized.AdditionalDirs != nil create a new slice and copy elements into it,
otherwise leave AdditionalDirs as nil) so that newWireRequest.AdditionalDirs
reflects the original nil-vs-empty distinction.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  - Same validation as issue 001: `append([]string(nil), normalized.AdditionalDirs...)` preserves `nil`.
  - The new-session payload already omits `additional_dirs` when the normalized slice is empty or nil.
  - No production change is needed here.
