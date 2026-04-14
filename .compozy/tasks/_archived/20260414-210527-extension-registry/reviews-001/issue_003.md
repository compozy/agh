---
status: resolved
file: internal/cli/skill_marketplace.go
line: 255
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562WUb,comment:PRRC_kwDOR5y4QM63madh
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep skill updates pinned to the recorded registry source.**

Line 222 already has the persisted registry provenance, but Lines 228 and 254 ignore it and go back through the aggregated `skillRegistry`. Since `loadSkillRegistry` builds a `MultiRegistry` over all configured sources, the same slug can be resolved from a different backend during update, silently switching the installed skill’s origin and sidecar metadata. Please select/filter the original source from `installed.Provenance.Registry` before calling `CheckUpdate` and reinstalling.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/skill_marketplace.go` around lines 214 - 255, The update flow
must use the original registry recorded in installed.Provenance.Registry rather
than the aggregated registry; before calling registry.CheckUpdate and
installMarketplaceSkill in updateMarketplaceSkill, look up or filter the
aggregated skillRegistry to get the specific backend matching
installed.Provenance.Registry (return an error if not found) and call that
specificRegistry.CheckUpdate(ctx, slug, currentVersion) and
specificRegistry.Install/ or pass specificRegistry into installMarketplaceSkill
so the update and reinstall come from the same provenance; update calls
referencing registry.CheckUpdate and installMarketplaceSkill to use the selected
specific registry instead.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `updateMarketplaceSkill()` uses the aggregated `skillRegistry` for both `CheckUpdate()` and reinstall, so an installed skill can silently switch backends when multiple sources expose the same slug. The persisted `installed.Provenance.Registry` is the correct pin. I will resolve a source-specific registry from that provenance, use it for update checks and reinstall, and add regression coverage in `internal/cli/skill_test.go` even though the test file is outside the listed scope because no in-scope file covers this flow.
- Resolution: Added provenance-pinned registry resolution for marketplace skill updates and reinstalls in `internal/cli/skill_marketplace.go`, plus regression coverage in `internal/cli/skill_test.go`.
- Verification: `go test ./internal/cli`; `make verify`
