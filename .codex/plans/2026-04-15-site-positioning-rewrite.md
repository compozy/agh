# Reposition AGH Site Around Runtime + AGH Network

## Summary

- Rewrite the public-facing `packages/site` entry surfaces so AGH is sold through user outcomes first, not through ACP or transport internals.
- Use a balanced split between `AGH Runtime` and `AGH Network`, but keep `Get Started` as the primary conversion path for cold visitors.
- Use `AGH Network` in marketing and navigation copy; reserve `AGH Network Protocol` for spec/reference pages.
- Add an explicit named comparison section on the homepage, but keep it factual and job-to-be-done oriented rather than benchmark-heavy or adversarial.
- Update the source-of-truth planning artifact in `.compozy/tasks/site/_techspec.md` so future content work does not regress back to README-style engineering framing.

## Public-Facing Changes

- Change the homepage narrative from "agent runtime with protocol internals" to "durable runtime for real agent work + open network for agent coordination."
- Make the primary hero CTA point to `/runtime` with a `Get Started` action; keep `/protocol` as the secondary path.
- Rename public marketing references from generic `Protocol` to `AGH Network` while keeping `/protocol` URLs and spec/reference terminology intact.
- Keep `Agent Operating System` only as supporting language, not as the lead category claim; competitors already crowd that phrase.
- Remove top-level copy that leads with `ACP`, `JSON-RPC`, `stdio`, `UDS`, `NATS`, `SQLite`, `SKILL.md`, `AGENT.md`, or "Wave 1 / Wave 2" roadmap language.
- Replace those terms with outcome-led claims:
  - Run real agent CLIs as durable teammates, not disposable terminal tabs.
  - Resume, audit, and replay work instead of losing context.
  - Standardize agent behavior across projects without custom glue.
  - Let specialized agents discover each other, delegate work, and report back cleanly.
  - Put agents where work already happens through bridges/channels, without selling the adapter matrix first.

## Implementation Changes

### 1. Reframe the source of truth

- Update `.compozy/tasks/site/_techspec.md` so the executive summary, homepage strategy, and any copy appendix reflect the new positioning:
  - Balanced `Runtime + AGH Network` story.
  - Runtime-first conversion path.
  - `AGH Network` branding for marketing copy.
  - Explicit rule that ACP is an implementation detail, not a homepage differentiator.
- Replace any landing-copy guidance that currently hardcodes protocol-first or implementation-first messaging.

### 2. Rewrite the homepage conversion path

- Rewrite the landing components under `packages/site/components/landing/` in this order of intent:
  - `hero`: establish the core problem and two-part solution; primary CTA `Get Started`, secondary CTA `Explore AGH Network`.
  - `two-pillars`: present `AGH Runtime` and `AGH Network` as equal product surfaces, but describe them in user language, not protocol jargon.
  - `runtime-features`: focus on durable sessions, replay, memory, automation, observability, and real CLI orchestration.
  - `protocol-section`: explain why AGH Network matters in terms of coordination and interoperability; move message-kind detail out of the center of the sales story.
  - `comparison`: replace "typical harness" with a named comparison against `OpenClaw`, `OpenFang`, and `GoClaw`.
  - `final-cta`: reinforce runtime adoption first, AGH Network exploration second.
- Reorder or simplify sections if needed so the page flows:
  1. problem/outcome
  2. runtime + network split
  3. runtime proof
  4. network proof
  5. named comparison
  6. final CTA
- Keep architecture diagrams and deep implementation proof below the main conversion argument, not before it.

### 3. Rewrite the runtime intro docs

- Rewrite the runtime entry surfaces in `packages/site/content/runtime/` so they explain the operator problem before the architecture:
  - `index.mdx`
  - `core/index.mdx`
  - `core/overview/what-is-agh.mdx`
  - `core/overview/comparison.mdx`
