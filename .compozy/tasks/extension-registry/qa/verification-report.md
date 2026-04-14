VERIFICATION REPORT
-------------------
Claim: The grouped registry, adapter, installer, and security case matrix commands (`G7` through `G11`) passed on the current tree.
Command: `go test ./internal/registry -count=1 -run 'TestMultiRegistrySearchQueriesSourcesConcurrently|TestMultiRegistrySearchReturnsHealthyResultsOnPartialFailure|TestMultiRegistrySearchSkipsNonSearchableSources|TestMultiRegistryInfoResolvesHighestPrioritySource|TestMultiRegistryDownloadDelegatesToResolvedSource|TestMultiRegistryCheckUpdate|TestVersionIsNewer|TestMultiRegistryCloseClosesAllSources'`; `go test ./internal/registry -count=1 -run 'TestInstallerInstallExtensionArchiveReturnsChecksum|TestInstallerInstallRejectsCompressedArchiveOverLimit|TestInstallerInstallRejectsDecompressedArchiveOverLimit|TestExtractArchive_ValidArchiveProducesDirectoryStructure|TestExtractArchive_EnforcesLimitsAndRejectsUnsafeEntries|TestInstallerInstallBlocksCriticalVerificationContent|TestMoveInstalledDir|TestInstallerCleansStaleTempDirs'`; `go test ./internal/registry -count=1 -run 'TestExtractArchiveStripsSpecialPermissionBits|TestManifestPathAtRootRejectsSymlinkedManifest|TestCleanArchiveEntryPath|TestPathWithinRoot|TestInstallerInstallRejectsUnexpectedContentType'`; `go test ./internal/registry/clawhub -count=1 -run 'TestClientSearchParsesListingsAndLimit|TestClientSearchEmptyResultsReturnsEmptySlice|TestClientDownloadUsesLatestEndpointWhenVersionEmpty|TestClientDownloadUsesVersionedEndpointWhenVersionSpecified|TestClientRetriesHTTP500WithBackoff|TestClientSearchReturnsEmptyForExtensionFilter'`; `go test ./internal/registry/github -count=1 -run 'TestClientInfoFetchesLatestAndVersions|TestClientDownloadSingleTarballAsset|TestClientDownloadMultipleAssetsRequiresSelection|TestClientDownloadSelectsRequestedAsset|TestClientDownloadFallsBackToSourceArchive|TestClientFetchRequestedReleaseByTag|TestClientFetchRequestedReleaseRejectsPrerelease|TestClientRateLimitExceeded|TestClientUsesGitHubToken|TestCheckRateLimitWarnsWithoutFailing'`
Executed: 2026-04-14T19:24:55Z
Exit code: 0
Output summary: `internal/registry`, `internal/registry/clawhub`, and `internal/registry/github` all passed with the targeted functional, integration-style adapter, and security coverage commands used by the case matrix.
Warnings: none
Errors: none
Verdict: PASS

