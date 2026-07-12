package globaldb

import "github.com/compozy/agh/internal/store"

var globalSchemaTailMigrations = []store.Migration{
	{
		Version:  64,
		Name:     "add_model_catalog_curation",
		Up:       migrateModelCatalogCuration,
		Checksum: "2026-07-09-add-model-catalog-curation",
	},
	{
		Version:  65,
		Name:     "add_model_catalog_explicit_curation",
		Up:       migrateModelCatalogExplicitCuration,
		Checksum: "2026-07-10-add-model-catalog-explicit-curation",
	},
	{
		Version:  66,
		Name:     "add_model_catalog_curation_presence",
		Up:       migrateModelCatalogCurationPresence,
		Checksum: "2026-07-10-add-model-catalog-curation-presence",
	},
	networkChannelProjectionsMigration,
	sessionCatalogPagingMigration,
	loopCatalogPagingMigration,
	automationCatalogProjectionMigration,
	networkTimelineSequenceMigration,
	sessionsCatalogIndexCleanupMigration,
	networkDirectRoomCreatedIndexMigration,
	goalDurableStateMigration,
	loopRunControlActorMigration,
	goalCheckpointControlCauseMigration,
	sessionCreationProfilesMigration,
	loopRunOriginIdentityMigration,
	goalRecoveryCleanupMigration,
	goalBindingReplayFencesMigration,
	goalAdoptionAttemptMigration,
}
