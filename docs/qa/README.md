# QA — Living Docs

Canonical QA tree for AGH. Owned by the `qa-report` (planning) + `qa-execution` (sessions) skill pair; `real-scenario-qa` (playbook lab + runtime observation) also lands its findings here. One tree, forever: rounds append, ids never reset, history lives in dated reports.

## Layout

- `state.csv` — scenario tracker (schema + enums: `.agents/skills/qa-report/references/state-schema.md`)
- `bugs/BUG-NNNN.md` — global bug registry (monotonic ids; dedup before filing)
- `journeys/J-NN-<slug>.md` — journey maps + Mermaid flows (flows before matrix)
- `charters/CH-NNN.md` — session charters; per-run debriefs appended
- `reports/<YYYY-MM-DD>-<scope>.md` — one per run, never overwritten
- `evidence/<date>-<scope>/` — checkpoint/failure screenshots only (lean; bulk stays lab-side, indexed by path)
- `automation-backlog.md` — AB-NNN entries; automation intent lives here only
- `templates/` — project copies of bug/charter/report templates

## Area codes (scenario id prefixes)

| Code | Area |
|---|---|
| RT | Runtime & sessions (daemon, session lifecycle, providers) |
| TA | Tasks & automation (task runs, leases, scheduling, loops) |
| ET | Extensibility & tools (extensions, hooks, skills, registries, bundles) |
| NB | Network & bridges (channels, threads, bridge SDKs, delivery) |
| MS | Memory & settings (memory, config lifecycle, sandbox/env) |
| LP | Loops (workflow runs, catalog, configure/fork, editor) |

New areas: define the code here first, then mint ids.

## Entry points

- CLI: `agh` (structured output; UDS + HTTP parity)
- Web: `make web-dev` (export `AGH_WEB_API_PROXY_TARGET` from the bootstrap manifest for isolated labs)
- Release/scenario labs: `agh-qa-bootstrap` skill (isolated `AGH_HOME`/ports/provider homes; see CLAUDE.md Workflow Rules)

## Adopted from (migration inventory, 2026-07-05)

- `state.csv` seeded from the feature-stories tracker (253 stories, cycle 2026-06). The frozen origin CSV and the 5 subsystem analyses now live in `_seeds/feature-stories/` (the old `.compozy/tasks/feature-stories/` dir was retired 2026-07-05). Original prose statuses preserved per row in `notes` (`migrated-status:`). `journey` column intentionally empty — journeys get mapped by the first planning cycle (flows before matrix); do not backfill from old TC-style checks.
- `bugs/BUG-0001..0017` re-minted from the feature-stories registry (old per-round `BUG-001..017`); impact tiers unclassified — classify on next touch.
- **Evidence caveat:** the origin lab (`~/dev/qa-labs/agh-feature-stories-20260621-...-lab/`) was accidentally deleted during the 2026-07-05 cleanup, so the lab-relative `qa/evidence/...` and `qa/issues/...` paths in `state.csv` are dangling. Treat migrated `pass` verdicts as historical claims backed by the surviving `file:line` code citations; the next Full cycle re-validates with fresh evidence.
- `_seeds/final-qa/` — pre-release master plan (283 scenarios across 15 modules) + openclaw/hermes QA pattern libraries, from the retired `.compozy/tasks/final-qa/`. Mine into journeys/charters as cycles touch each module, then prune.
- `_seeds/qa-e2e-playbook.md` — the 2026-04 E2E playbook (evidence standard, execution profiles, suite matrix, automation backlog seed), formerly `docs/ideas/qa-e2e/`.
- Historical per-round QA trees (29 under `.compozy/tasks/_archived/*/qa/`), `final-qa/_runs/` evidence, and 28 stale external labs were deleted on 2026-07-05 (no live references; ids collided across rounds and were never migrated).

## Changelog

- 2026-07-06 — Loops QA cycle **planned** (task 24, Full tier): `personas.md` (5 personas); 10 journeys `journeys/J-01..J-10`; scenarios `LP-001..LP-038` in `state.csv` (all `untested`); charters `CH-001..CH-013`; automation backlog `AB-001..AB-004`; journey→`_tests.md` E2E matrix + completeness in `reports/2026-07-06-loops-plan.md`. Planning only — execution is task 25. LP area code already defined below.
- 2026-07-05 — Tree bootstrapped; feature-stories tracker migrated (253 rows, 17 bugs re-minted); seeds adopted; legacy per-round trees retired.
