# Autonomous AGH Web and Site Impact Research

You are researching gaps in the current autonomy TechSpec around the product UI (`web/`) and public/docs site (`packages/site/`).

This is a research/review task. Do not modify any file except the requested output report.

## Goal

Determine which parts of `web/` and `packages/site/` must be changed, improved, or added so the autonomy work can be implemented and documented coherently. The current TechSpec focuses heavily on backend/CLI/daemon work; check whether it under-plans runtime UI, docs, generated contracts, tests, navigation, and user-facing explanations.

## Required Inputs To Read First

- `.compozy/tasks/autonomous/_techspec.md`
- `.compozy/tasks/autonomous/adrs/adr-001.md`
- `.compozy/tasks/autonomous/adrs/adr-002.md`
- `.compozy/tasks/autonomous/adrs/adr-003.md`
- `.compozy/tasks/autonomous/adrs/adr-004.md`
- `.compozy/tasks/autonomous/adrs/adr-005.md`
- `.compozy/tasks/autonomous/adrs/adr-006.md`
- `.compozy/tasks/autonomous/adrs/adr-007.md`
- `.compozy/tasks/autonomous/adrs/adr-008.md`
- `.compozy/tasks/autonomous/adrs/adr-009.md`
- `.compozy/tasks/autonomous/adrs/adr-010.md`
- `.compozy/tasks/autonomous/reviews/opus-techspec-review-round2.md`
- `.compozy/tasks/autonomous/reviews/gpt54mini-agh-code-analysis.md`
- `web/AGENTS.md`
- `web/CLAUDE.md`
- `web/package.json`
- `packages/site/package.json`

## Areas To Inspect

For `web/`, inspect only the files needed to understand current architecture and affected surfaces. Prioritize:

- `web/src/systems/`
- `web/src/routes/`
- `web/src/lib/api-client.ts`
- `web/src/lib/api-contract.ts`
- `web/src/generated/agh-openapi.d.ts`
- `web/src/components/app-sidebar.tsx`
- `web/e2e/tasks.spec.ts`
- `web/e2e/network.spec.ts`
- `web/e2e/combined-flows.spec.ts`
- `web/src/storybook/`

For `packages/site/`, inspect only the files needed to understand current docs architecture and affected content/navigation. Prioritize:

- `packages/site/content/runtime/`
- `packages/site/content/protocol/`
- `packages/site/lib/runtime-navigation.ts`
- `packages/site/lib/site-config.ts`
- `packages/site/components/docs/`
- `packages/site/components/landing/`
- `packages/site/app/runtime/`
- `packages/site/app/protocol/`

## Context To Preserve

The current backend architecture decisions should remain stable unless you find a concrete UI/docs contradiction:

- Autonomy is additive; manual task/session flows remain first-class.
- Task creation does not trigger coordinator startup or create claimable work.
- Run enqueue is the coordinator trigger.
- `ClaimNextRun` is the sole authoritative next-work primitive.
- Scheduler is sweep/notify/recovery only, not a direct claimant.
- Task-run ownership lives in `task_runs`; no durable scheduler queue.
- Hooks use `coordinator.*`, `spawn.*`, and `task.run.*`; scheduler hooks are not in MVP.
- MVP implementation scope is TechSpec steps 1-10.
- Web UI is currently listed as "modified later" / post-MVP visibility, but the user suspects web/docs impact is under-planned. Validate that assumption.

## Research Questions

Answer concretely:

1. What exact `web/` systems/routes/components/API contracts/tests will likely need changes for the autonomy MVP?
2. Should any web work move into MVP steps 1-10, or should it remain post-MVP? Explain why.
3. What read models or API contract DTOs must exist before `web/` can show coordinator, leases, spawned children, claim state, scheduler alerts, or manual/autonomous differences?
4. Which web flows must remain manual-first and how should UI avoid implying that task creation starts orchestration?
5. What exact `packages/site/` docs/content/navigation pages should be added or updated?
6. Should docs/site work be part of MVP task decomposition or a post-MVP/docs-release step?
7. Are there generated OpenAPI or package-level contract implications for `web/src/generated/agh-openapi.d.ts` or `packages/site` docs examples?
8. Is a new ADR needed for product surface/documentation strategy, or can the TechSpec absorb it?
9. What changes should be made to `_techspec.md` and existing ADRs before `$cy-create-tasks`?
10. What is overengineering and should stay out of MVP?

## Output

Write the full report to:

`.compozy/tasks/autonomous/reviews/opus-web-site-impact-research.md`

Use this format:

```md
# Opus Research: Web and Site Impact for Autonomous AGH

## Verdict

Brief verdict: missing / partially covered / sufficiently covered.

## Executive Recommendation

What to add to the TechSpec now, what to leave post-MVP, and whether a new ADR is needed.

## Web Impact

Concrete files/areas, required backend contracts/read models, MVP vs post-MVP split, and test implications.

## Site/Docs Impact

Concrete docs/navigation/content areas, examples that must change, MVP vs post-MVP split, and test implications.

## Required TechSpec Edits

Specific edits grouped by section.

## ADR Recommendation

Whether to add a new ADR. If yes, propose title, decision, alternatives, consequences.

## Overengineering To Avoid

Specific UI/docs features that should not be included in MVP.

## References

List local files inspected.
```

Be strict. The goal is to prevent `$cy-create-tasks` from missing important UI/docs work, while still avoiding overengineering.
