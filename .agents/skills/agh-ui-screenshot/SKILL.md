---
name: agh-ui-screenshot
description: Captures deterministic PNG screenshots of AGH runtime UI surfaces — Storybook stories rendered via the isolation iframe at /iframe.html?id=<story-id>&viewMode=story, and arbitrary local URLs — by driving headless Chrome through the Chrome DevTools Protocol with chrome-launcher plus chrome-remote-interface. Waits explicitly for the load event, an extra React/Storybook settle window, and document.fonts.ready before capturing, so JetBrains Mono and Inter never render mid-FOUT. Use when the redesign-v2 audit, visual regression diff, or design-parity comparison against docs/design/new-proposal needs PNGs of stories from web/ and packages/ui Storybook instances, or any local dev-server URL. Do not use for production E2E flows that need clicks or interactivity (use Playwright or agent-browser instead), for capturing remote authenticated sites, or as a substitute for Storybook's own test-runner.
---

# AGH UI Screenshot

Capture deterministic PNGs of AGH runtime UI surfaces (Storybook isolation iframes, the proposal mock at `docs/design/new-proposal/agh-refined-7.html`, or any local dev-server URL) via the Chrome DevTools Protocol. Chrome's built-in `--screenshot` flag hangs on Storybook iframes and races font loading; this skill avoids both by driving Chrome explicitly through `chrome-launcher` + `chrome-remote-interface`.

## Procedures

**Step 1: Confirm dev servers are reachable.**
1. Run `curl -s -o /dev/null -w "%{http_code}\n" http://localhost:6006/` for the `web/` Storybook and `http://localhost:6007/` for the `packages/ui/` Storybook. Expect `200`.
2. If a server is down, start it from its workspace: `cd web && bun run storybook` (defaults to 6006) or `cd packages/ui && bun run storybook` (defaults to 6007). Wait for the "Storybook started" banner before proceeding.
3. Read `references/storybook-urls.md` for the canonical port table, iframe URL grammar, and story-id naming patterns.

**Step 2: Prepare the capture workdir.**
1. Execute the bootstrap helper: `bash .agents/skills/agh-ui-screenshot/scripts/setup-workdir.sh /tmp/agh-ui-screenshot`. This is a bootstrap helper — it creates the directory and installs `chrome-launcher` + `chrome-remote-interface` via bun. It is idempotent; re-running on an existing workdir is a no-op.
2. Capture its stdout — the printed path is the workdir to `cd` into for every later `bun run` invocation.

**Step 3: Resolve the story ids to capture.**
1. Skip this step when capturing a non-Storybook URL (e.g., the proposal mock).
2. For Storybook captures, run the read-only helper `bun run .agents/skills/agh-ui-screenshot/scripts/list-stories.mjs http://localhost:6006 [--filter <substring>]`. The script fetches `index.json` and emits one story id per line, optionally filtered.
3. Confirm each story id intended for capture appears in the output. A missing id will land on Storybook's "Couldn't find story" fallback and produce a tiny (under 20 KB) PNG. If unsure of the naming pattern, read `references/storybook-urls.md`.

**Step 4: Capture screenshots via the CDP helper.**
1. From the workdir created in Step 2, run the mutating helper `bun run .agents/skills/agh-ui-screenshot/scripts/cap.mjs --out <output-dir> --width <W> --height <H> --wait <ms> --shot <name> <url> [--shot <name> <url> ...]`. Each `--shot` pair writes `<output-dir>/<name>.png`.
2. Use `1440 × 900` for full-route surfaces, `1680 × 1050` for wide-breakpoint validation, `1100 × 700` for primitive previews, `320 × 800` for collapsed sidebar. Defaults documented in `references/cdp-flow.md`.
3. Use `--wait 2200` as the empirical floor for AGH's `routes-app-stories-*` surfaces. Bump to `4000` if any capture shows fallback fonts.
4. Verify the helper's stdout: each successful capture prints `saved <name>`; failures print `FAIL <name> <message>` and the script exits 0 — scan for `FAIL` lines.
5. Inspect output PNG sizes. Anything under 20 KB is suspicious — see `references/troubleshooting.md`.

**Step 5 (conditional): Capture the new-proposal mock.**
1. Only run this step when the audit needs proposal-side screenshots beyond what `docs/design/new-proposal/agh-refined-7.html` renders by default.
2. Read `references/proposal-mock-capture.md` for the clone-and-patch wrapper that exposes view state via URL query params, the recommended view matrix, and the cleanup step.
3. Boot a static server: `cd docs/design/new-proposal && python3 -m http.server 8765` (background it).
4. Capture via the same `scripts/cap.mjs` helper, pointing each `--shot` at `http://localhost:8765/_proposal-capture.html?view=…`.

**Step 6: Confirm captures land where intended.**
1. List the output dir and verify every expected PNG exists with a plausible size (typically 80 KB – 250 KB for full-route surfaces).
2. Spot-check at least one capture by reading the PNG (an image-aware tool or visual diff) to confirm the rendered viewport matches the expected surface.

## Error Handling

* If a capture hangs longer than 90 s, the Storybook story is likely throwing inside React and never firing `load`. Read `references/troubleshooting.md` "Symptom: capture hangs indefinitely on a specific story" for recovery.
* If every PNG renders in fallback system fonts, bump `--wait` and re-read `references/troubleshooting.md` "Symptom: PNGs render in fallback system fonts".
* If the bootstrap helper fails on `bun add`, the workdir cache may be corrupted — delete `<workdir>/node_modules` and `<workdir>/bun.lock`, then re-run `scripts/setup-workdir.sh`.
* If Chrome zombies pile up after a SIGKILL, run `pkill -9 -f "Google Chrome.*headless" && pkill -9 -f "Chrome Helper.*headless"` and retry.
* For any other symptom, consult `references/troubleshooting.md` before re-implementing the capture pipeline by hand.
