---
status: completed
title: Web setup orchestrator with checklist, manifest, and verify
type: frontend
complexity: medium
---

# Task 5: Web setup orchestrator with checklist, manifest, and verify

## Overview
Turns the bridges web UI into a setup orchestrator — Slack manifest step, verify/send-test actions, state-aware checklist, progress config fields — truthful to daemon capabilities only. The bridges web UI is a capable manager but offers zero setup acceleration today; this slice surfaces the task_04 manifest/verify APIs and task_01 progress types so users stop bouncing between the UI and external docs.

<critical>
- ALWAYS READ the PRD, the TechSpec, and their catalogs (`_user_stories.md`, `_tests.md`) before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>
- MUST add a Slack manifest step to the create wizard's provider stage (fetch from the task_04 manifest endpoint; copy button + "open api.slack.com" deep link) — only for providers that expose one.
- MUST render task_04 verify results inline on the detail panel's secret-binding cards ("token valid" / failure remediation) and add a "Verify" action; expose "Send test" beside the existing dry-run test-delivery, labeled to distinguish them.
- MUST add a per-platform setup checklist section to the detail panel that derives state from the bridge itself (provider installed → secrets bound → webhook path configured/registered → verified → enabled → healthy) with deep links to the platform dashboards; for Telegram the checklist offers the webhook-registration action via the task_04 HTTP route (no CLI needed).
- MUST add `delivery_defaults.progress` fields (mode/grouping/typing/reactions) to create/edit dialogs using the regenerated types from task_01.
- MUST render only what the daemon supports — no invented controls (SD-007); checklist items map 1:1 to real API-observable state.
- MUST pull all styling from tokens (`packages/ui/src/tokens.css` via DESIGN.md grammar).
</requirements>

## Subtasks
- [x] 5.1 Manifest step in create wizard (adapter + hook + component) — present only when the provider exposes a manifest endpoint
- [x] 5.2 Verify action + inline results on secret cards; send-test action wired beside dry-run test-delivery with distinguishing labels
- [x] 5.3 State-aware setup checklist section in `bridge-detail-panel` (incl. Telegram webhook-register action via HTTP)
- [x] 5.4 Progress config fields (mode/grouping/typing/reactions) in create/edit dialogs using regenerated types
- [x] 5.5 Screenshot verification of the new surfaces via `agh-ui-screenshot`; cite captures in completion notes

## Implementation Details
Depends on task_01 (progress fields / regenerated types) and task_04 (manifest/verify/send-test/webhook-register APIs). Reference `_techspec.md` §8 (Web/Docs Impact). Uses regenerated TS types from `make codegen` artifacts — consume, do not hand-edit. Activate `agh-design` + `ui-craft` for the checklist/verify visual language; verify with `agh-ui-screenshot` before completion. Cross-refs updated: old task_08/10 → task_04; progress types from task_01.

### Relevant Files
- `web/src/systems/bridges/adapters/bridges-api.ts` — manifest/verify/send-test/webhook-register calls
- `web/src/systems/bridges/components/bridge-create-dialog.tsx` — manifest step + progress fields
- `web/src/systems/bridges/components/bridge-detail-panel.tsx` — checklist, verify, send-test
- `web/src/systems/bridges/hooks/` — new queries/mutations for manifest/verify/send-test
- `web/src/systems/bridges/components/` — new checklist/manifest components as needed

### Dependent Files
- `web/src/systems/bridges/mocks/` — MSW handlers for the new endpoints
- `web/src/generated/` — regenerated types (consumed, not edited)
- `web/src/systems/bridges/` component/hook test suites — vitest cases below

### Competitor References
- `.resources/hermes/website/docs/user-guide/messaging/slack.md:19-118` — step taxonomy the checklist encodes (translated into a live, state-aware UI checklist rather than prose)

## Deliverables
- Create wizard with manifest hand-off; detail panel with verify, send-test, and live checklist; progress config fields
- Screenshot captures cited in the completion notes
- Every test case assigned in `## Tests` implemented and passing **(REQUIRED)**

## Tests
Cases assigned from `_tests.md` (old task_11 ownership). Read each case's full definition there before writing tests. Logic/behavior only — no snapshot/render-shape filler. Visual parity is `agh-ui-screenshot`'s job (subtask 5.5).

