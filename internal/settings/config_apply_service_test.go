package settings

import (
	"context"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/config/lifecycle"
	diagnosticcontract "github.com/pedronauck/agh/internal/diagnosticcontract"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/store/globaldb"
)

func TestConfigApplyServiceRecordsLiveApplyAndAdvancesGeneration(t *testing.T) {
	t.Parallel()

	t.Run("Should persist applied live record and advance active generation", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		homePaths := testHomePaths(t)
		writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
		db, err := globaldb.OpenGlobalDB(ctx, homePaths.DatabaseFile)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if err := db.Close(ctx); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})

		service := testService(t, homePaths, Dependencies{
			SkillsRuntime: newFakeSkillsRuntime(testSkill("alpha", false)),
			ApplyRecords:  NewConfigApplyRecordRepository(db.DB(), nil),
		})

		cfg, err := aghconfig.LoadForHome(homePaths)
		if err != nil {
			t.Fatalf("LoadForHome() error = %v", err)
		}
		cfg.Skills.DisabledSkills = []string{"alpha"}

		result, err := service.ApplySection(WithMutationSource(ctx, "http"), SectionUpdateRequest{
			SectionRequest: SectionRequest{Section: SectionSkills},
			Skills:         &cfg.Skills,
		})
		if err != nil {
			t.Fatalf("ApplySection(skills) error = %v", err)
		}
		if !result.Applied {
			t.Fatal("ApplySection(skills).Applied = false, want true")
		}
		if got, want := result.Record.Lifecycle, lifecycle.Live; got != want {
			t.Fatalf("Lifecycle = %q, want %q", got, want)
		}
		if got, want := result.Record.Status, lifecycle.StatusApplied; got != want {
			t.Fatalf("Status = %q, want %q", got, want)
		}
		if got, want := result.Record.Generation, int64(1); got != want {
			t.Fatalf("Generation = %d, want %d", got, want)
		}
		active, err := service.ActiveConfig(ctx)
		if err != nil {
			t.Fatalf("ActiveConfig() error = %v", err)
		}
		if len(active.Skills.DisabledSkills) != 1 || active.Skills.DisabledSkills[0] != "alpha" {
			t.Fatalf("ActiveConfig().Skills.DisabledSkills = %#v, want [alpha]", active.Skills.DisabledSkills)
		}

		records, err := service.ListApplyRecords(ctx, ApplyRecordFilter{Status: lifecycle.StatusApplied})
		if err != nil {
			t.Fatalf("ListApplyRecords(applied) error = %v", err)
		}
		if len(records) != 1 {
			t.Fatalf("ListApplyRecords(applied) len = %d, want 1", len(records))
		}
		if got, want := records[0].Actor, "http"; got != want {
			t.Fatalf("Actor = %q, want %q", got, want)
		}
	})
}

func TestConfigApplyServiceRecordsRestartRequiredWithoutAdvancingGeneration(t *testing.T) {
	t.Parallel()

	t.Run("Should persist blocked record for restart-required change", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		homePaths := testHomePaths(t)
		writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
		db, err := globaldb.OpenGlobalDB(ctx, homePaths.DatabaseFile)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if err := db.Close(ctx); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})

		service := testService(t, homePaths, Dependencies{
			ApplyRecords: NewConfigApplyRecordRepository(db.DB(), nil),
		})

		cfg, err := aghconfig.LoadForHome(homePaths)
		if err != nil {
			t.Fatalf("LoadForHome() error = %v", err)
		}
		general := GeneralSettings{
			Defaults:       cfg.Defaults,
			Limits:         cfg.Limits,
			Permissions:    cfg.Permissions,
			SessionTimeout: cfg.Session.Limits.Timeout,
			HTTP:           cfg.HTTP,
			Daemon:         cfg.Daemon,
		}
		general.HTTP.Port = 2124

		result, err := service.ApplySection(WithMutationSource(ctx, "uds"), SectionUpdateRequest{
			SectionRequest: SectionRequest{Section: SectionGeneral},
			General:        &general,
		})
		if err != nil {
			t.Fatalf("ApplySection(general) error = %v", err)
		}
		if result.Applied {
			t.Fatal("ApplySection(general).Applied = true, want false")
		}
		if got, want := result.Record.Lifecycle, lifecycle.RestartRequired; got != want {
			t.Fatalf("Lifecycle = %q, want %q", got, want)
		}
		if got, want := result.Record.Status, lifecycle.StatusBlocked; got != want {
			t.Fatalf("Status = %q, want %q", got, want)
		}
		if got, want := result.NextAction, lifecycle.NextActionRestartDaemon; got != want {
			t.Fatalf("NextAction = %q, want %q", got, want)
		}
		if got, want := result.Record.Generation, int64(0); got != want {
			t.Fatalf("Generation = %d, want %d", got, want)
		}

		records, err := service.ListApplyRecords(ctx, ApplyRecordFilter{Status: lifecycle.StatusBlocked})
		if err != nil {
			t.Fatalf("ListApplyRecords(blocked) error = %v", err)
		}
		if len(records) != 1 {
			t.Fatalf("ListApplyRecords(blocked) len = %d, want 1", len(records))
		}
		if len(records[0].Diagnostics) == 0 {
			t.Fatal("Diagnostics len = 0, want restart-required diagnostic")
		}
	})
}

