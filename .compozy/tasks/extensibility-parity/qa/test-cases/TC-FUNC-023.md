# TC-FUNC-023: Automation projector Build does not mutate live scheduler

**Priority:** P0
**Type:** Functional
**Package:** internal/automation
**Related Tasks:** 10

## Objective

Validate that the automation projector follows the Build/Apply two-phase commit pattern. Calling Build must compute a new scheduler configuration from automation.job and automation.trigger resource records but must not mutate the live scheduler that is actively dispatching jobs. The live scheduler's job list, cron entries, and trigger registrations must remain unchanged until Apply is explicitly called. This prevents partially configured automations from firing during reconfiguration.

## Preconditions

- Automation projector is instantiated with an initial scheduler state (e.g., one existing cron job "daily-cleanup" running at midnight).
- The resource store contains the corresponding automation.job record for "daily-cleanup".
- New automation.job and automation.trigger records are prepared but not yet processed by the projector.

## Test Steps

1. Snapshot the live scheduler state: list all registered jobs and triggers.
   **Expected:** Returns exactly one job: "daily-cleanup" with its cron schedule. No triggers beyond what was initially configured.

2. Insert a new automation.job resource record "hourly-report" with a cron schedule of every hour.
   **Expected:** Record persisted in the resource store.

3. Insert a new automation.trigger resource record "on-session-end" that triggers a job when a session terminates.
   **Expected:** Record persisted in the resource store.

4. Call the automation projector's Build method.
   **Expected:** Build returns a plan/delta without error. The plan includes adding "hourly-report" and "on-session-end".

5. Immediately query the live scheduler state.
   **Expected:** The scheduler still has exactly one job: "daily-cleanup". "hourly-report" is NOT scheduled. "on-session-end" trigger is NOT registered. No cron entries were added.

6. Wait for the next minute boundary (or simulate a cron tick).
   **Expected:** Only "daily-cleanup" fires if its schedule matches. "hourly-report" does NOT fire because it is not in the live scheduler.

7. Call the automation projector's Apply method with the Build result.
   **Expected:** Apply completes without error. The live scheduler is atomically updated.

8. Query the live scheduler state after Apply.
   **Expected:** Three entries: "daily-cleanup" (unchanged), "hourly-report" (newly added), "on-session-end" trigger (newly added).

9. Simulate a session end event.
   **Expected:** The "on-session-end" trigger fires its associated job. This confirms the trigger is wired in the live scheduler.

## Edge Cases

- Build with conflicting job names (duplicate automation.job records with same name): Build returns an error, does not produce a partial plan.
- Apply called with a stale Build result (resources changed between Build and Apply): Apply should detect the staleness and either reject or re-Build (verify specified behavior).
- Removing an automation.job via resource delete, then Build + Apply: the job is removed from the live scheduler without affecting other jobs.
- Build with an automation.trigger referencing a non-existent job: Build returns a validation error.
- Scheduler under load during Apply: in-flight job executions from the old schedule complete normally; new schedule takes effect for subsequent ticks.