- Keep architecture material available, but move daemon / ACP / transport language after the product promise is established.
- Reframe runtime docs around:
  - running real agent CLIs
  - durable sessions
  - replayable history
  - workspace-aware memory and skills
  - one operator surface across CLI, HTTP/SSE, and web UI
- Keep deep ACP detail in agent/reference pages, not in first-contact overview copy.

### 4. Rewrite the protocol intro docs as AGH Network

- Rewrite the protocol entry surfaces in `packages/site/content/protocol/` so they sell the problem AGH Network solves before diving into spec structure:
  - `index.mdx`
  - `overview/index.mdx`
  - `specification/index.mdx`
- Use `AGH Network` in introductory prose and section headers; keep `AGH Network Protocol` in technical/spec contexts.
- Position AGH Network as:
  - the interoperable agent-network layer
  - separate from the AGH runtime
  - adoptable without adopting AGH wholesale
  - the place where cross-harness discovery, delegation, and communication become standardized
- Keep ACPX-style low-level protocol framing out of the homepage and reserve it for deeper reference.

### 5. Add an explicit named comparison section

- Add or rewrite one homepage comparison section that explicitly names competitors:
  - `OpenClaw`
  - `OpenFang`
  - `GoClaw`
- Structure the comparison by product lens, not by attack copy:
  - what they sell best
  - where AGH is different
  - why AGH matters for durable, distributed agent work
- Keep `ACPX` out of the homepage comparison; use it only as a reference model for protocol docs, because it is too low-level for the main product story.
- Avoid unverified benchmark claims. Use positioning differences only:
  - OpenClaw: assistant/gateway across channels
  - OpenFang: autonomous agent OS
  - GoClaw: multi-tenant AI gateway/platform
  - AGH: local-first runtime + open agent network for durable, interoperable work

### 6. Align navigation and terminology

- Update shared site labels so the top-level public wording reflects the new positioning:
  - navigation label `Protocol` becomes `AGH Network`
  - CTA/button copy uses runtime-adoption language first
  - protocol docs still live under `/protocol`
- Standardize wording across intro pages:
  - marketing: `AGH Runtime`, `AGH Network`
  - reference/spec: `AGH Runtime`, `AGH Network Protocol`
- Remove placeholder/program language such as "coming in Wave 1," "ships in Wave 2," and "adapted from RFCs" from public-facing entry pages.

## Test Plan

- Content acceptance:
  - homepage hero contains no `ACP`, `JSON-RPC`, `stdio`, `UDS`, `NATS`, or `SQLite`
  - primary CTA on the homepage points to `/runtime`
  - secondary CTA points to `/protocol`
  - homepage includes an explicit named comparison section for `OpenClaw`, `OpenFang`, and `GoClaw`
  - runtime intro pages establish user problem and product outcome before architecture details
  - protocol intro pages explain AGH Network's value and adoption path before spec mechanics
  - public entry pages contain no roadmap-placeholder language
- Consistency checks:
  - `AGH Network` is used consistently in marketing copy
  - `AGH Network Protocol` appears only where the page is clearly spec/reference oriented
  - `Agent Operating System` is supporting language, not the hero/category lead
- Repo verification after implementation:
  - update landing-page tests/snapshots that assert section headings and CTA labels
  - run the site test suite and a full site build
  - manually review `/`, `/runtime`, and `/protocol` for narrative flow and broken links

## Assumptions and Defaults

- `.resources/*` is the primary competitor corpus for this rewrite.
- Internal truth sources are `.compozy/tasks/site/analysis/*` and `docs/plans/2026-04-08-agh-network-design.md`.
- Scope is limited to `packages/site` intro/marketing surfaces plus `.compozy/tasks/site/_techspec.md`; deep CLI/API reference pages are not part of this rewrite unless they contain obvious placeholder language.
- The rewrite should not reuse old README framing or treat README text as a copy source.
- Named comparison on the homepage should stay factual and positioning-based, not benchmark-based, unless the repo already contains sourced benchmark evidence that the team wants to publish.
