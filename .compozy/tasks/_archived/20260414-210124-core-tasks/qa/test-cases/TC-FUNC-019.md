## TC-FUNC-019: Invalid transition (queued to running, skipping claim)

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that the run lifecycle state machine enforces valid transitions. Specifically, attempting to start (transition to "running") a run that is still "queued" (skipping the "claimed" step) must be rejected with ErrInvalidStatusTransition. All invalid transition paths are covered.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing task with a queued run (status="queued")
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Attempt to start a queued run (skip claim)**
   - Call StartRun(ctx, runID, StartRun{}, actor)
   - **Expected:** Error returned; `errors.Is(err, ErrInvalidStatusTransition)` == true

2. **Verify run status is unchanged**
   - Read the run from store
   - **Expected:** Status still == "queued"

3. **Attempt to complete a queued run (skip claim+start)**
   - Call CompleteRun(ctx, runID, RunResult{}, actor)
   - **Expected:** Error returned; ErrInvalidStatusTransition

4. **Attempt to fail a queued run (skip claim+start)**
   - Call FailRun(ctx, runID, RunFailure{Error: "forced"}, actor)
   - **Expected:** Error returned; ErrInvalidStatusTransition

5. **Attempt to complete a claimed run (skip start)**
   - Claim the run first, then call CompleteRun
   - **Expected:** Error returned; ErrInvalidStatusTransition (must be "running" to complete)

6. **Verify valid transition path works: queued -> claimed -> running -> completed**
   - Create new run, claim it, start it, complete it
   - **Expected:** All transitions succeed in order

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| queued -> running | StartRun on queued | ErrInvalidStatusTransition |
| queued -> completed | CompleteRun on queued | ErrInvalidStatusTransition |
| queued -> failed | FailRun on queued | ErrInvalidStatusTransition |
| claimed -> completed | CompleteRun on claimed | ErrInvalidStatusTransition |
| completed -> running | StartRun on completed | ErrInvalidStatusTransition |
| failed -> running | StartRun on failed | ErrInvalidStatusTransition |
| cancelled -> claimed | ClaimRun on cancelled | ErrInvalidStatusTransition |
| running -> claimed | ClaimRun on running | ErrInvalidStatusTransition |

---

### Related Test Cases
- TC-FUNC-014: Enqueue run on ready task
- TC-FUNC-015: Claim queued run
- TC-FUNC-016: Start claimed run
- TC-FUNC-017: Complete running run with result
