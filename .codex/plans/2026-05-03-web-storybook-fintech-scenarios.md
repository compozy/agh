# Storybook Launch-Week Scenario Upgrade

## Summary

- Replace the current thin Northstar Pay Storybook dataset with a denser, cross-functional launch-week scenario built for screenshots, not just fixture realism.
- The default and populated stories should feel like a startup in motion: executive, finance, product, engineering, GTM, support, compliance, and operations roles all active at once.
- This is a Storybook-only data and story-composition upgrade. No runtime behavior, API contracts, or production schemas change.

## Interfaces

- No public API or runtime contract changes.
- Expand the internal Storybook scenario layer in `web/src/storybook/fintech-scenario.ts` so it exports a fuller company registry: roles, people, workspaces, sessions, channels, launch milestones, KPIs, campaigns, incidents, and reusable message/task catalogs.
- Shared fixture modules such as `web/src/systems/network/mocks/fixtures.ts` and `web/src/systems/tasks/mocks/fixtures.ts` should consume those richer scenario exports rather than hardcoding thin, surface-specific data.

## Implementation Changes

- Recast Northstar Pay as a launch-week company, not only a risk-ops desk.
  Use one coherent company with a flagship launch narrative: launch countdown, campaign rollout, pricing and revenue monitoring, support escalations, merchant operations, and executive check-ins.
- Expand the agent roster to 10-12 specialized roles.
  Include at minimum: `cto`, `cfo`, `frontend-engineer`, `backend-engineer` or `platform-engineer`, `product-manager`, `marketing-lead`, `copywriter`, `support-lead`, `fraud-ops`, `compliance`, and `release-manager`.
- Increase default populated density across the app shell.
  Sidebars and workspace fixtures should show multiple active workspaces, multiple sessions in flight, and enough agent rows that the shell reads like a real operating environment instead of a small demo.
- Make `Network` the hero screenshot surface.
  Replace the current 2-channel, low-message scene with a launch-week collaboration graph: 6-10 channels, 20-40 messages in the main room, 5-8 visible members, and room detail panels fully populated.
  The default active room should be a cross-functional launch room, not a narrow ops-only room.
  Message content must show real coordination across roles: CTO risk callouts, CFO burn and revenue concerns, frontend release notes, marketing launch timing, copywriter headline tweaks, support escalations, and partner or merchant follow-ups.
- Increase task depth and breadth.
  Grow tasks from a small handful to a realistic launch backlog: 15-30 tasks spanning release verification, pricing rollout, landing-page fixes, campaign approvals, support macros, finance sign-off, compliance review, and incident follow-ups.
  Populate parent and child dependencies, approval-gated tasks, blocked tasks, queued runs, and cross-role ownership so list, detail, run, dashboard, and editor stories all look busy and credible.
- Enrich sessions and transcripts.
  Sessions should exist for engineering, executive, GTM, support, and operations roles, with believable prompts, tool activity, and transcript snippets.
  The session/chat stories should surface richer markdown, more tool results, and more business context so screenshots look like active agent work, not placeholder chat.
- Enrich knowledge, skills, automation, bridges, and settings.
  Knowledge should include launch briefs, pricing notes, support playbooks, KPI definitions, and executive summaries.
  Skills should include role-specific capabilities such as launch-copy polish, frontend QA pass, executive brief synthesis, burn-report prep, and merchant escalation handling.
  Automation should include campaign send windows, launch checklists, finance alerts, support escalations, and release guardrails.
  Bridges should reflect realistic integrations such as Slack, Linear, HubSpot, Stripe-adjacent operations, or CRM/support lanes where appropriate.
- Add screenshot-oriented populated variants where the default state is still too sparse.
  For the highest-value surfaces, create explicit hero populated states that maximize above-the-fold information density without changing the UI itself.
  The goal is not more story count everywhere; it is better default screenshots on the few surfaces marketing will actually capture.

## Test Plan

- Update Storybook regression tests to assert density and role coverage, not just placeholder removal.
  Add expectations for specialized roles such as CTO, CFO, frontend, marketing, and copywriter.
  Add minimum populated counts for high-value fixtures such as channels, visible messages, sessions, and tasks.
  Keep bans for old self-referential placeholders and placeholder registry URLs.
- Run focused verification on the Storybook-heavy surfaces and shared fixture tests first.
- Run `make web-lint`, `make web-typecheck`, targeted Storybook/Vitest suites, `bun run --cwd web build-storybook`, and finally `make verify`.

## Assumptions And Defaults

- Use one canonical company: Northstar Pay.
- Use one canonical narrative: launch week, not a pure incident-response scene and not a board-review-only scene.
- Keep all screenshot-facing product copy in English.
- Preserve the current UI layout and component structure; this plan upgrades data richness, populated defaults, and screenshot composition, not the design system.
- Prefer coherence over randomness: every populated story should feel like it belongs to the same week, same company, and same operating context.
