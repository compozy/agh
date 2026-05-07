# TC-FUNC-004: Catalog Merge Determinism (Priority + Freshness + Source-ID Tie-Break)

**Priority:** P0
**Type:** Functional
**Module:** `internal/modelcatalog` merge
**Requirement:** TechSpec Proposed Design / Architectural Boundaries.
**Status:** Not Run

## Objective

Verify that the merge projection is deterministic and follows the documented priority order, freshness tie-break, and source-id tie-break, with lower-priority sources filling missing fields.

## Preconditions

- [ ] Catalog seeded via test harness with crafted source rows for one provider/model.
- [ ] All rows written through `internal/modelcatalog.Store.ReplaceSourceRows`.

## Test Steps

1. **Higher-priority source wins conflicting non-empty field.**
   - Seed: `config` (priority 120) `display_name="Config Name"`, `models_dev` (priority 50) `display_name="DevName"`.
   - **Expected:** Projected `display_name="Config Name"`.
2. **Lower-priority source fills missing field.**
   - Seed: `config` row sets only `default_reasoning_effort`; `models_dev` row sets `cost_input_per_million`.
   - **Expected:** Projected model exposes both fields.
3. **Freshness tie-break.**
   - Seed: two rows with identical priority but different `refreshed_at`.
   - **Expected:** Fresher row wins.
4. **Source-id tie-break.**
   - Seed: two rows with identical priority and `refreshed_at`.
   - **Expected:** Ascending `source_id` wins.
5. **Sources array sorted deterministically.**
   - **Expected:** `sources` ordered `(priority DESC, refreshed_at DESC, source_id ASC)`.
6. **Projection top-level sorted by `(provider_id ASC, model_id ASC)`.**
   - **Expected:** Stable across repeated calls.
7. **Availability state derivation.**
   - Seed: live row `available=true stale=false` + models_dev row.
   - **Expected:** `availability_state="available_live"`.
   - Replace: live row `available=true stale=true` → `available_stale`.
   - Replace: live row `available=false stale=true` → `unavailable_stale`.
   - Remove live/extension row → `unknown`.
8. **`models.dev` and `builtin` never elevate availability above `unknown`.**
   - Seed only `models_dev` + `builtin`.
   - **Expected:** `availability_state="unknown"` and `available=null`.

## Audit Coverage

- C6 task tree (Task 03).
- SI-5 (`models.dev` not authority), SI-13 (partial success).

## Pass Criteria

- Every assertion holds across two consecutive runs (determinism).

## Failure Criteria

- Any tie-break diverges from the documented order.
- `models.dev`/`builtin` ever yield `available=true` directly.
