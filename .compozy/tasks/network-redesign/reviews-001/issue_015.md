---
status: resolved
file: web/src/hooks/routes/use-network-page.ts
line: 475
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIec,comment:PRRC_kwDOR5y4QM66CAlC
---

# Issue 015: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Normalize `channel`/`peer` search params to a single target.**

`validateNetworkSearch()` can return both `channel` and `peer`, but `activeRoomItem` always checks `search.peer` first. If a stale `peer` param is present and does not resolve, it suppresses an otherwise valid `channel` selection and leaves the page without an active room. Either make the params mutually exclusive during validation or explicitly fall back to `channel` when the peer lookup misses.

<details>
<summary>🔧 One simple fix</summary>

```diff
 function validateNetworkSearch(search: Record<string, unknown>): NetworkRouteSearch {
+  const channel = normalizeSearchValue(search.channel);
+  const peer = normalizeSearchValue(search.peer);
   const kindValue = normalizeSearchValue(search.kind);
   const normalizedKind =
     kindValue === "all" || (kindValue && toNetworkKindFilter(kindValue))
       ? (kindValue as NetworkKindFilter)
       : undefined;

   return {
-    channel: normalizeSearchValue(search.channel),
+    channel: peer ? undefined : channel,
     details: search.details === "closed" ? "closed" : undefined,
     kind: normalizedKind === "all" ? undefined : normalizedKind,
-    peer: normalizeSearchValue(search.peer),
+    peer,
   };
 }
```
</details>


Also applies to: 562-573

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/hooks/routes/use-network-page.ts` around lines 462 - 475,
validateNetworkSearch currently can return both channel and peer which lets a
stale peer param suppress a valid channel selection; update
validateNetworkSearch to make channel and peer mutually exclusive by preferring
channel when both are present: compute normalizedChannel =
normalizeSearchValue(search.channel) and normalizedPeer =
normalizeSearchValue(search.peer), then if normalizedChannel is truthy return
channel=normalizedChannel and peer=undefined, else return peer=normalizedPeer
and channel=undefined (keeping existing handling for kind/details). Apply the
same change to the other identical validation block referenced in the diff so
only one of channel/peer is ever returned.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `validateNetworkSearch()` currently allows both `channel` and `peer`, but `activeRoomItem` resolves `search.peer` first. A stale peer param can therefore suppress a valid channel selection and leave the page with no active room.
- Fix plan: normalize the route search so `channel` and `peer` are mutually exclusive, preferring the explicit channel target when both are present, and add route coverage in the existing network page test file. This requires a minimal test update outside the listed scope because the route behavior is already exercised there.
- Resolution: normalized network route search so `channel` and `peer` are mutually exclusive, preferring a valid channel target when both are present, and added route coverage for the stale-peer case.
- Verification: `bun run test:raw src/routes/_app/-network.test.tsx src/systems/network/components/network-create-channel-dialog.test.tsx`, `make web-lint`, `make web-typecheck`, and `make verify`
