---
status: resolved
file: internal/session/manager.go
line: 66
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dM9,comment:PRRC_kwDOR5y4QM65IPER
---

# Issue 025: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Clear synthetic queue state when a target/session is removed.**

These maps add per-key synthetic dispatch state, but the existing `remove`/`removeActive` paths only delete from `sessions`, `pending`, and `finalizing`. A stopped session can therefore leave queued prompts or a stale `dispatching=true` marker behind indefinitely, which leaks memory and can block later dispatch for the same key. Please purge the synthetic maps alongside the session maps.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/manager.go` around lines 64 - 66, The remove/removeActive
code paths currently only clear sessions/pending/finalizing and must also purge
per-key synthetic state to avoid leaks; under syntheticMu lock, delete entries
for the session key from syntheticQueues and syntheticDispatching (and ensure
any queuedSyntheticPrompt resources are dropped) whenever a session is removed
(in the remove and removeActive functions), and ensure syntheticMu is held while
reading/modifying syntheticQueues/syntheticDispatching to prevent races.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `remove` and `removeActive` only purge `sessions`, `pending`, and `finalizing`. They do not clear `syntheticQueues` or `syntheticDispatching`, so a removed session can retain queued synthetic prompts and/or a stale dispatch marker indefinitely.
- Fix approach: purge the per-session synthetic state under `syntheticMu` whenever a session is removed, and drop any queued synthetic prompt channels so callers do not hang on removed sessions.
