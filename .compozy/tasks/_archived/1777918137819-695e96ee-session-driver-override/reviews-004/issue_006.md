---
status: resolved
file: internal/observe/observer_test.go
line: 519
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59R11d,comment:PRRC_kwDOR5y4QM663fBt
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Provider-path tests are over-constrained to a single happy path.**

Line 503-519 and Line 581 hardcode `"claude"` in both workspace resolution and session creation, so these tests won’t catch regressions in provider override, mismatch, or legacy-empty-provider repair paths introduced by this PR.

<details>
<summary>Suggested refactor to make provider scenarios testable</summary>

```diff
-func newSession(id string, state session.State, workspace string, now time.Time) *session.Session {
+func newSession(id string, state session.State, workspace string, provider string, now time.Time) *session.Session {
 	return &session.Session{
 		ID:           id,
 		Name:         strings.ToUpper(id),
 		AgentName:    "coder",
-		Provider:     "claude",
+		Provider:     provider,
 		WorkspaceID:  observerWorkspaceID,
 		Workspace:    workspace,
 		State:        state,
 		ACPSessionID: "acp-" + id,
 		CreatedAt:    now,
 		UpdatedAt:    now,
 	}
}
```

```diff
-			Agents: []aghconfig.AgentDef{{
-				Name:     "coder",
-				Provider: "claude",
-				Prompt:   "You are a coding assistant.",
-			}},
+			Agents: []aghconfig.AgentDef{
+				{
+					Name:     "coder",
+					Provider: "claude",
+					Prompt:   "You are a coding assistant.",
+				},
+				{
+					Name:     "reviewer",
+					Provider: "openai",
+					Prompt:   "You are a reviewer.",
+				},
+			},
```
</details>



As per coding guidelines, "Focus on critical paths: workflow execution, state management, error handling."


Also applies to: 581-581

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/observer_test.go` around lines 503 - 519, The test hardcodes
the provider "claude" in the fakeObserveWorkspaceResolver
ResolvedWorkspace.AgentDef and in session creation, which misses
provider-override/mismatch/empty-provider scenarios; update the test to
parameterize the provider value (use a table-driven approach) and add cases for
(1) matching provider, (2) workspace provider mismatch vs session override, and
(3) legacy/empty workspace provider that should be repaired or defaulted;
specifically modify the
fakeObserveWorkspaceResolver.expectedRef/ResolvedWorkspace.Agents entry and the
session creation in the test harness to read the provider from the test case,
and assert the resulting provider used by the observe/session code in each case
to catch regressions in provider resolution logic.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
- The provider-repair and provider-persistence paths mentioned in the comment are already covered in `internal/observe/reconcile_test.go` and `internal/observe/helpers_test.go`.
- The cited helper in `observer_test.go` is used for live-session notifier flows, not legacy metadata repair, so adding mismatch or empty-provider cases there would duplicate existing coverage without exercising new production branches.
