# QA — Living Docs

Canonical QA tree for AGH. Owned by the `qa-report` (planning) + `qa-execution` (sessions) skill pair; `real-scenario-qa` (playbook lab + runtime observation) also lands its findings here. One tree, forever: rounds append, ids never reset, history lives in dated reports.

## Layout

- `state.csv` — scenario tracker (schema + enums: `.agents/skills/qa-report/references/state-schema.md`)
- `bugs/BUG-NNNN.md` — global bug registry (monotonic ids; dedup before filing)
- `journeys/J-NN-<slug>.md` — journey maps + Mermaid flows (flows before matrix)
- `charters/CH-NNN.md` — session charters; per-run debriefs appended
- `reports/<YYYY-MM-DD>-<scope>.md` — one per run, never overwritten
- `evidence/<date>-<scope>/` — checkpoint/failure screenshots + cited run artifacts only (lean). **Skeeper-managed** (`git@github.com:compozy/specs.git`, namespace `agh`, pattern `docs/qa/evidence/**`): gitignored from the main repo, mirrored to the sidecar, restored via `skeeper restore --all`. Reports reference evidence by repo-relative path, which resolves after restore. Uncited bulk dumps are pruned before sync.
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
| GL | Goal (conversational convergence, controls, context, recovery) |

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

