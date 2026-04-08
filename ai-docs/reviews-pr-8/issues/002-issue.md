# Issue 2 - Review Thread Comment

**File:** `internal/daemon/notifier.go:144`
**Date:** 2026-04-08 11:02:40 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: the compile-time assertion was missing, and marketplace hooks were running without the same trust gate used for MCP servers. The fix keeps the notifier path unchanged structurally and enforces the marketplace gate inside `HookRunner`.

## Body

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "== Hook dispatch call sites =="
rg -n -C3 --type=go '\bRunHooks\s*\('

echo
echo "== SkillRegistry.ForWorkspace definitions / call sites =="
rg -n -C4 --type=go '\bForWorkspace\s*\('

echo
echo "== Marketplace trust / consent filtering =="
rg -n -C4 --type=go 'SourceMarketplace|consent|allow.*marketplace|trusted.*skill|filter.*hook'
```

Repository: compozy/agh

Length of output: 49661

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Registry.ForWorkspace implementation ==="
sed -n '135,180p' internal/skills/registry.go

echo
echo "=== skillsHookDispatcher definition and interface verification ==="
rg -n -A20 'type skillsHookDispatcher' internal/daemon/notifier.go

echo
echo "=== sessionHookPhase interface definition ==="
rg -n -B2 -A8 'type sessionHookPhase' internal/daemon/notifier.go

echo
echo "=== Check for marketplace/consent filtering in ForWorkspace ==="
sed -n '135,220p' internal/skills/registry.go | head -90
```

Repository: compozy/agh

Length of output: 4770

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Check if DisabledSkills filtering is applied to hooks ==="
rg -n -B3 -A3 'DisabledSkills' internal/skills/registry.go

echo
echo "=== Check ForWorkspace return path - does it filter anything? ==="
sed -n '180,200p' internal/skills/registry.go

echo
echo "=== Check mergedSkillList implementation ==="
rg -n -B2 -A15 'func mergedSkillList' internal/skills/registry.go

echo
echo "=== Search for any hook-level marketplace filtering ==="
rg -n -C5 'RunHooks.*marketplace|hook.*marketplace|hook.*consent|hook.*filter' --type=go
```

Repository: compozy/agh

Length of output: 2505

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Check if DisabledSkills filtering is applied in List() or exported from ForWorkspace ==="
sed -n '115,135p' internal/skills/registry.go

echo
echo "=== Check RunHooks implementation to see if it filters marketplace hooks ==="
sed -n '50,120p' internal/skills/hooks.go

echo
echo "=== Verify: does skillsHookDispatcher have compile-time interface assertion? ==="
grep -n 'var _ sessionHookPhase.*skillsHookDispatcher' internal/daemon/notifier.go || echo "NOT FOUND: No compile-time interface assertion"
```

Repository: compozy/agh

Length of output: 2630

---

</details>

**Add compile-time interface assertion and implement marketplace trust gate for hooks.**

Two issues found:

1. **Missing compile-time interface verification**: `skillsHookDispatcher` (line 75) implements `sessionHookPhase` but lacks the assertion `var _ sessionHookPhase = (*skillsHookDispatcher)(nil)`. Add this at package level per guidelines.

2. **No marketplace trust gate for hook execution**: `Registry.ForWorkspace()` returns all skills without filtering marketplace sources, and `RunHooks()` executes any skill passed to it without consent checks. Unlike MCP server resolution (which filters via `allowedMarketplace` in `mcp.go`), lifecycle hooks run against all returned skills—including untrusted marketplace ones. This enables automatic third-party code execution on session create/stop.

Add explicit marketplace filtering to the hook dispatch path before calling `RunHooks()`, or filter marketplace hooks within `RunHooks()` itself (similar to how MCP servers require an allowlist).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/notifier.go` around lines 128 - 144, Add a package-level
compile-time assertion for the sessionHookPhase interface by adding: var _
sessionHookPhase = (*skillsHookDispatcher)(nil) so the compiler verifies
skillsHookDispatcher implements sessionHookPhase; then prevent untrusted
marketplace skills from being invoked by filtering the results of
Registry.ForWorkspace(...) before calling d.runner.RunHooks: replicate the
allowlist behavior used in mcp.go (allowedMarketplace) — either filter
marketplace-origin skills out in the notifier path that calls RunHooks or add
the same allowlist gating inside RunHooks itself so only marketplace-trusted
skills are dispatched during hook execution.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Resolve

Thread ID: `PRRT_kwDOR5y4QM55lKhB`

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=PRRT_kwDOR5y4QM55lKhB
```

---

_Generated from PR review - CodeRabbit AI_
