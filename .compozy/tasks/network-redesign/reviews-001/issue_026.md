---
status: resolved
file: web/src/systems/network/mocks/handlers.ts
line: 82
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIew,comment:PRRC_kwDOR5y4QM66CAlb
---

# Issue 026: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Rewrite both conversation endpoints when swapping fixture peer IDs.**

Only `peer_from` is remapped here. If `peerId !== networkPeerFixture.peer_id`, sent messages can become self-addressed or still reference the old fixture peer on `peer_to`, which makes the mock DM timeline inconsistent.


<details>
<summary>Suggested fix</summary>

```diff
     return HttpResponse.json({
       messages: networkPeerMessagesFixture.map(message => ({
         ...message,
         peer_from: message.peer_from === networkPeerFixture.peer_id ? peerId : message.peer_from,
+        peer_to: message.peer_to === networkPeerFixture.peer_id ? peerId : message.peer_to,
       })),
     });
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/network/mocks/handlers.ts` around lines 70 - 82, The handler
for GET "/api/network/peers/:peer_id/messages" only remaps peer_from causing
inconsistent DM timelines; update the mapping to remap both peer_from and
peer_to when swapping fixture peer IDs so neither side keeps the original
networkPeerFixture.peer_id. In the http.get handler (callback using params)
change the networkPeerMessagesFixture.map callback to replace occurrences of
networkPeerFixture.peer_id with the runtime peerId for both message.peer_from
and message.peer_to (leave other fields unchanged), and apply the same change to
the other conversation endpoint handler that performs the same fixture-id swap.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `web/src/systems/network/mocks/handlers.ts:77-81` rewrites participant ids when serving `/api/network/peers/:peer_id/messages`, and that rewrite is what makes the DM timeline inconsistent for non-default peer ids.
- The base fixtures already use real peer ids (`peer_storybook_local` / `peer_storybook_remote`), so mutating either side of those messages at handler time is unnecessary and can manufacture self-addressed messages.
- The file only has one peer-conversation endpoint with this behavior. Fix approach: stop rewriting participant ids in that handler and add regression coverage that proves the remote-peer timeline preserves the original local/remote participants.

## Resolution

- Removed the peer-id rewrite from the peer-conversation handler so it now returns the fixture timeline with its original local/remote participants intact.
- Reused the minimal out-of-scope regression coverage in `web/src/systems/network/mocks/network-mocks.test.ts` to verify the remote-peer handler response matches the fixture contract exactly.
- Verified with `bun x vitest run src/systems/network/mocks/network-mocks.test.ts`, `make web-test`, and `make verify`.
