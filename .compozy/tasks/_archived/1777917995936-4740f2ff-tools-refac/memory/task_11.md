# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Aligned site docs, regenerated CLI reference, and added regression coverage so packages/site reflects the canonical tool-first surface implemented by tasks 01-10. No raw `claim_token` examples remain in user-facing docs.

## Important Decisions

- Kept the parallel CLI commands (`agh task next/heartbeat/...`) alongside the new `agh__autonomy` tool family in autonomy docs — both call the same writers, so docs show both routes instead of preferring one. The previous `cli-only` framing was the only thing that needed deletion.
- Did not edit the auto-generated CLI reference pages by hand. `make cli-docs` plus `bun run format` is enough to refresh content; only the hand-authored `cli-reference/index.mdx` and `meta.json` needed manual updates for tool/toolsets navigation.
- Wrote a single new test file `packages/site/lib/runtime-tools-canonical-docs.test.ts` rather than scattering assertions across existing tests. It checks the key tool/toolset names, deterministic denial codes, MCP auth status framing, and the absence of `--claim-token` examples in autonomy CLI pages.

## Learnings

- The existing `runtime-autonomy-docs.test.ts` asserts the literal substring `Never send raw lease credentials through `agh ch send`` — when expanding the sentence, lead with `agh ch send` so the substring stays intact rather than inserting `agh__network_send,` before it.
- `oxfmt` re-aligns markdown table column padding after `make cli-docs`. Run `bun run format` whenever CLI doc generation runs against `packages/site/content/runtime/cli-reference/`.
- The hand-authored `packages/site/content/runtime/cli-reference/index.mdx` is preserved across `make cli-docs` runs (the OperatorNote in the file says so) — it must be edited explicitly when new top-level commands appear.
- `packages/site/content/runtime/cli-reference/meta.json` is the only place the navigation order for tool/toolsets is declared; without it, the new pages exist but do not surface in the docs sidebar.
- `make codegen-check` and the OpenAPI spec already only contain `claim_token_hash`; tasks 01-10 cleared the contract drift so this task did not need to touch `internal/codegen` or `web/src/generated`.

## Files / Surfaces

- `packages/site/content/runtime/core/autonomy/task-runs-and-leases.mdx`
- `packages/site/content/runtime/core/autonomy/index.mdx`
- `packages/site/content/runtime/core/configuration/agent-md.mdx`
- `packages/site/content/runtime/core/configuration/config-toml.mdx`
- `packages/site/content/runtime/core/agents/definitions.mdx`
- `packages/site/content/runtime/core/hooks/index.mdx`
- `packages/site/content/runtime/core/hooks/declaration.mdx`
- `packages/site/content/runtime/core/automation/index.mdx`
- `packages/site/content/runtime/core/extensions/install.mdx`
- `packages/site/content/runtime/core/skills/index.mdx`
- `packages/site/content/runtime/core/skills/bundled.mdx`
- `packages/site/content/runtime/cli-reference/index.mdx`
- `packages/site/content/runtime/cli-reference/meta.json`
- `packages/site/content/runtime/cli-reference/tool/**` (new)
- `packages/site/content/runtime/cli-reference/toolsets/**` (new)
- `packages/site/content/runtime/cli-reference/**` (regenerated and reformatted)
- `packages/site/lib/runtime-tools-canonical-docs.test.ts` (new)

## Errors / Corrections

- First test draft expected the literal phrase "Login and logout remain operator-only management flows", which broke after `oxfmt` line-wrapped the prose. Loosened the assertion to "operator-only management flows" before re-running.
- Initial doc edit of the lease-credentials warning re-ordered the surfaces so the existing autonomy-docs test could no longer find `Never send raw lease credentials through `agh ch send``. Restored the original lead and appended `agh__network_send` afterwards.

## Ready for Next Run

- All `make verify` stages green: oxlint 0/0, golangci-lint 0 issues, 7094 Go tests, 49 site vitest tests, package boundaries clean, site `next build` succeeded.
- Implementation tasks 01-10 are now fully reflected in the runtime docs, including default discovery, tool-callable hook/automation/extension/config management, autonomy reason codes, MCP auth status-only framing, the new `agh-tools-guide` bundled skill, and the `agh tool` / `agh toolsets` CLI pages.
- Next dependency task `task_12` (QA Plan and Test Coverage) can consume the current docs without further alignment work.
