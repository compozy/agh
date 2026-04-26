---
status: resolved
file: internal/network/audit.go
line: 279
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQk,comment:PRRC_kwDOR5y4QM67VX7C
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Prune stale presence keys or this cache will grow forever.**

`presence.lastSeen` only ever inserts/updates entries. In a long-lived daemon with peer/channel churn, every distinct greet tuple stays resident indefinitely even after it has aged past `presence.duration`, so this becomes an unbounded in-memory index.



<details>
<summary>Possible direction</summary>

```diff
 func (w *FileAuditWriter) shouldWriteTimelineMessage(entry store.NetworkMessageEntry) bool {
   if strings.TrimSpace(entry.Kind) != string(KindGreet) {
     return true
   }
   if w == nil || w.presence.duration <= 0 {
     return true
   }

   key := strings.Join([]string{
     strings.TrimSpace(entry.Direction),
     strings.TrimSpace(entry.Channel),
     strings.TrimSpace(entry.PeerFrom),
     strings.TrimSpace(entry.PeerTo),
   }, "\x00")

   at := entry.Timestamp.UTC()
   w.presence.mu.Lock()
   defer w.presence.mu.Unlock()

   if w.presence.lastSeen == nil {
     w.presence.lastSeen = make(map[string]time.Time)
   }
+  cutoff := at.Add(-w.presence.duration)
+  for existingKey, seenAt := range w.presence.lastSeen {
+    if seenAt.Before(cutoff) {
+      delete(w.presence.lastSeen, existingKey)
+    }
+  }

   lastSeen, ok := w.presence.lastSeen[key]
   w.presence.lastSeen[key] = at
   if !ok {
     return true
   }
   return at.Sub(lastSeen) > w.presence.duration
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (w *FileAuditWriter) shouldWriteTimelineMessage(entry store.NetworkMessageEntry) bool {
	if strings.TrimSpace(entry.Kind) != string(KindGreet) {
		return true
	}
	if w == nil || w.presence.duration <= 0 {
		return true
	}

	key := strings.Join([]string{
		strings.TrimSpace(entry.Direction),
		strings.TrimSpace(entry.Channel),
		strings.TrimSpace(entry.PeerFrom),
		strings.TrimSpace(entry.PeerTo),
	}, "\x00")

	at := entry.Timestamp.UTC()
	w.presence.mu.Lock()
	defer w.presence.mu.Unlock()

	if w.presence.lastSeen == nil {
		w.presence.lastSeen = make(map[string]time.Time)
	}
	cutoff := at.Add(-w.presence.duration)
	for existingKey, seenAt := range w.presence.lastSeen {
		if seenAt.Before(cutoff) {
			delete(w.presence.lastSeen, existingKey)
		}
	}

	lastSeen, ok := w.presence.lastSeen[key]
	w.presence.lastSeen[key] = at
	if !ok {
		return true
	}
	return at.Sub(lastSeen) > w.presence.duration
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/audit.go` around lines 251 - 279, shouldWriteTimelineMessage
currently only inserts/updates w.presence.lastSeen and never removes old keys,
causing an unbounded map growth; modify
FileAuditWriter.shouldWriteTimelineMessage (and/or the presence struct) to prune
stale entries from w.presence.lastSeen when holding the lock: iterate the map
and delete any entries where at.Now().UTC().Sub(t) > w.presence.duration (or
t.Add(w.presence.duration).Before(at)), and perform this pruning
opportunistically (e.g., each call that updates lastSeen or when the map exceeds
a threshold) to avoid locking cost spikes while ensuring entries older than
presence.duration are removed. Ensure you keep the existing logic that updates
lastSeen[key] = at and that pruning is protected by the same presence.mu lock.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `shouldWriteTimelineMessage()` updates `presence.lastSeen` but never removes keys that have already aged out of the configured presence window.
  - Root cause: the presence suppression cache is append/update only, so long-lived daemons accumulate stale `(direction, channel, peerFrom, peerTo)` entries forever.
  - Fix plan: opportunistically prune stale entries while holding `presence.mu` before updating the current key, then add a regression that asserts expired presence keys are evicted.
