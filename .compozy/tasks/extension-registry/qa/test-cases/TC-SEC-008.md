# TC-SEC-008: Registry Source Configuration Validation

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Security |
| **Estimated Time** | 3 min |
| **Module** | `internal/config/config.go` |
| **OWASP** | A05:2021 — Security Misconfiguration |

## Objective

Validate that registry source configuration is properly validated, especially `base_url` values.

## Preconditions

- TOML config files with various `base_url` values.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Config with `base_url = "https://api.github.com"` | **Expected:** Accepted. |
| 2 | Config with `base_url = "http://localhost:8080"` | **Expected:** Accepted (valid for development). |
| 3 | Config with `base_url = ""` (empty) | **Expected:** Uses default for the registry type. |
| 4 | Config with `base_url = "not-a-url"` | **Expected:** Validation error on config load. |
| 5 | Config with `registry = "unknown"` | **Expected:** Error: unsupported registry type. |

## Edge Cases

- URL with trailing slash: should be normalized (strip or keep consistently).
- URL with embedded credentials: should be warned against or rejected.
- Config missing `[extensions.marketplace]` section entirely: should use defaults.
