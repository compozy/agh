# Task Memory: task_16.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task 16 by adding a production Linear bridge provider under `extensions/bridges/linear` on top of `internal/bridgesdk`, with provider-owned auth/mode switching in `provider_config`, unit coverage, conformance coverage, integration coverage, clean verification, and tracking updates.

## Important Decisions
- Treat the task spec plus `_techspec.md`/ADRs as the approved design baseline for this run.
- Mirror the production provider pattern used by `extensions/bridges/github` and other existing providers instead of inventing a Linear-specific runtime shape.
- Keep provider-owned mode branching fully local to Linear via `provider_config` (`comments` vs `agent_sessions`, `api_key` vs `oauth`).
- Leave unrelated dirty worktree files untouched; only task-local memory/tracking files and Linear provider surfaces should change.
- Use OAuth client-credentials for Linear `oauth` auth mode so the provider can derive and refresh bearer tokens from `client_id` / `client_secret` without introducing daemon-global token state.
- Require `provider_config` to carry Linear tenant/mode ownership (`organization_id`, `mode`, `auth_mode`, webhook settings, optional API/token URLs) and validate that shared bridge semantics remain unchanged.
- Use chat-sdk-compatible Linear thread IDs so comments route by root comment and agent sessions route by `(issue, root comment, session)` for follow-up delivery.
- Keep agent-session outbound behavior append-only: emit new activities from progressive deltas and reject edit/delete attempts in that mode.

## Learnings
- The `.resources/chat` Linear adapter uses a single webhook endpoint with `linear-signature` HMAC verification, comment vs agent-session ingress split, issue/comment/session thread IDs, and append-only agent-session delivery semantics.
- The existing AGH providers all reconcile managed instance configs after initialize, report per-instance initial state through the shared Host API, and use provider-local metadata in delivery requests to preserve follow-up routing context.
- Linear webhook payloads include `organizationId`, a plain hex `linear-signature`, and an optional `webhookTimestamp`; the chat-sdk tests treat timestamps older than one minute as invalid.
- Reaction webhooks lack `issueId`; task 16 can stay within scope by focusing on comment and agent-session ingress/delivery while leaving richer reaction lookup as later follow-up work if needed.
- Provider-local unit coverage had to exercise runtime startup, shared-webhook ingress, delivery markers, shutdown, and helper branches inside `extensions/bridges/linear` because the separate `internal/extension` integration suite does not count toward the package coverage target.

## Files / Surfaces
- `.compozy/tasks/bridge-adapters/memory/task_16.md`
- `extensions/bridges/github/*`
- `extensions/bridges/slack/*`
- `extensions/bridges/gchat/*`
- `extensions/bridges/linear/*`
- `internal/bridgesdk/*`
- `internal/extensiontest/bridge_adapter_harness.go`
- `internal/extension/linear_provider_integration_test.go`
- `.resources/chat/packages/adapter-linear/src/index.ts`
- `.resources/chat/packages/adapter-linear/src/types.ts`

## Errors / Corrections
- Linear initially failed conformance after initialize because `isNotInitializedRPCError` did not recognize `subprocess.RPCError`; aligning it with the shared provider pattern restored the retry behavior for early `bridges/instances/get` calls.
- Initial package coverage stalled below the required threshold until runtime-local tests were added for initialize, webhook ingress, delivery error paths, shutdown, and helper branches.

## Ready for Next Run
- Task 16 is implementation-complete and verified. Fresh evidence: `go test -count=1 ./extensions/bridges/linear -cover` passed at `80.2%`, `go test -tags integration ./internal/extension -run 'TestLinearProvider' -count=1` passed, and `make verify` passed after the last code change.
