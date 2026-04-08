# Issue 4 - Review Thread Comment

**File:** `internal/skills/mcp.go:80`
**Date:** 2026-04-08 12:09:55 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: the dedupe key was trimmed, but the stored `resolvedServer.Name` was not, so downstream consumers could still observe inconsistent names. The fix normalizes `resolvedServer.Name` before replacement/appending so the stored server definition matches the dedupe key.

## Body

_⚠️ Potential issue_ | _🟡 Minor_

**Normalize the stored MCP name before deduping.**

Line 64 trims only the temporary lookup key. The untrimmed `resolvedServer.Name` still gets appended/replaced, so `" github "` and `"github"` collapse to one slot here but can diverge again in downstream merges/lookups that use `Name` verbatim.

<details>
<summary>🛠 Proposed fix</summary>

```diff
 			resolvedServer := toConfigMCPServer(server)
-			name := strings.TrimSpace(resolvedServer.Name)
+			resolvedServer.Name = strings.TrimSpace(resolvedServer.Name)
+			name := resolvedServer.Name
```

</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
			resolvedServer := toConfigMCPServer(server)
			resolvedServer.Name = strings.TrimSpace(resolvedServer.Name)
			name := resolvedServer.Name
			if idx, ok := index[name]; ok && name != "" {
				resolved[idx] = resolvedServer
				origins[idx] = mcpOrigin{
					skillName: skill.Meta.Name,
					source:    skill.Source,
				}
				continue
			}

			resolved = append(resolved, resolvedServer)
			origins = append(origins, mcpOrigin{
				skillName: skill.Meta.Name,
				source:    skill.Source,
			})
			if name != "" {
				index[name] = len(resolved) - 1
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/mcp.go` around lines 63 - 80, The code trims the temporary
lookup key `name` but leaves `resolvedServer.Name` unnormalized, causing entries
like " github " and "github" to dedupe but persist different `Name` values;
update the `resolvedServer` produced by `toConfigMCPServer(server)` so its
`Name` field is normalized (e.g., set `resolvedServer.Name =
strings.TrimSpace(name)` or otherwise normalize `name`) before using it for
replacement (`resolved[idx] = resolvedServer`) or appending (`resolved =
append(resolved, resolvedServer)`), keeping the same `index`/`mcpOrigin` logic
intact.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Resolve

Thread ID: `PRRT_kwDOR5y4QM55mbaM`

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=PRRT_kwDOR5y4QM55mbaM
```

---

_Generated from PR review - CodeRabbit AI_
