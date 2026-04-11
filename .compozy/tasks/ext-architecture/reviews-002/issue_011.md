---
status: resolved
file: internal/extension/manifest.go
line: 535
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QU6I,comment:PRRC_kwDOR5y4QM620Ap9
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reject blank map keys after trimming.**

Whitespace-only MCP server names and env keys are normalized into `""` and kept. That lets malformed manifests survive normalization, and trimming can silently collapse two distinct raw keys onto the same normalized key.



<details>
<summary>Suggested fix</summary>

```diff
 func normalizeMCPServers(src map[string]MCPServerConfig) map[string]MCPServerConfig {
 	if len(src) == 0 {
 		return nil
 	}
 
 	dst := make(map[string]MCPServerConfig, len(src))
 	for name, server := range src {
-		dst[strings.TrimSpace(name)] = MCPServerConfig{
+		trimmedName := strings.TrimSpace(name)
+		if trimmedName == "" {
+			continue
+		}
+		dst[trimmedName] = MCPServerConfig{
 			Command: strings.TrimSpace(server.Command),
 			Args:    normalizeStrings(server.Args),
 			Env:     normalizeStringMap(server.Env),
 		}
 	}
+	if len(dst) == 0 {
+		return nil
+	}
 	return dst
 }
@@
 func normalizeStringMap(src map[string]string) map[string]string {
 	if len(src) == 0 {
 		return nil
 	}
 
 	dst := make(map[string]string, len(src))
 	for key, value := range src {
-		dst[strings.TrimSpace(key)] = strings.TrimSpace(value)
+		trimmedKey := strings.TrimSpace(key)
+		if trimmedKey == "" {
+			continue
+		}
+		dst[trimmedKey] = strings.TrimSpace(value)
 	}
+	if len(dst) == 0 {
+		return nil
+	}
 	return dst
 }
```
</details>


Also applies to: 557-565

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/manifest.go` around lines 522 - 535, normalizeMCPServers
currently trims names and env keys but keeps entries whose trimmed key becomes
empty, and can silently collapse distinct raw keys; change normalizeMCPServers
to skip any entry whose trimmed name is empty (i.e., if strings.TrimSpace(name)
== "" continue) and when building Env use/adjust normalizeStringMap to similarly
drop any env entries whose trimmed key is empty so whitespace-only keys are
rejected; apply the same empty-key rejection to the other similar normalizer
referenced at lines 557-565 (the companion function that normalizes MCP
env/string maps) and ensure behavior on key collisions is deterministic (e.g.,
last write wins).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The finding is accurate. `normalizeMCPServers()` and `normalizeStringMap()` trim keys but still keep entries whose trimmed key becomes empty, and collisions between differently spaced keys collapse according to nondeterministic Go map iteration order.
  - Root cause: normalization happens while iterating unsorted maps and does not explicitly reject blank keys after trimming.
  - Fix approach: drop entries whose trimmed key is empty, make collision handling deterministic by normalizing in a stable key order, and add manifest coverage in `internal/extension/manifest_test.go`.
  - Resolution: implemented in `internal/extension/manifest.go` with regression coverage in `internal/extension/manifest_test.go`, then verified with focused package tests plus `make verify`.
