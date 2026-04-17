# Reference Docs and Homepage IA Analysis

## Overview

This analysis compares how strong agent and harness projects split marketing, getting-started content, conceptual docs, reference material, and protocol-specific documentation. The main pattern is consistent: the homepage sells the product at a high level, the docs home routes users into a few clear paths, and deep reference material stays separate from onboarding.

The most important AGH-specific conclusion is structural, not stylistic: AGH should treat `AGH Runtime` and `AGH Network Protocol` as separate top-level documentation surfaces. The runtime docs should explain how to use the product. The protocol docs should explain how other harnesses can implement and adopt the network standard.

## Compared projects

### Chat SDK

Chat SDK has the cleanest docs IA of the set. Its docs home is a routing page, not a prose-heavy essay. It splits content into:

- Getting started
- Usage
- Adapters
- Features
- Guides
- API reference
- Contributing

The flow is progressive disclosure: overview -> getting started -> usage and core patterns -> adapters and state -> guides and API reference. The API docs are split into narrow pages per concept, which keeps reference pages short and purpose-built.

### OpenFang

OpenFang uses a broader product-docs model. Its README acts as a docs home and a product index at the same time. The information hierarchy is:

- Getting started
- Core concepts
- Integrations
- Reference
- Release and operations

This project is a strong example of how to support a complex agent OS. It separates onboarding from architecture, then keeps operational material like production release checklists out of the main conceptual flow.

### Harnss

Harnss is more marketing-led than docs-led. Its README carries the product pitch, screenshots, quick start, engine comparison, agent store, MCP server setup, install, development, and contributing. It is a useful example of product positioning for an agent harness, but it does not provide a strong multi-page documentation taxonomy.

### T3 Code

T3 Code is the simplest model here: a focused marketing homepage plus a download page. It is useful as a contrast case. The homepage has one headline, one primary CTA, one visual proof point, and one secondary path for platform variants. It keeps the experience narrow and decisive.

## Reusable documentation taxonomy patterns

1. Start with a docs home that routes, not explains. Chat SDK and OpenFang both use the top-level docs page as a table of contents for the rest of the product.
2. Put onboarding first. Every strong project has a dedicated getting-started path before deeper conceptual material.
3. Separate concepts from reference. Concepts explain the model. Reference pages list commands, APIs, or structured fields.
4. Split operational docs out of core docs. Troubleshooting, production checklists, release steps, and diagnostics should live in their own sections.
5. Use narrow reference pages when a surface is dense. Chat SDK breaks API docs into one page per core type or format, which keeps the material scannable.
6. Make integrations a first-class section when the product depends on them. OpenFang does this for providers, channels, MCP/A2A, and skills.
7. Keep protocol docs isolated from product docs when the protocol is meant to be implemented elsewhere. That is the cleanest path for AGH Network.

## Reusable homepage / positioning patterns

1. Lead with a single sentence that says what the product is and why it matters.
2. State the product advantage in plain language before listing features.
3. Use a short set of differentiated bullets, not a long feature dump.
4. Show one strong visual proof point early.
5. Make the primary action obvious. T3 Code uses a single download CTA; Harnss uses a direct install path; OpenFang leads users into start and status flows.
6. For agent harnesses, position around workflow ownership, context continuity, and control over tools and state.
7. For protocol products, position around interoperability, implementation clarity, and ecosystem adoption.

## What AGH should copy vs avoid

### Copy

- A docs home that divides the system into clear entry points instead of one long page.
- A dedicated getting-started path with the first success milestone first.
- Separate reference areas for runtime, protocol, and operational material.
- A homepage that speaks to two audiences: people who want to run AGH now, and people who want to implement the protocol in another harness.
- A strong, visible protocol section that makes AGH Network a product feature, not a footnote.

### Avoid

- Burying the protocol inside runtime documentation.
- Mixing marketing copy, onboarding, and reference in the same page.
- Making the docs feel like a giant README with no hierarchy.
- Overloading the homepage with every feature at once.
- Treating the network protocol as a niche appendix when it is the main ecosystem differentiator.

## Evidence

- `.resources/chat/apps/docs/content/docs/index.mdx` - docs home with clear routes into usage, adapters, features, guides, API, and contributing.
- `.resources/chat/apps/docs/content/docs/getting-started.mdx` - getting-started page that clusters first steps and guide paths.
- `.resources/chat/apps/docs/content/docs/api/index.mdx` - compact API landing page that maps exports to narrow reference pages.
- `.resources/chat/apps/docs/content/docs/usage.mdx` - example of concept-first docs with prerequisites and related links.
- `.resources/chat/apps/docs/content/docs/adapters.mdx` - feature matrix plus adapter model explanation.
- `.resources/chat/apps/docs/content/docs/state.mdx` - separate state concept page with focused behavioral explanation.
- `.resources/chat/apps/docs/content/docs/concurrency.mdx` - deep guide page for one behavioral concern.
- `.resources/chat/apps/docs/content/docs/error-handling.mdx` - focused guide for one class of problems.
- `.resources/chat/apps/docs/content/docs/meta.json` - top-level docs tree and section ordering.
- `.resources/chat/apps/docs/content/docs/api/meta.json` - API reference subtree.
- `.resources/chat/apps/docs/content/docs/guides/meta.json` - guide subtree with concrete use-case pages.
- `.resources/chat/apps/docs/content/docs/contributing/meta.json` - contributor docs as a separate section.
- `.resources/openfang/docs/README.md` - docs home that doubles as a product index with getting started, concepts, integrations, reference, and operations.
- `.resources/openfang/docs/getting-started.md` - step-by-step onboarding flow from install to first agent session.
- `.resources/openfang/docs/architecture.md` - architecture and system-boundary documentation separated from onboarding.
- `.resources/openfang/docs/api-reference.md` - exhaustive API reference split by endpoint family and protocol type.
- `.resources/openfang/docs/cli-reference.md` - CLI reference as a dedicated command catalog.
- `.resources/openfang/docs/skill-development.md` - standalone integration and extension docs for skills.
- `.resources/openfang/docs/workflows.md` - conceptual guide for a major product capability.
- `.resources/openfang/docs/providers.md` - provider/catalog documentation separated from core onboarding.
- `.resources/openfang/docs/mcp-a2a.md` - protocol integration docs split from the rest of the system.
- `.resources/openfang/docs/production-checklist.md` - operational release content kept out of the core learning path.
- `.resources/harnss/README.md` - marketing-first harness narrative with screenshots, quick start, engine matrix, and install instructions.
- `.resources/t3code/apps/marketing/src/pages/index.astro` - minimal marketing homepage with one CTA, one core claim, and one visual proof point.
- `.resources/t3code/apps/marketing/src/pages/download.astro` - narrow follow-up page that handles platform-specific install paths.
- `docs/rfcs/003_agh-network-v0.md` - AGH's protocol differentiator, useful for splitting runtime docs from protocol docs.
