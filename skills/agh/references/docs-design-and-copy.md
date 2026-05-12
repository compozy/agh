# Docs, Design, And Copy

## Contents

- Truthful docs
- Copy authority
- Design authority
- Site docs
- Vocabulary
- Generated docs

## Truthful Docs

Document implemented daemon behavior. When an RFC, plan, or prompt disagrees with runtime truth, runtime truth wins. If behavior is aspirational, label it as future work or do not document it as supported.

Public docs should help a reader complete a concrete task or understand a concrete runtime contract.

## Copy Authority

COPY.md is the product-language authority for marketing copy, docs prose, package metadata, UI microcopy, CLI help, release text, OpenGraph text, and SEO descriptions.

Before using words such as today, supported, live, complete, or shipping, verify the claim against implemented behavior and release state.

## Design Authority

packages/ui/src/tokens.css is the runtime token source. DESIGN.md is generated and carries design-system specification and rationale. Do not invent colors, radii, spacing, typography, motion, or shadows for AGH surfaces.

Do not hand-edit generated DESIGN.md token regions. Run the design sync/codegen path when changing tokens.

## Site Docs

Fumadocs runtime docs live under packages/site/content/runtime/. CLI reference pages are generated from command sources. If generated references are wrong, fix source commands, not generated output.

Site validation uses Turbo-backed commands from the repo root. Do not rely on package-local Bun commands as final evidence.

## Vocabulary

Use canonical AGH terms:

- capability, not recipe, workflow, procedure, or playbook for current AGH behavior
- AGENT.md for agent definitions
- AGH Network for the protocol surface
- skill for reusable instruction bundles

Do not rename product concepts without updating code, docs, specs, task artifacts, and tests together.

## Generated Docs

Generated docs and generated design regions are contracts only through their source generators. Do not add tests that merely freeze generated prose unless generated prose itself is the product artifact under test.
