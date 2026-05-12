# Troubleshooting capture failures

## Symptom: every PNG is 5–10 KB and renders "Couldn't find story"

Cause: the story id passed to `cap.mjs` does not exist in the running Storybook.

Fix: list valid ids first.

```
bun run .agents/skills/agh-ui-screenshot/scripts/list-stories.mjs http://localhost:6006 --filter <substring>
```

Match the exact id (case-sensitive, dashes only). Common slip: `routes-app-stories-tasks-id--overview` vs `routes-app-stories-tasks--id-overview` (the `id-` segment is part of the story file name, not a separator).

## Symptom: PNGs render in fallback system fonts (Arial / Helvetica)

Cause: `document.fonts.ready` resolved before Inter or JetBrains Mono finished decoding, OR the WARN line `fonts.ready timeout` appeared in stdout.

Fix:

1. Bump `--wait` from `2200` to `4000` and re-run.
2. Confirm the Storybook process has loaded font assets (check `<storybook-host>/sb-common-assets/*` 200s in the network tab of a real browser visit).
3. On corporate networks, verify external font CDNs are not blocked. AGH ships fonts locally via `@agh/ui`, but third-party deps may pull from CDNs.

## Symptom: `chrome-launcher` fails with "No usable sandbox"

Cause: a hardened Linux environment refuses `--no-sandbox`.

Fix: the script already passes `--no-sandbox`. If still failing, the host's seccomp profile is blocking Chrome. Run under Docker with `--cap-add SYS_ADMIN` or use `--user-data-dir=$(mktemp -d)` to avoid the protected `~/.config/chromium` location.

## Symptom: capture hangs indefinitely on a specific story

Cause: that story throws a runtime error inside React and never fires the `load` event.

Fix:

1. Visit the URL in a real browser; check the iframe devtools console for errors.
2. If the story crashes, fix the story (its `play` function may be running before refs settle).
3. As a workaround, wrap the `cap.mjs` invocation in `timeout 90 bash -c '...'` so the orchestrator can move on.

## Symptom: Chrome zombies pile up after a crashed run

Cause: `cap.mjs` died before reaching its `finally` block (SIGKILL, OOM).

Fix:

```
pkill -9 -f "Google Chrome.*headless"
pkill -9 -f "Chrome Helper.*headless"
```

Then re-run.

## Symptom: PNG is correct but cropped at unexpected width

Cause: `--window-size` was set but `setDeviceMetricsOverride` was not (the script always applies it; this happens if `cap.mjs` was modified).

Fix: confirm `Emulation.setDeviceMetricsOverride({ width, height, deviceScaleFactor: 1, mobile: false })` is still called before the per-target loop.

## Symptom: bun install errors in the workdir

Cause: the workdir's `package.json` declared a dep range that no longer resolves, or bun's cache is corrupted.

Fix:

```
rm -rf <workdir>/node_modules <workdir>/bun.lock
bash .agents/skills/agh-ui-screenshot/scripts/setup-workdir.sh <workdir>
```

## Symptom: Storybook tells the user the port is unavailable

Cause: a previous `bun run storybook` is still running.

Fix: list the offender and either reuse it or kill it.

```
ps aux | grep storybook | grep -v grep
# kill the duplicate, NOT a healthy long-running instance the operator is using
```

Two healthy Storybook instances (web on 6006, packages/ui on 6007) is the expected state. If both ports are already serving valid `/index.json`, do not start new ones.
