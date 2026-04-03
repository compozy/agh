---
status: resolved
file: web/src/lib/stores/topology.ts
line: 36
severity: medium
author: claude-reviewer
---

# Issue 160: Topology store rebuild races with itself on rapid WebSocket updates



## Review Comment

The `rebuild` function in the topology store uses a layout token to prevent stale layouts from being applied:

```typescript
const rebuild = async (snapshot: TopologySnapshot): Promise<void> => {
    const layoutToken = ++activeLayoutToken;
    const graph = await computeDashboardGraph(snapshot);
    if (layoutToken !== activeLayoutToken) {
        return;
    }

    update((state) => ({
        ...state,
        graph,
        shapeSignature: topologyShapeSignature(snapshot)
    }));
};
```

However, `rebuild` is called from inside the `update` callback in `onMessage`:

```typescript
update((state) => {
    const nextSnapshot = applyTopologyEvent(state.snapshot, payload);
    if (!nextSnapshot) {
        return state;
    }
    void rebuild(nextSnapshot);  // fire-and-forget async
    return {
        ...state,
        snapshot: nextSnapshot,
        shapeSignature: topologyShapeSignature(nextSnapshot),
        error: null
    };
});
```

The `rebuild` call uses `void` (fire-and-forget), meaning the store update returns before the layout is computed. If multiple WebSocket messages arrive rapidly, each triggers a `rebuild` that competes with previous ones. While the `activeLayoutToken` check prevents stale results, it means that intermediate snapshots never get their layout applied. Only the last one in a burst wins.

Note that `shapeSignature` is now computed eagerly in both the `update` callback and the `rebuild` callback, which means the signature updates immediately while the graph layout lags behind.

This could cause visible jank: the `snapshot` and `shapeSignature` in the store are updated immediately (triggering sidebar/counter updates), but the `graph` lags behind until ELKjs finishes computing. During the interim, the sidebar and header show new agent counts while the canvas still shows the old layout.

**Suggested fix**: Debounce the `rebuild` call to batch rapid topology updates, or use a queue that processes layouts sequentially and always computes the latest snapshot.

## Triage

- Decision: `invalid`
- Notes:
  - The store intentionally uses a monotonic layout token so only the latest asynchronous layout result can win. That is not a race bug; it is the mechanism preventing stale ELK layouts from overwriting newer topology state.
  - The temporary state where `snapshot` updates before `graph` is recomputed is an expected tradeoff of async layout work, and I did not find a broken invariant or stale graph overwrite in the current implementation.
  - Debouncing or queueing layouts could be a future UX optimization, but this review comment does not point to a correctness defect that needs fixing in this batch.
  - Resolution: closed as a design/UX tradeoff rather than a correctness bug.
