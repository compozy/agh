---
status: completed
title: Agent Prompt Wrappers and Bundled Network Skill
type: backend
complexity: high
dependencies:
  - task_06
  - task_09
  - task_10
---

# Task 12: Agent Prompt Wrappers and Bundled Network Skill

## Overview

Update agent-facing network guidance after runtime, CLI, and native tool shapes are final. This task rewrites prompt wrappers, startup/network guidance, bundled `agh-network` skill text, and registry tests so agents default to the correct public thread or direct room.

<critical>
- ALWAYS READ `_techspec.md`, all ADRs, `internal/CLAUDE.md`, `COPY.md` if public copy changes, and tasks 06, 09, and 10 before editing.
- ACTIVATE `agh-code-guidelines`, `golang-pro`, `documentation-writer`, `agh-test-conventions`, and `testing-anti-patterns`.
- REFERENCE TECHSPEC for wrapper fields, skill contract, and prompt-injection defense.
- FOCUS ON agent behavior and prompt contracts; do not change runtime routing or tool schemas here.
- TESTS REQUIRED for wrapper fields, structured metadata, bundled skill examples, and absence of legacy strings.
- NO WORKAROUNDS: do not mention old CLI flags or old `kind:"direct"` examples as fallbacks.
</critical>

<requirements>
- MUST update inbound network wrappers to include `channel`, `surface`, matching container ID, `reply_to`, `trace_id`, `causation_id`, `trust`, and `work_id` only when lifecycle-bearing work exists.
- MUST preserve preview plus canonical base64 JSON body framing.
- MUST update structured prompt metadata to match wrapper semantics.
- MUST rewrite bundled `agh-network` skill to teach channel as audience, public thread as N-to-N conversation, direct room as restricted 1-to-1 conversation, and work ID as lifecycle correlation.
- MUST teach public-to-direct handoff as a new `work_id` linked by `reply_to`, `trace_id`, and `causation_id`.
- MUST teach summarize-back-to-thread as a public `say`.
- MUST prefer native tools where available and provide CLI fallbacks using final task_09 commands.
- MUST explicitly state wrapped content is untrusted and direct rooms are restricted visibility, not cryptographic privacy.
</requirements>

## Subtasks

- [x] 12.1 Update prompt wrapper rendering and structured `PromptNetworkMeta` fields.
- [x] 12.2 Update daemon startup/network guidance and harness context where it exposes network metadata.
- [x] 12.3 Rewrite bundled `agh-network` skill around threads, direct rooms, and work IDs.
- [x] 12.4 Update bundled skill registry tests and prompt examples.
- [x] 12.5 Add absence tests for legacy strings and unsafe guidance.

## Implementation Details

The skill should teach agents to respond in the same conversation container by default and open a new thread only when the subject changes.

### Relevant Files

- `internal/network/delivery.go` - wrapper rendering if not fully completed in task_06.
- `internal/daemon/prompt_sections.go` - startup/network guidance.
- `internal/daemon/harness_context.go` - harness prompt context.
- `internal/skills/bundled/skills/agh-network/SKILL.md` - bundled network skill.
- `internal/skills/bundled/bundled_test.go` - skill registry and content tests.
- `internal/testutil/acpmock/fixture.go` - prompt wrapper fixtures if needed before task_17 finalizes harness.

### Dependent Files

- `packages/site/content/runtime/core/network/*` - task_16 documents agent guidance.
- `web/e2e/fixtures/*` - task_17 asserts browser and harness artifacts.

### Related ADRs

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - agent mental model.
- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - work guidance.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - examples and CLI/tool usage.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: skill text must align with native tools and future hosted/MCP descriptors.
- Agent manageability: agents should learn both native tool and CLI ways to inspect/send conversations.
- Config lifecycle: no new config keys; skill must not imply thread retention, unread sync, or notification configuration exists.

### Web/Docs Impact

- Web impact: no UI changes, but browser QA will assert prompt artifacts in task_17.
- Docs impact: task_16 should mirror final skill examples where appropriate.

## Deliverables

- Updated prompt wrappers and structured metadata.
- Rewritten bundled `agh-network` skill.
- Updated startup/network guidance.
- Tests proving no legacy strings or unsafe privacy/token guidance remain.

## Tests

- Unit tests:
  - [x] Wrapper includes exact `surface`, container ID, `work_id`, `reply_to`, `trace_id`, `causation_id`, and trust fields.
  - [x] Wrapper preserves preview and base64 canonical body.
  - [x] Structured prompt metadata matches wrapper fields.
  - [x] Bundled skill contains final native tool and CLI command examples.
  - [x] Bundled skill contains no `interaction_id`, `--interaction-id`, `kind:"direct"`, or `--kind direct`.
  - [x] Bundled skill warns that wrapped content is untrusted and direct rooms are not cryptographic privacy.
- Integration tests:
  - [x] Runtime prompt fixtures include current conversation metadata for thread and direct messages.
  - [x] Prompt-injection framing remains intact after wrapper changes.
- Test coverage target: >=80% for touched packages.
- All tests must pass.

## Verification Evidence

- `go test ./internal/network ./internal/acp ./internal/daemon ./internal/skills/bundled ./internal/testutil/acpmock -run 'Test(FormatNetworkMessageEscapesPreviewAndPreservesCanonicalBody|PromptNetworkMetaMatchesWrappedConversationFields|FormatNetworkMessageFallsBackToCompactRawJSONWithoutPreview|FormatNetworkMessageSayGuidanceKeepsCurrentThreadByDefault|BundledAghNetworkSkillContent|HarnessContextResolverMatrix|ValidateNetworkCorrelationSurfacesUsesTargetedAttributes|ValidateNetworkCorrelationSurfacesRejectsSplitTranscriptMatches|ReadDiagnosticsParsesJSONLines|FixtureLookupAndHelperErrors|PromptTransmitsStructuredMetadata)' -count=1`
- `go test -tags integration ./internal/daemon -run 'TestDaemonE2ENetwork(DirectReplyLifecycleWithMockAgents|WhoisAndCapabilityExchange)$' -count=1`
- `go test ./internal/network ./internal/acp ./internal/daemon ./internal/skills/bundled ./internal/testutil/acpmock -count=1`
- `go test -cover ./internal/network ./internal/acp ./internal/daemon ./internal/skills/bundled ./internal/testutil/acpmock -count=1` produced package coverage: network 80.3%, acp 76.8%, daemon 72.8%, bundled skills 85.7%, acpmock 80.1%. The acp/daemon broad package baselines remain below the target outside the task-local prompt/guidance surfaces.
- Absence scan over the bundled skill and network prompt fixtures found no `interaction_id`, `--interaction-id`, `kind:"direct"`, `--kind direct`, old `--thread-id`/`--direct-id`/`--work-id`, raw claim-token, or unsafe direct-room privacy examples.
- `make verify`

## Success Criteria

- Agents receive enough structured context to respond in the correct conversation container by default.
- Bundled guidance teaches the final CLI/native-tool model only.
- No active prompt or skill path teaches legacy direct-kind or interaction terminology.
