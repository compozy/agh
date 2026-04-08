# Issue 1 - Review Thread Comment

**File:** `internal/daemon/boot.go:72`
**Date:** 2026-04-08 11:02:40 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: assigning nil concrete `*skills.Registry` / `*skills.MCPResolver` pointers into interface fields does produce non-nil interfaces in Go. The fix is to pass interface-typed nils from boot when skills are disabled.

## Body

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo '--- SessionManagerDeps field types ---'
rg -n -C2 'type SessionManagerDeps struct|SkillRegistry\s+session\.SkillRegistry|MCPResolver\s+session\.MCPResolver' internal/daemon/daemon.go

echo
echo '--- boot() concrete locals and assignments ---'
rg -n -C2 'skillsRegistry\s+\*skills\.Registry|mcpResolver\s+\*skills\.MCPResolver|SkillRegistry:\s+skillsRegistry|MCPResolver:\s+mcpResolver' internal/daemon/boot.go

echo
echo '--- Manager nil guard that depends on true nil interfaces ---'
rg -n -C2 'resolveStartMCPServers|skillRegistry == nil && mcpResolver == nil|skillRegistry == nil \|\| mcpResolver == nil' internal/session/manager_lifecycle.go
```

Repository: compozy/agh

Length of output: 1645

---

</details>

**Avoid the typed-nil interface trap when assigning nil skill pointers to interface fields.**

`skillsRegistry` and `mcpResolver` are concrete pointer variables declared at lines 70–71. When skills are disabled, both remain `nil`, but assigning them directly to the `SessionManagerDeps` interface fields at lines 203–204 creates non-nil interface values (Go interfaces with a concrete type and nil value). This breaks the `nil` check at `internal/session/manager_lifecycle.go:406` (`case m.skillRegistry == nil && m.mcpResolver == nil:`), causing the session manager to attempt MCP server resolution even when skills are disabled, leading to failure or panic.

Assign nil concrete pointers through interface-typed intermediates that remain truly `nil`:

```diff
+	var skillRegistryDep session.SkillRegistry
+	if skillsRegistry != nil {
+		skillRegistryDep = skillsRegistry
+	}
+	var mcpResolverDep session.MCPResolver
+	if mcpResolver != nil {
+		mcpResolverDep = mcpResolver
+	}
+
 	sessions, err := d.newSessionManager(ctx, SessionManagerDeps{
 		HomePaths:         d.homePaths,
 		Logger:            logger,
 		Notifier:          &fanout,
 		PromptAssembler:   promptAssembler,
-		SkillRegistry:     skillsRegistry,
-		MCPResolver:       mcpResolver,
+		SkillRegistry:     skillRegistryDep,
+		MCPResolver:       mcpResolverDep,
 		WorkspaceResolver: workspaceResolver,
 	})
```

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Resolve

Thread ID: `PRRT_kwDOR5y4QM55lKg6`

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=PRRT_kwDOR5y4QM55lKg6
```

---

_Generated from PR review - CodeRabbit AI_
