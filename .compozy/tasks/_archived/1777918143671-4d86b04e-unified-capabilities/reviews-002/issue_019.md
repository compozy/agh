---
status: resolved
file: internal/network/capability_catalog.go
line: 246
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyyW,comment:PRRC_kwDOR5y4QM654NpJ
---

# Issue 019: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Compare filtered capability IDs as a set, not by position.**

`projectWhoisCapabilityCatalog()` returns matches in catalog order, so this positional check rejects a valid response whenever the caller asks for the same IDs in a different order. That can incorrectly downgrade a complete filtered catalog to “unknown”.

<details>
<summary>Suggested fix</summary>

```diff
 func capabilityCatalogAlignsWithCapabilityIDs(
 	capabilityIDs []string,
 	capabilityCatalog []sessionpkg.NetworkPeerCapability,
 ) bool {
 	normalizedIDs := normalizeCapabilityIDList(capabilityIDs)
 	if len(normalizedIDs) != len(capabilityCatalog) {
 		return false
 	}
 
-	for idx, capability := range capabilityCatalog {
-		if normalizedIDs[idx] != strings.TrimSpace(capability.ID) {
-			return false
-		}
-	}
-	return true
+	remaining := make(map[string]int, len(normalizedIDs))
+	for _, id := range normalizedIDs {
+		remaining[id]++
+	}
+	for _, capability := range capabilityCatalog {
+		id := strings.TrimSpace(capability.ID)
+		if remaining[id] == 0 {
+			return false
+		}
+		remaining[id]--
+	}
+	for _, count := range remaining {
+		if count != 0 {
+			return false
+		}
+	}
+	return true
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func capabilityCatalogAlignsWithCapabilityIDs(
	capabilityIDs []string,
	capabilityCatalog []sessionpkg.NetworkPeerCapability,
) bool {
	normalizedIDs := normalizeCapabilityIDList(capabilityIDs)
	if len(normalizedIDs) != len(capabilityCatalog) {
		return false
	}

	remaining := make(map[string]int, len(normalizedIDs))
	for _, id := range normalizedIDs {
		remaining[id]++
	}
	for _, capability := range capabilityCatalog {
		id := strings.TrimSpace(capability.ID)
		if remaining[id] == 0 {
			return false
		}
		remaining[id]--
	}
	for _, count := range remaining {
		if count != 0 {
			return false
		}
	}
	return true
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/capability_catalog.go` around lines 232 - 246,
capabilityCatalogAlignsWithCapabilityIDs currently compares normalizedIDs and
capabilityCatalog by index, which fails when the caller requests the same IDs in
a different order; change it to compare as sets: build a set
(map[string]struct{}) of normalized IDs returned by normalizeCapabilityIDList
and then iterate capabilityCatalog (use strings.TrimSpace(capability.ID)) to
check each catalog ID exists in that set, also ensure lengths match (or count
matches) to guarantee equality; update function to return true only when every
catalog ID is present in the normalized set and counts are equal.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `capabilityCatalogAlignsWithCapabilityIDs` compares requested IDs and returned catalog entries by index, but `projectWhoisCapabilityCatalog` deliberately preserves catalog order. A valid filtered response can therefore be marked unknown when the request order differs.
- Fix plan: compare normalized IDs as a multiset instead of by position and add a regression test in `internal/network/capability_catalog_test.go` for permuted request order.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