Claim: The grouped CLI, skill migration, config, schema, and capability case matrix commands (`G12` through `G20`) passed on the current tree.
Command: `go test ./internal/cli -count=1 -run 'TestExtensionSearchCommandUsesSearchableRegistrySources|TestExtensionSearchCommandAppliesSourceFilter|TestExtensionSearchCommandRejectsNonPositiveLimit|TestExtensionInstallCommandInstallsMarketplaceExtensionAndPrintsRestartMessage|TestExtensionInstallCommandPassesAssetToRegistryDownload|TestExtensionInstallAndRemoveOfflinePreservesSourceDirectory|TestExtensionRemoveCommandDeletesDirectoryAndRegistryRecord|TestExtensionRemoveCommandReturnsClearErrorForMissingExtension|TestRemoveInstalledExtensionRollsBackRegistryOnCommitFailure|TestExtensionUpdateCommandCheckOnlyShowsAvailableUpdatesWithoutDownloading|TestExtensionUpdateCommandReportsAlreadyUpToDate|TestExtensionUpdateCommandReinstallsNewerVersion|TestExtensionUpdateCommandAllUpdatesMarketplaceExtensions'`; `go test -tags integration ./internal/cli -count=1 -run 'TestExtensionSearchCommandIntegrationReturnsRegistryListings|TestExtensionInstallCommandIntegrationCreatesManagedInstallAndRegistryRecord|TestExtensionUpdateAndRemoveIntegration|TestExtensionRemoveMissingIntegrationReturnsClearError'`; `go test ./internal/cli -count=1 -run 'TestSkillSearchCommandPassesLimitAndRendersTable|TestSkillSearchCommandRejectsNonPositiveLimit|TestSkillInstallCommandInstallsMarketplaceSkill|TestSkillUpdateCommandCheckOnlyReportsUpdateWithoutDownloading|TestSkillUpdateCommandAllUpdatesMarketplaceSkills|TestSkillUpdateCommandReportsAlreadyUpToDate|TestSkillRemoveCommandDeletesMarketplaceSkillDirectory|TestSkillRemoveCommandRefusesNonMarketplaceSkill'`; `go test -tags integration ./internal/cli -count=1 -run 'TestSkillSearchInstallListRemoveIntegrationFlow|TestSkillInstallCommandIntegrationCreatesSkillDirectoryAndSidecar|TestSkillInstallAndRemoveIntegrationRefreshesRegistry|TestSkillInstallCommandIntegrationWritesMatchingHash|TestSkillInstallCommandIntegrationReplacesExistingSkillDirectory'`; `go test ./internal/store/globaldb -count=1 -run 'TestOpenGlobalDBCreatesExtensionsTableWithExpectedColumns|TestOpenGlobalDBMigratesLegacyExtensionsTableColumns'`; `go test ./internal/config -count=1 -run 'TestDefaultWithHomeLeavesMarketplaceConfigEmpty|TestExtensionsConfigValidateMarketplaceConfig|TestSkillsConfigValidateMarketplaceConfig|TestApplyConfigOverlayFileLeavesMarketplaceDefaultsWhenOverlayOmitsFields'`; `go test ./internal/extension -count=1 -run 'TestCapabilityCheckerMarketplaceShouldDenyRestrictedCapabilities|TestCapabilityCheckerMarketplaceShouldAllowDefaultReadCapabilities|TestCapabilityCheckerRegisterShouldApplyMarketplaceTierCeiling|TestRegistryInstallPersistsMarketplaceMetadata|TestRegistryInstallClearsRemoteMetadataForNonMarketplaceSources'`; `go test -tags integration ./internal/extension -count=1 -run 'TestRegistryIntegrationLifecycle|TestRegistryIntegrationMultipleSourcesCoexist'`; `go test -tags integration ./internal/registry -count=1 -run 'TestInstallerInstallPipelineWithInMemoryDownloader'`
Executed: 2026-04-14T19:24:55Z
Exit code: 0
Output summary: Targeted CLI, skill-migration, config, globaldb, extension capability, extension integration, and installer integration groups all passed; this included the new regressions for extension remove rollback compensation, `--asset` forwarding, `update --all`, and already-up-to-date behavior.
Warnings: none
Errors: none
Verdict: PASS

Claim: The legacy marketplace migration checks (`G21`) passed on the current tree.
Command: `find internal/skills -maxdepth 1 -type d -name marketplace`; `rg -n "internal/skills/marketplace|skills/marketplace|marketplace\\.New|marketplace\\.Client" internal cmd`
Executed: 2026-04-14T19:24:55Z
Exit code: 0
Output summary: `find` returned no `internal/skills/marketplace` directory, and `rg` returned no legacy package references under `internal/` or `cmd/`.
Warnings: none
Errors: none
Verdict: PASS

