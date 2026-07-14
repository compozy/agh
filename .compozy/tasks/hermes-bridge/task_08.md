---
status: completed
title: Docs parity (8/8) and truthfulness fixes
type: docs
complexity: medium
---

# Task 8: Docs parity (8/8) and truthfulness fixes

## Overview
Brings bridge docs to full-provider parity, fixes Discord truthfulness violations, adds the
`ADDING_A_BRIDGE` author guide, and updates `skills/agh` for new verbs/config. Five of eight
providers lack setup guides, the slots table omits the same five, and Discord's README
documents Slack APIs — this slice makes docs match runtime truth and turns task_07's drift
invariants green.

<critical>
- ALWAYS READ the PRD, the TechSpec, and their catalogs (`_user_stories.md`, `_tests.md`) before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>
- MUST fix `extensions/bridges/discord/README.md`: Ed25519 public-key verification (not
  "signing-secret"); Discord REST endpoints (not `chat.postMessage`/`chat.update`/`chat.delete`).
- MUST complete the `index.mdx` "Supported provider slots" table to all 8 providers (matching
  each `extension.toml` secret_slots exactly — SD-007).
- MUST add setup sections (or per-provider pages) for Teams, WhatsApp, GChat, GitHub, Linear —
  behavior-first, then setup steps, auth-mode-dependent slots explained for GitHub/Linear.
- MUST rewrite setup flows CLI-first (`agh bridge create` / `setup` / `manifest` / `verify`)
  with curl as a fallback appendix; delete the "examples use the HTTP API because…" framing.
- MUST add `internal/bridges/ADDING_A_BRIDGE.md` (every touch point for a new provider + a grep
  verification recipe) and a mirroring Fumadocs bridge-author page with copy-pasteable
  `extension.toml` + adapter templates.
- MUST update the official bundled AGH skill: `skills/agh/` documents bridge surfaces this
  program changes — add the new `manifest`/`setup`/`verify`/`send-test` verbs, the
  webhook-registration route, and the `delivery_defaults.progress` block (`native-tools.md` +
  `runtime-operations.md` as applicable). This is the §7 Impact Audit "Official AGH skill"
  owner.
- MUST follow `COPY.md` and `docs/_memory/glossary.md` vocabulary throughout.
- MUST make task_07 drift invariants green (slots table, Slack scope list vs task_04
  constants, setup-section presence) — no new prose-freezing tests here; the drift suite from
  task_07 is the owning gate.
</requirements>

## Subtasks
- [x] 8.1 Truthfulness fixes (Discord README, slots table to 8/8, CLI-first framing in
      `setup.mdx`)
- [x] 8.2 Five new provider setup sections/pages (Teams, WhatsApp, GChat, GitHub, Linear)
- [x] 8.3 `ADDING_A_BRIDGE.md` + Fumadocs bridge-author page with templates + grep recipe
- [x] 8.4 `skills/agh/` bridge references updated for the new verbs, routes, and progress
      config
- [x] 8.5 Cross-check every claim against provider code (runtime truth beats copy — L-018
      delegated-docs audit posture); task_07 drift suite green

## Implementation Details
Reference `_techspec.md` §8 and §4 delete targets (curl-first framing, 3-provider table).
New-verb reference pages (`manifest`/`setup`/`verify`/`send-test`) regenerate from cobra
inside their own tasks (task_04); this task owns the conceptual/setup content. Drift harness
ownership: task_07; this task is the content that turns it green. Docs-only where no
generated artifact is touched — `make verify` exception applies only if zero code changes.
Skills: `documentation-writer`, `copywriting`, `skill-best-practices` (for `skills/agh/`).

### Relevant Files
- `packages/site/content/runtime/core/bridges/{index,setup,routing}.mdx` (+ new per-provider
  pages under `core/bridges/`)
- `extensions/bridges/discord/README.md` — Ed25519 + Discord REST truthfulness
- `internal/bridges/ADDING_A_BRIDGE.md` (new) — author checklist + grep recipe
- `skills/agh/references/native-tools.md` (+ `runtime-operations.md` as applicable)

### Dependent Files
- `packages/site/content/runtime/core/bridges/meta.json` — nav entries
- `internal/extension/bridge_docs_conformance_test.go` (task_07) — turns green
- Other provider READMEs for consistency (optional touch)

### Competitor References
- `.resources/hermes/website/docs/user-guide/messaging/{slack,telegram,discord,whatsapp}.md`
  — guide structure (Overview → generated-artifact path → manual → scope table → configure →
  validate → troubleshoot)
- `.resources/hermes/gateway/platforms/ADDING_A_PLATFORM.md:293` — grep verification recipe
- `.resources/hermes/website/docs/developer-guide/adding-platform-adapters.md:36-451` —
  author-guide shape

## Deliverables
- 8/8 provider setup coverage; corrected Discord README; complete slots table; CLI-first setup
- Bridge-author guide (`ADDING_A_BRIDGE.md` + Fumadocs page)
- `skills/agh/` bridge references updated (official-skill co-ship, §7 Impact Audit owner)
- Task_07 drift tests green; site build green

## Tests

Cases assigned from `_tests.md` (remap: old task_15 → this task; drift suite owned by
task_07). No new prose-freezing tests — docs-artifact tests are forbidden except where the
artifact is the product contract and the stronger gate already owns it (task_07 drift suite).

- Unit tests (suite: Not applicable — no production code change; prose-freezing tests
  forbidden by test-placement rules):
- Integration tests (suite: `internal/extension/bridge_docs_conformance_test.go` from
  task_07 — the owning gate — `_tests.md` §5 case 17):
  - [x] Slots-table completeness: every discovered provider appears in `index.mdx`
  - [x] Slack docs scope list matches the task_04 scope constants
  - [x] Every discovered provider has a setup section (or per-provider page)
  - [x] Adding provider+docs together stays green; the harness already fails one-sided adds
- E2E / site gate:
  - [x] Focused site owner gates green: Oxfmt, Turbo typecheck/test (the package defines no
        Turbo `lint` task), and Turbo build
  - [x] Link/nav integrity verified by the site build (`meta.json` entries resolve)
- Spot-audit (manual, recorded in completion notes — not a frozen prose test):
  - [x] Discord Ed25519 + Discord REST claims match provider code
  - [x] Teams Bot Framework, GitHub/Linear auth modes match each `extension.toml`
- Test coverage target: n/a (docs)
- All assigned gates must pass

## Success Criteria
- Every assigned test case / gate implemented and passing
- Task_07 drift suite green; site builds clean
- Zero claims contradicting provider code (spot-audit: Discord Ed25519, Teams Bot Framework,
  GitHub/Linear auth modes match `extension.toml`)
- Official AGH skill documents the new bridge verbs, webhook-register route, and
  `delivery_defaults.progress` block
