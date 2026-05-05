## TC-FUNC-003: Environment Repair And Extension Env Diagnostics

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 55 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify explicit `.env` inspection/repair behavior and extension `requires_env` diagnostics across CLI, API, generated web contracts, settings UI, and docs without leaking environment values.

### Traceability

- Task: task_09, Environment, Extension, and Release Hardening.
- TechSpec: issues 57 and 59; Testing Approach config/setup `.env` repair and `requires_env` behavior.
- Task 10 extra coverage: repair only bounded structured issues, refuse unsupported lines/symlinks/directories, no secret values; `requires_env` and `missing_env` through CLI JSON/human, HTTP/API, generated TypeScript, settings page, and docs.
- Surfaces: `internal/config/dotenv.go`, `internal/cli/config.go`, `internal/extension/manifest.go`, `internal/cli/extension.go`, `internal/api/contract/settings.go`, `internal/api/core`, web hooks/extensions settings, site config/env/extension docs.

### Preconditions

- Temp workspace with `.env` files covering valid entries, malformed multi-key values, non-ASCII/sanitized values, unsupported lines, symlinked `.env`, directory `.env`, and sentinel secret values.
- Test extension manifests include valid, invalid, duplicate, present, and missing `requires_env` values.
- Settings API and web fixtures are available for extension status.

### Test Steps

1. Run `agh config validate --repair-env` against a bounded malformed structured `.env`.
   - **Expected:** Command repairs only supported structured entries, writes safely through a temp file, reports diagnostics, and never prints secret values.

2. Run repair against unsupported `.env` content, symlink `.env`, and directory `.env`.
   - **Expected:** Command refuses to rewrite, reports a clear diagnostic, and leaves user-owned content unchanged.

3. Load extension manifests with valid, invalid, and duplicate `requires_env`.
   - **Expected:** Valid names are accepted; invalid or duplicate names fail validation with path-specific diagnostics.

4. Run extension list/status in human, JSON, and toon modes with missing environment variables.
   - **Expected:** Output includes `requires_env` and `missing_env` names only, and no environment values.

5. Query settings/API payloads for the same extension.
   - **Expected:** API payloads expose `requires_env` and `missing_env` arrays consistently with CLI and contain no values.

6. Validate web settings hooks/extensions surface.
   - **Expected:** Generated TypeScript contract includes env fields; settings page renders missing env names as diagnostics without exposing values.

7. Review site docs.
   - **Expected:** Config validate/check, environment variables, extension install/status, and installation docs describe repair/env requirements accurately.

### Evidence To Capture

- `qa/logs/TC-FUNC-003/go-test-env-extension.log`
- `qa/logs/TC-FUNC-003/config-repair.json`
- `qa/logs/TC-FUNC-003/extension-status.json`
- `qa/logs/TC-FUNC-003/settings-extension.json`
- `qa/screenshots/TC-FUNC-003/settings-missing-env-desktop.png` if browser validation is executed

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Symlink `.env` | `.env -> outside` | Refuse rewrite |
| Directory `.env` | `.env/` directory | Refuse rewrite |
| Duplicate env requirement | `["TOKEN", "TOKEN"]` | Manifest validation error |
| Secret value present | `TOKEN=secret` | Name may appear, value never appears |

### Related Test Cases

- TC-UI-002: Web settings redaction.
- TC-REG-002: Site env/extension docs.
