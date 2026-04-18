# Tasks UI redesign plan

## Summary

- Convert `Tasks` from a modal-heavy flow into a route-driven workspace with a persistent master-detail shell.
- Keep the global app sidebar and the task list rail visible while create, edit, detail, and run detail render in the main content pane.
- Preserve the existing product language and tokens, but simplify hierarchy, reduce accent noise, strengthen empty states, and reuse shared UI primitives.

## Implementation changes

- Refactor `web/src/routes/_app/tasks.tsx` so the shell stays mounted for `/tasks`, `/tasks/new`, `/tasks/$id`, `/tasks/$id/edit`, and `/tasks/$id/runs/$runId`.
- Add route-based task editor screens for create and edit, using one shared editor component and keeping task actions in the main panel instead of dialogs.
- Rework task list, preview, detail, dashboard, inbox, and empty states around shared primitives such as `Button`, `Empty`, `Panel`, and `PillButton`.
- Calm the visual system inside `Tasks`: one accent at a time, quieter metadata, clearer section breaks, and consistent spacing and surfaces.

## Public interfaces / routes

- Add `/tasks/new` and `/tasks/$id/edit`.
- Keep `/tasks/$id` and `/tasks/$id/runs/$runId`, but render them inside the persistent `Tasks` shell.
- Reuse existing task APIs for create, update, publish, enqueue, approve, reject, archive, dismiss, and retry.

## Test plan

- Update routing tests to cover `/tasks`, `/tasks/new`, `/tasks/$id`, `/tasks/$id/edit`, and `/tasks/$id/runs/$runId` with the shell still mounted.
- Update editor tests to cover create, template switching, draft save, submit, and edit prefill.
- Update surface tests for the redesigned list, empty states, dashboard, and inbox views.
- Run focused web tests for the modified route and task components.

## Assumptions

- Keep the existing app-wide tokens, font setup, and icon set to avoid a cross-app rebrand in this task.
- Keep `dashboard` and `inbox` as top-level task modes; the persistent master-detail flow applies to list, create, edit, detail, and run detail.
- Move heavy form flows to routes, while lightweight actions remain inline.