### Web Unit / Component (`bunx turbo run test --filter=./web`, vitest)
- [x] Should render the manifest step for a Slack provider selection and copy the fetched JSON; the step is ABSENT for providers without a manifest endpoint
- [x] Should render per-check status on the matching secret card on verify-mutation success and the remediation text on failure
- [x] Should derive every checklist item from mocked bridge state (unbound secret → unchecked with CTA; enabled+healthy → all checked) — items map 1:1 to daemon-observable facts, no invented controls (SD-007)
- [x] Should serialize the progress fields (mode/grouping/typing/reactions) into `delivery_defaults.progress` on both create and edit
- [x] Should produce the correct POST body from the full create-wizard MSW flow with manifest step + progress fields

### E2E — Web (`make test-e2e-web`, Playwright; mocked daemon contract)
- [x] Bridges create flow — Should open the create wizard, select a provider WITH a manifest endpoint (Slack), surface the manifest step (copy + deep link), complete creation, and show the state-aware setup checklist on the detail panel
- [x] Verify flow — Should run the Verify action from the detail panel and render per-check status inline on the matching secret-binding cards, including a failure remediation
- [x] Send-test vs dry-run — Should expose "Send test" beside the existing dry-run test-delivery with distinguishing labels, and reflect the send-test result (SD-007 truthful-UI)

Test coverage target: >=80% for touched packages. All tests must pass under the repo gates.

## Success Criteria
- Every assigned test case implemented and passing
- `bunx turbo run lint typecheck test --filter=./web` green
- Screenshot captured via `agh-ui-screenshot` and cited for wizard + detail panel
- Every checklist item corresponds to a daemon-observable fact (SD-007 audit in PR/completion notes)

## Completion Notes (2026-07-12)

- The create wizard persists disabled bridges, fetches Slack manifests only after the create response supplies the persisted ID, and retains the committed bridge through manifest/copy failure recovery. Non-Slack providers never request the endpoint.
- The detail route projects checklist state only from provider discovery, bridge/binding/health facts, and current-session verify/register evidence. Semantic fingerprints preserve Telegram registration across lifecycle and non-provider edits while invalidating all evidence for provider-config or binding changes.
- Dry-run target resolution and real provider send-test use distinct endpoints, labels, request/result contracts, and pending states. Create/edit progress controls serialize the typed block and delete it entirely when provider default is restored.
- Focused post-freeze evidence: 14 owner suites / 192 tests passed; exact mocked-daemon Playwright passed 3/3; Web package typecheck passed; final-patch oxfmt/oxlint passed. Contract parity returned PASS and deslop returned SHIP.
- The task's broad Web gate is intentionally deferred by explicit user direction: global suites and the single `make verify` run once after every task, QA action, and review remediation. No global gate was repeated for this task.
- QA impact: `NB-039` was reset to `untested`; new scenario `NB-052` is `untested`. The tracker validates at 434 unique rows with 16 columns.

### Visual evidence

- Fully configured detail: `/tmp/agh-ui-screenshot/task05-final/bridge-detail-configured-isolated.png`
- Daemon-shaped Slack manifest handoff: `/tmp/agh-ui-screenshot/task05-final/bridge-slack-manifest-isolated.png`
- Failure and unavailable states inspected in the same capture directory: `bridge-detail-failed-verify-isolated.png` and `bridge-detail-bindings-unavailable-focused.png`
- Storybook returned HTTP `000` after teardown; no lab daemon, watcher, Storybook, or headless Chrome process remained.

### AGH Impact Audit

- Native tools: no impact; checked `agh__*` IDs, toolsets, descriptors, schemas/digests, risk flags, availability diagnostics, capability gates, and CLI/API fallbacks. Task 05 only consumes the existing typed Task 04 HTTP contracts.
- Extensibility and hooks: no impact; checked extension handshakes, hooks, skills/capabilities, tools/resources, bundles, registries, bridge SDKs, MCP sidecars, and `config.toml` lifecycle. The Web setup-profile map is presentation-only and introduces no runtime capability.
- Workspace data isolation: manifest, verify, register, bindings, send-test, route state, and transient evidence remain keyed to one bridge instance. Dynamic route state remounts on bridge ID, query keys carry the instance ID, and no global/workspace/session/agent datum or cache scope was added.
- Official AGH skill: no impact; checked `skills/agh/` against the consumed public verbs/routes. No public behavior, tool ID, CLI path, hook event, capability, bundle/resource, or memory/network/task semantic changed in this frontend-only task.