Claim: The isolated daemon-backed local extension install/remove replay (`G22`) preserved the source directory and cleaned only the managed install copy.
Command: `go build -o ./sdk/examples/secret-guard/bin/secret-guard ./sdk/examples/secret-guard`; `AGH_HOME=/tmp/agh-ext-reg-case.MyAerc ./bin/agh daemon start -o json`; `AGH_HOME=/tmp/agh-ext-reg-case.MyAerc ./bin/agh extension install sdk/examples/secret-guard -o json`; `sqlite3 /tmp/agh-ext-reg-case.MyAerc/agh.db "select manifest_path, registry_slug, registry_name, remote_version from extensions where name = 'secret-guard';"`; `AGH_HOME=/tmp/agh-ext-reg-case.MyAerc ./bin/agh extension remove secret-guard -o json`; `sqlite3 /tmp/agh-ext-reg-case.MyAerc/agh.db "select count(*) from extensions where name = 'secret-guard';"`; `AGH_HOME=/tmp/agh-ext-reg-case.MyAerc ./bin/agh extension list -o json`; `AGH_HOME=/tmp/agh-ext-reg-case.MyAerc ./bin/agh extension remove secret-guard -o json`; `AGH_HOME=/tmp/agh-ext-reg-case.MyAerc ./bin/agh daemon stop -o json`
Executed: 2026-04-14T19:24:55Z
Exit code: 0
Output summary: The daemon started in isolated mode; install returned `secret-guard` as `active` and `healthy`; SQLite recorded `/tmp/agh-ext-reg-case.MyAerc/extensions/secret-guard/extension.toml` with NULL registry provenance; the managed manifest existed while the source manifest under `sdk/examples/secret-guard/extension.toml` remained present; remove returned `/tmp/agh-ext-reg-case.MyAerc/extensions/secret-guard`; SQLite row count returned `0`; `extension list` returned `[]`; a second remove failed with `extension: extension not found`; the daemon stopped cleanly.
Warnings: none
Errors: none
Verdict: PASS

Claim: The repository smoke/build gate (`G1` through `G6`) passed from the current tree after the final code changes.
Command: `make fmt`; `make lint`; `make test`; `make build`; `go clean -testcache`; `make verify`
Executed: 2026-04-14T19:24:55Z
Exit code: 0
Output summary: `make lint` reported `0 issues`; `make test` finished with `2897 tests`; `make build` completed for backend and web assets; after a clean test cache, `make verify` again finished with `2897 tests in 13.181s` and `OK: all package boundaries respected`.
Warnings: none
Errors: none
Verdict: PASS

