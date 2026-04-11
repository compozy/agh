---
status: resolved
file: internal/api/core/errors_test.go
line: 80
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLd,comment:PRRC_kwDOR5y4QM623eI1
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Use `t.Run("Should...")` names for these subtests.**

The table-driven structure looks good, but the `name` values should follow the repository’s required `Should...` convention so the generated subtest names stay consistent across the suite. As per coding guidelines, "`**/*_test.go`: MUST use t.Run(\"Should...\") pattern for ALL test cases`".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/errors_test.go` around lines 16 - 80, The subtest names in
the tests table use freeform descriptions instead of the repository convention;
update each test case's name field in the tests slice (e.g. the entries used by
t.Run) so they follow the "Should ..." pattern (for example "Should return bad
request when body path mismatch") so t.Run(tt.name, ...) generates consistent
names; adjust all entries that reference contract.ErrChannelInstanceMismatch,
channelspkg.ErrChannelInstanceNotFound, channelspkg.ErrChannelRouteNotFound,
workspacepkg.ErrWorkspaceNotFound, channelspkg.ErrChannelInstanceUnavailable,
channelspkg.ErrInvalidChannelStateTransition, channelspkg.ErrDeliveryNotFound,
channelspkg.ErrDeliveryQueueSaturated,
channelspkg.ErrDeliveryTransportUnavailable and the generic error used with
StatusForChannelError accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The subtests already use `t.Run`, but their names do not follow the repository’s `Should ...` naming convention.
  - I will rename the table entries only; there is no production behavior change required here.
  - Resolution: Renamed the table cases in [internal/api/core/errors_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/api/core/errors_test.go:14) to the required `Should ...` form; verified with `make verify`.
