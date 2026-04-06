# Fix xterm Hidden/Reveal Rendering in Dashboard Nodes

## Summary

- Root cause is in the current dashboard terminal lifecycle, not the PTY stream. `web/src/lib/components/dashboard/Terminal.svelte` always does an initial `fit()` after mount, while `web/src/lib/components/dashboard/AgentNode.svelte` can keep the terminal mounted under a `display:none` ancestor during compact zoom mode.
- Runtime inspection of the live page showed the terminal subtree mounted while hidden and, after zooming back in, the visible xterm canvases still sitting at the default `300x150` size instead of fitting the node. That matches xterm's known hidden-fit failure mode described in xterm.js PR #525 and issue #3029.
- The relevant Collaborator reference is `.resources/collaborator-ai/collab-electron/packages/components/src/Terminal/TerminalTab.tsx`, which uses an explicit `visible` prop and re-fits on visibility changes, while the broader app also isolates terminals from the canvas host through a webview layer. For this fix, copy the visibility discipline, not the full webview architecture.

## Key Changes

- Add a `visible?: boolean` prop to `web/src/lib/components/dashboard/Terminal.svelte`, defaulting to `true`.
- Refactor terminal sizing inside `web/src/lib/components/dashboard/Terminal.svelte` so all fitting goes through one guarded helper:
  - Only call `fitAddon.fit()` when `visible === true`.
  - Only call `fitAddon.fit()` when the container is measurable with positive rendered dimensions.
  - Schedule fits via double `requestAnimationFrame` on initial visible mount and single/double `requestAnimationFrame` on reveal, so fitting happens after the DOM has left `display:none`.
- Keep the xterm instance, websocket, and PTY session alive while hidden. Do not unmount the terminal on compact zoom and do not reconnect the socket just because visibility changed.
- Make the `ResizeObserver` in `web/src/lib/components/dashboard/Terminal.svelte` visibility-aware:
  - Ignore zero-sized observations.
  - Ignore observations while hidden.
  - Re-fit once on the first visible frame after `visible` flips back to `true`.
- Update `web/src/lib/components/dashboard/AgentNode.svelte` to pass `visible={!compact}` into `Terminal`.
- Preserve the existing compact-mode behavior:
  - Compact threshold stays `zoom < 0.5`.
  - Terminal stays mounted for state preservation.
  - The parent layout can continue hiding the full section; the fix is that terminal measurement and fitting must no longer happen while hidden.
- Do not add CSS canvas width/height overrides, manual delays, forced refresh hacks, or dependency upgrades in this change. Those are symptom patches, not root-cause fixes.

## Public API / Interface Changes

- `Terminal.svelte` gains one prop:
  - `visible?: boolean` with default `true`
- No changes to websocket payloads, PTY backend APIs, topology data, or session models.

## Test Plan

- Extend `web/src/lib/components/dashboard/Terminal.spec.ts` with a regression for the hidden-then-revealed path:
  - Mount `Terminal` with `visible=false`.
  - Assert the initial fit is not executed while hidden.
  - Flip to `visible=true`.
  - Flush animation frames/timers.
  - Assert `fit()` runs on reveal.
- Add or adjust the test harness in `web/src/test/setup.ts` so `ResizeObserver` behavior can be driven deliberately for hidden and visible states, instead of only auto-emitting a fixed positive size on `observe()`.
- Update `web/src/test/mocks/MockTerminalChild.svelte` and `web/src/lib/components/dashboard/AgentNode.spec.ts` so the node test can assert:
  - Compact mode still keeps the terminal mounted.
  - Compact mode passes `visible=false`.
  - Returning above the compact threshold passes `visible=true`.
- Keep the existing terminal lifecycle tests passing:
  - websocket connects on mount
  - binary frame buffering still writes correctly
  - destroy still closes socket and unregisters terminal
- Verification after implementation:
  - `npm --prefix web run check`
  - `npm --prefix web test -- --run`
  - `npm --prefix web run build`
  - `make verify`

## Assumptions

- Keep the current embedded xterm-in-node architecture for this fix.
- Do not port Collaborator's webview isolation layer in this change.
- Do not upgrade `@xterm/xterm` unless the hidden/reveal regression still reproduces after the visibility-aware lifecycle fix.
- The success criterion is: zooming out and back in no longer leaves xterm canvases stuck at default dimensions or visually corrupt, and PTY state remains continuous across the transition.
