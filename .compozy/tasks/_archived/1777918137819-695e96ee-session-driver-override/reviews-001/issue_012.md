---
status: resolved
file: internal/session/manager_prompt.go
line: 231
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM581azf,comment:PRRC_kwDOR5y4QM66RFPG
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Verify the call site now passes request context.
rg -nP 'lookupPromptSession\(' internal/session/manager_prompt.go

# Verify no runtime use of context.Background() remains in this lookup path.
rg -nP 'readMetaWithContext\(context\.Background\(\),\s*target\)' internal/session/manager_prompt.go
```

Repository: compozy/agh

Length of output: 259

---



</details>

**Thread `context.Context` through prompt session lookup to avoid dropping cancellation**

Line 231 uses `context.Background()` instead of the available request context, bypassing caller cancellation and deadlines. Update `lookupPromptSession` to accept context as first parameter and thread it through to `readMetaWithContext`.

<details>
<summary>Suggested fix</summary>

```diff
@@
-func (m *Manager) submitPromptRequest(ctx context.Context, req promptRequest) (<-chan acp.AgentEvent, error) {
-	session, err := m.lookupPromptSession(req.target)
+func (m *Manager) submitPromptRequest(ctx context.Context, req promptRequest) (<-chan acp.AgentEvent, error) {
+	session, err := m.lookupPromptSession(ctx, req.target)
@@
-func (m *Manager) lookupPromptSession(target string) (*Session, error) {
+func (m *Manager) lookupPromptSession(ctx context.Context, target string) (*Session, error) {
@@
-	meta, metaErr := m.readMetaWithContext(context.Background(), target)
+	meta, metaErr := m.readMetaWithContext(ctx, target)
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/manager_prompt.go` at line 231, lookupPromptSession
currently calls m.readMetaWithContext(context.Background(), target) which drops
caller cancellation/deadlines; change lookupPromptSession to accept ctx
context.Context as its first parameter and pass that ctx through to
m.readMetaWithContext(ctx, target). Update the lookupPromptSession signature and
all its call sites to thread the incoming context through so
cancellation/deadlines propagate to readMetaWithContext and any downstream
operations.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `lookupPromptSession` falls back to `context.Background()` when it checks persisted metadata, so prompt submission can ignore caller cancellation/deadlines on the not-active lookup path.
- Fix plan: thread the incoming context through prompt-session lookup. This requires a minimal scope exception to update the shared helper call sites in `internal/session/synthetic_prompt.go`, because they rely on the same lookup function.
