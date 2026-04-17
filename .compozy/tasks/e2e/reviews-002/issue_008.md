---
status: resolved
file: internal/testutil/acpmock/fixture.go
line: 174
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57y10m,comment:PRRC_kwDOR5y4QM644c8n
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`Fixture.Validate` and `Fixture.Agent` disagree on name normalization.**

Line 167 validates with `TrimSpace`, but Line 190 compares raw `agent.Name`. A fixture entry like `"name": " demo "` can pass validation and still fail lookup via `Agent("demo")`.



<details>
<summary>🩹 Proposed fix</summary>

```diff
 	for idx, agent := range f.Agents {
 		name := strings.TrimSpace(agent.Name)
 		if name == "" {
 			return fmt.Errorf("acpmock: agents[%d].name is required", idx)
 		}
+		if agent.Name != name {
+			return fmt.Errorf("acpmock: agents[%d].name must not contain leading or trailing whitespace", idx)
+		}
 		if _, ok := seen[name]; ok {
 			return fmt.Errorf("acpmock: duplicate agent %q", name)
 		}
@@
 	for _, agent := range f.Agents {
-		if agent.Name == target {
+		if strings.TrimSpace(agent.Name) == target {
 			return agent, nil
 		}
 	}
```
</details>


Also applies to: 189-193

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/acpmock/fixture.go` around lines 167 - 174,
Fixture.Validate and Fixture.Agent disagree on name normalization: ensure both
use the same normalized key by trimming whitespace before validation, storage,
and lookup. In Fixture.Validate (where you build seen), normalize agent.Name
with strings.TrimSpace before checking empty/duplicate and store the trimmed
name in seen; in Fixture.Agent, compare using strings.TrimSpace(agent.Name) (or
trim the lookup key) so lookups for names like " demo " succeed; apply the same
trimming normalization to the other similar block referenced around the Agent
lookup (the code at the other duplicate-check/lookup sites).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `Fixture.Validate()` trims agent names for empty/duplicate checks, but `Fixture.Agent()` compares the raw stored name. A fixture with surrounding whitespace in `agents[].name` can validate successfully and still fail lookup by the normalized name.
- Fix plan: canonicalize stored agent names during fixture validation so validation, storage, and lookup all use the same trimmed identifier.
- Test impact: requires a focused update in `internal/testutil/acpmock/fixture_test.go` to cover normalized names.
- Resolution: implemented. fixture validation now canonicalizes stored agent names, and `fixture_test.go` covers lookup through a whitespace-padded fixture name.
- Verification: `go test ./internal/testutil/acpmock`, `make verify`.
