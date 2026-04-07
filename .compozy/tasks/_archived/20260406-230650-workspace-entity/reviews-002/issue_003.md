---
status: resolved
file: internal/acp/client.go
line: 441
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoB1,comment:PRRC_kwDOR5y4QM61T6HG
---

# Issue 003: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Redundant slice copy.**

`normalizeAdditionalDirs` at line 431 already returns a freshly allocated slice. The subsequent copy at lines 439-441 is unnecessary.


<details>
<summary>Proposed fix</summary>

```diff
 	normalized.AdditionalDirs = additionalDirs
 	if normalized.Permissions == "" {
 		normalized.Permissions = aghconfig.PermissionModeApproveReads
 	}
-	if normalized.AdditionalDirs != nil {
-		normalized.AdditionalDirs = append([]string(nil), normalized.AdditionalDirs...)
-	}
 	if normalized.Env != nil {
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/client.go` around lines 439 - 441, Remove the redundant slice
copy: in the code path where normalizeAdditionalDirs already returns a newly
allocated slice, do not re-copy normalized.AdditionalDirs using
append([]string(nil), ...); instead keep the slice as-is (remove the
append-based copy) — locate this in the same function around
normalizeAdditionalDirs and the normalized.AdditionalDirs handling and delete
the additional append copy to avoid unnecessary allocation.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  - `normalizeAdditionalDirs` can return a non-nil zero-length slice when every input directory is filtered out.
  - The follow-up `append([]string(nil), normalized.AdditionalDirs...)` intentionally normalizes that case back to `nil`, which keeps `omitempty` behavior consistent and still detaches the slice from caller-owned storage.
  - Removing the copy would change that behavior, so this is not a redundant allocation in practice.
