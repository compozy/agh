---
status: resolved
file: web/src/lib/stores/topology.ts
line: 83
severity: high
author: claude-reviewer
---

# Issue 144: Topology store WebSocket message parsing has no error handling for malformed JSON



## Review Comment

In the topology store's `onMessage` handler, `JSON.parse` is called on the raw WebSocket data without any try/catch:

```typescript
onMessage(event) {
    if (typeof event.data !== 'string') {
        return;
    }

    const payload = JSON.parse(event.data) as TopologySnapshot | TopologyEvent;

    update((state) => {
        const nextSnapshot = applyTopologyEvent(state.snapshot, payload);
        ...
    });
}
```

If the server sends malformed JSON (e.g., during a partial message delivery, a server bug, or a protocol change), this will throw an unhandled exception. Since this runs inside a Svelte store update, the error will propagate up and could crash the entire dashboard or leave the store in an inconsistent state.

**Suggested fix**: Wrap the `JSON.parse` call in a try/catch and log or ignore malformed messages:

```typescript
let payload: TopologySnapshot | TopologyEvent;
try {
    payload = JSON.parse(event.data);
} catch {
    return;
}
```

## Triage

- Decision: `valid`
- Notes: Confirmed in `web/src/lib/stores/topology.ts`: `JSON.parse(event.data)` is called without protection inside the websocket message handler. A malformed frame would throw through the store update path and can break the dashboard client. This is a straightforward correctness fix with a regression test.