TEST CASE COVERAGE
--------------------------
Test cases found: 59 reference cases under `.compozy/tasks/extension-registry/test-cases/`
Executed: 59 exercised during this QA run
Results:
  - SMOKE-001: PASS | Evidence: G1,G2,G3,G4,G5,G6 | Bug: none
  - SMOKE-002: PASS | Evidence: G12,G13 | Bug: none
  - SMOKE-003: PASS | Evidence: G11,G12,G13 | Bug: none
  - SMOKE-004: PASS | Evidence: G13,G22 | Bug: none
  - SMOKE-005: PASS | Evidence: G14,G15 | Bug: none
  - TC-FUNC-001: PASS | Evidence: G7 | Bug: none
  - TC-FUNC-002: PASS | Evidence: G7 | Bug: none
  - TC-FUNC-003: PASS | Evidence: G7 | Bug: none
  - TC-FUNC-004: PASS | Evidence: G7 | Bug: none
  - TC-FUNC-005: PASS | Evidence: G7 | Bug: none
  - TC-FUNC-006: PASS | Evidence: G7 | Bug: none
  - TC-FUNC-007: PASS | Evidence: G7 | Bug: none
  - TC-FUNC-008: PASS | Evidence: G7 | Bug: none
  - TC-FUNC-009: PASS | Evidence: G8,G20 | Bug: none
  - TC-FUNC-010: PASS | Evidence: G8 | Bug: none
  - TC-FUNC-011: PASS | Evidence: G8 | Bug: none
  - TC-FUNC-012: PASS | Evidence: G8 | Bug: none
  - TC-FUNC-013: PASS | Evidence: G8 | Bug: none
  - TC-FUNC-014: PASS | Evidence: G8 | Bug: none
  - TC-FUNC-015: PASS | Evidence: G8 | Bug: none
  - TC-FUNC-016: PASS | Evidence: G8 | Bug: none
  - TC-FUNC-017: PASS | Evidence: G12,G13 | Bug: none
  - TC-FUNC-018: PASS | Evidence: G12 | Bug: none
  - TC-FUNC-019: PASS | Evidence: G12 | Bug: none
  - TC-FUNC-020: PASS | Evidence: G11,G12,G13 | Bug: none
  - TC-FUNC-021: PASS | Evidence: G11,G12,G13 | Bug: none
  - TC-FUNC-022: PASS | Evidence: G11,G12 | Bug: none
  - TC-FUNC-023: PASS | Evidence: G12,G22 | Bug: none
  - TC-FUNC-024: PASS | Evidence: G12 | Bug: none
  - TC-FUNC-025: PASS | Evidence: G12,G13 | Bug: none
  - TC-FUNC-026: PASS | Evidence: G12,G13 | Bug: none
  - TC-FUNC-027: PASS | Evidence: G14,G15 | Bug: none
  - TC-FUNC-028: PASS | Evidence: G14,G15,G21 | Bug: none
  - TC-FUNC-029: PASS | Evidence: G14,G15 | Bug: none
  - TC-FUNC-030: PASS | Evidence: G21 | Bug: none
  - TC-INT-001: PASS | Evidence: G10 | Bug: none
  - TC-INT-002: PASS | Evidence: G10 | Bug: none
  - TC-INT-003: PASS | Evidence: G10 | Bug: none
  - TC-INT-004: PASS | Evidence: G10 | Bug: none
  - TC-INT-005: PASS | Evidence: G11 | Bug: none
  - TC-INT-006: PASS | Evidence: G11 | Bug: none
  - TC-INT-007: PASS | Evidence: G11 | Bug: none
  - TC-INT-008: PASS | Evidence: G11 | Bug: none
  - TC-INT-009: PASS | Evidence: G11 | Bug: none
  - TC-INT-010: PASS | Evidence: G13,G19 | Bug: none
  - TC-REG-001: PASS | Evidence: G14,G15 | Bug: none
  - TC-REG-002: PASS | Evidence: G14,G15 | Bug: none
  - TC-REG-003: PASS | Evidence: G12,G22 | Bug: none
  - TC-REG-004: PASS | Evidence: G16 | Bug: none
  - TC-REG-005: PASS | Evidence: G17 | Bug: none
  - TC-REG-006: PASS | Evidence: G18 | Bug: none
  - TC-SEC-001: PASS | Evidence: G8,G9 | Bug: none
  - TC-SEC-002: PASS | Evidence: G8 | Bug: none
  - TC-SEC-003: PASS | Evidence: G8 | Bug: none
  - TC-SEC-004: PASS | Evidence: G8 | Bug: none
  - TC-SEC-005: PASS | Evidence: G18 | Bug: none
  - TC-SEC-006: PASS | Evidence: G8,G9 | Bug: none
  - TC-SEC-007: PASS | Evidence: G8,G9 | Bug: none
  - TC-SEC-008: PASS | Evidence: G17 | Bug: none
Not executed: none

ISSUES FILED
-------------
Total: 1
By severity:
  - Critical: 1
  - High: 0
  - Medium: 0
  - Low: 0
Details:
  - BUG-001: Local extension remove deleted the original source directory | Severity: Critical | Priority: P0 | Status: Fixed
