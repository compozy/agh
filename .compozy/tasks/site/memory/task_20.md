# Task Memory: task_20.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Create four protocol Reference pages for Trust Profile v1, verification, NATS binding, and conformance under `packages/site/content/protocol/`.
- Ground content in RFC 004, current `internal/network/`, Task 19 flat protocol docs, and TechSpec Appendix B.
- Required evidence before closeout: site build, browser QA for touched `/protocol/*` routes, self-review/content checks, full `make verify`, tracking updates, and local commit only after clean verification.

## Important Decisions

- Use RFC 004 as the normative semantic source for v1 trust, NATS additions, and conformance; distinguish current implementation gaps if `internal/network/` does not implement those v1 features yet.
- Use the current docs package selector `bunx turbo run build --filter=@agh/site`; the task's literal `--filter=packages/site` selector is known stale from shared memory but will be checked/reported.
- Keep the new protocol pages flat under `packages/site/content/protocol/`, matching Task 19 route conventions.

## Learnings

- QMD archived/ledger/plan searches surfaced mostly stale older NATS planning and prior docs workflow notes; current RFC 004 plus current source remain authoritative.
- Current `internal/network/` is v0-only: it accepts only `agh-network/v0`, preserves `proof` opaquely, derives v0 route tokens from SHA-256 over Peer ID, and does not accept `nickname@fingerprint` verified handles.
- Deterministic Ed25519/JCS fixture in `ed25519-jcs.mdx` was independently verified locally with Node crypto.
- Browser QA confirmed all four new routes render at `http://localhost:3007/protocol/{ed25519-jcs,verification,nats,conformance}/`; the internal `/protocol/ed25519-jcs/#worked-example` anchor also resolves and `agent-browser errors` returned no output.
- Correct site build selector is still `bunx turbo run build --filter=@agh/site`; the literal task selector `--filter=packages/site` fails because there is no workspace package named `packages/site`.

## Files / Surfaces

- Added docs: `packages/site/content/protocol/{ed25519-jcs,verification,nats,conformance}.mdx`.
- Updated sidebar metadata: `packages/site/content/protocol/meta.json`.

## Errors / Corrections

- `bunx turbo run build --filter=packages/site` fails with `No package found with name 'packages/site' in workspace`; used and verified `bunx turbo run build --filter=@agh/site` plus direct `bun run build` in `packages/site`.
- Full `make verify` remains blocked outside Task 20 in `web/src/styles.test.ts`: 3 failed token assertions expect `#121212`, `#1C1C1E`, and `#2C2C2E`, while current CSS has `#141312`, `#1e1c1b`, and `#2e2c2b`.
- Because the full gate failed, task tracking was not marked complete and no local commit was created.

## Ready for Next Run

- Task-scoped docs/content/browser checks passed; resolve or authorize the unrelated design-token gate before updating task completion tracking or committing.
