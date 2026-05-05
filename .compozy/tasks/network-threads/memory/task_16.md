# Task Memory: task_16.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Public site, runtime docs, examples, API reference, CLI reference, landing visuals/copy, and blog post all rewritten around `surface`, `thread_id`, `direct_id`, and `work_id`. `direct` no longer advertised as a wire kind anywhere in active site docs.

## Important Decisions

- Kept `protocol/interactions.mdx` filename to preserve `/protocol/interactions/` deep links; rewrote the page as Work Lifecycle and updated frontmatter title only. Cross-references inside other docs use the new "Work Lifecycle" label but keep the `/protocol/interactions` URL.
- ed25519-jcs.mdx worked example switched from a `say` envelope (which now requires conversation surface fields) to a `greet` envelope so the canonical-bytes section did not need fabricated SHA/Ed25519 values. Implementers are pointed to `internal/network/v1trust/` for the cryptographic golden fixture.
- `bridges/routing.mdx` `--thread-id` flag is bridge-provider routing metadata, not the AGH network thread; left unchanged. Legacy-term scan in `runtime-manual-cli-examples.test.ts` is intentionally scoped to `runtime/core/network/`, `runtime/use-cases/`, `protocol/` to avoid false positives on bridge docs.
- New legacy-term scan only flags inside fenced code blocks. Tombstone narrative ("rejects envelopes that try to set `kind:"direct"`") is allowed because it documents the rejection rule rather than advertising the term.

## Learnings

- `make cli-docs` regenerates the entire `packages/site/content/runtime/cli-reference/` tree, not just the changed surface; commit those re-generated files together.
- `bun run typecheck` chains `generate:openapi → source:generate → content:generate → tsgo --noEmit`. Test suite re-runs the same generators; running `bun run test` after edits is sufficient to refresh `.source/` and `.velite/`.
- Fumadocs MDX rejects `surface:\"thread\"` inside JSX attribute strings (the backslash escape is invalid in attribute values). Use plain text in `RouteRow` descriptions.
- The `content-code-block-quality` and `content-diagram-quality` site tests forbid raw ` ```mermaid ` fences in MDX; use the `<Mermaid chart={...} caption="..." />` component instead.
- `content-frontmatter-quality` caps page descriptions at ~160 characters.

## Files / Surfaces

- Protocol: `envelope`, `message-kinds`, `interactions`, `examples`, `overview`, `delivery`, `conformance`, `nats`, `capability-discovery`, `ed25519-jcs`; guides `minimal-sender`, `nats-transport`, `trust-verification`, `testing`.
- Runtime: `core/network/{index,protocol,channels-and-peers,delivery-and-safety,task-ingress}`; new `core/network/{threads,directs,work}.mdx`; `core/network/meta.json`; `core/agents/capabilities.mdx`; `use-cases/handoff-between-agents.mdx`; `guides/coordinate-agents-over-network.mdx`.
- Generated: `runtime/api-reference/network.mdx` regenerated via `make codegen` + `bun run generate:openapi`; `runtime/cli-reference/network/` regenerated via `make cli-docs` (added `directs/`, `threads/`, `work/` subtrees and refreshed `send.mdx`).
- Landing & blog: `components/landing/network-section.tsx`, `components/landing/primitives/network-kinds.ts`, `components/landing/network-protocol-visual.tsx`, `components/blog/kind-chip.tsx`, `content/blog/posts/introducing-agh-the-first-agent-network-protocol.mdx`.
- Tests: `lib/landing-cli-snippets.test.ts`, `lib/runtime-manual-cli-examples.test.ts` (added removed-flags + scoped legacy-term scan + restricted-visibility scan), `components/landing/__tests__/landing.test.tsx`.

## Errors / Corrections

- Initial pass declared the minimal-sender example as a `say` without surface, which is now invalid; corrected to `say` + `surface:"thread"` + `thread_id`. Same correction applied to the testing-guide echo-peer fixture.
- First docs-validation patch flagged any document containing `kind:"direct"` even in tombstone language. Refined to scan only fenced code blocks under network/use-case/protocol roots.
- Initial RouteRow descriptions used `surface:\"thread\"` JSX-escaped strings; Turbopack rejected the attribute. Replaced with plain prose descriptions.

## Ready for Next Run

- Active legacy vocabulary (`interaction_id`, `kind:"direct"`, `--interaction-id`, `--thread-id` for AGH network, `--direct-id`, `--work-id`) no longer appears as advertised usage in active site docs. Bridge `--thread-id` flag is intentional and left in place.
- Generated outputs (`openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, `packages/site/content/runtime/cli-reference/`) regenerated and in sync with source. Re-running `make codegen-check` and `make cli-docs` confirms zero drift.
- `make verify` passed (8386 Go tests, 257 site vitest tests, all package boundaries respected).
