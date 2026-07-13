# Task Memory: task_09

## Objective Snapshot

- Turn `_qa.md` into an executable targeted cycle in the living `docs/qa/` tree: seven bridge journeys, persona-driven charters, tracker mappings/resets, risk hunts, TTFM measurement, automation candidates, and an explicit Task 10 handoff.

## Corpus and Authority

- The workflow has no PRD, `_user_stories.md`, ADRs, or handoffs. The current contract is `_qa.md`, `_tasks.md`, `_techspec.md`, `_tests.md`, `task_01.md` through `task_10.md`, `state.yaml`, and the realized task memories.
- Historical peer-review artifacts use the retired 18-task numbering and are provenance only. Current scope is implementation Tasks 01–08, planning Task 09, execution Task 10.
- `docs/qa/` is the only living QA tree. This task updates planning artifacts only; it does not run sessions or assign pass/fail verdicts.

## Conflict Resolutions

- `_qa.md` provisional `NB-047..NB-073` seeds collide with live rows. Tasks 01–06 already created content-addressed canaries `NB-bridge-tool-progress`, `NB-long-bridge-replies`, `NB-provider-progress-rendering`, `NB-bridge-provider-setup`, `NB-web-bridge-setup`, `NB-bridge-edit-reply`, and `NB-bridge-restart-recovery`. Keep those canonical scenarios, map every provisional seed to them in the planning report, and do not mint 27 duplicated invariants.
- Use the exact final-diff reset set from `_qa.md` (`NB-024`, `025`, `026`, `028`, `036`, `037`, `038`, `039`) plus Task 07's realized `NB-029` and `NB-031` resets. Preserve historical bugs, fixes, retests, evidence, and reports.
- J-30 must branch across all eight providers: Slack manifest; WhatsApp/Telegram/Discord guided setup; Teams/GChat/GitHub/Linear generic create/bind/configure. GitHub/Linear finish with issue/comment behavior, not a fabricated chat-message path.
- Telegram `setWebhook` is a Telegram-only decision branch. WhatsApp and Discord require their distinct manual provider-console handoffs.
- Every journey needs a persona abandonment/resume path; valid mode-off, capability boundaries, and product failures are not abandonment by themselves.
- Do not freeze “6000 chars on Discord = 3 chunks.” Suffix/fence overhead is inside the 2,000-code-point wire cap. Assert bounded ordered numbered chunks, lossless content, valid fences, and a runtime-computed count across all six chat providers.
- Restart QA targets the realized checkpoint-only behavior: universal visible fail-open, no text replay/prefix duplication, durable metrics, and reconciliation before new registrations.

## Planned IDs and Ownership

- Journeys: content-addressed files for setup, progress, remediation, long replies, restart, Web setup, and inbound edit/reply.
- Charters: nine content-addressed immutable files; the first three carry the TTFM protocol. One journey and one canonical tour per charter.
- Automation backlog: one content-addressed file each for Web failure/remediation, progress-storm soak, restart fail-open, and TTFM replay; legacy `AB-013..AB-016` references remain metadata only.
- Personas: add three bridge-specific humans and extend Ada for structured bridge setup; security remains a lens, not a persona.

## Validation Contract

- Validate living scenario frontmatter, unique content-addressed IDs, valid statuses, real journey references, and every changed/consolidated bridge scenario covered by at least one Task 09 charter. `state.csv` is generated and absent from the committed tree.
- Validate all seven journeys contain Mermaid, a true end state, and abandonment/resume; all nine charters contain one tour, one time-box, concrete inputs/observables, and real scenario IDs.
- No production code, E2E, global suite, or `make verify` in Task 09. Task 10 owns execution in a fresh isolated lab and mandatory teardown.

## Completion Evidence

- Added bridge persona ownership in `docs/qa/personas.md`; wrote seven content-addressed journeys and nine immutable content-addressed charters.
- Mapped/reset 17 changed or canonical living scenarios while preserving historical bug/fix/retest/evidence fields. All 27 provisional `_qa.md` seeds are reconciled explicitly in the planning report; no duplicate `NB` scenario was minted.
- Added four per-file automation candidates and `docs/qa/reports/2026-07-12-hermes-bridge-plan.md`, including TTFM definitions/baselines, risk ownership, taxonomy coverage, Task 10 evidence rules, isolation, and mandatory teardown.
- Focused validation covers scenario frontmatter and references, seven journeys with Mermaid/true-end/abandonment, nine charters with one tour covering all changed scenarios, all 27 provisional seeds in the consolidation matrix, and `git diff --check`.

## Working Set

- `docs/qa/{personas.md,scenarios/,journeys/J-<slug>.md,charters/CH-<slug>.md,automation-backlog/,reports/2026-07-12-hermes-bridge-plan.md,README.md}`
- `.compozy/tasks/hermes-bridge/{task_09.md,state.yaml,memory/MEMORY.md,memory/task_09.md}`
