# TC-REG-002: `make cli-docs` Regenerates Without Drift

**Priority:** P0 (Critical)
**Type:** Regression / Docs
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove `make cli-docs` regenerates the CLI reference tree under `packages/site/content/runtime/cli-reference/` with no diff against the committed tree (after `bun run format`). Confirm the hand-authored `index.mdx` and `meta.json` keep `tool/` and `toolsets/` groups listed and that no `--claim-token` examples remain.

## Traceability

- Task: task_11.
- TechSpec: "Docs And Generated Surfaces", "Delete Targets".
- ADR: ADR-001 (tool-first surface), ADR-005 (autonomy hard cut).
- Surfaces: `Makefile` (`cli-docs`), `packages/site/content/runtime/cli-reference/`.

## Preconditions

- Working tree clean.
- Bun deps installed.

## Test Steps

1. Capture pre-state:
   ```bash
   git status --porcelain packages/site/content/runtime/cli-reference/ | tee qa/logs/TC-REG-002/pre-status.txt
   ```

2. Regenerate:
   ```bash
   make cli-docs | tee qa/logs/TC-REG-002/make-cli-docs.log
   ```

3. Format:
   ```bash
   bun run --cwd packages/site format | tee qa/logs/TC-REG-002/bun-format.log
   ```

4. Compare:
   ```bash
   git status --porcelain packages/site/content/runtime/cli-reference/ | tee qa/logs/TC-REG-002/post-status.txt
   git diff --stat packages/site/content/runtime/cli-reference/ | tee qa/logs/TC-REG-002/post-diff.txt
   ```
   - **Expected:** Empty diff.

5. Confirm hand-authored landing pages list new groups:
   ```bash
   grep -nE "tool/|toolsets/" packages/site/content/runtime/cli-reference/index.mdx \
     | tee qa/logs/TC-REG-002/index-grep.txt
   grep -nE "tool|toolsets" packages/site/content/runtime/cli-reference/meta.json \
     | tee qa/logs/TC-REG-002/meta-grep.txt
   ```
   - **Expected:** Both files mention `tool/` and `toolsets/` groups.

6. Confirm no stale autonomy CLI examples:
   ```bash
   grep -RIn -- "--claim-token" packages/site/content/runtime/cli-reference/ \
     | tee qa/logs/TC-REG-002/claim-token-flag-grep.txt
   grep -RIn "claim_token[^_]" packages/site/content/runtime/cli-reference/ \
     | tee qa/logs/TC-REG-002/claim-token-grep.txt
   ```
   - **Expected:** Zero matches.

7. Spot-check new mutable command docs exist and are linked:
   - `cli-reference/config/{set,unset,diff}.mdx`
   - `cli-reference/hooks/{create,update,delete,enable,disable}.mdx`
   - `cli-reference/automation/jobs/{create,update,delete,trigger,history}.mdx`
   - `cli-reference/extension/{install,update,remove,enable,disable}.mdx`
   - `cli-reference/mcp/auth/status.mdx` (and `login`/`logout` retained for operator).
   - `cli-reference/task/{next,heartbeat,complete,fail,release}.mdx` post hard cut.
   - `cli-reference/tool/*.mdx` (list, search, info, invoke, mcp).
   - `cli-reference/toolsets/{list,info}.mdx`.

## Evidence To Capture

- Logs, diffs, grep outputs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Generator output diverges due to oxfmt re-alignment | tables not formatted | Re-run `bun run format`; commit the formatted version |
| New top-level CLI group added | e.g., new family in a future task | Hand-author `index.mdx`/`meta.json` entry to keep it listed |
| `claim_token_hash` appears | observability docs | Allowed |

## Channels Exercised

- `cobra` CLI JSON export.
- `packages/site/content/runtime/cli-reference/` tree.

## Related Test Cases

- TC-REG-001 (codegen drift).
- TC-REG-003 (site build).
- TC-SEC-001 (cross-channel claim_token sweep).
