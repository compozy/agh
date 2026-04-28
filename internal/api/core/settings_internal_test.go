package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	automationmodel "github.com/pedronauck/agh/internal/automation/model"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
	settingspkg "github.com/pedronauck/agh/internal/settings"
)

func TestSettingsHelperFunctionsAndNilErrorWrappers(t *testing.T) {
	t.Parallel()

	if got := NewSettingsValidationError(nil); got != nil {
		t.Fatalf("NewSettingsValidationError(nil) = %v, want nil", got)
	}
	if got := NewSettingsNotFoundError(nil); got != nil {
		t.Fatalf("NewSettingsNotFoundError(nil) = %v, want nil", got)
	}
	if got := NewSettingsConflictError(nil); got != nil {
		t.Fatalf("NewSettingsConflictError(nil) = %v, want nil", got)
	}
	if got := NewSettingsForbiddenError(nil); got != nil {
		t.Fatalf("NewSettingsForbiddenError(nil) = %v, want nil", got)
	}

	if duration, err := parseOptionalDuration("", "path"); err != nil || duration != 0 {
		t.Fatalf("parseOptionalDuration(empty) = %v, %v", duration, err)
	}
	if duration, err := parseOptionalDuration("1m", "path"); err != nil || duration != time.Minute {
		t.Fatalf("parseOptionalDuration(valid) = %v, %v", duration, err)
	}
	if _, err := parseOptionalDuration("bad", "path"); err == nil {
		t.Fatal("parseOptionalDuration(invalid) error = nil, want non-nil")
	}

	if got := settingsLogTailPollInterval(0); got != defaultPollInterval {
		t.Fatalf("settingsLogTailPollInterval(0) = %v, want %v", got, defaultPollInterval)
	}
	if got := settingsLogTailPollInterval(15 * time.Millisecond); got != 15*time.Millisecond {
		t.Fatalf("settingsLogTailPollInterval(custom) = %v, want 15ms", got)
	}

	if _, ok := findSettingsProvider([]settingspkg.ProviderItem{{Name: "openai"}}, "openai"); !ok {
		t.Fatal("findSettingsProvider() = false, want true")
	}
	if _, ok := findSettingsSandbox([]settingspkg.SandboxItem{{Name: "local"}}, "local"); !ok {
		t.Fatal("findSettingsSandbox() = false, want true")
	}

	fireLimit := automationFireLimitFromPayload(automationmodel.FireLimitConfig{Max: 5, Window: "1m"})
	if fireLimit.Max != 5 || fireLimit.Window != "1m" {
		t.Fatalf("automationFireLimitFromPayload() = %#v", fireLimit)
	}
}

func TestSettingsLogTailFileHelpers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "agh.log")
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	file, info, err := openSettingsLogTailFile(path)
	if err != nil {
		t.Fatalf("openSettingsLogTailFile() error = %v", err)
	}
	defer file.Close()

	rotated, err := settingsLogTailRotated(path, info, file)
	if err != nil {
		t.Fatalf("settingsLogTailRotated(no rotation) error = %v", err)
	}
	if rotated {
		t.Fatal("settingsLogTailRotated(no rotation) = true, want false")
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	rotated, err = settingsLogTailRotated(path, info, file)
	if err != nil {
		t.Fatalf("settingsLogTailRotated(missing path) error = %v", err)
	}
	if !rotated {
		t.Fatal("settingsLogTailRotated(missing path) = false, want true")
	}
}

func TestSettingsConversionErrorBranches(t *testing.T) {
	t.Parallel()

	if _, err := SettingsSectionResponseFromEnvelope(settingspkg.SectionEnvelope{}); err == nil {
		t.Fatal("SettingsSectionResponseFromEnvelope(unknown) error = nil, want non-nil")
	}
	if _, err := SettingsCollectionResponseFromEnvelope(settingspkg.CollectionEnvelope{}); err == nil {
		t.Fatal("SettingsCollectionResponseFromEnvelope(unknown) error = nil, want non-nil")
	}
	if got := StatusForSettingsError(errors.New("settings exploded")); got != 500 {
		t.Fatalf("StatusForSettingsError(default) = %d, want 500", got)
	}
}

