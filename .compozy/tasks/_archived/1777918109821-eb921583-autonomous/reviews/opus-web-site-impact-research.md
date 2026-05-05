# Opus Research: Web and Site Impact for Autonomous AGH

Reviewer: Opus (architect-advisor lens)
Date: 2026-04-26
Inputs: `_techspec.md`, `adrs/adr-001..010.md`, `reviews/opus-techspec-review-round2.md`, `reviews/gpt54mini-agh-code-analysis.md`, `web/AGENTS.md`, `web/CLAUDE.md`, `web/package.json`, `web/src/lib/api-client.ts`, `web/src/lib/api-contract.ts`, `web/src/generated/agh-openapi.d.ts`, `web/src/components/app-sidebar.tsx`, `web/src/systems/{tasks,session,network,agent}/`, `web/src/routes/_app/{tasks*,session.$id,network}.tsx`, `web/e2e/{tasks,network,combined-flows}.spec.ts`, `web/src/storybook/`, `packages/site/package.json`, `packages/site/lib/{runtime-navigation,site-config}.ts`, `packages/site/components/{docs,landing}/`, `packages/site/app/{runtime,protocol,(home)}/`, `packages/site/content/{runtime,protocol}/`.

## Verdict

**Partially covered.** The TechSpec correctly flags that web visibility is "modified later" (step 15) and post-MVP, and that operator task/session surfaces remain first-class. That posture is right for the runtime UI. But the spec under-plans three concrete impact surfaces that `cy-create-tasks` will otherwise miss:

1. **Generated OpenAPI / contract DTO surface.** Every new autonomy endpoint added in MVP step 7 (and the data-model fields added in step 6) will land in `internal/api/contract`, regenerate `openapi/agh.json`, and propagate into `web/src/generated/agh-openapi.d.ts`. Even with the UI deferred, `web-typecheck` and `web-test` will break the moment task-run DTOs gain `claim_token`, `lease_until`, `heartbeat_at`, capability side-table projections, or session lineage fields, because existing components (`task-run-detail-panels.tsx`, `tasks-dashboard-cards.tsx`, `tasks-detail-header.tsx`, e2e fixtures) read those exact shapes today. This is an MVP integration cost, not post-MVP.

2. **Operator task UI honesty.** The current Tasks UI (`web/src/systems/tasks/`, `e2e/tasks.spec.ts`) already has a publish/approve/enqueue/claim path that the user actually walks through. ADR-005, ADR-010, and the Coordinator Trigger section make a load-bearing claim that "task creation does not start orchestration; the run-enqueue boundary does." The current UI labels and behaviors (e.g., `task-card-publish`, `tasks-detail-enqueue`, "Create & enqueue" button) are *ambiguous* about that boundary. Without one MVP web copy/labeling pass, the UI will silently contradict the trigger semantics the daemon team is committing to. This is a small change but it must be in MVP because it is the operator's mental model of the new contract.

3. **Documentation site narrative.** The runtime docs (`packages/site/content/runtime/core/{sessions,agents,network,hooks,automation}/`) and the CLI reference (`packages/site/content/runtime/cli-reference/{task,session}/`) describe the *current* substrate. There is **zero** mention of `coordinator`, `autonomy`, `claim_token`, `lease_until`, `ClaimNextRun`, `spawn_depth`, or `agh me/spawn/task next/release`. Once the kernel ships, every CLI page under `cli-reference/task/run/` and the entire `core/sessions/` and `core/agents/spawning.mdx` page describe stale behavior. Site docs are not wired into MVP build-order step 10, but the *demo milestone* after step 7 is unsellable without minimum docs and CLI ref pages for the new verbs. This is best treated as a co-shipping requirement, not a separate post-MVP TechSpec.

The rest — full coordinator dashboards, lease graphs, scheduler visualizations, autonomy alerts UI, marketing pages — should stay post-MVP per ADR-001 and the existing Impact Analysis row for "Web UI: modified later".

## Executive Recommendation

**Add to the TechSpec now (MVP scope creep that is unavoidable):**

