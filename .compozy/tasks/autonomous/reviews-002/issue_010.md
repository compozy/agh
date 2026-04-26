---
status: resolved
file: internal/cli/agent_kernel_test.go
line: 720
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59q6tn,comment:PRRC_kwDOR5y4QM67YhqO
---

# Issue 010: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Use the repo’s `Should...` subtest pattern throughout this new test file.**

Several new tests here run assertions directly in the top-level test function (`TestMeCommandJSONReturnsValidatedIdentity`, `TestMeContextCommandJSONKeepsStableSectionOrder`, `TestSpawnCommandMapsBoundedChildRequest`, etc.) instead of wrapping the case in `t.Run("Should...")`. Please move each standalone case into a named subtest and keep `t.Parallel()` inside the subtest body.


As per coding guidelines, `MUST use t.Run("Should...") pattern for ALL test cases`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/agent_kernel_test.go` around lines 16 - 720, Several top-level
test functions (e.g., TestMeCommandJSONReturnsValidatedIdentity,
TestMeContextCommandJSONKeepsStableSectionOrder,
TestSpawnCommandMapsBoundedChildRequest,
TestChannelSendRejectsMissingInputsAndInvalidIdentity,
TestAgentCommandsRejectMissingIdentityBeforeAgentCalls,
TestChannelListCommandJSONReturnsVisibleChannels,
TestChannelSendPreservesCoordinationMetadataAndRejectsClaimToken,
TestChannelReplySendsOnlyMessageIDAndBodyWhenMetadataIsResolvedServerSide,
TestChannelRecvJSONLOutputEmitsOneObjectPerMessage,
TestAgentCommandsRenderHumanAndToonOutputs) contain assertions directly in the
top-level test; wrap each of these logical cases in t.Run("Should ...") subtests
(use descriptive "Should..." titles) and move any t.Parallel() calls from the
top-level function into the body of each subtest so each subtest calls
t.Parallel() at its start; ensure any per-case setup (stubClient, deps, and
client.fn assignments) is inside the subtest body so tests remain isolated and
still call the same functions (e.g., agentMeFn, agentContextFn, agentSpawnFn,
agentChannelSendFn, agentChannelRecvFn, agentChannelReplyFn) as before.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: Several new tests in `internal/cli/agent_kernel_test.go` execute assertions directly in top-level test bodies, and some table cases use names that do not start with `Should`. This violates the repository test-shape convention. The fix is to wrap each logical case in `t.Run("Should ...")` and move `t.Parallel()` into the subtest body while preserving isolated setup.
- Resolution: Wrapped/renamed the affected CLI agent kernel tests with `Should...` subtests and verified with focused tests plus full `make verify`.
