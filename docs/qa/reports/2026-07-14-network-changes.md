# QA Run Report — 2026-07-14 — network-changes

- **Scope:** Opt-in Agent Network participation release cycle: Local defaults, bounded Live admission and usage, future-run coordination, administration/extensibility, discoverability, and an adjacent ordinary-session canary.
- **Cadence tier:** targeted + adjacent canary + release-grade real-scenario companion
- **Build:** `ed16bc191579bf52c1d35ecc32ebf91d17a7e961` · **Environment:** fresh isolated manifest per charter; source checkpoint passed `make verify` at 2026-07-14 20:17 BRT
- **Started:** 2026-07-14T20:40:07-03:00 · **Status:** in-progress

## Personas

| Persona | Base | Device / Network / Locale | Sessions |
| --- | --- | --- | --- |
| Ada | Agent operator | desktop / flaky or wifi-fast / en-US | CH-live-bounds-agent-path, CH-network-local-session-canary |
| Bruno | Autonomy and Network operator | desktop / wifi-fast / en-US | CH-network-admin-lifecycle, CH-coordination-future-runs |
| Nia | Solo builder discovering AGH | laptop / wifi-fast / en-US | CH-network-local-default |
| Mateo Rivera | Helix CLI founder/operator | desktop / production-like provider path / en-US | devtool-oss-launch companion inside CH-live-bounds-agent-path |

## Flows in Scope

- `J-run-bounded-live-collaboration` — explicitly Live work admits only eligible, bounded, workspace-scoped wake sources (`../journeys/J-run-bounded-live-collaboration.md`).
- `J-administer-network-live` — an administrator changes availability and Live policy without enrolling executions (`../journeys/J-administer-network-live.md`).
- `J-enable-coordinated-conversations` — an invitation affects future coordinated runs while current task state stays authoritative (`../journeys/J-enable-coordinated-conversations.md`).
- `J-network-local-default` — ordinary owners complete Local work with no hidden Network artifact or usage (`../journeys/J-network-local-default.md`).
- `J-15` — the ordinary session CLI/API lifecycle remains independent of Network (`../journeys/J-15-operate-session-via-cli-api.md`).

## Session Matrix & Results

| # | Charter | Journey / Scenario | Persona | Tour | Status | Issue | Fix commit |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CH-live-bounds-agent-path | J-run-bounded-live-collaboration / NB-run-bounded-live-collaboration; NB-agent-manages-participation; NB-020 | Ada + Mateo | Interrupt Tour | Pending | | |
| 2 | CH-network-admin-lifecycle | J-administer-network-live / NB-network-live-config-lifecycle; NB-network-availability-toggle; NB-001; NB-002; MS-037; ET-025; ET-026; ET-027; ET-028; ET-030; ET-network-participation-hooks; NB-023 | Bruno | Multi-Tab Tour | Pending | | |
| 3 | CH-coordination-future-runs | J-enable-coordinated-conversations / NB-coordination-invitation-future-runs; NB-run-conversation-bounds-usage | Bruno | Back-Button Tour | Pending | | |
| 4 | CH-network-local-default | J-network-local-default / NB-execution-participation-defaults; NB-network-empties-onboarding-settings; NB-participation-controls-serialize; RT-010; RT-031; TA-001; TA-004; TA-007; TA-049 | Nia | Feature Tour | Pending | | |
| 5 | CH-network-local-session-canary | J-15 / RT-023; RT-042 | Ada | Feature Tour | Pending | | |

Status legend: `Pending | Pass | Fixed | Skipped | Blocked (needs human verify) | Blocked (human decision)`

## Preconditions and Execution Register

- **Automated precondition:** complete `make verify` passed on the current runtime source freeze at 20:17 BRT; Task 07 changed only QA planning documents afterward.
- **Playbook rotation:** previous real-scenario run used `consumer-saas-growth` and recorded clean teardown; this run selects validated `devtool-oss-launch`.
- **Open-registry awareness:** the existing autonomous one-kickoff completion stall remains open. A recurrence is appended to that bug as re-found; the observer never sends a second prompt.
- **Runtime E2E lane:** PASS — `rtk make test-e2e-runtime` exited 0 after the canonical Network collaboration E2E expectations were corrected from the removed `running` transport state to the public `active` state. Focused `-race` proof passed 4 tests before the complete rerun.
- **Web E2E lane:** PASS — `rtk make test-e2e-web` exited 0 after fresh codegen check, Web build, and the complete daemon-served Playwright lane.
- **Manifest/teardown register:** Pending — one row will be added per charter with manifest, audit, provider, and `teardown.json` paths.
- **Production-parity deviations:** Pending observation; every deviation qualifies the corresponding verdict.

## Session Debriefs

Pending. One block will be written within five minutes of each charter ending.

## Experiential Lens Pass

The two widest UI journeys are `J-administer-network-live` and `J-network-local-default`. Usability, accessibility, perceived performance, compatibility, error recoverability, and production parity remain Pending until both base walks finish.

## What Was Fixed

### Runtime E2E contract drift: Network status expected a removed transport state

- **Symptom:** two real Network collaboration E2E scenarios failed after receiving `enabled=true, status=active` with two local peers because their assertions still required `status=running`.
- **Root cause:** the E2E suite did not co-ship the Task 06 status hard cut from broker-oriented `running` to Local/Live `disabled|ready|active`; production and the lower canonical status suite already matched the TechSpec.
- **Fix:** current Task 08 local diff; two existing assertions now require the public `active` state. No production path or assertion strength changed.
- **Regression test:** existing `TestDaemonE2ENetworkDirectReplyLifecycleWithMockAgents` and `TestDaemonE2ENetworkWhoisAndCapabilityExchange`; focused `-race` run passed 4 tests and full `make test-e2e-runtime` then exited 0.
- **Retested:** automated owning suite only; the in-persona Live charter remains Pending.

## Paper Cuts

| Persona | Where (journey/step) | Felt | Sharpness | Outcome |
| --- | --- | --- | --- | --- |
| — | — | Not run yet | — | Pending |

## Runtime Errors Observed

- None recorded yet; sessions have not started.

## Human Verifications Needed

- None identified yet.

## Decisions for a Human

- None identified yet.

## Learnings

- Pending execution.

## Final Status

- **Exit gate (full automated suite):** Pending.
- **Issues by user impact:** Blocks-Completion 0 · Data-Loss 0 · Trust-Damage 0 · Friction 0 · Cosmetic 0
- **Coverage:** 0/5 charters walked; 5 Pending.
- **Verdict:** pending — no release-readiness claim exists before the matrix, strict audit, automated lanes, and teardown register are terminal.
