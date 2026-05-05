# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 12 agent-facing network prompt/guidance hard cut after runtime, CLI, and native tools are final.
- Completion requires wrapper/structured metadata updates, bundled `agh-network` skill rewrite, prompt/fixture tests, absence tests for legacy/unsafe strings, clean verification, tracking updates, and one local commit when staging can stay task-scoped.

## Important Decisions
- Scope stays on prompt contracts and agent guidance; runtime routing and tool schemas are treated as already finalized by tasks 06, 09, and 10.
- `PromptNetworkMeta` should carry the wrapper trust marker so structured metadata matches the rendered `<network-message trust="untrusted">` contract.
- Wrapper/metadata `work_id` should be emitted only through the lifecycle-bearing reuse rule; non-lifecycle kinds do not echo an incidental `WorkID`.
- E2E prompt assertions should verify wrapper/transcript metadata and delivered file-audit rows for durable conversation messages; conversation-store writes own the sent side.

## Learnings
- Pre-change signal: `internal/acp.PromptNetworkMeta` has no trust field, while `internal/network/delivery.go` renders `trust="untrusted"` only in XML.
- Pre-change signal: bundled `agh-network` does not mention the final thread/direct/work native tool IDs or final `agh network threads/directs/work` CLI read commands.
- Pre-change signal: active daemon/acpmock integration fixtures still contain old `--thread-id`, `--direct-id`, `--work-id`, and `kind:"direct"` fixture expectations even though task 09/10 hard-cut those surfaces.
- Baseline focused tests pass before implementation: `go test ./internal/network ./internal/acp ./internal/daemon ./internal/skills/bundled ./internal/testutil/acpmock -run 'Test(FormatNetworkMessageEscapesPreviewAndPreservesCanonicalBody|BundledAghNetworkSkillContent|HarnessContextResolverMatrix|PromptInputCompositeIntegrationPreservesStoredMessagesAcrossUserAndNetworkTurns|ValidateNetworkCorrelationSurfacesUsesTargetedAttributes|TurnMatchNetwork)' -count=1`.
- Implemented signal: `PromptNetworkMeta` and acpmock network matchers now include `trust`, wrapper rendering uses the shared untrusted marker, and `work_id` is suppressed for non-lifecycle-bearing messages.
- Implemented signal: bundled `agh-network` now teaches channel/thread/direct/work mental model, final native tool IDs, final task_09 CLI commands, public-to-direct handoff with new work, summarize-back-to-thread, untrusted wrapper framing, and direct-room visibility limits.
- Integration correction: public-thread `capability` with `work_id` needs `--to` because runtime lifecycle work is directed even when the container is a public thread.
- Repeatable unrelated integration signal: `TestPromptInputCompositeIntegrationPreservesStoredMessagesAcrossUserAndNetworkTurns` and `TestHarnessContextIntegrationStartupAndPromptShareResolverPolicy` currently miss `<current-available-skills>` before this task's network fixture path; task-specific network E2E passes after the wrapper/fixture updates.
- Verification evidence: focused task tests passed; task-specific network integration E2E passed with `-tags integration`; touched package tests passed; coverage was `internal/network` 80.3%, `internal/acp` 76.8%, `internal/daemon` 72.8%, `internal/skills/bundled` 85.7%, `internal/testutil/acpmock` 80.1%.
- Final verification evidence: `make verify` passed before commit and again after local commit `96996f7b` (`feat: align network prompt wrappers and skill`), including frontend format/lint/typecheck/test/build, Go lint with 0 issues, 8,386 Go tests, package-boundary check, and build.

## Files / Surfaces
- Production surfaces touched: `internal/acp/types.go`, `internal/network/delivery.go`, `internal/skills/bundled/skills/agh-network/SKILL.md`, `internal/testutil/acpmock/fixture.go`.
- Test/fixture surfaces touched: `internal/network/delivery_test.go`, `internal/acp/client_test.go`, `internal/testutil/acpmock/testdata/network_collaboration_fixture.json`, `internal/testutil/acpmock/fixture_test.go`, `internal/daemon/daemon_network_collaboration_integration_test.go`, `internal/daemon/network_e2e_assertions_test.go`, `internal/daemon/prompt_input_composite_integration_test.go`, and `internal/skills/bundled/bundled_test.go`.

## Errors / Corrections
- Fixed initial integration E2E failures caused by old fixture assumptions: durable conversation messages should be asserted through delivered file-audit rows, and public-thread capability work must include a directed `to` peer.
- The `agh-test-conventions` heuristic script reports broad pre-existing naming/inline-case issues in touched test files; no task-scoped rewrite was performed because `make verify` is the repository gate and the findings are unrelated cleanup.

## Ready for Next Run
- Task code, tests, workflow memory, tracking, task-scoped commit, and post-commit verification are complete. Tracking/memory files remain unstaged by design; task code is committed in `96996f7b`.
