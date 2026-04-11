---
status: resolved
file: internal/config/provider.go
line: 233
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Q89L,comment:PRRC_kwDOR5y4QM6200jf
---

# Issue 010: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Minor inconsistency in index key handling.**

`OverrideMCPServers` uses `strings.TrimSpace(server.Name)` for lookups (line 220) but stores the untrimmed `server.Name` in the index (line 216). This differs from `MergeMCPServers` which also stores untrimmed names but trims only during iteration (line 185).

If a base server has name `"  github  "` and overlay has `"github"`, the lookup will fail because the index key is `"  github  "` but lookup uses `"github"`.

Consider trimming consistently when building the index:

<details>
<summary>♻️ Suggested fix</summary>

```diff
 	for i, server := range merged {
-		if strings.TrimSpace(server.Name) == "" {
+		name := strings.TrimSpace(server.Name)
+		if name == "" {
 			continue
 		}
-		index[server.Name] = i
+		index[name] = i
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
// OverrideMCPServers overlays MCP servers by name, replacing the full server object
// on collision instead of field-merging it.
func OverrideMCPServers(base []MCPServer, overlay []MCPServer) []MCPServer {
	merged := cloneMCPServers(base)
	index := make(map[string]int, len(merged))
	for i, server := range merged {
		name := strings.TrimSpace(server.Name)
		if name == "" {
			continue
		}
		index[name] = i
	}

	for _, server := range overlay {
		name := strings.TrimSpace(server.Name)
		if idx, ok := index[name]; ok && name != "" {
			merged[idx] = cloneMCPServer(server)
			continue
		}

		merged = append(merged, cloneMCPServer(server))
		if name != "" {
			index[name] = len(merged) - 1
		}
	}

	return merged
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/provider.go` around lines 207 - 233, OverrideMCPServers
builds the map index using the raw server.Name but looks up using
strings.TrimSpace(server.Name), causing mismatches for names with surrounding
whitespace; update the index population to use trimmedName :=
strings.TrimSpace(server.Name) (skip if trimmedName == "") when iterating merged
to set index[trimmedName] = i, and likewise use trimmedName for adding new
entries after append so lookups and stored keys are consistent with
MergeMCPServers; keep existing cloneMCPServer calls and the rest of the merge
logic unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `OverrideMCPServers` builds its lookup index from raw `server.Name` values but performs overlay lookups with `strings.TrimSpace(server.Name)`, so whitespace-padded base names fail to collide with their trimmed overlays.
- The same normalized-name root cause exists in the adjacent merge path, so the safest fix is to normalize index keys consistently where MCP server name matching is performed.
- Fix approach: build the name index from trimmed names, keep empty names skipped, and add regression tests covering whitespace-normalized collisions.
