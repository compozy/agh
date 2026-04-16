# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Cut over `hook.binding` to the shared resource runtime and make the projected hook snapshot authoritative for dispatch.
- Close the shipped session-runtime gaps for `tool.*` and `permission.*`.
- Remove the legacy hook-binding authority path in the same tranche after the resource-backed runtime is wired.

## Important Decisions

- Reuse `hookspkg.HookDecl` normalization/matcher/ordering behavior behind a typed `hook.binding` codec and projector rather than introducing a second hook-binding spec model.
- Use the approved PRD/TechSpec as the design authority for this implementation task; no extra brainstorming approval loop is needed before coding.
- Keep `internal/hooks` resource-agnostic: the daemon owns `hook.binding` resource codecs/stores/projectors and ACP event translation, while hooks only expose the build/apply seam.
- Preserve a direct build/apply fallback only when no shared resource kernel exists; the resource-backed projector is authoritative for migrated bindings whenever the kernel/codecs are available.
- Widen tool and permission matchers to honor `agent_name`, `workspace_id`, and `workspace_root` so migrated bindings can scope those events the same way as other hook families.

## Learnings

- `internal/hooks/agent_event.go` is currently a no-op, which is the concrete reason the real session notifier flow does not dispatch `tool.*` or `permission.*` hooks today.
- `internal/daemon/boot.go` still feeds `hooks.Rebuild()` from declaration providers (`config`, `agent`, `skill`, `extension`) instead of the shared resource runtime.
- The default daemon reconcile driver currently starts with no projector registrations, so this task must wire the first real family into that runtime.
- The session notifier only had the session ID in its base interface, so wiring `tool.*` and `permission.*` end to end required a narrow optional notifier extension that carries the active `*session.Session` for agent-event dispatch.
- The repo-wide `make verify` gate also surfaced unrelated-but-blocking lint debt in `internal/session/resume_repair.go`, so the final verified tree includes those constant cleanups in addition to the hook-runtime cutover.
- Daemon unit coverage needed explicit tests around boot helper seams and notifier forwarding to reach the project floor after the new hook-resource runtime landed.

## Files / Surfaces

- `internal/hooks/`
- `internal/session/`
- `internal/daemon/{boot.go,daemon.go,extensions.go,hooks_bridge.go,hook_agent_events.go,hook_binding_resources.go,hook_bindings.go}`
- `internal/extension/{host_api.go,host_api_resources.go}`
- `internal/api/core/resources.go`
- `internal/resources/`

## Errors / Corrections

- `make verify` initially failed on daemon/session lint issues (`funlen`, `revive`, `unused`, `goconst`); corrected by splitting boot helpers, renaming the session-aware notifier interface, cleaning the agent-event helper signature, removing dead test state, and extracting repeated stop-detail constants.
- Daemon unit coverage initially stayed below the required floor; corrected with targeted tests around hook agent-event helpers, resource-backed hook publication, daemon boot helper seams, notifier forwarding, and extension-runtime attachment behavior.

## Ready for Next Run

- Implementation and verification are complete.
- Fresh evidence:
  - `go test ./internal/hooks ./internal/session ./internal/daemon ./internal/extension ./internal/api/core`
  - `go test -tags integration ./internal/hooks ./internal/daemon`
  - `go test -cover ./internal/hooks ./internal/session ./internal/daemon` -> `81.5% / 80.9% / 80.0%`
  - `make verify`
