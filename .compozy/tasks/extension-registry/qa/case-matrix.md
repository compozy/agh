# Extension Registry Case Matrix

## Command Groups

| Group | Command |
|-------|---------|
| G1 | `make fmt` |
| G2 | `make lint` |
| G3 | `make test` |
| G4 | `make build` |
| G5 | `go clean -testcache` |
| G6 | `make verify` |
| G7 | `go test ./internal/registry -count=1 -run 'TestMultiRegistrySearchQueriesSourcesConcurrently|TestMultiRegistrySearchReturnsHealthyResultsOnPartialFailure|TestMultiRegistrySearchSkipsNonSearchableSources|TestMultiRegistryInfoResolvesHighestPrioritySource|TestMultiRegistryDownloadDelegatesToResolvedSource|TestMultiRegistryCheckUpdate|TestVersionIsNewer|TestMultiRegistryCloseClosesAllSources'` |
| G8 | `go test ./internal/registry -count=1 -run 'TestInstallerInstallExtensionArchiveReturnsChecksum|TestInstallerInstallRejectsCompressedArchiveOverLimit|TestInstallerInstallRejectsDecompressedArchiveOverLimit|TestExtractArchive_ValidArchiveProducesDirectoryStructure|TestExtractArchive_EnforcesLimitsAndRejectsUnsafeEntries|TestInstallerInstallBlocksCriticalVerificationContent|TestMoveInstalledDir|TestInstallerCleansStaleTempDirs'` |
| G9 | `go test ./internal/registry -count=1 -run 'TestExtractArchiveStripsSpecialPermissionBits|TestManifestPathAtRootRejectsSymlinkedManifest|TestCleanArchiveEntryPath|TestPathWithinRoot|TestInstallerInstallRejectsUnexpectedContentType'` |
| G10 | `go test ./internal/registry/clawhub -count=1 -run 'TestClientSearchParsesListingsAndLimit|TestClientSearchEmptyResultsReturnsEmptySlice|TestClientDownloadUsesLatestEndpointWhenVersionEmpty|TestClientDownloadUsesVersionedEndpointWhenVersionSpecified|TestClientRetriesHTTP500WithBackoff|TestClientSearchReturnsEmptyForExtensionFilter'` |
| G11 | `go test ./internal/registry/github -count=1 -run 'TestClientInfoFetchesLatestAndVersions|TestClientDownloadSingleTarballAsset|TestClientDownloadMultipleAssetsRequiresSelection|TestClientDownloadSelectsRequestedAsset|TestClientDownloadFallsBackToSourceArchive|TestClientFetchRequestedReleaseByTag|TestClientFetchRequestedReleaseRejectsPrerelease|TestClientRateLimitExceeded|TestClientUsesGitHubToken|TestCheckRateLimitWarnsWithoutFailing'` |
| G12 | `go test ./internal/cli -count=1 -run 'TestExtensionSearchCommandUsesSearchableRegistrySources|TestExtensionSearchCommandAppliesSourceFilter|TestExtensionSearchCommandRejectsNonPositiveLimit|TestExtensionInstallCommandInstallsMarketplaceExtensionAndPrintsRestartMessage|TestExtensionInstallCommandPassesAssetToRegistryDownload|TestExtensionInstallAndRemoveOfflinePreservesSourceDirectory|TestExtensionRemoveCommandDeletesDirectoryAndRegistryRecord|TestExtensionRemoveCommandReturnsClearErrorForMissingExtension|TestRemoveInstalledExtensionRollsBackRegistryOnCommitFailure|TestExtensionUpdateCommandCheckOnlyShowsAvailableUpdatesWithoutDownloading|TestExtensionUpdateCommandReportsAlreadyUpToDate|TestExtensionUpdateCommandReinstallsNewerVersion|TestExtensionUpdateCommandAllUpdatesMarketplaceExtensions'` |
| G13 | `go test -tags integration ./internal/cli -count=1 -run 'TestExtensionSearchCommandIntegrationReturnsRegistryListings|TestExtensionInstallCommandIntegrationCreatesManagedInstallAndRegistryRecord|TestExtensionUpdateAndRemoveIntegration|TestExtensionRemoveMissingIntegrationReturnsClearError'` |
| G14 | `go test ./internal/cli -count=1 -run 'TestSkillSearchCommandPassesLimitAndRendersTable|TestSkillSearchCommandRejectsNonPositiveLimit|TestSkillInstallCommandInstallsMarketplaceSkill|TestSkillUpdateCommandCheckOnlyReportsUpdateWithoutDownloading|TestSkillUpdateCommandAllUpdatesMarketplaceSkills|TestSkillUpdateCommandReportsAlreadyUpToDate|TestSkillRemoveCommandDeletesMarketplaceSkillDirectory|TestSkillRemoveCommandRefusesNonMarketplaceSkill'` |
| G15 | `go test -tags integration ./internal/cli -count=1 -run 'TestSkillSearchInstallListRemoveIntegrationFlow|TestSkillInstallCommandIntegrationCreatesSkillDirectoryAndSidecar|TestSkillInstallAndRemoveIntegrationRefreshesRegistry|TestSkillInstallCommandIntegrationWritesMatchingHash|TestSkillInstallCommandIntegrationReplacesExistingSkillDirectory'` |
| G16 | `go test ./internal/store/globaldb -count=1 -run 'TestOpenGlobalDBCreatesExtensionsTableWithExpectedColumns|TestOpenGlobalDBMigratesLegacyExtensionsTableColumns'` |
| G17 | `go test ./internal/config -count=1 -run 'TestDefaultWithHomeLeavesMarketplaceConfigEmpty|TestExtensionsConfigValidateMarketplaceConfig|TestSkillsConfigValidateMarketplaceConfig|TestApplyConfigOverlayFileLeavesMarketplaceDefaultsWhenOverlayOmitsFields'` |
| G18 | `go test ./internal/extension -count=1 -run 'TestCapabilityCheckerMarketplaceShouldDenyRestrictedCapabilities|TestCapabilityCheckerMarketplaceShouldAllowDefaultReadCapabilities|TestCapabilityCheckerRegisterShouldApplyMarketplaceTierCeiling|TestRegistryInstallPersistsMarketplaceMetadata|TestRegistryInstallClearsRemoteMetadataForNonMarketplaceSources'` |
| G19 | `go test -tags integration ./internal/extension -count=1 -run 'TestRegistryIntegrationLifecycle|TestRegistryIntegrationMultipleSourcesCoexist'` |
| G20 | `go test -tags integration ./internal/registry -count=1 -run 'TestInstallerInstallPipelineWithInMemoryDownloader'` |
| G21 | Shell migration checks: absence of `internal/skills/marketplace`, zero references to legacy import/types under `internal/` and `cmd/`, then `make build` |
| G22 | Isolated daemon replay for managed local extension install/remove using `sdk/examples/secret-guard` and a dedicated `AGH_HOME` |

