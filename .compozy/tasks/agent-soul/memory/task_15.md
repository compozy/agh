# Task Memory: task_15.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Update Fumadocs site to document the implemented authored-context surfaces: `SOUL.md`, `HEARTBEAT.md`, session health, `[agents.soul]`, `[agents.heartbeat]`, CLI/HTTP/UDS/Host API parity, extension surfaces, AGH Network greet boundary, and validation tests.

## Important Decisions
- Created two new runtime concept pages instead of expanding `agent-md.mdx`: `core/agents/soul.mdx`, `core/agents/heartbeat.mdx`. Kept `AGENT.md` as the executable-authority reference.
- Added a dedicated `core/sessions/health.mdx` page rather than embedding health into `lifecycle.mdx`; lifecycle now links out to it.
- Added `[agents.soul]` and `[agents.heartbeat]` reference sections plus annotated example to `config-toml.mdx` to match `internal/config` defaults (Soul `2048` projection, Heartbeat `25 wakes_per_cycle`, `168h` retention, `5m`/`30m` intervals).
- Network protocol page got a "Network presence is independent from authored context" section enforcing the greet boundary.
- Extension `develop.mdx` got Host API method tables, hook events, and the three native tools (`agh__session_health`, `agh__agent_heartbeat_status`, `agh__agent_heartbeat_wake`) — no Soul native tool listed.
- Created `lib/runtime-authored-context-docs.test.ts` to assert canonical docs and CLI references exist and that no `agh session heartbeat` command leaks into docs.

## Learnings
- The CLI doc generator (`internal/cli/docpost`) emits unaligned tables and a stray blank line; oxfmt re-aligns markdown tables on save. Run `make cli-docs && bunx oxfmt packages/site/content/runtime/cli-reference` to keep diffs minimal.
- `oxfmt` reformats `.mdx` so always run it on edited Fumadocs pages before committing; lint-staged config only covers `.md` automatically.
- `bun run test` runs every workspace's vitest suite from repo root; `cd packages/site && bun run test` is enough for the docs-only loop.

## Files / Surfaces
- Authored: `packages/site/content/runtime/core/agents/soul.mdx`, `packages/site/content/runtime/core/agents/heartbeat.mdx`, `packages/site/content/runtime/core/sessions/health.mdx`, `packages/site/lib/runtime-authored-context-docs.test.ts`.
- Updated: `packages/site/content/runtime/core/agents/index.mdx`, `packages/site/content/runtime/core/agents/meta.json`, `packages/site/content/runtime/core/sessions/index.mdx`, `packages/site/content/runtime/core/sessions/lifecycle.mdx`, `packages/site/content/runtime/core/sessions/meta.json`, `packages/site/content/runtime/core/configuration/config-toml.mdx`, `packages/site/content/runtime/core/extensions/develop.mdx`, `packages/site/content/runtime/core/network/protocol.mdx`.
- Regenerated/formatted: `packages/site/content/runtime/cli-reference/**` (covers agent soul/heartbeat, session health/status/inspect/soul subcommands and the rest of the tree).

## Errors / Corrections
- First test pass failed on `subordinate` and `wake_event_retention` strings that I did not actually include in the docs; replaced with assertions that match the prose (`Agent Heartbeat`, `wake_coalesced`, plus a positive assertion for the no-`agh agent heartbeat refresh` sentence).

## Ready for Next Run
- Task 16 QA can rely on the published docs/CLI surfaces; ensure QA evidence quotes match the documented diagnostic codes (`heartbeat_if_match_header_unsupported`, `session_prompt_active_race`, etc.) and the `[agents.heartbeat]` defaults documented here.
