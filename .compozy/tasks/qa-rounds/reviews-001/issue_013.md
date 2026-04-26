---
status: resolved
file: internal/observe/observer.go
line: 632
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vP,comment:PRRC_kwDOR5y4QM67Z0NK
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Registry recovery should not re-cache stopped sessions.**

`OnSessionStopped` explicitly evicts the session from `o.sessions`, but this branch adds it back for any persisted registry row. A late event for a completed session will therefore repopulate the in-memory cache and keep it there indefinitely, because stopped sessions remain in the registry. Consider making registry recovery a one-shot lookup, or only caching sessions that are still live.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/observer.go` around lines 621 - 632, The registry recovery
currently re-adds stopped sessions into the in-memory cache by calling
o.trackSession in the ListSessions loop; change this to avoid caching sessions
that are no longer live. Specifically, in the loop that iterates sessions
returned by o.registry.ListSessions, only call o.trackSession for sessions whose
persisted metadata indicates they are active (e.g., check a liveliness field
like info.State/Status not equal to "stopped" or !info.Stopped); otherwise,
return the observed snapshot without calling o.trackSession (or treat recovery
as a one-shot lookup). Also ensure this behavior is consistent with
OnSessionStopped which evicts from o.sessions so stopped sessions are never
re-added.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `OnSessionStopped` evicts a session from the observer cache, but registry recovery currently calls `trackSession` for every persisted row, including `stopped` sessions. A late event can therefore re-cache stopped sessions indefinitely. Fix by treating stopped registry rows as one-shot recovery snapshots and only tracking non-stopped rows.
