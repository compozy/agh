---
status: resolved
file: internal/store/globaldb/global_db_network_channels_test.go
line: 300
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59ReBE,comment:PRRC_kwDOR5y4QM662-g3
---

# Issue 005: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Use `t.Run("Should ...")` subtests for this new suite.**

These scenarios are all top-level tests right now, but this repo’s Go test convention requires subtests as the default pattern. Please group them under parent tests with `t.Run("Should ...")`, and keep `t.Parallel()` on the independent cases after the split.



As per coding guidelines, `**/*_test.go`: MUST use `t.Run("Should...")` pattern for ALL test cases and use table-driven tests with subtests (`t.Run`) as default pattern for Go tests.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_network_channels_test.go` around lines 15 -
300, You need to convert the top-level Test* functions for network channel
behavior into subtests using the t.Run("Should ...") pattern: create a parent
test (e.g. TestNetworkChannelsSuite) and move each existing test body (from
TestOpenGlobalDBCreatesNetworkChannelsSchema,
TestGlobalDBWriteAndListNetworkChannels, TestGlobalDBGetNetworkChannelNotFound,
TestGlobalDBDeleteNetworkChannel,
TestGlobalDBDeleteWorkspaceCascadesNetworkChannels,
TestGlobalDBListNetworkChannelsWrapsTimestampParseFailures,
TestMigrateGlobalSchemaRebuildsNetworkChannelsWithWorkspaceForeignKey) into its
own t.Run("Should <describe action>") { ... } block, keeping each subtest's
t.Parallel() where present and preserving all calls to helpers like
openTestGlobalDB, registerWorkspaceForGlobalTests, globalDB.WriteNetworkChannel,
globalDB.GetNetworkChannel, ListNetworkChannels, migrateGlobalSchema,
tableHasForeignKey, etc.; leave helper names and assertions unchanged, only
refactor test entry points into subtests under a single parent test function.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: The network-channel database coverage in `internal/store/globaldb/global_db_network_channels_test.go` is split across several top-level `Test*` functions even though the scenarios belong to one cohesive suite. That diverges from the repository preference for `t.Run("Should ...")`-style subtests as the default test shape.
- Fix plan: Group the network-channel scenarios under a single parent suite test, keep each scenario as a descriptive `t.Run("Should ...")`, and preserve `t.Parallel()` for independent cases.
- Outcome: Grouped the network-channel database scenarios under a single `TestNetworkChannels` suite with `Should ...` subtests. Verified with `go test ./internal/store/globaldb -count=1` and `make verify`.
