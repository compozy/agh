---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/config/hooks.go
line: 61
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_0Dru,comment:PRRC_kwDOR5y4QM6-RRYo
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Deep-copy the new matcher pointers in `cloneHookDecl()`.**

These new nested matchers are now heap-backed, but `cloneHookDecl()` still only copies `ToolReadOnly`. Cloned hook declarations will alias the original `NetworkMatcher` and `CompactionMatcher`, so later mutation can bleed back into the source config.

 

<details>
<summary>Suggested fix</summary>

```diff
 func cloneHookDecl(src hookspkg.HookDecl) hookspkg.HookDecl {
 	cloned := src
 	cloned.Args = cloneStrings(src.Args)
 	cloned.Env = mergeStringMaps(nil, src.Env)
 	cloned.SecretEnv = mergeStringMaps(nil, src.SecretEnv)
 	cloned.Metadata = mergeStringMaps(nil, src.Metadata)
 	cloned.Enabled = cloneBoolPtr(src.Enabled)
+	if src.Matcher.NetworkMatcher != nil {
+		networkMatcher := *src.Matcher.NetworkMatcher
+		cloned.Matcher.NetworkMatcher = &networkMatcher
+	}
+	if src.Matcher.CompactionMatcher != nil {
+		compactionMatcher := *src.Matcher.CompactionMatcher
+		cloned.Matcher.CompactionMatcher = &compactionMatcher
+	}
 	if src.Matcher.ToolReadOnly != nil {
 		value := *src.Matcher.ToolReadOnly
 		cloned.Matcher.ToolReadOnly = &value
 	}
 	return cloned
 }
```
</details>


Also applies to: 216-226

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/config/hooks.go` around lines 57 - 61, cloneHookDecl currently only
copies ToolReadOnly and leaves NetworkMatcher and CompactionMatcher pointers
shared; update cloneHookDecl (and the similar clone at the other location
handling lines ~216-226) to deep-copy the nested matcher pointers: check for
nil, allocate new matcher structs (or call their Clone/Copy method if available)
and copy their fields into the new HookDecl so NetworkMatcher and
CompactionMatcher are not aliased back to the original; keep the existing copy
of ToolReadOnly intact.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `internal/config/hooks.go:259-270` deep-copies slices/maps and `ToolReadOnly`, but `Matcher.NetworkMatcher` and `Matcher.CompactionMatcher` remain shared pointers after cloning. Mutating a cloned declaration can therefore bleed into the source config declaration. Deep-copy the new matcher pointers and add a regression test.
