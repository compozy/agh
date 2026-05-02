# Curl Installer for AGH

## Summary

Add a first-class Unix curl installer served from `https://agh.network/install.sh`, modeled after Hermes' user-facing install path but adapted to AGH's single-binary release model.

Root cause to fix: AGH currently has release artifacts and installation docs, but no stable one-line bootstrap URL. A docs-only curl snippet would be a workaround; the implementation must wire the installer through the site, release assets, CI, and docs so it stays real.

Chosen defaults:

- Canonical command: `curl -fsSL https://agh.network/install.sh | sh`
- Platforms: macOS and Linux only.
- Flow: install the `agh` binary, verify `agh version`, then run interactive `agh install` automatically when `/dev/tty` is available.
- Non-interactive mode: skip bootstrap and print the exact next command.
- Canonical script source: `packages/site/public/install.sh`, because the site is static-exported and `/install.sh` must be served as a real static asset.

## Public Interfaces

- Add installer flags:
  - `--version vX.Y.Z`: install a specific release tag instead of latest.
  - `--dir PATH`: install `agh` into a specific directory.
  - `--skip-bootstrap`: install binary only and do not run `agh install`.
  - `--dry-run`: resolve platform, asset URL, target path, and bootstrap behavior without downloading or writing files.
  - `-h` / `--help`: print usage.
- Add environment overrides:
  - `AGH_VERSION`
  - `AGH_INSTALL_DIR`
  - `AGH_RELEASE_REPO`, defaulting to `compozy/agh`
  - `AGH_SKIP_BOOTSTRAP=1`
- Use stable release archive asset names so `latest/download` works without brittle JSON parsing:
  - `agh_linux_x86_64.tar.gz`
  - `agh_linux_arm64.tar.gz`
  - `agh_darwin_x86_64.tar.gz`
  - `agh_darwin_arm64.tar.gz`
- No daemon API, OpenAPI contract, generated TypeScript contract, config schema, or CLI command contract changes.

## Implementation Changes

- Add `packages/site/public/install.sh` as a POSIX `sh` installer:
  - Detect `linux|darwin` and `x86_64|arm64`.
  - Download the matching tarball and `checksums.txt` from GitHub Releases.
  - Verify SHA-256 using `sha256sum` or `shasum -a 256`; fail if no checksum tool exists or the asset is absent from `checksums.txt`.
  - Extract in a temp directory with cleanup traps, install `agh` with executable permissions, and avoid partially replacing the target binary on failed downloads.
  - Default install dir: `/usr/local/bin` when writable, otherwise `$HOME/.local/bin`; never mutate shell rc files.
  - Warn if the final directory is not on `PATH`, but still run bootstrap via the absolute binary path.
  - Run `agh install` through `/dev/tty` when bootstrap is enabled and an interactive terminal exists; otherwise print `agh install` as the next step.
- Update site headers in `packages/site/public/_headers` for `/install.sh`:
  - `Content-Type: text/plain; charset=utf-8`
  - short cache TTL, e.g. `Cache-Control: public, max-age=300, must-revalidate`
- Update GoReleaser config:
  - Change archive naming to the stable asset names above.
  - Upload `packages/site/public/install.sh` as release asset `install.sh` via `release.extra_files`.
  - Align release/homepage metadata with the selected public install path and release repo.
  - Keep Go linker module paths unchanged unless a separate module-path hard cut is explicitly scoped.
- Update release notes templates:
  - Make curl install the primary release install path.
  - Keep checksum/cosign verification notes for release artifacts.
  - Remove or demote stale `go install` snippets that conflict with the current module/repo state.
- Update documentation and site UI:
  - `packages/site/content/runtime/core/getting-started/installation.mdx`: make curl the primary binary install path, document flags, explain bootstrap behavior, keep Homebrew/Linux package/source checkout as alternatives.
  - `packages/site/components/landing/install-section.tsx`: make the first tab the curl command, keep source/package alternatives only where accurate.
  - `packages/site/components/landing/__tests__/landing.test.tsx`: update tab labels, ids, keyboard expectations, and command assertions.
  - `packages/site/content/blog/posts/introducing-agh-the-first-agent-network-protocol.mdx`: replace prominent install snippets with the new curl command if the post is meant to remain current product guidance.
  - Do not hand-edit generated CLI reference pages under `packages/site/content/runtime/cli-reference/`.
- Add verification hooks:
  - Add a Mage target such as `InstallerCheck` that runs `sh -n packages/site/public/install.sh` and `sh packages/site/public/install.sh --dry-run --skip-bootstrap`.
  - Add `InstallerCheck` to `Verify()` so `make verify` and CI both exercise the installer.
  - Add a `Makefile` target if needed for direct local use.
  - Update `.github/workflows/release.yml` release validation to check the installer exists, passes syntax, and is included as a GoReleaser extra release file.
- Update release config tests:
  - Extend `internal/config/release_config_test.go` to assert stable archive naming, release extra file `install.sh`, checksum signing, SBOM coverage, and package targets remain present.

## Test Plan

- Installer local checks:
  - `sh -n packages/site/public/install.sh`
  - `sh packages/site/public/install.sh --dry-run --skip-bootstrap`
  - Optional manual dry-run with overrides: `AGH_VERSION=v0.0.0 AGH_INSTALL_DIR=/tmp/agh-bin sh packages/site/public/install.sh --dry-run --skip-bootstrap`
- Go/release config:
  - `go test ./internal/config -run TestGoReleaserConfigPreservesTrustArtifactsAndPackageTargets -count=1`
  - `go test -tags mage ./... -run 'Test.*Installer|TestShouldEnsureWebBundle|TestRunRaceEnabledGoCommand' -count=1` if Mage tests are updated.
- Site/docs:
  - `cd packages/site && bun run test -- landing`
  - `cd packages/site && bun run test`
  - `cd packages/site && bun run typecheck`
  - `cd packages/site && bun run build`
- Release validation:
  - Run the existing GoReleaser dry-run path used by the release workflow, or the closest local equivalent available with the repo's release tooling.
  - Confirm `dist/` contains the stable archive names and that `install.sh` is attached as an extra release artifact.
- Final gate:
  - `make verify`

## Assumptions

- `agh.network` is the canonical public install host, and the site remains static-exported, so `/install.sh` must be a static file in `packages/site/public`.
- The release asset repository for installer downloads should be `compozy/agh`, matching the current `origin` remote and site GitHub URL; the Go module path is not renamed in this task.
- Windows and PowerShell installers are out of scope for this pass.
- The installer must not silently edit shell startup files; it installs the binary and prints PATH guidance when needed.
- The existing dirty worktree includes user changes in release, docs, Go, site, and web files. Implementation must preserve those edits and only touch files required for this installer.
- GoReleaser supports `release.extra_files` for adding pre-existing release assets; implementation should follow the official GoReleaser release documentation: https://goreleaser.com/customization/publish/scm/
