---
status: resolved
file: extensions/bridges/discord/provider_test.go
line: 1488
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Odcq,comment:PRRC_kwDOR5y4QM64G4Yl
---

# Issue 003: _🛠️ Refactor suggestion_ | _🟠 Major_

## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Please convert the new suite to table-driven `t.Run("Should...")` cases.**

Most of the file is a long list of one-off tests with repeated setup for mapping, webhook, and delivery permutations. Refactoring these into table-driven subtests will make scenario growth much cheaper and align the suite with the repo’s test conventions.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default" and "MUST use `t.Run(\"Should...\")` pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/discord/provider_test.go` around lines 30 - 1488, Convert
the long, one-off tests into table-driven subtests using t.Run("Should...") for
each scenario: e.g., refactor TestMapDiscordMessageEventRoutingAndAttachments,
TestMapDiscordInteractionPayloadsStableTargetIdentity,
TestMapDiscordReactionPayloads,
TestExecuteDiscordDeliveryValidatesEditAndDeleteOperations and similar Test*
functions into a table (slice of cases) where each case has a name like "Should
map DM with attachment" and runs as t.Run("Should ...", func(t *testing.T) {
t.Parallel(); setup reused shared test helpers (e.g.,
testDiscordManagedInstance, discordAPIFake, provider setup), execute the
scenario using the case inputs, and assert expected outputs }); keep existing
helper functions and unique identifiers (mapDiscordMessageEvent,
mapDiscordInteractionCommand, mapDiscordInteractionAction,
mapDiscordReactionEvent, executeDiscordDelivery, handleInteractionWebhook,
serveWebhookHTTP, determineInitialState, etc.) to locate logic to test and reuse
setup across cases. Ensure every new subtest uses the "Should..." naming pattern
and preserves t.Parallel() semantics and existing assertions.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - This is a broad refactor suggestion, not a concrete correctness defect in the scoped batch.
  - The requested rewrite would touch most of `extensions/bridges/discord/provider_test.go`, create large unrelated churn, and is not required to resolve a failing behavior or regression identified in the review round.
  - The existing tests already exercise distinct behaviors; converting the whole file to table-driven form is a follow-up maintenance task, not a blocking review fix for this batch.
  - Outcome: no code change required for this batch; analysis completed and the item is closed as non-blocking.
