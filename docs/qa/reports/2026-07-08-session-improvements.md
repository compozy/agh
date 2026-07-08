# QA Run Report - 2026-07-08 - session-improvements

- **Scope:** Session Experience Overhaul execution gate: blank-on-return, session open latency,
  live streaming/reconnect, transcript UI language, CLI/API/UDS session parity, and truthful UI
  invariants planned by task 42.
- **Cadence tier:** Full.
- **Build:** 4ea584c41 plus local task-42/session-improvements worktree changes. **Environment:** pending deterministic QA bootstrap.
- **Started:** 2026-07-08T15:20:58-03:00. **Status:** in-progress.

## Personas

| Persona | Base | Device / Network / Locale | Sessions |
|---|---|---|---|
| Théo | Power User | desktop / wifi-fast / en-US | CH-014, CH-016 |
| Ada | Power User (native-tool) | desktop / wifi-fast / en-US | CH-018 |
| Rafa | Casual User | desktop / wifi-fast / en-US | CH-017, CH-021 |
| Nia | New User | laptop / wifi-slow or wifi-fast / en-US | CH-015, CH-019 |
| Sol | Accessibility-Reliant | desktop / wifi-fast / en-US | CH-020 |

## Flows in Scope

- `J-11 Return to a running session` - blank-on-return hero, return paths, snapshot reconnect,
  lifecycle truth, and workspace redirect notice (`../journeys/J-11-return-to-running-session.md`).
- `J-12 Open a session fast` - cold open, deep link, warm remount, and long-history paging
  (`../journeys/J-12-open-session-fast.md`).
- `J-13 Follow a live run` - live streaming, scroll anchoring, composer queue, stop, and clear
  convergence (`../journeys/J-13-follow-a-live-run.md`).
- `J-14 Read a finished transcript` - grouped tool rows, inline I/O, folds, usage truth, clear,
  and gap-free paging (`../journeys/J-14-read-a-finished-transcript.md`).
- `J-15 Operate a session via CLI/API` - CLI, HTTP, and UDS parity with raw/transcript streams and
  bounded reads (`../journeys/J-15-operate-session-via-cli-api.md`).

## Session Matrix & Results

| # | Charter | Journey / Scenario | Persona | Tour | Status | Issue | Fix commit |
|---|---|---|---|---|---|---|---|
| 1 | CH-014 | J-11 / RT-045 RT-043 RT-023 RT-015 RT-024 RT-041 | Théo | Interrupt Tour | Pending | | |
| 2 | CH-018 | J-15 / RT-050 RT-051 RT-023 RT-022 RT-012 RT-042 | Ada | Feature / strategy-based | Pending | | |
| 3 | CH-016 | J-13 / RT-054 RT-058 RT-059 RT-018 RT-019 RT-020 RT-013 | Théo | Multi-Tab Tour | Pending | | |
| 4 | CH-017 | J-14 / RT-048 RT-049 RT-053 RT-055 RT-056 RT-057 RT-060 | Rafa | Feature Tour | Pending | | |
| 5 | CH-021 | J-14 / RT-047 RT-052 RT-022 RT-017 | Rafa | Garbage Tour | Pending | | |
| 6 | CH-015 | J-12 / RT-046 RT-047 RT-040 RT-012 RT-044 | Nia | Network Tour | Pending | | |
| 7 | CH-020 | J-13 / RT-054 RT-058 RT-057 RT-048 RT-059 | Sol | Back-Button Tour | Pending | | |
| 8 | CH-019 | J-11 / RT-010 RT-015 RT-021 RT-041 | Nia | Back-Button Tour | Pending | | |

Status legend: `Pending | Pass | Fixed | Skipped | Blocked (needs human verify) | Blocked (human decision)`.

## Bootstrap

Pending.

```text
[QA_BOOTSTRAP]
manifest_path=pending
lab_root=pending
runtime_home=pending
base_url=pending
verification_report=pending
health_status=pending
[/QA_BOOTSTRAP]
```

## Evidence Index

- Planned lean evidence root: `docs/qa/evidence/2026-07-08-session-improvements/`.
- Bulky lab artifacts will remain under the bootstrap-managed lab root and be indexed here by
  path.

## Automated Lanes

| Command | Status | Evidence |
|---|---|---|
| `make test-e2e-runtime` | Pending | |
| `make test-e2e-web` | Pending | |
| `make verify` | Pending | |

## What Was Fixed

None yet.

## Paper Cuts

| Persona | Where (journey/step) | Felt | Sharpness | Outcome |
|---|---|---|---|---|

## Runtime Errors Observed

None yet.

## Human Verifications Needed

None yet.

## Decisions for a Human

None yet.

## Learnings

- Task 43 starts from task 42's Full-cycle plan; every session-improvements `RT-*` row in scope is
  either `untested` or a historical canary row deliberately re-walked by CH-019/CH-018.

## Final Status

Pending until all matrix rows are terminal and the exit gate has run.
