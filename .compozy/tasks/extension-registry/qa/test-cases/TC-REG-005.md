# TC-REG-005: Config Loading With New Marketplace Fields Backward-Compatible

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Regression |
| **Estimated Time** | 2 min |
| **Module** | `internal/config/config.go` |
| **Changed In** | Task 04 — Config Changes |

## Objective

Validate that TOML configs without the new `[extensions.marketplace]` section still load correctly with defaults.

## Preconditions

- TOML config file without `[extensions.marketplace]` section (pre-feature config).

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Load config without `[extensions.marketplace]` | **Expected:** Config loads. Marketplace config uses zero values / defaults. |
| 2 | Load config with `[extensions.marketplace]` section | **Expected:** Marketplace fields populated from config. |
| 3 | Load config with only `[extensions]` but no `[extensions.marketplace]` | **Expected:** Extensions section present, marketplace defaults used. |

## Regression Risk

Medium — new nested TOML sections could break parsing if not optional.
