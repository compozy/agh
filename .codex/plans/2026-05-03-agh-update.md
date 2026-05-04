# GitHub-First Self-Update for `agh update`

## Summary

- Replace the current advisory `agh update` with a real self-update flow for direct-binary installs on macOS and Linux.
- Keep GitHub Releases as the canonical distribution source in v1. Do not add a Cloudflare Worker/R2 manifest layer in this change.
- Hold the security bar at or above the current installer: the updater must verify signed release provenance in-process with Sigstore, then verify the archive checksum before any swap.
- Scope v1 to the latest published non-prerelease release only. No channels, prerelease opt-in, or explicit version targeting in this first cut.

## Public Interfaces

- CLI:
  - `agh update` applies the latest stable release when the install method supports self-update.
  - `agh update --check` performs a status-only check and never mutates files.
  - Text/JSON output exposes `status`, `install_method`, `managed`, `current_version`, `latest_version`, `release_url`, `recommendation`, `daemon_restarted`, and `message`.
  - `status` values are fixed: `current`, `available`, `updated`, `deferred`, `unsupported`, `failed`.
- HTTP/UDS:
  - Add a read-only endpoint `GET /api/settings/update`.
  - Response shape is fixed: `supported`, `managed`, `install_method`, `current_version`, `latest_version`, `available`, `status`, `recommendation`, `release_url`, `checked_at`, `last_error`.
  - This endpoint is for web and agent visibility only in v1. It does not trigger installation.
- Release assets:
  - Publish `checksums.txt.sigstore.json` as the machine-consumable provenance artifact for `checksums.txt`.
  - Keep `checksums.txt` as the checksum catalog.
  - `install.sh` and the Go updater both verify the same `checksums.txt` + `checksums.txt.sigstore.json` pair.

## Implementation Changes

- Add a dedicated `internal/update` package for self-update logic.
- Detect install method by executable path, with `AGH_MANAGED` as an explicit override.
- Managed methods in v1: Homebrew, `apt`/`deb`, `dnf`/`rpm`, Scoop, and `go install`.
- Self-updatable method in v1: direct binary install on macOS and Linux.
- Windows direct-binary installs return `unsupported` with deterministic manual guidance in v1.
- Resolve releases from `compozy/agh` GitHub Releases, always choosing the latest non-draft, non-prerelease release.
- Persist update-check cache under AGH home at `cache/update-state.json` with a 24-hour TTL.
- Download archive, `checksums.txt`, and `checksums.txt.sigstore.json`.
- Verify Sigstore provenance with identity regexp `^https://github\.com/compozy/agh/\.github/workflows/release\.yml@refs/tags/v[0-9][A-Za-z0-9._-]*$` and issuer `https://token.actions.githubusercontent.com`.
- Verify the archive checksum from `checksums.txt` after provenance verification.
- If the daemon is running, atomically swap the binary and then trigger the existing daemon restart flow; on restart failure or timeout, restore the backup binary and perform one best-effort daemon start recovery.
- Add a “Software update” surface to General Settings that reads `GET /api/settings/update`.
- Do not add a web-triggered update button in v1.
- Rewrite `install.sh` to verify `checksums.txt.sigstore.json` instead of downloading separate `.sig` and `.pem` files.
- Extend the release pipeline so the release publishes `checksums.txt.sigstore.json`.

## Test Plan

- CLI and updater unit coverage for install-method detection, stable-only selection, `dev` build refusal, deferred/unsupported output, asset resolution, cache TTL, provenance verification, checksum mismatch, missing assets, corrupt archives, and network failures.
- Integration coverage for direct-binary update with daemon stopped, direct-binary update with daemon running and successful relaunch, restart failure rollback and recovery, and no-update behavior.
- API and web coverage for `GET /api/settings/update`, generated types, adapter/query/store behavior, and the General Settings update panel states.
- Site and release coverage for installer contract tests, release asset validation, and final `make verify`.

## Assumptions

- The reusable precedent from `../looper` is its client-side updater architecture and install-method detection, not Cloudflare distribution infrastructure.
- v1 intentionally omits `--channel`, `--tag`, explicit version targeting, and web-triggered binary mutation.
- v1 limits self-mutation to direct-binary macOS/Linux installs; other install methods remain deterministic guidance only.
