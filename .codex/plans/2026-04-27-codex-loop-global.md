# Globalize `codex-timed-loop` and Add `rounds=<num>`

## Summary

- Extract the plugin from AGH into `/Users/pedronauck/dev/ai/codex-loop-plugin`, using that repo as the marketplace root while keeping the publishable bundle under `plugins/codex-timed-loop/`.
- Replace repo-local `.codex` installation with a global `~/.codex` runtime, while keeping activation opt-in through `[[CODEX_LOOP ...]]` on the first prompt line.
- Evolve the header contract to accept exactly one limiter:
  - `min="<time>"`
  - `rounds="<num>"`

## Key Changes

- Distribution and installation:
  - Create a standalone repo at `/Users/pedronauck/dev/ai/codex-loop-plugin/`.
  - Add `.agents/plugins/marketplace.json` there with a neutral marketplace name.
  - Keep the plugin id as `codex-timed-loop`.
  - Install runtime files under `~/.codex/codex-timed-loop/`.
  - Register hooks in `~/.codex/hooks.json`.
  - Ensure `features.codex_hooks = true` in `~/.codex/config.toml`.

- Header contract:
  - Header must stay on the first line.
  - Valid:
    - `[[CODEX_LOOP name="qa" min="6h"]]`
    - `[[CODEX_LOOP name="qa" rounds="3"]]`
  - Invalid:
    - missing both `min` and `rounds`
    - passing both at the same time
  - `rounds` must be a positive integer.
  - Duration parsing expands to human-friendly aliases such as `30m`, `30min`, `1h 30m`, `2 hours`, `45sec`.

- Loop semantics:
  - Persist `limit_mode = "time" | "rounds"`.
  - Time mode keeps the current minimum-duration behavior.
  - Rounds mode persists:
    - `target_rounds`
    - `completed_rounds`
  - Each intercepted `Stop` counts as one completed round.
  - When `completed_rounds == target_rounds`, mark the loop `completed` and stop continuing.
  - Rapid-stop protection still applies to both modes and can cut the loop short.

- Continuation prompts:
  - Time mode continues to mention remaining time.
  - Rounds mode must use explicit round framing such as `Round 2 of 3 begins now`.
  - Remove AGH-specific `real-scenario-qa` behavior.
  - Replace it with optional global config in `~/.codex/codex-timed-loop/config.toml`:
    - `optional_skill_name`
    - `optional_skill_path`
    - `extra_continuation_guidance`

- AGH cleanup:
  - Remove `scripts/plugins/codex-timed-loop/`.
  - Remove `.agents/plugins/marketplace.json` if it only exists for this plugin.
  - Remove local `.codex` plugin leftovers created by the repo-local experiment.
  - Replace the existing `codex-timed-loop@agh-local-plugins` installation with the global marketplace source.

## Test Plan

- Unit tests:
  - parser accepts `min` and `rounds`, and rejects zero, negative, both together, or both absent
  - duration parsing covers short and long aliases
  - `rounds=3` increments `completed_rounds` on each `Stop` and completes on the third round
  - time mode still works
  - rapid-stop escalation and `cut_short` work in both modes
  - optional continuation config is applied only when valid

- Integration:
  - installer writes into `~/.codex` when `HOME` or `CODEX_HOME` is redirected to a temp directory
  - installer preserves unrelated hooks
  - installer rejects inline `[hooks]` in `~/.codex/config.toml`
  - uninstall removes only managed artifacts
  - marketplace in the standalone repo is discoverable by Codex

- Smoke tests:
  - install from `/Users/pedronauck/dev/ai/codex-loop-plugin`
  - validate `[[CODEX_LOOP ... min="30m"]]`
  - validate `[[CODEX_LOOP ... rounds="3"]]`
  - validate that prompts without the header remain no-op
  - validate behavior in at least two repositories with isolation by `session_id`

## Assumptions and Defaults

- Codex global hooks live in `~/.codex/hooks.json` / `~/.codex/config.toml`.
- Personal marketplaces are registered separately from repo-local marketplaces.
- The plugin remains explicitly opt-in and does not activate without the header.
- `rounds` means number of intercepted `Stop` events, not number of prompts or elapsed time.
