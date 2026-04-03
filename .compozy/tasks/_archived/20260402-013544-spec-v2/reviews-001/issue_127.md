---
status: resolved
file: internal/dashboard/events.go
line: 98
severity: medium
author: claude-reviewer
---

# Issue 127: TopologyBroadcaster.Publish drop-and-retry can silently lose events under contention



## Review Comment

The `TopologyBroadcaster.Publish` method implements backpressure by dropping the oldest event and retrying:

```go
for _, stream := range b.subscribers {
    select {
    case stream <- event:
    default:
        select {
        case <-stream:   // drop oldest
        default:
        }
        select {
        case stream <- event:  // retry send
        default:              // silently drop if still full
        }
    }
}
```

There are two issues with this pattern:

1. **Race between subscribers:** The drain (`<-stream`) and retry (`stream <- event`) are not atomic. Between the drain and the retry, another goroutine publishing concurrently (since `Publish` holds the mutex, this is only a concern if multiple publishers exist -- currently it seems single-publisher, so this may be acceptable) or the consumer reading could change the channel state.

2. **Silent event loss:** If after draining one message the channel is still full (possible with buffer size 1 when the consumer hasn't read yet), the event is silently dropped with no logging or metric. The consumer receives no indication that events were lost. While the `wsClient` has `frames_dropped` control messages, this drop happens in the broadcaster layer before the event reaches any client.

**Suggested fix:** Log or count when events are dropped at the broadcaster level so operators have visibility into backpressure:

```go
default:
    select {
    case <-stream:
    default:
    }
    select {
    case stream <- event:
    default:
        // TODO: emit metric or log for dropped topology event
    }
}
```

## Triage

- Decision: `invalid`
- Notes: `TopologyBroadcaster.Publish` already serializes publishers under `b.mu`, so the race concern described in the review does not apply to the current implementation. The remaining suggestion is observability for intentional drop-on-overflow behavior; that is a product/telemetry enhancement, not a correctness defect with a scoped root-cause fix in this batch.
