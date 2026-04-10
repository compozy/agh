---
status: resolved
file: internal/skills/registry_workspace_cache.go
line: 51
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrX_,comment:PRRC_kwDOR5y4QM62twde
---

# Issue 016: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "workspaceCacheKey callsites:"
rg -n --type=go '\bworkspaceCacheKey\s*\(' -C2

echo
echo "ResolvedWorkspace constructors/usages:"
rg -n --type=go 'ResolvedWorkspace\s*{' -C4

echo
echo "Assignments/accesses for ID/RootDir on ResolvedWorkspace:"
rg -n --type=go '\.(ID|RootDir)\b' -C2
```

Repository: compozy/agh

Length of output: 1585

---

🏁 Script executed:

```shell
# Read the full workspaceCacheKey function
sed -n '113,135p' internal/skills/registry_workspace_cache.go

# Also check the context around line 48 and 151
echo "=== Context at line 48 (cache read) ==="
sed -n '43,60p' internal/skills/registry_workspace_cache.go

echo
echo "=== Context around line 151 (cache write) ==="
sed -n '140,165p' internal/skills/registry.go

echo
echo "=== ResolvedWorkspace struct definition ==="
rg -n --type=go 'type ResolvedWorkspace struct' -A10
```

Repository: compozy/agh

Length of output: 2364

---

🏁 Script executed:

```shell
# Check where ResolvedWorkspace is created/populated
echo "=== ResolvedWorkspace creation ==="
rg -n --type=go 'ResolvedWorkspace\s*{' -A5 | head -60

echo
echo "=== Check if ID/RootDir are set during resolution ==="
rg -n --type=go '\.ID\s*=' -B2 -A2 | grep -A2 -B2 'Resolved'

echo
echo "=== Check workspace.Workspace struct for ID/RootDir ==="
rg -n --type=go 'type Workspace struct' -A15
```

Repository: compozy/agh

Length of output: 1289

---

🏁 Script executed:

```shell
# Search for ResolvedWorkspace instantiation/return
echo "=== ResolvedWorkspace assignments/returns ==="
rg -n --type=go 'ResolvedWorkspace' | grep -E '(return|:=|=)' | head -30

echo
echo "=== Resolver interface/implementations ==="
rg -n --type=go 'type.*Resolver' -A5 | head -40

echo
echo "=== workspaceLoadFromResolved function (line 151 caller) ==="
rg -n --type=go 'func.*workspaceLoadFromResolved' -A20
```

Repository: compozy/agh

Length of output: 7332

---

🏁 Script executed:

```shell
# Check buildResolvedWorkspace function
echo "=== buildResolvedWorkspace function ==="
sed -n '236,280p' internal/workspace/resolver.go

echo
echo "=== Check where workspaceSkillTargetLocked is called ==="
rg -n --type=go 'workspaceSkillTargetLocked' -B3 -A3

echo
echo "=== Check if workspace ID/RootDir are validated before use ==="
rg -n --type=go 'if.*ID.*==' -A2 -B2 | head -40
```

Repository: compozy/agh

Length of output: 5042

---

🏁 Script executed:

```shell
# Check full workspaceSkillTargetLocked function signature
sed -n '43,60p' internal/skills/registry_workspace_cache.go

# Check if ResolvedWorkspace.Skills is used elsewhere in the same file
echo "=== ResolvedWorkspace.Skills usage in registry_workspace_cache.go ==="
rg -n --type=go '\.Skills' internal/skills/registry_workspace_cache.go -B2 -A2

# Check workspaceLoad structure
echo "=== workspaceLoad struct ==="
rg -n --type=go 'type workspaceLoad struct' -A8

# Check how cache is populated vs accessed
echo "=== Cache population pattern ==="
sed -n '140,180p' internal/skills/registry.go
```

Repository: compozy/agh

Length of output: 2656

---



</details>

**Derive cache key from Skills in read path to match write path.**

At line 48, `workspaceCacheKey(*resolved, nil)` discards the workspace's Skills. When a workspace has empty `ID` and `RootDir`, the write path (line 151) generates a key using resolved skills paths, but the read path returns an empty key and skips cache lookup. This creates a cache miss for valid entries.

Extract `resolved.Skills` into paths and pass them to `workspaceCacheKey` at line 48 to ensure read and write paths generate identical cache keys.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/registry_workspace_cache.go` around lines 43 - 51, In
workspaceSkillTargetLocked, the read path calls workspaceCacheKey(*resolved,
nil) and drops resolved.Skills causing cache misses; modify
workspaceSkillTargetLocked to extract the skill paths from resolved.Skills
(e.g., map to []string or the same shape used by the write path) and pass those
paths as the second argument to workspaceCacheKey so the read-key generation
matches the write-path key generation (ensure you use the same transformation of
resolved.Skills that the write path uses).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `workspaceSkillTargetLocked` computes the read-side cache key with `workspaceCacheKey(*resolved, nil)`, which discards resolver-provided skill paths. For workspaces that have neither `ID` nor `RootDir`, that makes valid cache entries unreachable even though the write path keys them by workspace skill paths.
- Fix approach: Derive the same `workspaceSkillPath` slice shape from `resolved.Skills` for the read path and add a regression test for skill-only workspace cache keys.
