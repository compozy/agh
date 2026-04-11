---
status: resolved
file: internal/session/manager_helpers.go
line: 90
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZn,comment:PRRC_kwDOR5y4QM623eZ5
---

# Issue 026: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't treat network-join failure as warning-only.**

At this point the session is already activated and persisted. If `joinNetworkPeer` fails, the session stays "healthy" locally but never becomes discoverable in its space, which is a silent degraded state for the new network path. Please either fail/rollback activation here, or persist a retriable degraded status instead of only logging and continuing.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/manager_helpers.go` around lines 88 - 90, Do not treat
joinNetworkPeer failures as warnings only: when m.joinNetworkPeer(ctx, session)
returns an error, either abort/rollback activation by returning that error up
from the activation path (so activation is not marked persisted/healthy), or set
a retriable degraded state on the session and persist it before continuing.
Concretely, replace the m.sessionLogger(...).Warn branch with logic that (a)
returns the error from the enclosing activation function (to let the caller
rollback), or (b) calls a session status setter (e.g.
session.SetStatus(DegradedRetriable)) and then persists the change via your
session persistence method (e.g. m.store.UpdateSession or m.persistSession), and
log the failure including session.Info().Space and the error. Ensure the chosen
path consistently prevents the session from remaining marked healthy/activated
when joinNetworkPeer fails.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `activateAndWatch` currently activates the session, stores active metadata, and then logs `joinNetworkPeer` failures as warning-only. That leaves the process running locally while the session is undiscoverable in its configured space, which is a silent degraded state. The fix is to treat the join failure as activation failure: roll back the activation and return the error so create cleans up fully and resume restores stopped metadata through the existing failed-resume recovery path.
  Resolved by converting the warning-only branch into rollback-and-return behavior in `internal/session/manager_helpers.go`. Added create/resume lifecycle coverage in `internal/session/manager_hooks_test.go`. Verified with package tests and a clean `make verify`.
