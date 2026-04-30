---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/client_test.go
line: 810
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulJ1,comment:PRRC_kwDOR5y4QM680KIT
---

# Issue 016: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap these new cases in `t.Run("Should ...")` subtests.**

These additions only call `t.Parallel()` at the top level, so they bypass the test structure the repo requires for all cases. Keeping them as named subtests also makes it much easier to extend each API surface with more scenarios without growing one monolithic test body.


As per coding guidelines, `**/*_test.go`: `Use t.Run("Should ...") subtests with t.Parallel as default` and `MUST use t.Run("Should...") pattern for ALL test cases`.


Also applies to: 812-858, 860-914, 916-973

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/client_test.go` around lines 691 - 810, The test
TestUnixSocketClientHostedMCPMethods contains multiple assertions for different
API flows but they are all in one top-level test with a single t.Parallel(),
violating the repo rule to use named subtests; split the monolithic body into
separate t.Run("Should ...") subtests for each scenario (e.g., "Should bind
hosted MCP", "Should fetch projection", "Should stream projection", "Should call
hosted tool", "Should release hosted MCP") and call t.Parallel() at the start of
each subtest; move the relevant request setup, client call (BindHostedMCP,
HostedMCPProjection, StreamHostedMCPProjection, CallHostedMCP, ReleaseHostedMCP)
and assertions into their corresponding t.Run blocks while keeping the same
httpClient roundTripper behavior, and repeat this pattern for the other test
blocks noted (the other ranges mentioned) so every case uses t.Run +
t.Parallel().
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestUnixSocketClientHostedMCPMethods`, `TestUnixSocketClientStreamsSessionEvents`, `TestUnixSocketClientTaskExecutionMethods`, and `TestUnixSocketClientAgentContextAndSpawnMethods` run scenario bodies directly under the top-level test. They should be split into `Should ...` subtests with `t.Parallel()` where safe. Some round-tripper setup can stay inside each subtest to avoid sharing assertion state.