- A new **"Generated Contract Surface"** subsection (under Impact Analysis or Integration Points) that names: `openapi/agh.json` regeneration, `web/src/generated/agh-openapi.d.ts` regeneration through `bun run codegen`, `web/src/generated/compozy-openapi.d.ts` parity, and the requirement that `web-typecheck` and `web-test` pass after every MVP step that changes a contract DTO. Add a row to the Impact Analysis table for `web/src/generated/*` + `web/src/systems/tasks` + `web/src/systems/session` typecheck blast radius.
- A new **"Operator UI Honesty"** task scope (~1 task) for MVP step 10 (or step 7's demo milestone): minimal label/copy/disabled-state work in `tasks-detail-header.tsx`, `tasks-detail-preview-panel.tsx`, `task-editor-surface.tsx`, and `task-card.tsx` so the user-visible publish/approve/enqueue/start surface accurately reflects ADR-005's run-enqueue trigger boundary and ADR-010's manual-first contract.
- A new **"Documentation Co-Ship"** subsection naming the *minimum* MDX pages that must land alongside MVP step 10: a new `core/autonomy/` folder (or `core/coordinator/` + `core/claim-and-lease/` pair) with 3-5 pages, and a `cli-reference/me/`, `cli-reference/spawn/`, plus updates to `cli-reference/task/run/` for `next`, `heartbeat`, `release` verbs. Without it the demo milestone (after step 7) cannot be documented.

**Leave post-MVP (per ADR-001 / Build Order step 15):**

- All net-new UI for coordinator dashboards, lease/heartbeat visualization, idle-agent registry views, scheduler alert panels, spawn lineage trees, eval-replay viewers, and autonomy telemetry charts.
- Marketing site updates (landing page, `app/(home)/page.tsx`) for the autonomy story.
- Any `web/src/systems/coordinator/` or `web/src/systems/autonomy/` new system module.

**ADR posture:** A new ADR is *recommended* but not blocking. ADR-011: "Generated Contracts and Documentation Co-Ship As Part Of Each Autonomy Step" would lock the boundary. Alternatively, the TechSpec absorbs it under existing sections without an ADR; that is acceptable because it does not change architecture, only release discipline. My recommendation: add the ADR, because cy-create-tasks will otherwise omit the cross-package shipping cost from MVP task files.

## Web Impact

### Files that will require code changes for MVP

**Forced by contract regeneration (MVP, must compile):**

- `web/src/generated/agh-openapi.d.ts` — regenerated by `bun run codegen` after every contract change in step 1, 6, 7, 9, 10. New shapes for ClaimNextRun, ClaimedRun, SpawnOpts, CoordinatorConfig, lease fields on `task_runs` projections, capability side-table projections, lineage fields on session payloads.
- `web/src/lib/api-contract.ts` — already a thin generic over `aghOperations`; no manual change but consumers of `OperationResponse<"getTaskRun", 200>` will see new fields.
- `web/src/systems/tasks/types.ts` — derives all task/run/dashboard/inbox types from operations. New fields will surface here automatically. If the daemon team replaces existing claim shapes (B3 in opus-techspec-review-round2: drop duplicative columns), shapes change and downstream components must adapt.
- `web/src/systems/tasks/components/task-run-detail-panels.tsx` and `task-run-detail-header.tsx` — render `claimed_by`, `attempt`, `idempotency_key`. Will need at minimum: `lease_until` display, `claim_token` (hash only — never raw — see security note below), `heartbeat_at`. Likely a small TypeScript fix even if the UI stays minimal.
- `web/src/systems/tasks/components/tasks-dashboard-cards.tsx` and `tasks-dashboard-active-runs.tsx` — already read `claimed`, `claim_latency_ms`. New fields (idle agents available, lease expiry rate, scheduler wake counts) will be optional and ignorable, but if shapes are renamed under B3 the existing cards break.
- `web/src/systems/session/types.ts` and dependent components — session payload will gain lineage fields (`parent_session_id`, `root_session_id`, `spawn_depth`, `spawn_role`, `ttl_expires_at`). If these are added as optional, no UI work is forced; if any current required field is renamed, components like `chat-header.tsx` and `session-inspector.tsx` need a touch.
- `web/src/components/app-sidebar.tsx` — currently lists agents and sessions. If session payload gets a "spawned/coordinator/worker" type marker, the sidebar will eventually want to surface it (post-MVP), but for the MVP the *only* obligation is to not crash on the new shape.

**Forced by operator UI honesty (MVP, ~1 small task):**

- `web/src/systems/tasks/components/tasks-detail-header.tsx:120-135` — the "Publish" and "Enqueue" buttons exist as separate verbs. The button labels and disabled-state logic must reflect the new run-enqueue-as-coordinator-trigger semantics. Add a small visual cue ("This will start the coordinator" or similar copy) on `enqueue` for coordinated workspaces. Per ADR-010, do not imply that creating a draft starts orchestration.
- `web/src/systems/tasks/components/task-editor-surface.tsx:101-103, 488-535` — the "Create & enqueue" button label and the surrounding copy ("This template enqueues its first run as soon as the task is created") must stay accurate after the trigger change. Verify the UI does not imply orchestration starts at creation.
- `web/src/systems/tasks/components/tasks-detail-preview-panel.tsx:185-200` — same publish/enqueue duo.
- `web/src/systems/tasks/components/task-card.tsx:107` — `task-card-publish-${task.id}` button. Verify behavior after the publish→enqueue→coordinator trigger change.
- `web/e2e/tasks.spec.ts:97-110` — the e2e test asserts `POST /api/tasks/{id}/publish` returns `task.status: "ready"`. After ADR-005, publish-without-approval-policy auto-enqueues a run (per N1 in opus-techspec-review-round2). The assertion may need to assert both `status: "ready"` AND a freshly-enqueued run, or split into two scenarios.

**Forced by removed/renamed CLI verbs (low risk if naming stable):**

- No direct web impact unless backend renames `claim`/`complete`/`fail`/`cancel` operations. Keep the OpenAPI operation IDs stable per ADR-002.

### Files that should remain unchanged for MVP

- All `web/src/systems/{network,memory,knowledge,bridges,automation,workspace,settings,skill,agent}/` — autonomy MVP does not change these surfaces. Memory provenance (ADR-008) lands as new payload fields; the memory UI can ignore them for MVP.
- `web/src/routes/__root.tsx`, `web/src/routes/_app.tsx`, route tree — no new top-level routes for coordinator/scheduler/lease views in MVP.
- Storybook stories under `web/src/systems/tasks/components/stories/` — fixtures may need a token added so stories still render typecheck-clean, but no new stories required for MVP.

### Read models and DTOs that must exist before any web work

These are backend obligations *before* the web team can show coordinator/lease/spawn surfaces. Add to TechSpec:

- `ClaimedRun` DTO with `task_id`, `run_id`, `claim_token` (or `claim_token_hash` for read endpoints — never expose raw token), `lease_until`, `heartbeat_at`. Already in TechSpec as `ClaimedRun`.
- A `TaskRun` projection that exposes `claim_token_hash` (NOT raw token), `lease_until`, `heartbeat_at`, and effective `release_reason` on terminal runs. Confirm this projection is added to `getTaskRun` and `listTaskRuns` operation responses.
- A `SessionLineage` block on session detail responses with `parent_session_id`, `root_session_id`, `spawn_depth`, `spawn_role`, `ttl_expires_at`, `auto_stop_on_parent`. Specify whether these are required or optional fields on the existing session contract.
- A coordinator-aware session marker (e.g., `session.kind: "coordinator" | "worker" | "user" | "system" | "dream"` extending the current `user/dream/system` taxonomy in `core/sessions/lifecycle.mdx:246-256`).
- Operation IDs: `claimNextTaskRun`, `heartbeatTaskRun`, `releaseTaskRun`, `spawnSession`, `getCoordinatorContext` (optional), `getAgentMe`, `getAgentMeContext`. Lock IDs in step 7 so generated TypeScript types are stable.
- Read-only endpoints `GET /api/coordinator/{workspace_id}` (optional, for MVP demo if a UI peek is needed) and `GET /api/sessions/{id}/lineage` are *not* required for MVP since the UI can render lineage from existing `getSession` if those fields are added there.

### Manual flows the UI must keep first-class (per ADR-010)

The Tasks UI must continue to support — and clearly label — the manual-first paths:

- Creating a task without execution metadata (no orchestration prompt, no execution mode picker on the create form).
- Publishing a draft as a separate, explicit action.
- Approving an approval-gated task as a separate, explicit action.
- Enqueueing a run as the coordinator trigger; UI copy should say so.
- Starting a session manually from the sidebar (already supported via "New session for {agent}") and prompting it directly without any task involvement.

UI must avoid implying that task creation starts orchestration. The current `task-editor-surface.tsx` "Create & enqueue" template is the most likely surface to mislead operators after ADR-005 lands. A copy review pass is sufficient — no new components required.

### Test implications

- `web-typecheck` will block the daemon PR when contract DTOs change. This is by design.
- `web-test` (Vitest) will break wherever existing component tests rely on shapes that get reorganized by B3 (drop duplicative columns). Specifically `task-run-detail-panels.test.tsx` and `task-run-detail-header.test.tsx` use a hand-rolled fixture `claimed_by: { kind: "agent_session", ref: "Coder" }` that already matches the canonical `ClaimedBy` ActorIdentity shape; they should keep working.
- `web/e2e/tasks.spec.ts` and `web/e2e/network.spec.ts` are daemon-served Playwright tests. The tasks spec exercises publish→approve→active-run→session-drilldown. After the run-enqueue-as-coordinator-trigger change, the test must add an explicit enqueue step (or rely on auto-enqueue-on-publish per N1) and assert no coordinator session spawns when only publish happens (manual-first guarantee).
- A new e2e scenario should be added in MVP for ADR-010's bookends: (a) user creates → publishes → coordinator spawns → worker session claims → completes; (b) user starts a manual session → prompts it directly → no coordinator spawns. These exercise the integration tests already required by ADR-010 from the operator surface.
- Storybook MSW contract tests (`web-storybook-msw-contract.test.ts`) lock fixture shapes; they must be updated after contract changes.

### Security notes for web exposure

- **Never expose raw `claim_token` over HTTP/SSE.** The TechSpec uses `claim_token_hash` in structured logs but does not say so for read endpoints. Lock this: HTTP responses return `claim_token_hash`; only the issuing UDS reply returns the raw `claim_token` to the claimant. Web UI must never receive the raw token.
- Spawn permission policy fields (`permission_policy_json`) may contain sensitive atom data; web UI should render a redacted summary, not raw JSON.

## Site/Docs Impact

### Required co-ship pages (MVP, alongside step 10)

These are the minimum docs that must ship with the autonomy MVP demo milestone. Without them, the new CLI verbs and concepts are undocumented and the operator cannot self-serve.

**New `packages/site/content/runtime/core/autonomy/` folder (or split into 2-3 sibling folders):**

- `index.mdx` — what autonomy is, the four-layer model (Situation Surface, Agent Kernel, Autonomy Kernel, Memory/Self-Correction), the manual-first peer model from ADR-010.
- `coordinator.mdx` — what the coordinator-agent is, when it spawns (run-enqueue boundary, idempotent per workspace), how to configure provider/model under `[autonomy.coordinator]` and workspace overrides, how to manually start/stop one. Reference ADR-005.
- `claim-and-lease.mdx` — `task_runs` claim/lease model, `ClaimNextRun` semantics, lease expiry/heartbeat/recovery rules, per-session lease cap (default 1 — see N17), structured release reasons. Reference ADR-003.
- `safe-spawn.mdx` — lineage, TTL, max depth/children, permission narrowing atoms, parent-stop reaper behavior. Reference ADR-006.
- `manual-vs-autonomous.mdx` — explicit ADR-010 contract for users: when does the coordinator wake, what manual paths still exist, how to bypass autonomy for direct prompting.

**Update `packages/site/content/runtime/core/agents/spawning.mdx`:**

- Add a "Spawned worker sessions" section explaining lineage, TTL, and that spawned sessions are normal managed sessions with narrowed permissions.
- Add `AGH_AGENT` and any new spawn-time env vars to the env table at line 79-87.

**Update `packages/site/content/runtime/core/sessions/lifecycle.mdx`:**

- Add a row to the Session types table at line 246-253 for `coordinator` and `worker` (or extend with `spawn_role` documentation).
- Add a "Spawned and reaped" subsection covering parent-stop, TTL expiry, and lease release with structured reasons.

**Update `packages/site/content/runtime/core/network/index.mdx` and `task-ingress.mdx`:**

- Add a sentence pointing at the new autonomy section so operators understand that channels remain coordination primitives, not replacements for `task_runs` claim ownership.

**Update `packages/site/content/runtime/core/hooks/event-catalog.mdx`:**

- Add three new sections: `Coordinator Events`, `Spawn Events`, `Task Run Events`. Mirror the families from ADR-009 / TechSpec hook taxonomy. This is mechanically generated from `agh hooks events --family coordinator|spawn|task.run`, so it's a one-pass update.
- Note: per N5/N12 in opus-techspec-review-round2, do NOT document `scheduler.*` as hooks; they stay in metrics/logs.

**New CLI reference pages under `packages/site/content/runtime/cli-reference/`:**

- `me/` folder with `index.mdx`, `context.mdx` for `agh me` and `agh me context`.
- `spawn/` folder with `index.mdx` for `agh spawn`.
- Update `task/run/` to add `next.mdx`, `heartbeat.mdx`, `release.mdx`. The current `claim.mdx` page exists for explicit-id claims and should remain (renamed or annotated to clarify the difference vs `next`).
- Update `task/meta.json` and `task/run/meta.json` to include the new pages.
- Update `cli-reference/meta.json` to add `me` and `spawn` to the page list.

**Update `packages/site/content/runtime/core/configuration/config-toml.mdx`:**

- Add a `[autonomy.coordinator]` block with all keys: `enabled`, `agent_name`, `provider`, `model`, `max_children`, `default_ttl`, `tool_allowlist`. Note workspace override path.

**Optional but recommended:**

- A `core/autonomy/observability.mdx` page with the new metrics catalog from TechSpec (`scheduler.wake.count`, `task.run.claim.success`, `coordinator.spawned`, `spawn.created`, `spawn.rejected`, `spawn.reaped`, etc.) so operators have one place to look up what each metric means.

### Pages that must be left alone for MVP (post-MVP)

- `app/(home)/page.tsx` and all `components/landing/*.tsx` — the marketing landing page. The autonomy story is too early to lead with on the homepage. Keep the network-first positioning per the saved project memory.
- `content/protocol/*.mdx` — the protocol docs are about the wire-level network protocol and are independent from the autonomy work. No changes for MVP.
- `lib/runtime-navigation.ts` and `lib/site-config.ts` — no nav restructure required if new pages slot under existing `core/` and `cli-reference/` parents. The runtime navigation builder rewrites the Core Concepts folder; adding `autonomy` as a sibling folder under `runtime/core/` is enough, no helper changes.
- All Mermaid storyboard images under `public/images/runtime/` — no new posters required for MVP. Add them in a post-MVP polish pass.

### Test implications

- `packages/site/lib/source.test.ts` and any docs route tests — adding new MDX pages will require fumadocs to regenerate `.source` (`bun run source:generate`). The pretest hook does this. Verify new pages render in the test pass.
- `packages/site/components/docs/*.test.tsx` — no changes unless new mdx blocks are added.
- `packages/site/global.test.ts` — verify global metadata still resolves canonical URLs for new pages.

### Generated openapi parity for site examples

The CLI reference pages embed example commands. They are hand-written and not generated from OpenAPI, so DTO shape changes do not auto-break them. But the *content* must be updated when new task-run fields appear in JSON output examples. Audit the existing `task/run/{claim,start,enqueue}.mdx` example JSON blobs after step 6 lands.

## Required TechSpec Edits

### `_techspec.md` § Impact Analysis (table)

Add three rows:

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|----------------------|-----------------|
| `openapi/agh.json` + `web/src/generated/agh-openapi.d.ts` + `web/src/systems/tasks/types.ts` + `web/src/systems/session/types.ts` | modified (MVP) | Every contract DTO change in steps 1, 6, 7, 9, 10 propagates into web typecheck. Web tests and Storybook MSW fixtures break on rename. | Run `bun run codegen` after each contract change; gate `web-typecheck` and `web-test` in the same PR; never expose raw `claim_token` over HTTP. |
| Operator Tasks UI (`web/src/systems/tasks/components/{task-card,task-editor-surface,tasks-detail-header,tasks-detail-preview-panel}.tsx` + `web/e2e/tasks.spec.ts`) | modified (MVP, copy/labels only) | After ADR-005 the publish/approve/enqueue distinction becomes load-bearing. Existing UI is ambiguous about which action triggers the coordinator. | One MVP copy/labeling pass to make the publish→enqueue→coordinator boundary explicit; one new e2e scenario for ADR-010 manual-first integration tests. No new components. |
| `packages/site/content/runtime/{core/autonomy,core/agents/spawning,core/sessions/lifecycle,core/hooks/event-catalog,core/configuration/config-toml,cli-reference/{me,spawn,task/run}}` | modified (MVP, co-ship with step 10) | New CLI verbs and runtime concepts will be undocumented at the demo milestone without minimal MDX pages. | Add a `core/autonomy/` folder with 3-5 pages and the named CLI ref pages; update meta.json indexes. Marketing site stays unchanged. |

### `_techspec.md` § Integration Points

Add a new subsection **"Generated Contract Surface"** that names:

> `internal/api/contract` is the source of truth for transport-agnostic DTOs. `openapi/agh.json` is regenerated from it. `web/src/generated/agh-openapi.d.ts` is regenerated by `bun run codegen` and consumed by `web/src/lib/api-contract.ts` and the `systems/*/types.ts` derivations. Every MVP step that adds or renames a contract field must run codegen and pass `web-typecheck` + `web-test` in the same PR. `claim_token` is never exposed over HTTP; only `claim_token_hash` appears in read endpoints, and the raw token is returned only in the synchronous claim reply on the issuing transport.

### `_techspec.md` § Development Sequencing / Build Order

After step 10's bullet, append:

> **Co-ship requirement**: step 10 must land alongside (a) the operator UI copy/labeling pass for the publish/enqueue/coordinator-trigger boundary in `web/src/systems/tasks/components/`; (b) one new e2e scenario covering ADR-010's manual-first bookends; (c) the minimum docs set under `packages/site/content/runtime/core/autonomy/` plus the CLI reference pages for `agh me`, `agh spawn`, and the new `agh task` verbs. These three deliverables are MVP scope for step 10, not post-MVP.

### `_techspec.md` § "Manual Control Contract"

Add a sentence:

> Operator-facing UI surfaces (web Tasks UI, future operator HTTP/UDS clients) MUST visually distinguish task creation, publish/approval, run enqueue, and coordinator spawn so the operator never confuses drafting with executing. The web team owns the labeling pass; the daemon team owns ensuring the API contract supports independent transitions.

### `_techspec.md` § API Endpoints

Add:

> All endpoint responses use `claim_token_hash` (SHA-256 hex prefix or full hash, decided at step 6) instead of the raw `claim_token`. The raw `claim_token` is returned exactly once in the synchronous reply to the claim that issued it (e.g., `POST /agent/tasks/claim-next` body), and is never persisted in any read model exposed over HTTP.

### `adrs/adr-005.md`

No required edits. Optionally add a "UI implications" sentence that operators expect a clear publish→enqueue→coordinator chain in the UI.

### `adrs/adr-010.md`

Optional edit: under "Implementation Notes" reference that the web Tasks UI must keep create/publish/enqueue distinct as visible actions, and that the manual-first integration test (user creates a session and prompts it directly without coordinator) should also exist as an e2e scenario, not only a daemon integration test.

## ADR Recommendation

**Recommended: New ADR-011 — "Generated Contracts and Documentation Co-Ship Each Autonomy MVP Step."**

- **Decision**: Each MVP step that changes `internal/api/contract` regenerates `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` in the same PR, gates on `make verify` plus `web-typecheck` and `web-test`, and never exposes raw `claim_token` over HTTP. Step 10 additionally co-ships the minimum web copy/labeling pass and the minimum docs set under `packages/site/content/runtime/core/autonomy/` and `cli-reference/{me,spawn,task/run}`. Broader UI for coordinator dashboards, lease graphs, scheduler views, and marketing site updates remain post-MVP per ADR-001.

- **Alternatives**:
  1. Defer all web/docs to a post-MVP TechSpec. Rejected: web typecheck breaks on contract changes; docs gap leaves the demo milestone undocumented.
  2. Build full coordinator/lease/spawn dashboards in MVP. Rejected: violates ADR-001 phased scope and overengineers before kernel contracts settle.
  3. Absorb into the existing TechSpec without an ADR. Acceptable; chosen if the team prefers fewer ADRs. The risk is that `cy-create-tasks` reads "web modified later" and omits the cross-package shipping cost from MVP task files.

- **Consequences**:
  - Positive: kernel PRs cannot break web build; demo milestone is self-documenting; manual-first contract is visible in the UI, not only in ADR text.
  - Negative: every MVP step that touches contracts now has a small frontend obligation; PR size grows by typecheck fixes.
  - Risk: docs co-ship adds sequencing pressure on step 10. Mitigation: docs are mechanical (CLI ref tables, hook catalog tables, one autonomy folder) and parallelizable with kernel work.

## Overengineering To Avoid

These features are tempting but should NOT be in the autonomy MVP. They belong in step 15 (post-MVP web visibility) or never:

- **Real-time coordinator/lease/heartbeat live dashboards** with SSE-driven heartbeat countdowns, lease-expiry visualizations, or scheduler tick streams. The kernel must work without an observer pane in the UI.
- **A new `web/src/systems/coordinator/` system** with its own adapters, query layer, and components. The MVP renders coordinator state through the existing session detail surface (because the coordinator IS a session) — a new system is overkill.
- **A spawn lineage tree visualization component** showing parent/child relationships graphically. Render lineage as a flat field list in the session inspector for MVP; a tree component is post-MVP polish.
- **Idle agent registry view, capability matcher debug view, scheduler wake log viewer.** All scheduler state is rebuildable and observable through metrics/logs (per ADR-009 and N12). No UI for it in MVP.
- **Eval/replay UI panel** for the autonomy harness. Step 14 is post-MVP; no UI scaffolding needed in MVP.
- **Coordinator config GUI** under settings. Coordinator config is TOML-only for MVP; web settings UI for `[autonomy.coordinator]` overrides is post-MVP.
- **Marketing site rewrite for the autonomy story** — landing page hero, new bento sections, autonomy narrative. The current network-protocol-first positioning (saved project memory) should remain. The autonomy capability gets a docs section, not a homepage.
- **A `coordinator` or `autonomy` folder in `app/(home)/` or new landing components.** Post-MVP only.
- **Mermaid storyboard posters for autonomy concepts** (`autonomy-overview-storyboard-v1.png`, etc.). Co-ship a text-and-table version; commission the storyboard later.
- **Storybook stories for coordinator/spawn/lease components.** No new components in MVP, so no new stories.
- **Inline JSON viewer for raw `claim_token` / `permission_policy_json` / `spawn_budget_json`.** Render redacted summaries; raw structured fields are operator-debug only and belong in a CLI subcommand, not the UI.
- **CLI ref auto-generation pipeline.** Existing CLI ref pages are hand-written. Adding the new verbs as hand-written pages is fast; do not build a generator in MVP.
- **A new "Autonomy" sidebar nav item in the runtime UI.** Not required until step 15 ships actual screens. Existing Tasks/Sessions/Network nav items already cover the operator surface.

## References

Local files inspected:

- `.compozy/tasks/autonomous/_techspec.md`
- `.compozy/tasks/autonomous/adrs/adr-001.md` through `adr-010.md`
- `.compozy/tasks/autonomous/reviews/opus-techspec-review-round2.md`
- `.compozy/tasks/autonomous/reviews/gpt54mini-agh-code-analysis.md`
- `web/AGENTS.md`
- `web/CLAUDE.md`
- `web/package.json`
- `web/src/lib/api-client.ts`
- `web/src/lib/api-contract.ts`
- `web/src/generated/agh-openapi.d.ts` (paths and operations index)
- `web/src/components/app-sidebar.tsx`
- `web/src/systems/` (top-level inventory: agent, automation, bridges, daemon, knowledge, network, session, settings, skill, tasks, workspace)
- `web/src/systems/tasks/{adapters,components,hooks,types.ts}`
- `web/src/systems/session/{components,hooks}`
- `web/src/systems/network/{components,hooks}`
- `web/src/routes/__root.tsx` and `web/src/routes/_app/` listing
- `web/e2e/tasks.spec.ts`
- `web/e2e/network.spec.ts`
- `web/e2e/combined-flows.spec.ts`
- `web/src/storybook/` (config + msw + route stories)
- `web/src/systems/tasks/components/stories/` listing
- `packages/site/package.json`
- `packages/site/lib/runtime-navigation.ts`
- `packages/site/lib/site-config.ts`
- `packages/site/components/docs/` and `packages/site/components/landing/index.ts`
- `packages/site/app/{runtime,protocol,(home)}/` route layout
- `packages/site/content/runtime/index.mdx`
- `packages/site/content/runtime/core/meta.json` (top-level Core Concepts index)
- `packages/site/content/runtime/core/sessions/{index.mdx,lifecycle.mdx,meta.json}`
- `packages/site/content/runtime/core/agents/{spawning.mdx,meta.json}`
- `packages/site/content/runtime/core/network/meta.json`
- `packages/site/content/runtime/core/hooks/{event-catalog.mdx,meta.json}`
- `packages/site/content/runtime/core/automation/{index.mdx,meta.json}`
- `packages/site/content/runtime/cli-reference/{meta.json,task/meta.json,task/run/{claim.mdx},session/}`
- `packages/site/content/protocol/index.mdx`
- Grep confirming zero current docs mentions of `claim_token`, `lease_until`, `ClaimNextRun`, `autonomy`
