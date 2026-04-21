# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Replace direct sidebar quick-create with an always-open session-creation dialog that exposes an explicit provider picker.
- Thread provider through the web create mutation path and surface the effective provider in session UI.
- Add a first-class inline resume-failure state when the persisted provider is no longer visible in the workspace.

## Important Decisions
- Built `useSessionCreateDialog` around `useWorkspace(activeWorkspaceId)` so provider options come straight from `WorkspaceDetailPayload.providers` instead of any ad-hoc discovery path, matching task_05 handoff notes.
- Reused `NativeSelect` for the agent and provider pickers to keep behavior testable in JSDOM and consistent with existing admin-style forms.
- Parsed the resume infrastructure error on the client with a single regex anchored on `validate agent "<name>" with provider "<name>" for session "<id>"` — which is the stable format emitted by `session.validateResumeAgent`. If the pattern does not match, the UI still renders a dedicated failure panel but without provider detail.
- Kept the sidebar pending spinner semantics: `isCreatingSession`, `pendingSessionAgentName`, and `pendingSessionWorkspaceId` now only reflect the in-flight dialog submission rather than click-triggered create.
- Left `handleNewSession(agentName)` as the sidebar entrypoint, but it now opens the dialog (prefilled) instead of creating. This preserves every existing caller and keeps scope tight.

## Learnings
- `createSession` already accepts `provider` via the generated contract, so no adapter or type change was required on the web create path beyond threading the field through the dialog submit.
- `useCreateSession` internally calls `useQueryClient`, so hook tests that exercise the full dialog need a direct mock of `@/systems/session/hooks/use-session-actions` in addition to the barrel mock.
- The existing `_app.test.tsx` mocks `@/systems/session`, so new barrel exports (`useSessionCreateDialog`, `SessionCreateDialog`, `SessionResumeFailure`) must be added to that mock to keep the layout tests green.

## Files / Surfaces
- `web/src/systems/workspace/{types.ts,index.ts,mocks/fixtures.ts}`
- `web/src/systems/session/{index.ts,hooks/use-session-create-dialog.ts,components/session-create-dialog.tsx,components/session-create-dialog.test.tsx,components/session-resume-failure.tsx,components/session-resume-failure.test.tsx,components/chat-header.tsx}`
- `web/src/hooks/routes/{use-app-layout.ts,use-app-layout.test.tsx,use-session-page-controls.ts}`
- `web/src/routes/_app.tsx`
- `web/src/routes/_app/session.$id.tsx`
- `web/src/routes/_app/-session.$id.test.tsx`
- `web/src/routes/-_app.test.tsx` (updated session barrel mock)

## Errors / Corrections
- Initial `use-app-layout.test.tsx` passed the real `useSessionCreateDialog` through `vi.importActual` without mocking `./use-session-actions`, which blew up with `No QueryClient set`. Fixed by mocking `@/systems/session/hooks/use-session-actions` directly in the test so the real dialog hook gets a stub create mutation.
- Typecheck flagged an unused `current` argument in the agent-change updater and a dead `ResumeMutateOptions` interface in the route test; both removed.

## Ready for Next Run
- `make verify` passes (5608 Go tests + 1415 web vitests, zero warnings, zero errors).
- Task_07 QA planning can rely on the dialog opening from every sidebar agent `+`, provider persistence on submit via the existing `createSession` contract, and the inline resume-failure panel surfacing missing-provider detail for task_08 regression coverage.
