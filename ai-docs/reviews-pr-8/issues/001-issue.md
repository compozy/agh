# Issue 1 - Review Thread Comment

**File:** `internal/skills/hooks.go:76`
**Date:** 2026-04-08 12:09:54 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: the issue is real. Hooks were incorrectly reusing `allowed_marketplace_mcp`, which let MCP approval implicitly authorize lifecycle hooks, and marketplace gating relied on mutable `skill.Meta.Name`. The fix adds a dedicated `skills.allowed_marketplace_hooks` config field and keys marketplace consent to immutable provenance identifiers (`slug`, `registry:slug`, or `hash`) instead of display metadata.

## Body

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cat -n internal/skills/hooks.go | head -120
```

Repository: compozy/agh

Length of output: 3400

---

🏁 Script executed:

```shell
cat -n internal/config/config.go | grep -A 10 -B 5 "SkillsConfig\|AllowedMarketplaceMCP"
```

Repository: compozy/agh

Length of output: 2660

---

🏁 Script executed:

```shell
rg -n "func marketplaceSkillAllowed|func marketplaceAllowlist" internal/skills/
```

Repository: compozy/agh

Length of output: 263

---

🏁 Script executed:

```shell
rg -n "type Skill struct" internal/skills/ -A 20
```

Repository: compozy/agh

Length of output: 1204

---

🏁 Script executed:

```shell
sed -n '119,150p' internal/skills/mcp.go
```

Repository: compozy/agh

Length of output: 815

---

🏁 Script executed:

```shell
rg -n "type Provenance struct" internal/skills/ -A 15
```

Repository: compozy/agh

Length of output: 911

---

</details>

**Separate hook consent from MCP consent and key it to immutable marketplace identity.**

`HookRunner` sources subprocess-hook consent from `AllowedMarketplaceMCP` with no separate hook allowlist, so enabling a marketplace skill's MCP server also enables its lifecycle hooks—a much broader privilege. Additionally, `marketplaceSkillAllowed` gates on `skill.Meta.Name`, which is mutable SKILL.md metadata and spoofable. Split this into a dedicated hook allowlist and key the check to immutable provenance fields (`slug`, `registry`, or `hash`) instead.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/hooks.go` around lines 68 - 76, NewHookRunner currently
initializes HookRunner.allowedMarketplace from cfg.AllowedMarketplaceMCP which
conflates MCP consent with hook consent and marketplaceSkillAllowed checks
skill.Meta.Name (mutable); change this by adding a separate hook allowlist in
the config (e.g., AllowedMarketplaceHooks or AllowedHookMarketplaceIDs),
initialize HookRunner.allowedHooks from that list instead of
AllowedMarketplaceMCP in NewHookRunner, and update the gating function
marketplaceSkillAllowed (or create a new hookAllowed) to validate against
immutable marketplace identity fields (slug, registry, or hash) on the skill
rather than skill.Meta.Name; keep using cloneStrings for the new list to match
existing patterns and ensure all references to allowedMarketplace are adjusted
to use the distinct hook allowlist where lifecycle hooks are evaluated.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Resolve

Thread ID: `PRRT_kwDOR5y4QM55mbZm`

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=PRRT_kwDOR5y4QM55mbZm
```

---

_Generated from PR review - CodeRabbit AI_
