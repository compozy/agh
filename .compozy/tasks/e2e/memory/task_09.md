# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Add the browser operator-journey proof for workspace onboarding/selection through session lifecycle in the daemon-served SPA, using the shared Playwright harness from task_08.
- Cover visible session behavior only: onboarding, session creation, prompt submission, streaming visibility, approval UI, stop/resume, and reload hydration.

## Important Decisions

- Extend the shared browser runtime fixture with configurable seeding rather than launching bespoke daemons inside the task_09 spec.
- Use ACP fixture-backed mock agents for deterministic browser session behavior, registered into the isolated browser runtime home before daemon startup.
- Add route-state artifact capture and selector helpers in the shared browser fixture layer so later browser tasks can reuse them.
- Keep the operator journey assertions on shipped browser surfaces, but allow one direct daemon/runtime reproduction path when browser artifacts reveal a transport-level failure that is otherwise opaque from the UI.
- Fix the underlying browser session path instead of weakening the Playwright scenario: resume coverage depends on truthful ACP `loadSession` capabilities and AI SDK-compatible finish chunks.

## Learnings

- The current browser harness only boots an empty daemon; without pre-launch agent seeding there is no real session lifecycle to drive from the UI after onboarding.
- Most session-shell surfaces already expose stable `data-testid` hooks; onboarding buttons are the main missing automation seam.
- The browser lifecycle fixture's mock driver already implemented `session/load`; the resume 500 came from the initialize capability drift that advertised `loadSession: false`.
- The browser chat surface rejects streamed completion chunks that end with `stopReason`; the HTTP prompt stream has to send `finishReason` for the AI SDK-backed session UI to accept the completion.
- The browser runtime workspace resolve helper must post `{ path: "<root>" }`; `{ root_dir: ... }` does not match the shipped resolve-workspace API contract.

## Files / Surfaces

- `web/e2e/fixtures/runtime.ts`
- `web/e2e/fixtures/runtime-helpers.ts`
- `web/e2e/fixtures/runtime-seed.ts`
- `web/e2e/fixtures/artifacts.ts`
- `web/e2e/fixtures/browser-artifact-session.ts`
- `web/e2e/fixtures/test.ts`
- `web/e2e/session-onboarding.spec.ts`
- `web/src/systems/workspace/components/workspace-setup.tsx`
- `web/src/routes/_app.tsx`
- `web/src/routes/_app/session.$id.tsx`
- `web/src/systems/session/components/permission-prompt.tsx`
- `internal/api/httpapi/prompt.go`
- `internal/api/httpapi/handlers_test.go`
- `internal/testutil/acpmock/driver/dist/index.js`
- `internal/testutil/acpmock/driver_test.go`

## Errors / Corrections

- Pre-existing unrelated diff in `.compozy/tasks/e2e/_meta.md`; leave untouched.
- Corrected browser resume failure: mock ACP initialize capabilities now advertise `loadSession: true` to match the implemented `session/load` path.
- Corrected browser streaming failure: HTTP prompt finish chunks now emit `finishReason` instead of `stopReason`.
- Corrected workspace seeding request shape: browser runtime resolve helper now sends `{ path: rootDir }`.

## Ready for Next Run

- Task implementation, verification, and the local code commit are complete (`efc451ba`, `test: add browser session lifecycle flow`).
- Later browser tasks can reuse the seeded runtime helpers, selector helpers, route-state artifact capture, and the browser session lifecycle fixture without adding route-specific daemon boot logic.