func TestConfigApplyServiceProviderOverlayForBuiltinRequiresRestart(t *testing.T) {
	t.Parallel()

	t.Run("Should block active generation when provider overlay replaces builtin provider", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		homePaths := testHomePaths(t)
		writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
		db, err := globaldb.OpenGlobalDB(ctx, homePaths.DatabaseFile)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if err := db.Close(ctx); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})

		service := testService(t, homePaths, Dependencies{
			ApplyRecords: NewConfigApplyRecordRepository(db.DB(), nil),
		})

		result, err := service.ApplyCollectionItem(
			WithMutationSource(ctx, "http"),
			CollectionItemPutRequest{
				CollectionRequest: CollectionRequest{Collection: CollectionProviders},
				Name:              "codex",
				Provider: &ProviderSettings{
					Command: "codex-browser",
				},
			},
		)
		if err != nil {
			t.Fatalf("ApplyCollectionItem(provider codex) error = %v", err)
		}
		if result.Applied {
			t.Fatal("ApplyCollectionItem(provider codex).Applied = true, want false")
		}
		if !result.RestartRequired {
			t.Fatal("ApplyCollectionItem(provider codex).RestartRequired = false, want true")
		}
		if got, want := result.Record.Lifecycle, lifecycle.RestartRequired; got != want {
			t.Fatalf("Lifecycle = %q, want %q", got, want)
		}
		if got, want := result.Record.Status, lifecycle.StatusBlocked; got != want {
			t.Fatalf("Status = %q, want %q", got, want)
		}
	})
}

func TestConfigApplyServiceReloadClassifiesUnknownPathsConservatively(t *testing.T) {
	t.Parallel()

	t.Run("Should block reload when desired config changed outside explicit diff coverage", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		homePaths := testHomePaths(t)
		writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
		db, err := globaldb.OpenGlobalDB(ctx, homePaths.DatabaseFile)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if err := db.Close(ctx); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})

		service := testService(t, homePaths, Dependencies{
			ApplyRecords: NewConfigApplyRecordRepository(db.DB(), nil),
		})
		initial, err := service.Reload(WithMutationSource(ctx, "boot"))
		if err != nil {
			t.Fatalf("Reload(initial) error = %v", err)
		}
		if !initial.Skipped {
			t.Fatal("Reload(initial).Skipped = false, want true")
		}
		records, err := service.ListApplyRecords(ctx, ApplyRecordFilter{})
		if err != nil {
			t.Fatalf("ListApplyRecords(initial) error = %v", err)
		}
		if len(records) != 0 {
			t.Fatalf("initial apply records len = %d, want 0", len(records))
		}

		writeFile(t, homePaths.ConfigFile, baseSettingsConfig()+`
[log]
level = "debug"
`)

		result, err := service.Reload(WithMutationSource(ctx, "cli"))
		if err != nil {
			t.Fatalf("Reload(log-level) error = %v", err)
		}
		if result.Applied {
			t.Fatal("Reload(log-level).Applied = true, want false")
		}
		if got, want := result.Record.Lifecycle, lifecycle.RestartRequired; got != want {
			t.Fatalf("Lifecycle = %q, want %q", got, want)
		}
		if got, want := result.Record.Status, lifecycle.StatusBlocked; got != want {
			t.Fatalf("Status = %q, want %q", got, want)
		}
		if got, want := result.Record.Generation, int64(0); got != want {
			t.Fatalf("Generation = %d, want %d", got, want)
		}
	})
}

