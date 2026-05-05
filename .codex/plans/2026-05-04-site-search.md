# Restore Site Search Across Home, Docs, and Blog

## Summary

- Keep the current custom headers (`HomeHeader` and `DocsHeader`) and preserve their Fumadocs `searchTrigger` slot wiring; the visible trigger is already present in the current repo state and does not need a bespoke redesign.
- Fix the real breakage by aligning the site with standard Next.js runtime search instead of static export: remove `output: "export"` from `packages/site/next.config.mjs`, keep the site on regular Next output, and switch `/api/search` back to query-driven `GET`.
- Expand the search index from Runtime + AGH Network to Runtime + AGH Network + Blog + Changelog, with blog/changelog indexed by title, description/summary, headings, and excerpt-style text.

## Implementation Changes

- Update `packages/site/next.config.mjs`:
  - Remove `output: "export"` and the deterministic static-export build-id dependency.
  - Keep current canonical URL behavior and `trailingSlash` unless a touched test proves it must change.
- Update `packages/site/app/layout.tsx`:
  - Make search configuration explicit in `RootProvider` instead of relying on defaults.
  - Keep the default Fumadocs dialog in fetch mode, pointed at `/api/search`, so both home and docs use the same provider-backed search flow.
- Update `packages/site/app/api/search/route.ts`:
  - Export the live `GET` handler from `createSearchAPI("advanced", ...)`, not `staticGET`.
  - Preserve Runtime and Protocol indexing.
  - Add Blog and Changelog entries built from Velite data:
    - Blog posts: `title`, `description`, `excerpt`, `toc` headings, `permalink`, breadcrumb `["Blog"]`, tag `Blog`.
    - Changelog releases: `version`, `summary`, `added/changed/fixed/breaking` bullet text, URL anchored into `/changelog`, breadcrumb `["Changelog"]`, tag `Changelog`.
- Add a small search-index builder/helper near the existing site data layer rather than inlining the whole mapping inside the route.
  - Runtime/Protocol remain sourced from Fumadocs loaders.
  - Blog/Changelog are sourced from `allPosts()` / `allReleases()` so the index follows the generated public content model already used by the site.
- Remove static-export-only test/contracts that become invalid after the hosting change.
  - Rewrite the static export determinism test into a runtime-search config test, or replace it with a test that asserts the site is not pinned to `output: "export"` anymore.
  - Keep `_headers` and other public-route contracts only where they still apply to the deployed Next app.

## Public APIs / Interfaces

- `/api/search` changes behavior:
  - Before: returns exported search DB payload suitable for static-client download.
  - After: returns live search results for `?query=...`, matching the Fumadocs default fetch client.
- Search result scope changes:
  - Before: Runtime + AGH Network only.
  - After: Runtime + AGH Network + Blog + Changelog.
- No new visible header API is introduced.
  - The current Fumadocs `searchTrigger` slot remains the single source of search UI in both the home shell and docs shell.

## Test Plan

- Update route-level tests to call `/api/search?query=...` and assert:
  - response shape is a result list, not exported DB JSON;
  - a Runtime/Protocol query returns docs results;
  - a Blog/Changelog query returns content from the new Velite-backed index;
  - tag/breadcrumb metadata stays stable where used.
- Add a provider/config regression test that asserts the root layout explicitly enables search against `/api/search`.
- Keep existing home/docs header tests and extend only if needed to prove the trigger remains wired after provider changes.
- Run targeted verification in `packages/site`:
  - `bun run typecheck`
  - `bun run test`
  - `bun run build`
- Run a manual smoke pass in dev:
  - open `/` and `/runtime`;
  - trigger search from both headers;
  - search for one docs term and one blog term;
  - confirm result navigation works.
- Run broader verification after the site-local checks; if unrelated failures remain from concurrent work or pre-existing non-search issues, report them separately and do not block this task on those failures per the user’s instruction.

## Assumptions

- The current header UI is acceptable; the requirement is to restore working Fumadocs search, not redesign the header.
- Blog indexing depth is limited to title + description + excerpt + headings; no full-body MDX parsing is added in this change.
- Changelog content is searchable via version + summary + list items, not by compiling the full rendered MDX body into search data.
- Trailing-slash and canonical URL conventions remain unchanged unless touched tests prove a necessary adjustment.
