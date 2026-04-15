# TC-FUNC-001: Bundle catalog lists all available bundles from installed extensions

**Priority:** P0 (Critical)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `Service.Catalog()`

## Objective

Validate that the bundle catalog correctly discovers and returns all bundle specs from all installed extensions, sorted by extension name and bundle name.

## Preconditions

- Two or more extensions installed, each with at least one bundle spec
- One extension with zero bundles (to verify it's excluded)
- Bundle service initialized with working extension lister and loader

## Test Steps

1. Install extension "ext-a" with bundles ["notify", "alerts"]
   **Expected:** Extension appears in registry

2. Install extension "ext-b" with bundle ["monitor"]
   **Expected:** Extension appears in registry

3. Install extension "ext-c" with zero bundles
   **Expected:** Extension appears in registry

4. Call `Service.Catalog(ctx)`
   **Expected:** Returns 3 CatalogEntry items (ext-a/alerts, ext-a/notify, ext-b/monitor). ext-c is excluded.

5. Verify sort order
   **Expected:** Sorted by (ExtensionName ASC, Bundle.Name ASC)

6. Verify each entry contains correct profile counts (JobCount, TriggerCount, BridgeCount)
   **Expected:** Profile summaries match manifest declarations

## Edge Cases

- Extension loader returns error for one extension → that extension is silently skipped, others still returned
- Extension loader returns nil extension → silently skipped
- Extension with empty name → trimmed and handled
- No extensions installed → returns empty slice (not nil)
