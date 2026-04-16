# TC-INT-002: HTTP GET /api/bundles/catalog returns extension bundles

**Priority:** P1 (High)
**Type:** Integration
**Component:** `internal/api/core/bundles.go` — `ListBundleCatalog`

## Objective

Validate the catalog endpoint returns all available bundles with correct profile summaries.

## Preconditions

- HTTP server with extensions installed
- At least one extension with bundles

## Test Steps

1. GET `/api/bundles/catalog`
   **Expected:** HTTP 200, response body contains `bundles` array

2. Verify each catalog entry has: extension_name, bundle_name, description, profiles[]
   **Expected:** All fields populated per contract.BundleCatalogPayload

3. Verify profile summaries include: name, description, primary_channel, channels[], job_count, trigger_count, bridge_count
   **Expected:** Counts match actual profile contents

4. GET when no extensions have bundles
   **Expected:** HTTP 200, `bundles: []` (empty array, not null)

5. GET when bundle service is nil
   **Expected:** HTTP 503

## Edge Cases

- Extension loader fails for one extension → that extension skipped, others returned
- Very large catalog (100+ bundles) → all returned, sorted
