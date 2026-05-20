package settings

import (
	"context"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/config/lifecycle"
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