func TestConfigApplyServiceRecordsRuntimeReconcileFailures(t *testing.T) {
	t.Parallel()

	t.Run("Should fail apply record without advancing generation when runtime reconcile fails", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		homePaths := testHomePaths(t)
		writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
		db, err := globaldb.OpenGlobalDB(ctx, homePaths.DatabaseFile)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if err := db.Close(ctx); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})

		applier := &fakeConfigRuntimeApplier{
			failures: []ApplyFailure{{
				Subsystem: "mcp",
				Diagnostic: diagnostics.NewItem(
					"config.apply.test_failure",
					diagnosticcontract.CodeConfigPartialFailure,
					diagnosticcontract.CategoryMCP,
					"Test runtime reconcile failed",
					"runtime reconcile failed",
					diagnosticcontract.SeverityError,
					diagnosticcontract.FreshnessLive,
				),
			}},
		}
		service := testService(t, homePaths, Dependencies{
			SkillsRuntime:  newFakeSkillsRuntime(testSkill("alpha", false)),
			ApplyRecords:   NewConfigApplyRecordRepository(db.DB(), nil),
			RuntimeApplier: applier,
		})

		cfg, err := aghconfig.LoadForHome(homePaths)
		if err != nil {
			t.Fatalf("LoadForHome() error = %v", err)
		}
		cfg.Skills.DisabledSkills = []string{"alpha"}

		result, err := service.ApplySection(WithMutationSource(ctx, "http"), SectionUpdateRequest{
			SectionRequest: SectionRequest{Section: SectionSkills},
			Skills:         &cfg.Skills,
		})
		if err != nil {
			t.Fatalf("ApplySection(skills) error = %v", err)
		}
		if result.Applied {
			t.Fatal("ApplySection(skills).Applied = true, want false")
		}
		if got, want := applier.calls, 1; got != want {
			t.Fatalf("runtime applier calls = %d, want %d", got, want)
		}
		if got, want := result.Record.Status, lifecycle.StatusFailed; got != want {
			t.Fatalf("Status = %q, want %q", got, want)
		}
		if got, want := result.Record.Lifecycle, lifecycle.Live; got != want {
			t.Fatalf("Lifecycle = %q, want %q", got, want)
		}
		if got, want := result.Record.Generation, int64(0); got != want {
			t.Fatalf("Generation = %d, want %d", got, want)
		}
		if result.Record.AppliedAt != nil {
			t.Fatalf("AppliedAt = %v, want nil", result.Record.AppliedAt)
		}
		if len(result.PartialFailures) != 1 {
			t.Fatalf("PartialFailures len = %d, want 1", len(result.PartialFailures))
		}
	})
}

func TestConfigApplyServiceFailedRecordsPreserveLifecycleIntent(t *testing.T) {
	t.Parallel()

	t.Run("Should record skills validation failure as live lifecycle", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		homePaths := testHomePaths(t)
		writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
		db, err := globaldb.OpenGlobalDB(ctx, homePaths.DatabaseFile)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if err := db.Close(ctx); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})

		service := testService(t, homePaths, Dependencies{
			ApplyRecords: NewConfigApplyRecordRepository(db.DB(), nil),
		})

		result, err := service.ApplySection(WithMutationSource(ctx, "cli"), SectionUpdateRequest{
			SectionRequest: SectionRequest{Section: SectionSkills},
		})
		if err == nil {
			t.Fatal("ApplySection(skills nil payload) error = nil, want validation error")
		}
		if result.Applied {
			t.Fatal("ApplySection(skills nil payload).Applied = true, want false")
		}
		if got, want := result.Record.Status, lifecycle.StatusFailed; got != want {
			t.Fatalf("Status = %q, want %q", got, want)
		}
		if got, want := result.Record.Lifecycle, lifecycle.Live; got != want {
			t.Fatalf("Lifecycle = %q, want %q", got, want)
		}
	})
}

type fakeConfigRuntimeApplier struct {
	failures []ApplyFailure
	calls    int
}

func (f *fakeConfigRuntimeApplier) ApplyActiveConfig(
	_ context.Context,
	_ *aghconfig.Config,
) []ApplyFailure {
	f.calls++
	return append([]ApplyFailure(nil), f.failures...)
}
