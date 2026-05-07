---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: pending
file: internal/modelcatalog/service.go
line: 127
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6tQ,comment:PRRC_kwDOR5y4QM6-6btD
---

# Issue 022: _⚠️ Potential issue_ | _🔴 Critical_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_ | _⚡ Quick win_

**Keep the refresh-flight entry alive until waiters are released, and make waiters cancelable.**

Two concurrency hazards are packed into this path: waiters block on `flight.done` without checking `ctx.Done()`, and the owner removes the map entry before closing `done`, which opens a window for a second refresh to start for the same provider while older waiters are still parked on the first flight. Pass `ctx` into `withRefreshFlight`, wait with `select`, and close `done` before deleting the map entry.
 

<details>
<summary>Suggested fix sketch</summary>

```diff
-	return s.withRefreshFlight(providerKey, scopeKey, func() ([]SourceStatus, error) {
+	return s.withRefreshFlight(ctx, providerKey, scopeKey, func() ([]SourceStatus, error) {
 		return s.refreshSources(ctx, sources, opts, now)
 	})
 }

 func (s *CatalogService) withRefreshFlight(
+	ctx context.Context,
 	providerID string,
 	scopeKey string,
 	fn func() ([]SourceStatus, error),
 ) ([]SourceStatus, error) {
 	for {
 		s.lockMu.Lock()
 		flight := s.refreshFlights[providerID]
 		if flight == nil {
 			flight = &refreshFlight{
 				scopeKey: scopeKey,
 				done:     make(chan struct{}),
 			}
 			s.refreshFlights[providerID] = flight
 			s.lockMu.Unlock()

 			flight.statuses, flight.err = fn()
 			s.lockMu.Lock()
-			delete(s.refreshFlights, providerID)
-			s.lockMu.Unlock()
 			close(flight.done)
+			delete(s.refreshFlights, providerID)
+			s.lockMu.Unlock()
 			return cloneSourceStatuses(flight.statuses), flight.err
 		}
 		s.lockMu.Unlock()
-		<-flight.done
+		select {
+		case <-flight.done:
+		case <-ctx.Done():
+			return nil, ctx.Err()
+		}
 		if flight.scopeKey == scopeKey {
 			return cloneSourceStatuses(flight.statuses), flight.err
 		}
 	}
 }
```
</details>

As per coding guidelines, "Use `context.Context` discipline: pass context through function calls and respect context cancellation in Go".


Also applies to: 350-377

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/modelcatalog/service.go` around lines 125 - 127, The current
refresh-flight flow doesn't respect context cancellation and removes the flight
entry before closing its done channel, causing waiters to block and a race that
can start a duplicate refresh; update the call site to pass ctx into
withRefreshFlight (replace s.withRefreshFlight(providerKey, scopeKey, ...) with
s.withRefreshFlight(ctx, providerKey, scopeKey, ...)), change withRefreshFlight
and its waiter logic to wait using select on ctx.Done() and flight.done so
waiters are cancelable, and ensure the owner closes flight.done before deleting
the flight map entry (close done then delete) so the flight stays alive until
all waiters are released.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
