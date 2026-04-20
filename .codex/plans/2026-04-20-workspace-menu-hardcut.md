# Web Workspace Navigation Hard-Cut: Network / Tasks / Bridges / Jobs / Triggers / Knowledge / Skills

## Summary

- Remove the `/automation` app surface and replace it with two sibling first-level pages: `/jobs` and `/triggers`.
- Reorder the workspace sidebar to this exact sequence: `Network`, `Tasks`, `Bridges`, `Jobs`, `Triggers`, `Knowledge`, `Skills`.
- Treat this as a hard-cut frontend change: no redirect, no compatibility layer, no hidden `/automation` fallback.
- Fix the structural cause instead of re-skinning the current page: the existing automation route couples two operational domains behind tabs and a shared route-local state model.

## Implementation Changes

- Navigation
  - Update `web/src/components/app-sidebar.tsx` to remove `Automation`, insert `Jobs` and `Triggers`, and apply the final order exactly.
  - Keep `Network` singular in both label and path (`/network`).
  - Update sidebar tests to assert presence, order, links, and active states for `/network`, `/tasks`, `/bridges`, `/jobs`, `/triggers`, `/knowledge`, and `/skills`.
- Routes
  - Remove `web/src/routes/_app/automation.tsx`.
  - Add `web/src/routes/_app/jobs.tsx` and `web/src/routes/_app/triggers.tsx` as sibling first-level routes under `/_app`.
  - Regenerate `web/src/routeTree.gen.ts` through the project tooling; do not edit it by hand.
  - Update route-tree tests that currently expect `/automation` under `/_app` so they instead require `/jobs` and `/triggers`, and confirm `/automation` is gone.
- View-model / hooks
  - Refactor the current route hook into a shared, parameterized view-model that accepts `kind: "jobs" | "triggers"`.
  - Remove all route-level kind tab state (`activeTab`, `handleTabChange`, kind pills).
  - Make each new route load only its own list/detail/run data:
    - `/jobs` loads jobs, job detail, and job runs.
    - `/triggers` loads triggers, trigger detail, and trigger runs.
  - Preserve existing behaviors for scope filtering, search, selection, create/edit/delete, and enabled toggling.
  - Keep `trigger now` available only on `Jobs`.
- Page UI
  - Reuse the existing automation page shell pattern, but without the kind switcher.
  - Keep only the scope pills (`all/global/workspace`), split pane, and contextual CTA per page.
  - Use explicit page titles and CTA labels: `Jobs` + `Job`, `Triggers` + `Trigger`.
  - Update empty/loading/error copy so page-specific operational surfaces no longer refer to the generic “automation” page.
- Dependent surfaces
  - Update `settings/automation` to expose two operational links: `Open Jobs` -> `/jobs` and `Open Triggers` -> `/triggers`.
  - Keep `settings/automation` itself unchanged as the runtime configuration page.
  - Replace Storybook route stories, router stubs, and tests that navigate to `/automation` with `/jobs` and `/triggers`.
  - Update any internal fixtures/mocks that still advertise `/automation` as the operational link.

## Public APIs / Interfaces / Types

- SPA routes
  - Add `/jobs`.
  - Add `/triggers`.
  - Remove `/automation`.
  - Keep `/network` unchanged.
- Internal route hook contract
  - Replace the tabbed automation route model with a `kind`-parameterized view-model.
  - Remove `activeTab` and `handleTabChange` from the route-level API.
- Domain/API boundary
  - Keep the backend API under `/api/automation/*` unchanged.
  - Keep the shared frontend system namespace as `@/systems/automation`; only the route structure changes.

## Test Plan

- Sidebar
  - Assert the exact workspace nav order.
  - Assert links and active states for `Network`, `Tasks`, `Bridges`, `Jobs`, `Triggers`, `Knowledge`, and `Skills`.
  - Assert `Automation` is no longer rendered.
- Routes
  - `/jobs`: loading, initial error, empty state, scope filter, create/edit/delete, enabled toggle, and `trigger now`.
  - `/triggers`: loading, initial error, empty state, scope filter, create/edit/delete, and enabled toggle.
  - No route test depends on a kind-tab UI.
- Settings / Storybook / route tree
  - Settings deep-links to `/jobs` and `/triggers`.
  - Storybook router stubs recognize the new routes.
  - Route tree contains `/jobs` and `/triggers` and no longer contains `/automation`.

## Assumptions

- Hard-cut is approved: `/automation` is removed outright, with no redirect.
- `Network` remains singular in both label and URL.
- Scope is web-only; no backend API changes are required.
- The settings section keeps the `Automation` name because it configures runtime behavior, not the operational route IA.