- 2026-07-11 — Goal QA cycle **planned** (task 07, Full tier): four composite journeys `J-23..J-26` preserve all eight `_qa.md` seed flows; 40 canonical `GL-001..GL-040` scenarios minted as `untested`; Task 03–05 impact flags `TA-087..TA-105` linked in place to personas/journeys/canonical GL overlaps; charters `CH-037..CH-045`; automation candidates `AB-010..AB-012`; completeness and journey→backbone matrix in `reports/2026-07-11-goal-plan.md`. Planning only — Task 08 executes the fresh isolated cycle.
- 2026-07-10 — `model-selector` QA cycle **planned** (Targeted + e2e-web + e2e-runtime): unified `RuntimeSelector` (web) + truthful catalog/curation/reasoning (backend) + agent surfaces. Six journeys `J-17..J-22` (J-22 is the adjacent settings/display canary); nine charters `CH-028..CH-036`, each with one canonical tour and a 30/60/90 minute box. Correction passes reconciled the five pre-existing orphan rows, separated Bruno's web agent-authoring walk (CH-029) from Ada's structured HTTP/UDS/CLI/`agh__agent_create` walk (CH-036), removed the fabricated native session-create path from CH-035, and minted `ET-053` so the published docs + bundled `skills/agh/` are journey-derived executable guidance. **Spec erratum:** no general agent-update API and no native agent read or session-create tool exist in this MVP; the plan covers only real create/read/session surfaces. Surface→scenario→charter and journey×five-dimension matrices live in `charters/_coverage-matrix.md` and `reports/2026-07-10-model-selector-plan.md`. `state.csv` is sorted and validates at 16 fields/row (365 rows, no duplicate/orphan); all 29 J-17..J-22 rows remain `untested` until execution. Planning only — execution is the later isolated real-user QA pass.
- 2026-07-11 — Frontend/runtime stability QA cycle **planned** (Targeted tier): 30 stale verdicts reset to `untested`, 24 already-untested rows linked/noted, and new recovery rows `NB-047` + `MS-059` minted without erasing historical bug/fix/evidence fields. Added journeys `J-23..J-25` and charters `CH-037..CH-039` for Network continuity, task/automation scale, and durable knowledge recovery; existing Session/Loop canaries are reused. Public expected contracts now describe counted pages, server-owned filters/order, exact totals/facets, fenced transcript SSE, workspace/identity isolation, paged bindings, and Memory dirty recovery. Repaired three pre-existing non-enum Loop `retest_status` cells to `pending` with blocker reasons preserved in notes. Plan and taxonomy audit: `reports/2026-07-11-frontend-stability-plan.md`. Planning only — execution uses a fresh isolated `northstar-pay` lab.
- 2026-07-08 — Session Improvements QA cycle **planned** (task 42, Full tier): `personas.md` adds session personas Théo/Nia/Rafa + extends Ada (Automation Agent = J-15) and Sol (session-thread a11y lens); 5 journeys `journeys/J-11..J-15`; charters `CH-014..CH-021` (8, incl. Sol a11y CH-020); automation backlog `AB-005..AB-008`; journey→`_tests.md` E2E+telemetry map + completeness in `reports/2026-07-08-session-improvements-plan.md`. `state.csv`: net-new `RT-045..RT-047` minted (return hero / open-fast / long-transcript); **placeholder `NEW-6/7/12..22` rows (added by tasks 21,22,25–37) minted to final `RT-048..RT-060`** with session persona + final journey + `overlaps` remap; 14 existing `RT` rows (RT-012/013/015/017/018/019/020/022/023/024/040/041/043/044) linked to journeys in place (already `untested` from their owning tasks — flag-don't-retest, not duplicated). Repaired a pre-existing unquoted-comma defect in the `NEW-6`→`RT-048` `entry_points` field. Planning only — execution is task 43. RT area code already defined below.
- 2026-07-08 — `loops-refac` QA cycle **planned** (task 15, Targeted tier + e2e-web): consolidated tasks 03/05/06/11/12/13/14 QA-impact flags. New `LP-040..LP-050` in `state.csv` (11 rows, all `untested`); reset `LP-003` (software-delivery load_tasks swap + gated sessions) and `LP-029` (reviews-watch gated fix_batch). New journey `J-16` (daemon-internal watch-events, distinct from J-08 extension watch_source); charters `CH-022..CH-027` + `CH-005` gating re-walk; `AB-009` (watch-events browser seed). J-01/J-08 refreshed for the gating/`load_tasks` behavior. Plan + flags↔tracker completeness audit: `reports/2026-07-08-loops-refac-plan.md`. Planning only — execution is task 16.
- 2026-07-09 — `loops-refac` QA cycle **executed** (task 16, Targeted tier + e2e-web): fresh isolated lab `loops-refac-task-16-20260709-20260709-034751-043179`; report `reports/2026-07-09-loops-refac.md`. `LP-040`/`LP-041`/`LP-042`/`LP-043`/`LP-044`/`LP-047`/`LP-048`/`LP-049`/`LP-050` moved to `pass`; `LP-045` moved to `pass` after BUG-0023 fix-loop; `LP-003`/`LP-046` moved to `blocked-verify` pending a configured isolated-lab provider; `LP-029` remains `blocked-verify` after BUG-0022 fix, pending a real git workspace plus CodeRabbit review event/account seed. `BUG-0023` and `BUG-0022` marked fixed.
- 2026-07-09 — `loops-refac` tracker **reconciled** (task 15 re-run, planning only): now that the watch-events substrate has landed, reset `LP-040`/`LP-041`/`LP-042` (task 11 phase A) `skipped→untested` for parity with the already-reset `LP-043/044` (task 12), `LP-047/048` (task 13), `LP-049/050` (task 14) — so task 16's next cycle actually covers phase A. **Repaired** a `state.csv` parse defect: the `LP-049`/`LP-050` `notes` field carried unquoted commas from the task-14 append, splitting those rows into 18/19 columns; re-quoted so every row is 16 fields. Real task-16 verdicts preserved: `LP-003`/`LP-045`/`LP-046` stay `fail` (`BUG-0023` open), `LP-029` stays `blocked-verify`.
- 2026-07-06 — Loops QA cycle **executed** (task 25, Full tier): report `reports/2026-07-06-loops.md`; `BUG-0018`/`BUG-0019` filed + verified; `state.csv` LP rows + `CH-001..013` debriefed. Evidence for `2026-07-06-loops` routed to skeeper (`docs/qa/evidence/**` now sidecar-managed, gitignored from main); 29 uncited catalog dumps pruned (6.8 MB/68 files → 5.1 MB/39 files), keeping all screenshots, logs, and report/bug/tracker-cited JSON.
- 2026-07-06 — Loops QA cycle **planned** (task 24, Full tier): `personas.md` (5 personas); 10 journeys `journeys/J-01..J-10`; scenarios `LP-001..LP-038` in `state.csv` (all `untested`); charters `CH-001..CH-013`; automation backlog `AB-001..AB-004`; journey→`_tests.md` E2E matrix + completeness in `reports/2026-07-06-loops-plan.md`. Planning only — execution is task 25. LP area code already defined below.
- 2026-07-05 — Tree bootstrapped; feature-stories tracker migrated (253 rows, 17 bugs re-minted); seeds adopted; legacy per-round trees retired.
