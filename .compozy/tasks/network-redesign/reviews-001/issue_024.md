---
status: resolved
file: web/src/systems/network/lib/query-options.ts
line: 57
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIet,comment:PRRC_kwDOR5y4QM66CAlY
---

# Issue 024: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Normalize the timeline query once and reuse it for both the key and the fetch.**

Right now `{}` and `undefined` hit the same endpoint payload (`limit` becomes `120` in `queryFn`) but produce different query keys because the raw `query` is passed into `networkKeys.*Messages(...)`. That splits the cache for identical requests and makes invalidation/refetch behavior inconsistent.


<details>
<summary>Suggested fix</summary>

```diff
 export function networkChannelMessagesOptions(
   channel: string,
-  query: NetworkChannelMessagesQuery = { limit: DEFAULT_TIMELINE_LIMIT },
+  query: NetworkChannelMessagesQuery = {},
   enabled = true
 ) {
+  const normalizedQuery = { limit: DEFAULT_TIMELINE_LIMIT, ...query };
+
   return queryOptions({
-    queryKey: networkKeys.channelMessages(channel, query),
-    queryFn: ({ signal }) =>
-      listNetworkChannelMessages(channel, { limit: DEFAULT_TIMELINE_LIMIT, ...query }, signal),
+    queryKey: networkKeys.channelMessages(channel, normalizedQuery),
+    queryFn: ({ signal }) => listNetworkChannelMessages(channel, normalizedQuery, signal),
     staleTime: 2_000,
     refetchInterval: MESSAGES_REFETCH_INTERVAL,
     enabled: Boolean(channel) && enabled,
   });
 }
@@
 export function networkPeerMessagesOptions(
   peerId: string,
-  query: NetworkPeerMessagesQuery = { limit: DEFAULT_TIMELINE_LIMIT },
+  query: NetworkPeerMessagesQuery = {},
   enabled = true
 ) {
+  const normalizedQuery = { limit: DEFAULT_TIMELINE_LIMIT, ...query };
+
   return queryOptions({
-    queryKey: networkKeys.peerMessages(peerId, query),
-    queryFn: ({ signal }) =>
-      listNetworkPeerMessages(peerId, { limit: DEFAULT_TIMELINE_LIMIT, ...query }, signal),
+    queryKey: networkKeys.peerMessages(peerId, normalizedQuery),
+    queryFn: ({ signal }) => listNetworkPeerMessages(peerId, normalizedQuery, signal),
     staleTime: 2_000,
     refetchInterval: MESSAGES_REFETCH_INTERVAL,
     enabled: Boolean(peerId) && enabled,
   });
 }
```
</details>


Also applies to: 84-92

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/network/lib/query-options.ts` around lines 49 - 57, Normalize
the timeline query object once inside networkChannelMessagesOptions: create a
single normalizedQuery that applies the default (DEFAULT_TIMELINE_LIMIT) to
undefined/missing fields, then use that normalizedQuery both when building the
cache key via networkKeys.channelMessages(channel, normalizedQuery) and when
calling listNetworkChannelMessages(channel, normalizedQuery, signal). Do the
same refactor for the analogous function around the later block (the other
*MessagesOptions function) so keys and fetch payloads match exactly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `web/src/systems/network/lib/query-options.ts:49-57` and `:84-92` build cache keys from the raw `query` argument but call the API with `{ limit: DEFAULT_TIMELINE_LIMIT, ...query }`.
- That means `undefined` and `{}` produce the same request payload while generating different query keys, which splits the cache and weakens invalidation/refetch consistency for identical timeline requests.
- Fix approach: normalize the timeline query once per function, then reuse the same normalized object for both `queryKey` and `queryFn`; add regression coverage for the normalized key shape.

## Resolution

- Refactored both timeline option factories to compute one normalized query object with `limit: query.limit ?? DEFAULT_TIMELINE_LIMIT`, then reuse it for both the cache key and the fetch payload.
- Added minimal out-of-scope regression coverage in `web/src/systems/network/lib/query-options.test.ts` because the batch scope did not include any existing query-options test file.
- Verified with `bun x vitest run src/systems/network/lib/query-options.test.ts`, `make web-typecheck`, `make web-test`, and `make verify`.
