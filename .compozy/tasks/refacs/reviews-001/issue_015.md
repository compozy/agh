---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/testutil/skills_stub.go
line: 89
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYRsN,comment:PRRC_kwDOR5y4QM6-67EQ
---

# Issue 015: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Add compile-time interface assertion.**

Per coding guidelines, Go implementations should have compile-time interface assertions to verify the struct satisfies the expected interface. Other stubs in this package (e.g., `StubWorkspaceService`, `StubObserver`) include this assertion.


<details>
<summary>Add interface assertion</summary>

```diff
 func (s StubSkillsRegistry) SetEnabledForAgent(
 	name string,
 	resolved *workspacepkg.ResolvedWorkspace,
 	agentName string,
 	enabled bool,
 ) error {
 	if s.SetEnabledForAgentFn != nil {
 		return s.SetEnabledForAgentFn(name, resolved, agentName, enabled)
 	}
 	if s.SetEnabledFn != nil {
 		return s.SetEnabledFn(name, resolved, enabled)
 	}
 	return nil
 }
+
+var _ core.SkillsRegistry = (*StubSkillsRegistry)(nil)
```

Note: You'll need to import `core "github.com/pedronauck/agh/internal/api/core"` and use the appropriate interface name that `StubSkillsRegistry` is meant to implement.
</details>

As per coding guidelines: "Use compile-time interface assertions in Go to verify that implementations satisfy interfaces".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/testutil/skills_stub.go` around lines 10 - 89, Add a
compile-time interface assertion to ensure StubSkillsRegistry implements the
intended interface: import core "github.com/pedronauck/agh/internal/api/core"
and add a top-level var assertion referencing the interface (e.g., assert that
core.<InterfaceName> is implemented by *StubSkillsRegistry) so the compiler will
catch mismatches; place this assertion near the other stubs' assertions in the
file.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  `StubSkillsRegistry` is meant to satisfy `core.SkillsRegistry`, but the file lacks a compile-time assertion. Adding the assertion makes interface drift compile-time visible, matching the other stubs in this package.
