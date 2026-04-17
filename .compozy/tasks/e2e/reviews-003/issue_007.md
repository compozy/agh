---
status: resolved
file: internal/daemon/tool_mcp_resources_test.go
line: 551
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57ziMT,comment:PRRC_kwDOR5y4QM645avB
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Assert the validation failure specifically.**

Both negative-path checks would pass on any unrelated error. Please assert the expected validation sentinel/type with `errors.Is`/`errors.As`, or at least a stable message fragment, so these tests fail for the right reason. As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".



Also applies to: 591-594

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/tool_mcp_resources_test.go` around lines 545 - 551, The test
currently only checks that validateAndEncodeTool(...) returned a non-nil error,
which is too broad; update the assertions to check for the specific validation
sentinel/type or stable message fragment (using errors.Is/errors.As or an
"ErrorContains" style check) so the failure is asserted for the intended
validation reason. Concretely, change the assertion around
validateAndEncodeTool(...) for the invalid Name case (and the similar case at
lines 591-594) to verify the returned error matches the expected validation
error value or contains the expected message fragment rather than simply being
non-nil, referencing validateAndEncodeTool and the test cases that pass
toolspkg.Tool with an invalid Name/Source.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The invalid-input checks in `TestValidateAndEncodeToolAndMCPServer` only assert that an error is non-nil, so they can pass on unrelated encode/decode failures.
  - The underlying resource codecs already surface stable validation sentinels and contextual messages, and the rest of the repo uses specific validation assertions for these paths.
  - Implemented: the invalid tool case now asserts `resources.ErrValidation` plus the stable `tool.name` fragment; the invalid MCP server case now asserts the stable `config: validate mcp resource spec` and `mcp_server.command is required` fragments.
  - Verification: `go test ./internal/daemon -run 'Test(ToolMCPSourceSyncerHandlesNilReceiverAndTriggerFailures|ToolMCPSourceSyncerSyncPropagatesProviderFailure|ToolMCPSourceSyncerReplacesCanonicalSnapshot|NewToolMCPPublisherFallsBackToNoopWithoutResourceRuntime|NewToolMCPPublisherBuildsSyncerWhenResourceRuntimeIsReady|NewToolMCPPublisherReturnsCodecResolutionErrors|ValidateAndEncodeToolAndMCPServer)$' -count=1`; `make verify`.
