package globaldb

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/modelcatalog"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

const (
	modelCatalogMigrationVersion                 = 23
	modelCatalogSourceConstraintMigrationVersion = 24
)

func TestGlobalDBModelCatalogSchemaMigration(t *testing.T) {
	t.Parallel()

	t.Run("Should create model catalog schema on fresh DB", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)

		assertModelCatalogSchema(t, globalDB.db)
		assertAppliedMigrationVersion(t, globalDB.db, modelCatalogSourceConstraintMigrationVersion)
	})

	t.Run("Should upgrade previous global schema by appending model catalog migrations", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		previousDB := openPreviousModelCatalogSchemaDB(t, path)
		beforeRecords, err := store.AppliedMigrations(ctx, previousDB)
		if err != nil {
			t.Fatalf("AppliedMigrations(previous) error = %v", err)
		}
		if got, want := len(beforeRecords), modelCatalogMigrationVersion-1; got != want {
			t.Fatalf("len(beforeRecords) = %d, want %d", got, want)
		}
		exists, err := tableExists(ctx, previousDB, "model_catalog_sources")
		if err != nil {
			t.Fatalf("tableExists(model_catalog_sources) error = %v", err)
		}
		if exists {
			t.Fatal("model_catalog_sources exists before v23 migration, want absent")
		}
		if err := previousDB.Close(); err != nil {
			t.Fatalf("previousDB.Close() error = %v", err)
		}

		globalDB, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(upgrade) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
				t.Errorf("Close(upgrade) error = %v", closeErr)
			}
		})

		assertModelCatalogSchema(t, globalDB.db)
		records, err := store.AppliedMigrations(ctx, globalDB.db)
		if err != nil {
			t.Fatalf("AppliedMigrations(upgrade) error = %v", err)
		}
		if got, want := len(records), len(globalSchemaMigrations); got != want {
			t.Fatalf("len(records) = %d, want %d", got, want)
		}
		if got := records[len(records)-1]; got.Version != modelCatalogSourceConstraintMigrationVersion ||
			got.Name != "rebuild_model_catalog_source_constraints" {
			t.Fatalf("tail migration = %#v, want model catalog source constraint v24", got)
		}
		for index, before := range beforeRecords {
			if !records[index].AppliedAt.Equal(before.AppliedAt) {
				t.Fatalf(
					"migration %d applied_at = %s, want unchanged %s",
					before.Version,
					records[index].AppliedAt,
					before.AppliedAt,
				)
			}
		}
	})

	t.Run("Should rebuild v23 model catalog tables into the v24 constrained shape", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		previousDB := openV23ModelCatalogSchemaDB(t, path)
		if _, err := previousDB.ExecContext(
			ctx,
			`INSERT INTO model_catalog_rows (
				source_id,
				provider_id,
				model_id,
				source_kind,
				priority,
				stale,
				refreshed_at,
				expires_at,
				display_name,
				last_error
			) VALUES (?, ?, ?, ?, ?, 0, ?, ?, '', '')`,
			"orphan-source",
			"codex",
			"gpt-5.4",
			string(modelcatalog.SourceKindConfig),
			120,
			time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
			"",
		); err != nil {
			t.Fatalf("ExecContext(insert orphan row) error = %v", err)
		}
		if err := previousDB.Close(); err != nil {
			t.Fatalf("previousDB.Close() error = %v", err)
		}

		globalDB, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(v24 upgrade) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
				t.Errorf("Close(v24 upgrade) error = %v", closeErr)
			}
		})

		assertAppliedMigrationVersion(t, globalDB.db, modelCatalogSourceConstraintMigrationVersion)

		var rowCount int
		if err := globalDB.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM model_catalog_rows`).
			Scan(&rowCount); err != nil {
			t.Fatalf("QueryRowContext(model_catalog_rows count) error = %v", err)
		}
		if got, want := rowCount, 0; got != want {
			t.Fatalf("model_catalog_rows count = %d, want %d after rebuild", got, want)
		}

		_, err = globalDB.db.ExecContext(
			ctx,
			`INSERT INTO model_catalog_rows (
				source_id,
				provider_id,
				model_id,
				source_kind,
				priority,
				stale,
				refreshed_at,
				expires_at,
				display_name,
				last_error
			) VALUES (?, ?, ?, ?, ?, 0, ?, ?, '', '')`,
			"missing-source",
			"codex",
			"gpt-5.4",
			string(modelcatalog.SourceKindConfig),
			120,
			time.Date(2026, 5, 7, 12, 1, 0, 0, time.UTC).Format(time.RFC3339Nano),
			"",
		)
		requireSQLiteConstraintError(t, err)
	})

	t.Run("Should keep model catalog migration record stable after reopen", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		first, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(first) error = %v", err)
		}
		firstRecords, err := store.AppliedMigrations(ctx, first.db)
		if err != nil {
			t.Fatalf("AppliedMigrations(first) error = %v", err)
		}
		if err := first.Close(ctx); err != nil {
			t.Fatalf("Close(first) error = %v", err)
		}

		second, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(second) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := second.Close(testutil.Context(t)); closeErr != nil {
				t.Errorf("Close(second) error = %v", closeErr)
			}
		})
		secondRecords, err := store.AppliedMigrations(ctx, second.db)
		if err != nil {
			t.Fatalf("AppliedMigrations(second) error = %v", err)
		}
		if got, want := len(secondRecords), len(firstRecords); got != want {
			t.Fatalf("len(secondRecords) = %d, want %d", got, want)
		}
		for index, firstRecord := range firstRecords {
			if !secondRecords[index].AppliedAt.Equal(firstRecord.AppliedAt) {
				t.Fatalf(
					"migration %d applied_at = %s, want unchanged %s",
					firstRecord.Version,
					secondRecords[index].AppliedAt,
					firstRecord.AppliedAt,
				)
			}
		}
	})

	t.Run("Should keep model catalog migration identity at global registry tail", func(t *testing.T) {
		t.Parallel()

		if len(globalSchemaMigrations) < modelCatalogSourceConstraintMigrationVersion {
			t.Fatalf(
				"len(globalSchemaMigrations) = %d, want at least %d",
				len(globalSchemaMigrations),
				modelCatalogSourceConstraintMigrationVersion,
			)
		}
		tail := globalSchemaMigrations[len(globalSchemaMigrations)-1]
		if tail.Version != modelCatalogSourceConstraintMigrationVersion ||
			tail.Name != "rebuild_model_catalog_source_constraints" ||
			tail.Checksum != "2026-05-07-rebuild-model-catalog-source-constraints" {
			t.Fatalf(
				"tail migration = version %d name %q checksum %q, want model catalog source constraint v24",
				tail.Version,
				tail.Name,
				tail.Checksum,
			)
		}
		if previous := globalSchemaMigrations[len(globalSchemaMigrations)-2]; previous.Version != modelCatalogMigrationVersion {
			t.Fatalf("previous migration version = %d, want %d", previous.Version, modelCatalogMigrationVersion)
		}
	})

	t.Run("Should keep model catalog schema out of migration v1 statements", func(t *testing.T) {
		t.Parallel()

		if !slices.Equal(globalSchemaStatements, schemaStatementsWithoutModelCatalog()) {
			t.Fatalf("globalSchemaStatements unexpectedly include model catalog schema statements")
		}
	})
}

func TestGlobalDBModelCatalogStore(t *testing.T) {
	t.Parallel()

	t.Run("Should replace source rows and reasoning efforts atomically", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		first := modelCatalogRow("config", "codex", "gpt-5.4", modelcatalog.SourceKindConfig, 120)
		first.ReasoningEfforts = []modelcatalog.ReasoningEffort{
			modelcatalog.ReasoningEffortLow,
			modelcatalog.ReasoningEffortHigh,
		}
		replaceModelCatalogRows(
			t,
			globalDB,
			"config",
			"codex",
			modelcatalog.SourceKindConfig,
			120,
			[]modelcatalog.ModelRow{first},
		)

		second := modelCatalogRow("config", "codex", "gpt-5.5", modelcatalog.SourceKindConfig, 120)
		second.ReasoningEfforts = []modelcatalog.ReasoningEffort{modelcatalog.ReasoningEffortMinimal}
		replaceModelCatalogRows(
			t,
			globalDB,
			"config",
			"codex",
			modelcatalog.SourceKindConfig,
			120,
			[]modelcatalog.ModelRow{second},
		)

		rows, err := globalDB.ListRows(
			ctx,
			modelcatalog.ListOptions{ProviderID: "codex", SourceID: "config", IncludeStale: true},
		)
		if err != nil {
			t.Fatalf("ListRows() error = %v", err)
		}
		if got, want := len(rows), 1; got != want {
			t.Fatalf("len(rows) = %d, want %d: %#v", got, want, rows)
		}
		if rows[0].ModelID != "gpt-5.5" || !slices.Equal(rows[0].ReasoningEfforts, second.ReasoningEfforts) {
			t.Fatalf("rows[0] = %#v, want replacement row with minimal effort", rows[0])
		}

		var oldEffortCount int
		if err := globalDB.db.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM model_catalog_reasoning_efforts WHERE source_id = ? AND provider_id = ? AND model_id = ?`,
			"config",
			"codex",
			"gpt-5.4",
		).Scan(&oldEffortCount); err != nil {
			t.Fatalf("QueryRowContext(old efforts) error = %v", err)
		}
		if oldEffortCount != 0 {
			t.Fatalf("old effort count = %d, want 0", oldEffortCount)
		}
		statuses, err := globalDB.ListSourceStatus(ctx, "codex")
		if err != nil {
			t.Fatalf("ListSourceStatus() error = %v", err)
		}
		if len(statuses) != 1 || statuses[0].RowCount != 1 {
			t.Fatalf("statuses = %#v, want one status with row_count 1", statuses)
		}
	})

	t.Run("Should roll back source replacement when reasoning effort insert fails", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		original := modelCatalogRow("config", "codex", "gpt-5.4", modelcatalog.SourceKindConfig, 120)
		original.ReasoningEfforts = []modelcatalog.ReasoningEffort{modelcatalog.ReasoningEffortLow}
		replaceModelCatalogRows(
			t,
			globalDB,
			"config",
			"codex",
			modelcatalog.SourceKindConfig,
			120,
			[]modelcatalog.ModelRow{original},
		)

		invalid := modelCatalogRow("config", "codex", "gpt-5.5", modelcatalog.SourceKindConfig, 120)
		invalid.ReasoningEfforts = []modelcatalog.ReasoningEffort{
			modelcatalog.ReasoningEffortHigh,
			modelcatalog.ReasoningEffortHigh,
		}
		err := globalDB.ReplaceSourceRows(
			ctx,
			"config",
			"codex",
			[]modelcatalog.ModelRow{invalid},
			modelCatalogStatus("config", "codex", modelcatalog.SourceKindConfig, 120),
		)
		if err == nil {
			t.Fatal("ReplaceSourceRows(duplicate efforts) error = nil, want constraint error")
		}

		rows, err := globalDB.ListRows(
			ctx,
			modelcatalog.ListOptions{ProviderID: "codex", SourceID: "config", IncludeStale: true},
		)
		if err != nil {
			t.Fatalf("ListRows(after failed replace) error = %v", err)
		}
		if len(rows) != 1 || rows[0].ModelID != "gpt-5.4" ||
			!slices.Equal(rows[0].ReasoningEfforts, original.ReasoningEfforts) {
			t.Fatalf("rows after failed replace = %#v, want original row preserved", rows)
		}
		statuses, err := globalDB.ListSourceStatus(ctx, "codex")
		if err != nil {
			t.Fatalf("ListSourceStatus(after failed replace) error = %v", err)
		}
		if len(statuses) != 1 || statuses[0].RowCount != 1 {
			t.Fatalf("statuses after failed replace = %#v, want original row_count 1", statuses)
		}
	})

	t.Run("Should filter rows by provider source and stale flag", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		fresh := modelCatalogRow("config", "codex", "codex-fresh", modelcatalog.SourceKindConfig, 120)
		stale := modelCatalogRow("config", "codex", "codex-stale", modelcatalog.SourceKindConfig, 120)
		stale.Stale = true
		replaceModelCatalogRows(
			t,
			globalDB,
			"config",
			"codex",
			modelcatalog.SourceKindConfig,
			120,
			[]modelcatalog.ModelRow{fresh, stale},
		)
		dev := modelCatalogRow("models_dev", "codex", "codex-dev", modelcatalog.SourceKindModelsDev, 50)
		replaceModelCatalogRows(
			t,
			globalDB,
			"models_dev",
			"codex",
			modelcatalog.SourceKindModelsDev,
			50,
			[]modelcatalog.ModelRow{dev},
		)
		claude := modelCatalogRow("config", "claude", "claude-fresh", modelcatalog.SourceKindConfig, 120)
		replaceModelCatalogRows(
			t,
			globalDB,
			"config",
			"claude",
			modelcatalog.SourceKindConfig,
			120,
			[]modelcatalog.ModelRow{claude},
		)

		rows, err := globalDB.ListRows(ctx, modelcatalog.ListOptions{ProviderID: "codex", SourceID: "config"})
		if err != nil {
			t.Fatalf("ListRows(fresh config) error = %v", err)
		}
		assertModelCatalogModelIDs(t, rows, []string{"codex-fresh"})

		rows, err = globalDB.ListRows(
			ctx,
			modelcatalog.ListOptions{ProviderID: "codex", SourceID: "config", IncludeStale: true},
		)
		if err != nil {
			t.Fatalf("ListRows(include stale) error = %v", err)
		}
		assertModelCatalogModelIDs(t, rows, []string{"codex-fresh", "codex-stale"})

		rows, err = globalDB.ListRows(
			ctx,
			modelcatalog.ListOptions{ProviderID: "codex", SourceID: "models_dev", IncludeAll: true},
		)
		if err != nil {
			t.Fatalf("ListRows(models_dev) error = %v", err)
		}
		assertModelCatalogModelIDs(t, rows, []string{"codex-dev"})

		rows, err = globalDB.ListRows(ctx, modelcatalog.ListOptions{ProviderID: "claude", IncludeStale: true})
		if err != nil {
			t.Fatalf("ListRows(claude) error = %v", err)
		}
		assertModelCatalogModelIDs(t, rows, []string{"claude-fresh"})
	})

	t.Run("Should store models dev status per provider without empty provider sentinel", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		replaceModelCatalogRows(t, globalDB, "models_dev", "codex", modelcatalog.SourceKindModelsDev, 50, nil)
		replaceModelCatalogRows(t, globalDB, "models_dev", "claude", modelcatalog.SourceKindModelsDev, 50, nil)

		statuses, err := globalDB.ListSourceStatus(ctx, "")
		if err != nil {
			t.Fatalf("ListSourceStatus(all) error = %v", err)
		}
		if got, want := len(statuses), 2; got != want {
			t.Fatalf("len(statuses) = %d, want %d: %#v", got, want, statuses)
		}
		for _, status := range statuses {
			if status.SourceID != "models_dev" {
				t.Fatalf("status.SourceID = %q, want models_dev", status.SourceID)
			}
			if status.ProviderID == "" {
				t.Fatalf("status has empty provider sentinel: %#v", status)
			}
		}
		codexStatuses, err := globalDB.ListSourceStatus(ctx, "codex")
		if err != nil {
			t.Fatalf("ListSourceStatus(codex) error = %v", err)
		}
		if len(codexStatuses) != 1 || codexStatuses[0].ProviderID != "codex" {
			t.Fatalf("codex statuses = %#v, want one codex status", codexStatuses)
		}
	})

	t.Run("Should reject empty provider source status sentinel writes", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		err := globalDB.ReplaceSourceRows(
			testutil.Context(t),
			"models_dev",
			"",
			nil,
			modelcatalog.SourceStatus{
				SourceID:   "models_dev",
				SourceKind: modelcatalog.SourceKindModelsDev,
				ProviderID: "",
				Priority:   50,
			},
		)
		if err == nil || !strings.Contains(err.Error(), "provider id is required") {
			t.Fatalf("ReplaceSourceRows(empty provider) error = %v, want provider id validation", err)
		}
	})

	t.Run("Should reject mismatched source status and row identities", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name   string
			rows   []modelcatalog.ModelRow
			status modelcatalog.SourceStatus
			want   string
		}{
			{
				name:   "Should reject mismatched status source",
				status: modelCatalogStatus("other", "codex", modelcatalog.SourceKindConfig, 120),
				want:   "status source id",
			},
			{
				name:   "Should reject missing status source kind",
				status: modelcatalog.SourceStatus{SourceID: "config", ProviderID: "codex", Priority: 120},
				want:   "source kind is required",
			},
			{
				name: "Should reject mismatched row provider",
				rows: []modelcatalog.ModelRow{
					modelCatalogRow("config", "claude", "gpt-5.4", modelcatalog.SourceKindConfig, 120),
				},
				status: modelCatalogStatus("config", "codex", modelcatalog.SourceKindConfig, 120),
				want:   "provider id",
			},
			{
				name: "Should reject mismatched row source kind",
				rows: []modelcatalog.ModelRow{
					modelCatalogRow("config", "codex", "gpt-5.4", modelcatalog.SourceKindModelsDev, 120),
				},
				status: modelCatalogStatus("config", "codex", modelcatalog.SourceKindConfig, 120),
				want:   "source kind",
			},
			{
				name: "Should reject blank reasoning effort",
				rows: []modelcatalog.ModelRow{
					func() modelcatalog.ModelRow {
						row := modelCatalogRow("config", "codex", "gpt-5.4", modelcatalog.SourceKindConfig, 120)
						row.ReasoningEfforts = []modelcatalog.ReasoningEffort{""}
						return row
					}(),
				},
				status: modelCatalogStatus("config", "codex", modelcatalog.SourceKindConfig, 120),
				want:   "reasoning effort",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				globalDB := openTestGlobalDB(t)
				err := globalDB.ReplaceSourceRows(testutil.Context(t), "config", "codex", tc.rows, tc.status)
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("ReplaceSourceRows() error = %v, want containing %q", err, tc.want)
				}
			})
		}
	})

	t.Run("Should reject nil contexts for catalog store methods", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		nilCtx := nilModelCatalogTestContext()
		if err := globalDB.ReplaceSourceRows(
			nilCtx,
			"config",
			"codex",
			nil,
			modelCatalogStatus("config", "codex", modelcatalog.SourceKindConfig, 120),
		); err == nil {
			t.Fatal("ReplaceSourceRows(nil context) error = nil, want validation error")
		}
		if _, err := globalDB.ListRows(nilCtx, modelcatalog.ListOptions{}); err == nil {
			t.Fatal("ListRows(nil context) error = nil, want validation error")
		}
		if _, err := globalDB.ListSourceStatus(nilCtx, "codex"); err == nil {
			t.Fatal("ListSourceStatus(nil context) error = nil, want validation error")
		}
	})

	t.Run("Should preserve null default reasoning effort", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		row := modelCatalogRow("config", "codex", "gpt-5.4", modelcatalog.SourceKindConfig, 120)
		row.ReasoningEfforts = []modelcatalog.ReasoningEffort{modelcatalog.ReasoningEffortLow}
		replaceModelCatalogRows(
			t,
			globalDB,
			"config",
			"codex",
			modelcatalog.SourceKindConfig,
			120,
			[]modelcatalog.ModelRow{row},
		)

		var raw sql.NullString
		if err := globalDB.db.QueryRowContext(
			ctx,
			`SELECT default_reasoning_effort FROM model_catalog_rows WHERE source_id = ? AND provider_id = ? AND model_id = ?`,
			"config",
			"codex",
			"gpt-5.4",
		).Scan(&raw); err != nil {
			t.Fatalf("QueryRowContext(default_reasoning_effort) error = %v", err)
		}
		if raw.Valid {
			t.Fatalf("default_reasoning_effort raw = %#v, want NULL", raw)
		}
		rows, err := globalDB.ListRows(
			ctx,
			modelcatalog.ListOptions{ProviderID: "codex", SourceID: "config", IncludeStale: true},
		)
		if err != nil {
			t.Fatalf("ListRows() error = %v", err)
		}
		if len(rows) != 1 || rows[0].DefaultReasoningEffort != nil {
			t.Fatalf("rows = %#v, want nil DefaultReasoningEffort", rows)
		}
	})

	t.Run("Should round trip nullable booleans and explicit default reasoning effort", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		available := false
		supportsReasoning := false
		defaultEffort := modelcatalog.ReasoningEffortHigh
		row := modelCatalogRow("", "", "manual-model", "", 120)
		row.Available = &available
		row.SupportsTools = nil
		row.SupportsReasoning = &supportsReasoning
		row.DefaultReasoningEffort = &defaultEffort
		row.ReasoningEfforts = []modelcatalog.ReasoningEffort{
			modelcatalog.ReasoningEffortLow,
			modelcatalog.ReasoningEffortHigh,
		}
		status := modelCatalogStatus("config", "codex", modelcatalog.SourceKindConfig, 120)
		status.RefreshState = ""
		if err := globalDB.ReplaceSourceRows(ctx, "config", "codex", []modelcatalog.ModelRow{row}, status); err != nil {
			t.Fatalf("ReplaceSourceRows() error = %v", err)
		}

		statuses, err := globalDB.ListSourceStatus(ctx, "codex")
		if err != nil {
			t.Fatalf("ListSourceStatus() error = %v", err)
		}
		if len(statuses) != 1 || statuses[0].RefreshState != modelcatalog.RefreshStateIdle {
			t.Fatalf("statuses = %#v, want default idle refresh state", statuses)
		}
		rows, err := globalDB.ListRows(
			ctx,
			modelcatalog.ListOptions{ProviderID: "codex", SourceID: "config", IncludeStale: true},
		)
		if err != nil {
			t.Fatalf("ListRows() error = %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("rows = %#v, want one row", rows)
		}
		got := rows[0]
		if got.SourceID != "config" || got.ProviderID != "codex" || got.SourceKind != modelcatalog.SourceKindConfig {
			t.Fatalf("row identity = %#v, want normalized source/provider/kind", got)
		}
		if got.Available == nil || *got.Available || got.SupportsTools != nil ||
			got.SupportsReasoning == nil || *got.SupportsReasoning ||
			got.DefaultReasoningEffort == nil || *got.DefaultReasoningEffort != modelcatalog.ReasoningEffortHigh {
			t.Fatalf("row nullable values = %#v, want false/nil/false/high", got)
		}
	})

	t.Run("Should update source status row count stale flag and last error", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		row := modelCatalogRow("provider_live:codex", "codex", "gpt-5.4", modelcatalog.SourceKindProviderLive, 110)
		replaceModelCatalogRows(
			t,
			globalDB,
			"provider_live:codex",
			"codex",
			modelcatalog.SourceKindProviderLive,
			110,
			[]modelcatalog.ModelRow{row},
		)

		failed := modelCatalogStatus("provider_live:codex", "codex", modelcatalog.SourceKindProviderLive, 110)
		failed.RefreshState = modelcatalog.RefreshStateFailed
		failed.LastError = "redacted refresh failed"
		failed.Stale = true
		if err := globalDB.ReplaceSourceRows(ctx, "provider_live:codex", "codex", nil, failed); err != nil {
			t.Fatalf("ReplaceSourceRows(failed) error = %v", err)
		}

		statuses, err := globalDB.ListSourceStatus(ctx, "codex")
		if err != nil {
			t.Fatalf("ListSourceStatus() error = %v", err)
		}
		if len(statuses) != 1 {
			t.Fatalf("statuses = %#v, want one status", statuses)
		}
		status := statuses[0]
		if status.RowCount != 0 || !status.Stale || status.LastError != "redacted refresh failed" ||
			status.RefreshState != modelcatalog.RefreshStateFailed {
			t.Fatalf("status = %#v, want failed stale status with row_count 0", status)
		}
		rows, err := globalDB.ListRows(
			ctx,
			modelcatalog.ListOptions{ProviderID: "codex", SourceID: "provider_live:codex", IncludeStale: true},
		)
		if err != nil {
			t.Fatalf("ListRows(after failed replace) error = %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("rows after failed replace = %#v, want empty", rows)
		}
	})

	t.Run("Should order equal freshness rows by source identity", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		refreshedAt := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
		zulu := modelCatalogRow("extension:z_source", "codex", "gpt-5.4", modelcatalog.SourceKindExtension, 100)
		zulu.RefreshedAt = refreshedAt
		alpha := modelCatalogRow("extension:a_source", "codex", "gpt-5.4", modelcatalog.SourceKindExtension, 100)
		alpha.RefreshedAt = refreshedAt
		replaceModelCatalogRows(
			t,
			globalDB,
			"extension:z_source",
			"codex",
			modelcatalog.SourceKindExtension,
			100,
			[]modelcatalog.ModelRow{zulu},
		)
		replaceModelCatalogRows(
			t,
			globalDB,
			"extension:a_source",
			"codex",
			modelcatalog.SourceKindExtension,
			100,
			[]modelcatalog.ModelRow{alpha},
		)

		rows, err := globalDB.ListRows(ctx, modelcatalog.ListOptions{ProviderID: "codex", IncludeStale: true})
		if err != nil {
			t.Fatalf("ListRows() error = %v", err)
		}
		if got, want := len(rows), 2; got != want {
			t.Fatalf("len(rows) = %d, want %d: %#v", got, want, rows)
		}
		gotSources := []string{rows[0].SourceID, rows[1].SourceID}
		if !slices.Equal(gotSources, []string{"extension:a_source", "extension:z_source"}) {
			t.Fatalf("source order = %#v, want extension:a_source before extension:z_source", gotSources)
		}
	})

	t.Run("Should surface corrupt persisted catalog timestamps and booleans", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		row := modelCatalogRow("config", "codex", "gpt-5.4", modelcatalog.SourceKindConfig, 120)
		replaceModelCatalogRows(
			t,
			globalDB,
			"config",
			"codex",
			modelcatalog.SourceKindConfig,
			120,
			[]modelcatalog.ModelRow{row},
		)

		if _, err := globalDB.db.ExecContext(
			ctx,
			`UPDATE model_catalog_rows SET refreshed_at = ? WHERE source_id = ? AND provider_id = ? AND model_id = ?`,
			"bad-timestamp",
			"config",
			"codex",
			"gpt-5.4",
		); err != nil {
			t.Fatalf("ExecContext(corrupt row timestamp) error = %v", err)
		}
		if _, err := globalDB.ListRows(
			ctx,
			modelcatalog.ListOptions{ProviderID: "codex", SourceID: "config", IncludeStale: true},
		); err == nil ||
			!strings.Contains(err.Error(), "refreshed_at") {
			t.Fatalf("ListRows(corrupt timestamp) error = %v, want refreshed_at parse error", err)
		}

		if _, err := globalDB.db.ExecContext(
			ctx,
			`UPDATE model_catalog_rows SET refreshed_at = ?, available = ? WHERE source_id = ? AND provider_id = ? AND model_id = ?`,
			store.FormatTimestamp(row.RefreshedAt),
			2,
			"config",
			"codex",
			"gpt-5.4",
		); err == nil {
			t.Fatal("ExecContext(corrupt boolean with checks enabled) error = nil, want constraint error")
		}
		if _, err := globalDB.db.ExecContext(ctx, `PRAGMA ignore_check_constraints = ON`); err != nil {
			t.Fatalf("enable ignore_check_constraints error = %v", err)
		}
		if _, err := globalDB.db.ExecContext(
			ctx,
			`UPDATE model_catalog_rows SET refreshed_at = ?, available = ? WHERE source_id = ? AND provider_id = ? AND model_id = ?`,
			store.FormatTimestamp(row.RefreshedAt),
			2,
			"config",
			"codex",
			"gpt-5.4",
		); err != nil {
			t.Fatalf("ExecContext(corrupt boolean) error = %v", err)
		}
		if _, err := globalDB.db.ExecContext(ctx, `PRAGMA ignore_check_constraints = OFF`); err != nil {
			t.Fatalf("disable ignore_check_constraints error = %v", err)
		}
		if _, err := globalDB.ListRows(
			ctx,
			modelcatalog.ListOptions{ProviderID: "codex", SourceID: "config", IncludeStale: true},
		); err == nil ||
			!strings.Contains(err.Error(), "available boolean") {
			t.Fatalf("ListRows(corrupt boolean) error = %v, want available boolean error", err)
		}

		if _, err := globalDB.db.ExecContext(
			ctx,
			`UPDATE model_catalog_sources SET last_refresh_at = ? WHERE source_id = ? AND provider_id = ?`,
			"bad-status-time",
			"config",
			"codex",
		); err != nil {
			t.Fatalf("ExecContext(corrupt status timestamp) error = %v", err)
		}
		if _, err := globalDB.ListSourceStatus(ctx, "codex"); err == nil ||
			!strings.Contains(err.Error(), "last_refresh_at") {
			t.Fatalf("ListSourceStatus(corrupt timestamp) error = %v, want last_refresh_at parse error", err)
		}
	})

	t.Run("Should read rows and reasoning efforts from one transaction snapshot", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)

		first := modelCatalogRow("config", "codex", "gpt-5.4", modelcatalog.SourceKindConfig, 120)
		first.ReasoningEfforts = []modelcatalog.ReasoningEffort{modelcatalog.ReasoningEffortLow}
		replaceModelCatalogRows(
			t,
			globalDB,
			"config",
			"codex",
			modelcatalog.SourceKindConfig,
			120,
			[]modelcatalog.ModelRow{first},
		)

		conn, err := globalDB.db.Conn(ctx)
		if err != nil {
			t.Fatalf("db.Conn() error = %v", err)
		}
		defer func() {
			if closeErr := conn.Close(); closeErr != nil {
				t.Errorf("conn.Close() error = %v", closeErr)
			}
		}()

		tx, err := conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			t.Fatalf("BeginTx() error = %v", err)
		}
		defer func() {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
				t.Errorf("tx.Rollback() error = %v", rollbackErr)
			}
		}()

		var rowCount int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM model_catalog_rows`).Scan(&rowCount); err != nil {
			t.Fatalf("QueryRowContext(count) error = %v", err)
		}
		if got, want := rowCount, 1; got != want {
			t.Fatalf("rowCount = %d, want %d", got, want)
		}

		second := modelCatalogRow("config", "codex", "gpt-5.4", modelcatalog.SourceKindConfig, 120)
		second.ReasoningEfforts = []modelcatalog.ReasoningEffort{modelcatalog.ReasoningEffortHigh}
		replaceModelCatalogRows(
			t,
			globalDB,
			"config",
			"codex",
			modelcatalog.SourceKindConfig,
			120,
			[]modelcatalog.ModelRow{second},
		)

		rows, err := listModelCatalogRows(ctx, tx, modelcatalog.ListOptions{
			ProviderID:   "codex",
			SourceID:     "config",
			IncludeStale: true,
		})
		if err != nil {
			t.Fatalf("listModelCatalogRows(snapshot) error = %v", err)
		}
		if got, want := len(rows), 1; got != want {
			t.Fatalf("len(rows) = %d, want %d: %#v", got, want, rows)
		}
		if !slices.Equal(rows[0].ReasoningEfforts, first.ReasoningEfforts) {
			t.Fatalf("snapshot ReasoningEfforts = %#v, want %#v", rows[0].ReasoningEfforts, first.ReasoningEfforts)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("tx.Commit() error = %v", err)
		}

		freshRows, err := globalDB.ListRows(ctx, modelcatalog.ListOptions{
			ProviderID:   "codex",
			SourceID:     "config",
			IncludeStale: true,
		})
		if err != nil {
			t.Fatalf("ListRows(fresh) error = %v", err)
		}
		if !slices.Equal(freshRows[0].ReasoningEfforts, second.ReasoningEfforts) {
			t.Fatalf("fresh ReasoningEfforts = %#v, want %#v", freshRows[0].ReasoningEfforts, second.ReasoningEfforts)
		}
	})
}

func assertModelCatalogSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	assertTablesPresent(t, db, "model_catalog_sources", "model_catalog_rows", "model_catalog_reasoning_efforts")
	assertTableColumns(t, db, "model_catalog_sources", []string{
		"source_id",
		"provider_id",
		"source_kind",
		"priority",
		"refresh_state",
		"last_refresh_at",
		"next_refresh_at",
		"last_success_at",
		"last_error",
		"row_count",
		"stale",
	})
	assertTableColumns(t, db, "model_catalog_rows", []string{
		"source_id",
		"provider_id",
		"model_id",
		"source_kind",
		"priority",
		"available",
		"stale",
		"refreshed_at",
		"expires_at",
		"display_name",
		"context_window",
		"max_input_tokens",
		"max_output_tokens",
		"supports_tools",
		"supports_reasoning",
		"default_reasoning_effort",
		"cost_input_per_million",
		"cost_output_per_million",
		"last_error",
	})
	assertTableColumns(t, db, "model_catalog_reasoning_efforts", []string{
		"source_id",
		"provider_id",
		"model_id",
		"effort",
		"rank",
	})
	assertIndexesPresent(
		t,
		db,
		"model_catalog_rows",
		"idx_model_catalog_rows_provider_model",
		"idx_model_catalog_rows_source_provider",
	)
	assertIndexesPresent(t, db, "model_catalog_sources", "idx_model_catalog_sources_provider")
	assertModelCatalogRowSourceForeignKey(t, db)
}

func openPreviousModelCatalogSchemaDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()

	ctx := testutil.Context(t)
	db, err := store.OpenSQLiteDatabase(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase(previous) error = %v", err)
	}
	if err := store.RunMigrations(ctx, db, previousModelCatalogMigrations()); err != nil {
		t.Fatalf("RunMigrations(previous) error = %v", err)
	}
	return db
}

func previousModelCatalogMigrations() []store.Migration {
	migrations := append([]store.Migration(nil), globalSchemaMigrations[:modelCatalogMigrationVersion-1]...)
	migrations[0].Statements = schemaStatementsWithoutModelCatalog()
	return migrations
}

func openV23ModelCatalogSchemaDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()

	ctx := testutil.Context(t)
	db, err := store.OpenSQLiteDatabase(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase(v23) error = %v", err)
	}
	if err := store.RunMigrations(
		ctx,
		db,
		append([]store.Migration(nil), globalSchemaMigrations[:modelCatalogSourceConstraintMigrationVersion-1]...),
	); err != nil {
		t.Fatalf("RunMigrations(v23) error = %v", err)
	}
	return db
}

func schemaStatementsWithoutModelCatalog() []string {
	blocked := make(map[string]struct{}, len(modelCatalogSchemaStatements()))
	for _, statement := range modelCatalogSchemaStatements() {
		blocked[strings.TrimSpace(statement)] = struct{}{}
	}
	filtered := make([]string, 0, len(globalSchemaStatements)-len(blocked))
	for _, statement := range globalSchemaStatements {
		if _, ok := blocked[strings.TrimSpace(statement)]; ok {
			continue
		}
		filtered = append(filtered, statement)
	}
	return filtered
}

func assertModelCatalogRowSourceForeignKey(t *testing.T, db *sql.DB) {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), `PRAGMA foreign_key_list(model_catalog_rows)`)
	if err != nil {
		t.Fatalf("PRAGMA foreign_key_list(model_catalog_rows) error = %v", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			t.Fatalf("rows.Close(foreign_key_list model_catalog_rows) error = %v", closeErr)
		}
	}()

	type foreignKeyRow struct {
		table string
		from  string
		to    string
	}

	refs := make([]foreignKeyRow, 0)
	for rows.Next() {
		var (
			id       int
			seq      int
			ref      foreignKeyRow
			onUpdate string
			onDelete string
			match    string
		)
		if err := rows.Scan(&id, &seq, &ref.table, &ref.from, &ref.to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatalf("Scan(foreign_key_list model_catalog_rows) error = %v", err)
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(foreign_key_list model_catalog_rows) error = %v", err)
	}

	if !slices.Contains(refs, foreignKeyRow{table: "model_catalog_sources", from: "source_id", to: "source_id"}) {
		t.Fatalf("model_catalog_rows foreign keys = %#v, want source_id -> model_catalog_sources(source_id)", refs)
	}
	if !slices.Contains(refs, foreignKeyRow{table: "model_catalog_sources", from: "provider_id", to: "provider_id"}) {
		t.Fatalf("model_catalog_rows foreign keys = %#v, want provider_id -> model_catalog_sources(provider_id)", refs)
	}
}

func replaceModelCatalogRows(
	t *testing.T,
	globalDB *GlobalDB,
	sourceID string,
	providerID string,
	sourceKind modelcatalog.SourceKind,
	priority int,
	rows []modelcatalog.ModelRow,
) {
	t.Helper()

	if err := globalDB.ReplaceSourceRows(
		testutil.Context(t),
		sourceID,
		providerID,
		rows,
		modelCatalogStatus(sourceID, providerID, sourceKind, priority),
	); err != nil {
		t.Fatalf("ReplaceSourceRows(%s/%s) error = %v", sourceID, providerID, err)
	}
}

func modelCatalogRow(
	sourceID string,
	providerID string,
	modelID string,
	sourceKind modelcatalog.SourceKind,
	priority int,
) modelcatalog.ModelRow {
	available := true
	supportsTools := true
	supportsReasoning := true
	contextWindow := int64(256000)
	maxOutputTokens := int64(32000)
	costInput := 1.25
	costOutput := 10.5
	return modelcatalog.ModelRow{
		SourceID:             sourceID,
		ProviderID:           providerID,
		ModelID:              modelID,
		DisplayName:          strings.ToUpper(modelID),
		SourceKind:           sourceKind,
		Priority:             priority,
		Available:            &available,
		RefreshedAt:          time.Date(2026, 5, 7, 11, 0, 0, 0, time.UTC),
		ExpiresAt:            time.Date(2026, 5, 8, 11, 0, 0, 0, time.UTC),
		ContextWindow:        &contextWindow,
		MaxOutputTokens:      &maxOutputTokens,
		SupportsTools:        &supportsTools,
		SupportsReasoning:    &supportsReasoning,
		CostInputPerMillion:  &costInput,
		CostOutputPerMillion: &costOutput,
	}
}

func modelCatalogStatus(
	sourceID string,
	providerID string,
	sourceKind modelcatalog.SourceKind,
	priority int,
) modelcatalog.SourceStatus {
	now := time.Date(2026, 5, 7, 11, 0, 0, 0, time.UTC)
	return modelcatalog.SourceStatus{
		SourceID:     sourceID,
		ProviderID:   providerID,
		SourceKind:   sourceKind,
		Priority:     priority,
		LastRefresh:  now,
		NextRefresh:  now.Add(24 * time.Hour),
		LastSuccess:  now,
		RefreshState: modelcatalog.RefreshStateSucceeded,
	}
}

func assertModelCatalogModelIDs(t *testing.T, rows []modelcatalog.ModelRow, want []string) {
	t.Helper()

	got := make([]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.ModelID)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("model ids = %#v, want %#v", got, want)
	}
}

func nilModelCatalogTestContext() context.Context {
	return nil
}
