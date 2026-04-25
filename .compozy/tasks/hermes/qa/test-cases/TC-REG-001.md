## TC-REG-001: Release Packaging Trust Artifacts

**Priority:** P1 (High)
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that release hardening adds package targets while preserving checksum signing and SBOM coverage, and that local validation limitations around GoReleaser Pro are documented.

### Traceability

- Task: task_09, Environment, Extension, and Release Hardening.
- TechSpec: issue 59; Testing Approach release config integrity and package targets.
- Surfaces: `.goreleaser.yml`, `.github/workflows/release.yml`, `internal/config/release_config_test.go`, site installation/release docs.

### Preconditions

- Repository checkout contains the final `.goreleaser.yml` and release workflow.
- Local environment can run Go tests.
- GoReleaser Pro dry-run may not be available locally; record this as an expected validation boundary if it occurs.

### Test Steps

1. Run the release config integrity test.
   - **Expected:** Test asserts `checksums.txt` with sha256, cosign checksum signing, Homebrew cask target, nFPM `deb` and `rpm` targets, and archive/source/package SBOM entries.

2. Inspect `.goreleaser.yml` package sections.
   - **Expected:** `homebrew_casks` and `nfpms` reference the AGH binary build without removing existing archives, source, checksum, signing, or SBOM configuration.

3. Inspect the release workflow.
   - **Expected:** CI retains release trust setup and GoReleaser Pro execution path where required by repository configuration.

4. Attempt local GoReleaser check only if the available edition supports this repo.
   - **Expected:** If OSS rejects Pro configuration, record the expected limitation and rely on the Go integrity test plus CI Pro dry-run requirement.

5. Review installation docs.
   - **Expected:** Docs mention archive checksums, cosign checksum signatures, SBOMs, Homebrew cask, and Linux `.deb`/`.rpm` packages.

### Evidence To Capture

- `qa/logs/TC-REG-001/release-config-test.log`
- `qa/logs/TC-REG-001/goreleaser-inspection.log`
- `qa/logs/TC-REG-001/release-docs-review.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Missing SBOM artifact | `.goreleaser.yml` omits package/source/archive | Integrity test fails |
| Removed signing | No checksum signer | Integrity test fails |
| Local OSS GoReleaser | Pro config present | Limitation documented, CI Pro dry-run required |
| Docs omit package trust | Install docs missing verification | Docs issue filed |

### Related Test Cases

- TC-FUNC-002: Managed update/install lifecycle.
- TC-REG-002: Site documentation consistency.
