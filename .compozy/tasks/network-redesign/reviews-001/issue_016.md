---
status: resolved
file: web/src/hooks/routes/use-network-page.ts
line: 573
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIeg,comment:PRRC_kwDOR5y4QM66CAlH
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep sidebar filtering separate from active-room selection.**

`activeRoomItem` is currently resolved from `filteredChannels` / `filteredPeers`, and the effect below immediately syncs that result back into the URL. That means typing a sidebar query can change the selected room (or clear it) just because the current room no longer matches the filter. Active-room resolution should come from the full room set, with filtering applied only to what the sidebar renders.



Also applies to: 575-593

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/hooks/routes/use-network-page.ts` around lines 522 - 573,
activeRoomItem is being derived from filteredChannels/filteredPeers so typing a
sidebar query can change/clear the selected room; instead compute the active
room from the full room set (allChannels/allPeers) and only use
filteredChannels/filteredPeers for what the sidebar renders. Concretely, build
unfiltered room arrays by mapping allChannels and allPeers with
makeChannelRoomItem/makePeerRoomItem (respecting starredChannels, readMarkers,
selectedRoomKey) and use those unfiltered arrays to compute activeRoomItem,
while leaving starredChannelRooms/channelRooms/directRooms (used for rendering)
to be created from filteredChannels/filteredPeers; update the activeRoomItem
useMemo dependencies to reference the new unfiltered arrays (and apply the same
fix around the similar block at lines 575-593).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `activeRoomItem` is derived from `filteredChannels` and `filteredPeers`, and the sync effect writes that derived room back into the URL. Typing a sidebar filter can therefore clear or change the active selection even though the user did not choose a new room.
- Fix plan: derive the active room from the full unfiltered room sets while keeping sidebar rendering filtered, then add route coverage proving the current room remains selected during sidebar filtering. This requires a minimal test update outside the listed scope because the route behavior is already exercised there.
- Resolution: derived active-room selection from the full unfiltered room sets while keeping the sidebar display filtered, and added route coverage proving the selected room survives sidebar filtering.
- Verification: `bun run test:raw src/routes/_app/-network.test.tsx src/systems/network/components/network-create-channel-dialog.test.tsx`, `make web-lint`, `make web-typecheck`, and `make verify`
