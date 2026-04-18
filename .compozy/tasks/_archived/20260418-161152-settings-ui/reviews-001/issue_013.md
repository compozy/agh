---
status: resolved
file: internal/config/mcpjson_write.go
line: 204
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kRo,comment:PRRC_kwDOR5y4QM65B60Q
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Delete can report success while leaving the server in the second collection.**

`loadMCPJSONCollection` only rejects duplicates within one top-level key, and `Delete` returns after removing the first match. If a document contains both `mcpServers.alpha` and `mcp_servers.alpha`, this path leaves one copy behind.


<details>
<summary>🩹 Minimal fix</summary>

```diff
 func (d *editableMCPJSONDocument) Delete(name string) bool {
-	if deleted := d.snake.delete(name); deleted {
-		return true
-	}
-	return d.camel.delete(name)
+	deletedSnake := d.snake.delete(name)
+	deletedCamel := d.camel.delete(name)
+	return deletedSnake || deletedCamel
 }
```
</details>


Also applies to: 226-230

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/mcpjson_write.go` around lines 168 - 204,
loadMCPJSONCollection only checks duplicates within the single collection
(collection.nameIndex) so servers that normalize to the same name across
different top-level keys (e.g. "mcpServers.alpha" vs "mcp_servers.alpha") can
slip through; change loadMCPJSONCollection to accept a shared map (e.g.
existingNames map[string]string or *map[string]string) or a pointer to a global
index and, after computing normalized := normalizeMCPServerName(actualName),
check that normalized against both collection.nameIndex and the shared
existingNames and return an error if already present (include prior name from
existingNames in the error); update the callers that load both collections (the
other load call around the lines noted 226-230) to pass the same shared map so
duplicates across top-level keys are detected and rejected.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in `loadEditableMCPJSONDocument` and `Delete`: camel-case and snake-case collections build independent normalized-name indexes, so duplicates can exist across top-level keys and `Delete` stops after the first match. I will enforce a shared normalized-name check across both collections and make deletion clear every matching copy so MCP sidecars cannot report a successful delete while leaving a duplicate behind.