func TestSettingsSectionResponseFromEnvelopeRequiresConcreteSectionPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		envelope settingspkg.SectionEnvelope
		want     string
	}{
		{
			name:     "general",
			envelope: settingspkg.SectionEnvelope{Section: settingspkg.SectionGeneral},
			want:     "settings general section is required",
		},
		{
			name:     "memory",
			envelope: settingspkg.SectionEnvelope{Section: settingspkg.SectionMemory},
			want:     "settings memory section is required",
		},
		{
			name:     "skills",
			envelope: settingspkg.SectionEnvelope{Section: settingspkg.SectionSkills},
			want:     "settings skills section is required",
		},
		{
			name:     "automation",
			envelope: settingspkg.SectionEnvelope{Section: settingspkg.SectionAutomation},
			want:     "settings automation section is required",
		},
		{
			name:     "network",
			envelope: settingspkg.SectionEnvelope{Section: settingspkg.SectionNetwork},
			want:     "settings network section is required",
		},
		{
			name:     "observability",
			envelope: settingspkg.SectionEnvelope{Section: settingspkg.SectionObservability},
			want:     "settings observability section is required",
		},
		{
			name:     "hooks extensions",
			envelope: settingspkg.SectionEnvelope{Section: settingspkg.SectionHooksExtensions},
			want:     "settings hooks-extensions section is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := SettingsSectionResponseFromEnvelope(tc.envelope)
			if err == nil {
				t.Fatal("SettingsSectionResponseFromEnvelope() error = nil, want non-nil")
			}
			if err.Error() != tc.want {
				t.Fatalf("SettingsSectionResponseFromEnvelope() error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestSettingsPayloadHelpersRejectInvalidInputs(t *testing.T) {
	t.Parallel()

	if _, err := generalSettingsFromPayload(contract.SettingsGeneralConfigPayload{
		Defaults:       contract.SettingsDefaultsPayload{Agent: "coder"},
		Limits:         contract.SettingsLimitsPayload{MaxSessions: 4, MaxConcurrentAgents: 2},
		Permissions:    contract.SettingsPermissionsPayload{Mode: contract.SettingsPermissionModeApproveReads},
		SessionTimeout: "bad",
		HTTP:           contract.SettingsHTTPPayload{Host: "127.0.0.1", Port: 2123},
		Daemon:         contract.SettingsDaemonPayload{Socket: "/tmp/agh.sock"},
	}); err == nil {
		t.Fatal("generalSettingsFromPayload(invalid timeout) error = nil, want non-nil")
	}

	if _, err := memoryConfigFromPayload(contract.SettingsMemoryConfigPayload{
		Enabled: true,
		Dream: contract.SettingsMemoryDreamPayload{
			Enabled:       true,
			Agent:         "dreamer",
			CheckInterval: "bad",
		},
	}); err == nil {
		t.Fatal("memoryConfigFromPayload(invalid interval) error = nil, want non-nil")
	}

	if _, err := skillsConfigFromPayload(contract.SettingsSkillsConfigPayload{
		Enabled:      true,
		PollInterval: "bad",
		Marketplace:  contract.SettingsMarketplacePayload{Registry: "clawhub"},
	}); err == nil {
		t.Fatal("skillsConfigFromPayload(invalid interval) error = nil, want non-nil")
	}

	if _, err := extensionRateLimitConfigFromPayload(
		contract.SettingsExtensionRateLimitPayload{Requests: 1, Window: "bad"},
		"extensions.resources.snapshot_rate_limit",
	); err == nil {
		t.Fatal("extensionRateLimitConfigFromPayload(invalid window) error = nil, want non-nil")
	}

	if _, err := sandboxProfileFromPayload(contract.SettingsSandboxProfilePayload{
		Backend: "invalid",
	}); err == nil {
		t.Fatal("sandboxProfileFromPayload(invalid backend) error = nil, want non-nil")
	}

	if _, err := hookDeclarationFromPayload(contract.SettingsHookDeclarationPayload{
		Name:         "capture",
		Event:        hookspkg.HookToolPreCall,
		Mode:         hookspkg.HookModeAsync,
		ExecutorKind: hookspkg.HookExecutorSubprocess,
		Command:      "/bin/capture",
		Timeout:      "bad",
		Matcher: hookspkg.HookMatcher{
			ToolID: "agh__read",
		},
	}); err == nil {
		t.Fatal("hookDeclarationFromPayload(invalid timeout) error = nil, want non-nil")
	}

	if _, err := parseSettingsTarget("invalid"); err == nil {
		t.Fatal("parseSettingsTarget(invalid) error = nil, want non-nil")
	}
	if _, err := requiredSettingsPathValue("", "name"); err == nil {
		t.Fatal("requiredSettingsPathValue(empty) error = nil, want non-nil")
	}
	if _, _, err := parseSettingsScope("invalid", ""); err == nil {
		t.Fatal("parseSettingsScope(invalid) error = nil, want non-nil")
	}

	if _, err := extensionsConfigFromPayload(contract.SettingsExtensionsConfigPayload{
		Marketplace: contract.SettingsMarketplacePayload{Registry: "github"},
		Resources: contract.SettingsExtensionResourcesPayload{
			AllowedKinds: []string{string(resources.ResourceKind("tool"))},
			MaxScope:     resources.ResourceScopeKindWorkspace,
			SnapshotRateLimit: contract.SettingsExtensionRateLimitPayload{
				Requests: 1,
				Window:   "1m",
			},
			OperatorWriteRateLimit: contract.SettingsExtensionRateLimitPayload{
				Requests: 1,
				Window:   "1m",
			},
		},
	}); err != nil {
		t.Fatalf("extensionsConfigFromPayload(valid) error = %v", err)
	}

	if _, err := automationSettingsFromPayload(contract.SettingsAutomationConfigPayload{
		Enabled:           true,
		Timezone:          "UTC",
		MaxConcurrentJobs: 1,
		DefaultFireLimit:  automationmodel.FireLimitConfig{Max: 1, Window: "1m"},
	}); err != nil {
		t.Fatalf("automationSettingsFromPayload(valid) error = %v", err)
	}
	if _, err := networkConfigFromPayload(contract.SettingsNetworkConfigPayload{
		Enabled:        true,
		DefaultChannel: "builders",
		Port:           4222,
		MaxPayload:     1024,
		GreetInterval:  5,
		MaxReplayAge:   10,
		MaxQueueDepth:  32,
	}); err != nil {
		t.Fatalf("networkConfigFromPayload(valid) error = %v", err)
	}
	if _, err := observabilityConfigFromPayload(contract.SettingsObservabilityConfigPayload{
		Enabled:        true,
		RetentionDays:  7,
		MaxGlobalBytes: 1024,
		Transcripts: contract.SettingsObservabilityTranscriptPayload{
			Enabled:            true,
			SegmentBytes:       2048,
			MaxBytesPerSession: 4096,
		},
	}); err != nil {
		t.Fatalf("observabilityConfigFromPayload(valid) error = %v", err)
	}
}
