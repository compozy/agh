# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task_06: store-backed runtime routing, delivery wrappers, prompt metadata, and task ingress metadata with tests, clean verification, tracking updates, and one local commit.
- Store conversation repository must commit before prompt delivery and outbound publish for conversation-bearing runtime sends.
- Task ingress may attach `metadata_json.network_work_id` only as correlation; task_runs keeps claim/lease/heartbeat/complete/fail/release/cancel authority.

## Important Decisions
- Manager is the runtime commit boundary: prepare/route envelopes, write the conversation repository, then perform prompt delivery or publish.
- Audit writer remains an audit sink; old audit-side `WriteNetworkMessage` mirroring is not the authoritative conversation path.
- Direct-surface router delivery must filter by deterministic direct-room membership in addition to channel and optional target peer.
- Task ingress metadata is server-derived from trusted ingress context; client-provided metadata remains opaque except for protected network correlation keys.

## Learnings
- Baseline targeted network tests pass: `go test ./internal/network -run 'TestFormatNetworkMessage|TestEnqueueRunFromPeer' -count=1`.
- Active gaps before implementation: manager sends/receives do not call the conversation repository before side effects, audit writer still mirrors timeline writes through `WriteNetworkMessage`, prompt metadata still has `interaction_id`, and task ingress forwards run metadata unchanged.
- Public API/native-tool `interaction_id` fields still exist; task_06 scope is runtime/prompt/task-ingress, while public contract hardening is owned by task_08 unless tests force narrower cleanup.
- First full `make verify` reached Go lint/typecheck and exposed one task-owned leftover in `internal/testutil/acpmock`: prompt-network fixture matching still referenced `PromptNetworkMeta.InteractionID`. The matcher now uses `surface`, `thread_id`, `direct_id`, and `work_id`.
- Later full `make verify` passed frontend format/lint/typecheck/tests/build and Go lint (`0 issues`) but failed during unrelated `internal/extension` race tests on TempDir cleanup. Exact failing tests passed in isolation under `-race`; the package-level failure reproduced in a different extension test, indicating a pre-existing async cleanup/load issue outside task_06 surfaces.
- A subsequent full `make verify` rerun passed end-to-end with `0 issues`, `DONE 8160 tests`, and `OK: all package boundaries respected`.
- After self-review cleanup and tracking updates, final full `make verify` passed with `0 issues`, `DONE 8160 tests`, and `OK: all package boundaries respected`.
- After recording final memory evidence, the pre-commit `make verify` gate passed again with `0 issues`, `DONE 8160 tests`, and `OK: all package boundaries respected`.
- Created local commit `3fae5a9b` (`feat: wire network conversation runtime routing`) containing only task-scoped runtime/prompt/task-ingress code and tests.
- Post-commit `make verify` passed: frontend format/oxlint/typecheck/tests/build, Go lint `0 issues`, `DONE 8160 tests`, and `OK: all package boundaries respected`.

## Files / Surfaces
- internal/network/manager.go
- internal/network/router.go
- internal/network/audit.go
- internal/network/delivery.go
- internal/network/tasks.go
- internal/acp/types.go
- internal/session/manager_prompt.go
- internal/daemon/boot.go
- internal/testutil/acpmock/fixture.go
- internal/network/*_test.go and internal/acp/*_test.go as needed

## Errors / Corrections
- Corrected ACP mock network fixture matching after lint/typecheck failure from the removed `PromptNetworkMeta.InteractionID` field.
- Corrected Go lint findings from audit decoupling: removed an unused audit presence mutex/test helper and formatted the inbound persistence rejection audit call.
- Self-review correction: removed the now-dead audit presence-window option and manager wiring after audit-side timeline writes moved to the conversation repository path.

## Ready for Next Run
- Implementation, tracking updates, local commit, and post-commit verification are complete; final report is ready.
