# BUG-20260712-reasoning-evidence-attribution: Background ACP sessions pollute the reasoning gate's protocol evidence

- **Status:** open
- **Impact (user-side):** Friction
- **Severity:** Medium · **Priority:** P2
- **Persona Affected:** Bruno and Ada as release operators relying on deterministic runtime gates
- **Journey Step:** J-21 apply advertised reasoning before first prompt
- **Scenarios:** RT-061 (automation evidence only; the previous user-session verdict is unchanged)
- **Found:** 2026-07-12 · **Report:** docs/qa/reports/2026-07-12-hermes-bridge.md
- **Origin:** n/a

## Summary

The release operator cannot complete the runtime E2E gate reliably because the reasoning suite reads one diagnostics file per registered agent while multiple ACP processes can write to it. A background memory-extractor session reuses the same fixture agent, and each new acpmock process restarts its provider-session counter at the same ID. The suite therefore sees two model/effort negotiations and fails even though the intended user session negotiated in the correct order.

## Reproduction

- **Charter:** CH-032 · **Tour:** Feature Tour, automated runtime precondition
- **Environment:** local Linux, race-enabled full `make test-e2e-runtime`, package parallelism enabled

1. Run `make test-e2e-runtime` from the repository root.
2. Observe `TestDaemonE2EProviderReasoningNegotiatesThroughAdvertisedACPOptions/Should_resolve_the_AGENT_reasoning_default_before_the_first_prompt`.
3. Compare the target diagnostics JSONL with the daemon process log.
4. Run the exact reasoning E2E repeatedly in isolation.

**Expected:** The evidence reader can attribute `model → effort → prompt` to the user session regardless of other daemon-owned sessions using the same agent definition.
**Actual:** Under the full lane, the diagnostics file contains `model → effort → prompt → model → effort`; the second process has a different AGH session ID but the same process-local acpmock session ID. The exact isolated suite passes 80/80 subtests across ten runs.

## Evidence

- Full-lane artifact: `/tmp/agh-e2e-testdaemone2eproviderreasoningnegotiatesthroughadvertisedacpoptions-3855578939`.
- Prior reproduction artifact: `/tmp/agh-e2e-testdaemone2eproviderreasoningnegotiatesthroughadvertisedacpoptions-3330044568`.
- Logs show the first user session `sess-1812f934e5b8fb46`, a second ACP process `sess-137d15a24ac5338e`, and a `memory extractor: extract failed` entry; both processes write provider session id `reasoning-claude-max-session-1` to the same registration diagnostics path.
- Focused evidence: `go test -race -tags integration -run '^TestDaemonE2EProviderReasoningNegotiatesThroughAdvertisedACPOptions$' -count=10 ./internal/daemon` passed 80/80 subtests.

## Fix

- **Root cause:** Diagnostics ownership is agent-file scoped, while ACP session IDs emitted by separate acpmock processes are only process-local. The E2E assertion has no stable process/invocation identity with which to distinguish the user session from daemon-owned background sessions. Workspace-local memory configuration also does not disable the global extractor path used here.
- **Fix commit:** pending; two local patch attempts (session-id filter and workspace memory disable) were disproved and removed. The two-touch rule requires a structural design before another change.
- **Regression test:** the canonical reasoning E2E plus a full-lane reproduction that permits concurrent daemon-owned sessions and still attributes protocol records to the intended invocation.

## Verification

- **Retested:** not yet
- **Result:** Open. The focused owner is stable in isolation, but the required full runtime E2E lane remains red.