## Case Mapping

| Case | Evidence Groups | Notes |
|------|-----------------|-------|
| SMOKE-001 | G1, G2, G3, G4, G5, G6 | Full repository gate. |
| SMOKE-002 | G12, G13 | CLI search behavior and searchable-source handling. |
| SMOKE-003 | G11, G12, G13 | GitHub install path, metadata, restart guidance. |
| SMOKE-004 | G13, G22 | Remove path, DB cleanup, not-found repeat case, live managed-dir replay. |
| SMOKE-005 | G14, G15 | Skill search/install/update through migrated pipeline. |
| TC-FUNC-001 | G7 | Concurrent aggregation and dedup. |
| TC-FUNC-002 | G7 | Partial source failure handling. |
| TC-FUNC-003 | G7 | Non-searchable source skip. |
| TC-FUNC-004 | G7 | Highest-priority info resolution. |
| TC-FUNC-005 | G7 | Download delegation. |
| TC-FUNC-006 | G7 | Update detection. |
| TC-FUNC-007 | G7 | Semantic version comparison. |
| TC-FUNC-008 | G7 | Close propagation. |
| TC-FUNC-009 | G8, G20 | Installer pipeline plus integration-tag pipeline proof. |
| TC-FUNC-010 | G8 | Compressed archive size limit. |
| TC-FUNC-011 | G8 | Decompressed size limit. |
| TC-FUNC-012 | G8 | File-count limit. |
| TC-FUNC-013 | G8 | Prompt-injection/content verification block. |
| TC-FUNC-014 | G8 | Atomic replace with backup. |
| TC-FUNC-015 | G8 | Stale temp cleanup. |
| TC-FUNC-016 | G8 | Single-root archive handling. |
| TC-FUNC-017 | G12, G13 | Default extension search flow. |
| TC-FUNC-018 | G12 | `--from` source filtering. |
| TC-FUNC-019 | G12 | `--limit` behavior and validation. |
| TC-FUNC-020 | G11, G12, G13 | Install from GitHub with explicit version. |
| TC-FUNC-021 | G11, G12, G13 | Install latest release without `--version`. |
| TC-FUNC-022 | G11, G12 | Asset selection from multi-asset GitHub release. |
| TC-FUNC-023 | G12, G22 | Local-path install preserved alongside marketplace flow. |
| TC-FUNC-024 | G12 | Remove rollback on final filesystem failure and not-found handling. |
| TC-FUNC-025 | G12, G13 | Update check mode without download. |
| TC-FUNC-026 | G12, G13 | Update install mode, `--all`, already-up-to-date path. |
| TC-FUNC-027 | G14, G15 | Skill install via migrated pipeline. |
| TC-FUNC-028 | G14, G15, G21 | Skill search via migrated pipeline and legacy package absence. |
| TC-FUNC-029 | G14, G15 | Skill update via migrated pipeline. |
| TC-FUNC-030 | G21 | Legacy marketplace package removal. |
| TC-INT-001 | G10 | ClawHub search parsing. |
| TC-INT-002 | G10 | ClawHub download latest/versioned. |
| TC-INT-003 | G10 | ClawHub retry/backoff. |
| TC-INT-004 | G10 | ClawHub extension-type filtering. |
| TC-INT-005 | G11 | GitHub release metadata parsing. |
| TC-INT-006 | G11 | GitHub asset selection. |
| TC-INT-007 | G11 | Pre-release and draft filtering. |
| TC-INT-008 | G11 | Rate limit handling and guidance. |
| TC-INT-009 | G11 | Auth token header usage. |
| TC-INT-010 | G13, G19 | End-to-end install/register/query/remove with real SQLite-backed flows. |
| TC-REG-001 | G14, G15 | Skill install behavior unchanged after migration. |
| TC-REG-002 | G14, G15 | Skill search output and limit behavior unchanged. |
| TC-REG-003 | G12, G22 | Local extension install path still works and is skipped by marketplace update flows. |
| TC-REG-004 | G16 | SQLite migration preserves existing rows and adds new columns. |
| TC-REG-005 | G17 | Backward-compatible config defaults and marketplace validation. |
| TC-REG-006 | G18 | Marketplace capability ceiling still enforced. |
| TC-SEC-001 | G8, G9 | Path traversal prevention helpers and extraction guardrails. |
| TC-SEC-002 | G8 | Symlink rejection in archive extraction. |
| TC-SEC-003 | G8 | Decompression-bomb protection. |
| TC-SEC-004 | G8 | Prompt-injection detection in manifests. |
| TC-SEC-005 | G18 | Marketplace capability ceiling. |
| TC-SEC-006 | G8, G9 | Compressed size enforcement and content-type validation. |
| TC-SEC-007 | G8, G9 | Safe permissions with special bits stripped. |
| TC-SEC-008 | G17 | Registry source config validation. |
