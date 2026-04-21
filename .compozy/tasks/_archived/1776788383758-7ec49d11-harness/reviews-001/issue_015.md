---
status: resolved
file: internal/daemon/harness_reentry_bridge.go
line: 223
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dM0,comment:PRRC_kwDOR5y4QM65IPEI
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Recovery ordering can violate FIFO after restart.**

The recovery pass sorts by `EndedAt` and then `run.ID`, but the real completion ordering signal is only fetched later via `latestDetachedTerminalSequence()`. If two detached runs share the same `EndedAt`, the bridge can dispatch the later completion first after boot because the queue starts draining before the earlier run is processed.

<details>
<summary>Suggested direction</summary>

```diff
- sort.SliceStable(runs, func(i, j int) bool {
-     if !runs[i].EndedAt.Equal(runs[j].EndedAt) {
-         return runs[i].EndedAt.Before(runs[j].EndedAt)
-     }
-     return runs[i].ID < runs[j].ID
- })
-
- for _, run := range runs {
-     metadata, ok, err := maybeDecodeDetachedHarnessRunMetadata(run.Metadata)
+ type recoveredRun struct {
+     run       taskpkg.Run
+     timestamp time.Time
+     sequence  int64
+ }
+
+ recovered := make([]recoveredRun, 0, len(runs))
+ for _, run := range runs {
+     metadata, ok, err := maybeDecodeDetachedHarnessRunMetadata(run.Metadata)
      if err != nil {
          return err
      }
      if !ok || detachedHarnessReentryProcessed(metadata.Reentry) {
          continue
      }
      sequence, timestamp, lookupErr := b.latestDetachedTerminalSequence(ctx, run.TaskID, run.ID)
      if lookupErr != nil {
          return lookupErr
      }
      if timestamp.IsZero() {
          timestamp = run.EndedAt
      }
-     if err := b.processTerminalRun(run.TaskID, run.ID, sequence, timestamp); err != nil {
+     recovered = append(recovered, recoveredRun{run: run, timestamp: timestamp, sequence: sequence})
+ }
+
+ sort.SliceStable(recovered, func(i, j int) bool {
+     if !recovered[i].timestamp.Equal(recovered[j].timestamp) {
+         return recovered[i].timestamp.Before(recovered[j].timestamp)
+     }
+     if recovered[i].sequence != recovered[j].sequence {
+         return recovered[i].sequence < recovered[j].sequence
+     }
+     return recovered[i].run.ID < recovered[j].run.ID
+ })
+
+ for _, item := range recovered {
+     if err := b.processTerminalRun(item.run.TaskID, item.run.ID, item.sequence, item.timestamp); err != nil {
          return err
      }
  }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
    type recoveredRun struct {
        run       taskpkg.Run
        timestamp time.Time
        sequence  int64
    }

    recovered := make([]recoveredRun, 0, len(runs))
    for _, run := range runs {
        metadata, ok, err := maybeDecodeDetachedHarnessRunMetadata(run.Metadata)
        if err != nil {
            return err
        }
        if !ok || detachedHarnessReentryProcessed(metadata.Reentry) {
            continue
        }
        sequence, timestamp, lookupErr := b.latestDetachedTerminalSequence(ctx, run.TaskID, run.ID)
        if lookupErr != nil {
            return lookupErr
        }
        if timestamp.IsZero() {
            timestamp = run.EndedAt
        }
        recovered = append(recovered, recoveredRun{run: run, timestamp: timestamp, sequence: sequence})
    }

    sort.SliceStable(recovered, func(i, j int) bool {
        if !recovered[i].timestamp.Equal(recovered[j].timestamp) {
            return recovered[i].timestamp.Before(recovered[j].timestamp)
        }
        if recovered[i].sequence != recovered[j].sequence {
            return recovered[i].sequence < recovered[j].sequence
        }
        return recovered[i].run.ID < recovered[j].run.ID
    })

    for _, item := range recovered {
        if err := b.processTerminalRun(item.run.TaskID, item.run.ID, item.sequence, item.timestamp); err != nil {
            return err
        }
    }
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/harness_reentry_bridge.go` around lines 199 - 223, The
sorting by EndedAt/ID can dispatch runs out of true completion order because the
actual completion sequence is provided by latestDetachedTerminalSequence; to
fix, obtain the terminal sequence for each run before ordering and sort runs by
(EndedAt, terminal sequence, ID) or primarily by terminal sequence when
available: call latestDetachedTerminalSequence(ctx, run.TaskID, run.ID) for each
run (skipping those with decode errors or
detachedHarnessReentryProcessed(metadata.Reentry)), store the returned sequence
and timestamp alongside the run, then sort the enriched entries using sequence
(fallback to EndedAt and then ID) and finally iterate calling processTerminalRun
with the pre-fetched sequence/timestamp to preserve FIFO after restart.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `recover` sorts runs by `EndedAt` and `run.ID` before consulting `latestDetachedTerminalSequence`, even though that later lookup is the durable ordering signal used by the wake queue.
  - If two runs share the same end timestamp, restart recovery can replay them in a different order from the live FIFO path.
  - I will prefetch recovery ordering metadata first, then sort recovered items by the same completion timestamp/sequence ordering used by the live wake queue.
