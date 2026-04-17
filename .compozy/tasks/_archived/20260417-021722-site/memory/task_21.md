# Task Memory: task_21.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Create four Diataxis Tutorial pages under `packages/site/content/protocol/guide/`: minimal sender, NATS transport, trust verification, and testing.
- Include runnable Go examples plus language-agnostic pseudocode, build progressively from envelope-only through transport, trust, and conformance testing.
- Verify docs build/render, spot-check Go examples compile, run browser QA on every touched route, update task tracking only after clean verification and self-review, then create one local commit if full verification passes.

## Important Decisions

- Do not claim a shipped built-in echo peer or standalone AGH Network conformance runner. Current source and existing conformance docs show neither exists. The testing tutorial will provide a tiny local echo peer and point to current package tests plus conformance expectations.
- Teach v0 NATS transport first because current AGH embeds NATS Core with `agh.network.v0`; introduce v1 NATS request/reply and verified route-token behavior by reference when trust is added.

## Learnings

- Prior Task 19/20 ledgers say protocol pages are flat under `packages/site/content/protocol/`, while this task intentionally creates nested guide pages under `packages/site/content/protocol/guide/`.
- Shared memory says the implemented v0 envelope uses `protocol`, `id`, `kind`, `channel`, `from`, `to`, `interaction_id`, `reply_to`, `trace_id`, `causation_id`, `ts`, `expires_at`, `body`, `proof`, and `ext`; avoid stale shorthand like `version`, `source`, or `target`.
- Shared memory says current AGH Network is v0-only in implementation: `proof` is preserved opaquely, Ed25519/JCS verifier and conformance runner are spec/future-oriented unless source review finds otherwise.
- QMD archived network memory confirms the current runtime exposes `agh network {status,peers,channels,send,inbox}` via daemon contracts and current docs should use `channel`, not older `space`, vocabulary.
- Subagent/source review confirmed no `internal/network/trust.go`, `sign.go`, public conformance runner, or echo peer exists.

## Files / Surfaces

- Added docs: `packages/site/content/protocol/guide/{minimal-sender,nats-transport,trust-verification,testing}.mdx`.
- Added sidebar metadata: `packages/site/content/protocol/guide/meta.json`; updated `packages/site/content/protocol/meta.json` to include `guide`.
- Source context: `docs/rfcs/003_agh-network-v0.md`, `docs/rfcs/004_agh-network-v1.md`, `internal/network/`.
- Existing protocol links to use: `/protocol/envelope/`, `/protocol/message-kinds/`, `/protocol/delivery/`, `/protocol/nats/`, `/protocol/ed25519-jcs/`, `/protocol/verification/`, `/protocol/conformance/`.

## Errors / Corrections

- Task wording mentions testing against AGH's built-in echo peer, but implementation search found no such peer. Correct docs by teaching a local echo peer and recording current implementation status instead of inventing commands.
- Task spec's `turbo run build --filter=packages/site` selector is stale; it fails with no package named `packages/site`. The real package filter is `@agh/site`, and `bunx turbo run build --filter=@agh/site` passed.
- Full `make verify` remains blocked by unrelated `web/src/styles.test.ts` design-token assertions expecting old neutral colors.

## Ready for Next Run

- Task-scoped validation passed: frontmatter/sections/links, extracted Go snippets, `@agh/site` build, and browser QA all passed.
- Do not mark Task 21 complete or commit until full `make verify` passes after the unrelated token-test mismatch is resolved or scope is expanded to fix it.
