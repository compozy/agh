# AGH Autonomous Model Review

## Verdict

The corrected autonomy model fits AGH's current architecture well. The codebase already separates task creation from execution start, keeps sessions explicit, and uses the daemon as the composition root. The main missing piece is not a new orchestration layer; it is durable claim fencing and lease state on `task_runs`, plus a scheduler that only sweeps, not owns work.

## Existing Paths

- Task creation is already a pure persistence step: [`internal/task/manager.go:172-233`](</Users/pedronauck/Dev/compozy/agh/internal/task/manager.go:172>).
- Approval today only reconciles task status; it does not bootstrap coordinator behavior yet: [`internal/task/manager.go:533-596`](</Users/pedronauck/Dev/compozy/agh/internal/task/manager.go:533>).
- Run lifecycle is already split into enqueue, claim, start, attach, complete, fail, and cancel: [`internal/task/manager.go:1329-1685`](</Users/pedronauck/Dev/compozy/agh/internal/task/manager.go:1329>).
- The current run schema has ownership fields, but no claim token or lease fields yet: [`internal/store/globaldb/global_db.go:335-376`](</Users/pedronauck/Dev/compozy/agh/internal/store/globaldb/global_db.go:335>).
- The task model does not carry an `orchestration_required` flag, which matches the corrected model: [`internal/task/types.go:228-278`](</Users/pedronauck/Dev/compozy/agh/internal/task/types.go:228>).
- Manual session creation, resume, and direct prompting are already explicit and separate from task execution: [`internal/session/manager_lifecycle.go:19-30`](</Users/pedronauck/Dev/compozy/agh/internal/session/manager_lifecycle.go:19>), [`internal/session/manager_workspace.go:40-71`](</Users/pedronauck/Dev/compozy/agh/internal/session/manager_workspace.go:40>), [`internal/cli/session.go:35-88`](</Users/pedronauck/Dev/compozy/agh/internal/cli/session.go:35>).
- The API surface already exposes distinct task, task-run, and session verbs instead of a merged autonomy primitive: [`internal/api/httpapi/routes.go:64-180`](</Users/pedronauck/Dev/compozy/agh/internal/api/httpapi/routes.go:64>), [`internal/api/udsapi/routes.go:60-185`](</Users/pedronauck/Dev/compozy/agh/internal/api/udsapi/routes.go:60>).
- The CLI has explicit task-run `enqueue`, `claim`, `start`, `attach-session`, `complete`, `fail`, and `cancel` commands, but no `task next` pull verb yet: [`internal/cli/task.go:481-760`](</Users/pedronauck/Dev/compozy/agh/internal/cli/task.go:481>).
- The daemon already owns task boot recovery and session bridging; that is the right place for a thin sweep/notify/recovery scheduler: [`internal/daemon/task_runtime.go:93-305`](</Users/pedronauck/Dev/compozy/agh/internal/daemon/task_runtime.go:93>).
- Hooks and resources are already extensible, but the current taxonomy has no autonomy-specific families yet: [`internal/hooks/events.go:8-97`](</Users/pedronauck/Dev/compozy/agh/internal/hooks/events.go:8>), [`internal/hooks/introspection.go:46-220`](</Users/pedronauck/Dev/compozy/agh/internal/hooks/introspection.go:46>), [`internal/daemon/hooks_bridge.go:22-120`](</Users/pedronauck/Dev/compozy/agh/internal/daemon/hooks_bridge.go:22>), [`internal/daemon/hook_binding_resources.go:14-89`](</Users/pedronauck/Dev/compozy/agh/internal/daemon/hook_binding_resources.go:14>).

## Conflicts Or Gaps

- `task_runs` still lacks the atomic next-work contract (`ClaimNextRun`, claim token, lease until, heartbeat), so the corrected model cannot be safe until the store and task service own that state.
- The current approval path does not yet trigger coordinator spawn/start behavior; that trigger needs to be wired where task execution actually starts, not at task creation.
- The scheduler MVP should stay mechanical. If it becomes a direct claimant, AGH would end up with two ownership authorities and the model becomes harder to reason about.

## Recommended Implementation Ownership

- `internal/task`: own `ClaimNextRun`, lease extension, token validation, and task-run state transitions.
- `internal/store/globaldb`: add the `task_runs` claim/lease columns, indexes, and atomic SQL claim/recovery behavior.
- `internal/daemon`: host the scheduler as a daemon-owned sweep/notify/recovery goroutine and own coordinator spawn/bootstrap wiring.
- `internal/session`: keep manual create/resume/prompt behavior unchanged; only add lineage/bootstrap helpers if spawn needs them.
- `internal/api/contract`, `internal/api/httpapi`, `internal/api/udsapi`, `internal/cli`: expose explicit start/approve/claim surfaces; add the future `task next` pull verb only if the agent-facing flow needs it.
- `internal/hooks`, `internal/daemon/hooks_bridge.go`, `internal/daemon/hook_binding_resources.go`: add only the autonomy hook families that need durable external extension points.

## TechSpec / ADR Changes Before `cy-create-tasks`

- No major rewrite is needed. The spec and ADR set already point in the right direction.
- I would only tighten `_techspec.md` so it states unambiguously that `ClaimNextRun` is the only authoritative next-work primitive and that the scheduler never claims runs directly.
- If you want one more small clarification, make the start/approve trigger explicit at the execution boundary, not task creation. ADR-005, ADR-004, and ADR-010 already support that reading.

## Overengineering Warnings

- Do not introduce a durable scheduler queue or a separate scheduler-owned ownership model.
- Do not add a first-class workflow package or store for MVP; workflow correlation should stay metadata until there is a real entity behind it.
- Do not add new hook families or resource kinds for transient scheduler state.
- Do not add an `orchestration_required` task-creation flag.
- Do not split manual and autonomous work into separate queues or separate execution contracts.

