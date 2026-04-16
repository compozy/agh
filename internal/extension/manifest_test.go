package extensionpkg

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/version"
)

func TestLoadManifest_ParsesTOMLAndJSONEquivalently(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	tomlDir := t.TempDir()
	writeFile(t, filepath.Join(tomlDir, manifestTOMLFileName), validManifestTOML)

	jsonDir := t.TempDir()
	writeFile(t, filepath.Join(jsonDir, manifestJSONFileName), validManifestJSON)

	gotTOML, err := LoadManifest(tomlDir)
	if err != nil {
		t.Fatalf("LoadManifest(toml): %v", err)
	}

	gotJSON, err := LoadManifest(jsonDir)
	if err != nil {
		t.Fatalf("LoadManifest(json): %v", err)
	}

	want := expectedManifest()
	t.Run("ShouldMatchExpectedManifestFromTOML", func(t *testing.T) {
		if !reflect.DeepEqual(*gotTOML, want) {
			t.Fatalf("unexpected TOML manifest\n got: %#v\nwant: %#v", *gotTOML, want)
		}
	})
	t.Run("ShouldMatchExpectedManifestFromJSON", func(t *testing.T) {
		if !reflect.DeepEqual(*gotJSON, want) {
			t.Fatalf("unexpected JSON manifest\n got: %#v\nwant: %#v", *gotJSON, want)
		}
	})
	t.Run("ShouldParseTOMLAndJSONEquivalently", func(t *testing.T) {
		if !reflect.DeepEqual(*gotTOML, *gotJSON) {
			t.Fatalf("TOML and JSON manifests differ\n toml: %#v\n json: %#v", *gotTOML, *gotJSON)
		}
	})
}

func TestLoadManifest_FiltersBlankStringEntries(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, manifestTOMLFileName), `[extension]
name = "filtered"
version = "0.2.1"
description = "Normalization coverage"
min_agh_version = "0.5.0"

[resources]
skills = ["skills/", "  ", ""]
agents = ["agents/", "\t"]

[capabilities]
provides = ["memory.backend", "   "]

[actions]
requires = ["sessions/list", ""]

[subprocess]
command = "agh-ext-filtered"
args = ["--config", " ", "\t", "config.toml"]

[security]
capabilities = ["memory.read", "   "]
`)

	manifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if !reflect.DeepEqual(manifest.Resources.Skills, []string{"skills/"}) {
		t.Fatalf("Resources.Skills = %#v, want %#v", manifest.Resources.Skills, []string{"skills/"})
	}
	if !reflect.DeepEqual(manifest.Resources.Agents, []string{"agents/"}) {
		t.Fatalf("Resources.Agents = %#v, want %#v", manifest.Resources.Agents, []string{"agents/"})
	}
	if !reflect.DeepEqual(manifest.Capabilities.Provides, []string{"memory.backend"}) {
		t.Fatalf("Capabilities.Provides = %#v, want %#v", manifest.Capabilities.Provides, []string{"memory.backend"})
	}
	if !reflect.DeepEqual(manifest.Actions.Requires, []string{"sessions/list"}) {
		t.Fatalf("Actions.Requires = %#v, want %#v", manifest.Actions.Requires, []string{"sessions/list"})
	}
	if !reflect.DeepEqual(manifest.Subprocess.Args, []string{"--config", "config.toml"}) {
		t.Fatalf("Subprocess.Args = %#v, want %#v", manifest.Subprocess.Args, []string{"--config", "config.toml"})
	}
	if !reflect.DeepEqual(manifest.Security.Capabilities, []string{"memory.read"}) {
		t.Fatalf("Security.Capabilities = %#v, want %#v", manifest.Security.Capabilities, []string{"memory.read"})
	}
}

func TestNormalizeMCPServersDropsBlankKeysAndUsesDeterministicCollisions(t *testing.T) {
	t.Parallel()

	got := normalizeMCPServers(map[string]MCPServerConfig{
		"  ": {
			Command: "ignored",
		},
		" foo": {
			Command: " first ",
			Env: map[string]string{
				" BAR ": " first ",
			},
		},
		"foo": {
			Command: " second ",
			Env: map[string]string{
				" ":     "ignored",
				" BAR ": "second",
				"BAR":   "final",
			},
		},
	})

	want := map[string]MCPServerConfig{
		"foo": {
			Command: "second",
			Env: map[string]string{
				"BAR": "final",
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeMCPServers() = %#v, want %#v", got, want)
	}
}

func TestLoadManifest_ParsesResourcePublishRequest(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, manifestTOMLFileName), `[extension]
name = "resource-grants"
version = "0.2.1"
min_agh_version = "0.5.0"

[resources.publish]
families = ["tools", "mcp_servers"]
max_scope = "workspace"

[subprocess]
command = "agh-ext-resource-grants"
`)

	manifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(manifest.Resources.Publish.Families, []string{"tools", "mcp_servers"}) {
		t.Fatalf("Resources.Publish.Families = %#v, want tools+mcp_servers", manifest.Resources.Publish.Families)
	}
	if got, want := manifest.Resources.Publish.MaxScope, resources.ResourceScopeKindWorkspace; got != want {
		t.Fatalf("Resources.Publish.MaxScope = %q, want %q", got, want)
	}
}

func TestNormalizeStringMapDropsBlankKeysAndUsesDeterministicCollisions(t *testing.T) {
	t.Parallel()

	got := normalizeStringMap(map[string]string{
		"   ":   "ignored",
		" KEY":  "first",
		"KEY":   "second",
		"\tKEY": "third",
	})

	want := map[string]string{
		"KEY": "second",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeStringMap() = %#v, want %#v", got, want)
	}
}

func TestNormalizeBridgeConfigTrimsSecretSlotsAndSchemaHints(t *testing.T) {
	t.Parallel()

	cfg := normalizeBridgeConfig(BridgeConfig{
		Platform:    " slack ",
		DisplayName: " Slack ",
		SecretSlots: []bridgepkg.BridgeSecretSlot{
			{Name: " bot_token ", Description: " Bot token ", Required: true},
		},
		ConfigSchema: &bridgepkg.BridgeProviderConfigSchema{
			Schema:  " agh.bridge.slack ",
			Version: " v1 ",
		},
	})

	if got, want := cfg.Platform, "slack"; got != want {
		t.Fatalf("cfg.Platform = %q, want %q", got, want)
	}
	if got, want := cfg.DisplayName, "Slack"; got != want {
		t.Fatalf("cfg.DisplayName = %q, want %q", got, want)
	}
	if got, want := cfg.SecretSlots[0].Name, "bot_token"; got != want {
		t.Fatalf("cfg.SecretSlots[0].Name = %q, want %q", got, want)
	}
	if cfg.ConfigSchema == nil {
		t.Fatal("cfg.ConfigSchema = nil, want value")
	}
	if got, want := cfg.ConfigSchema.Schema, "agh.bridge.slack"; got != want {
		t.Fatalf("cfg.ConfigSchema.Schema = %q, want %q", got, want)
	}
}

func TestCloneBoolPointer(t *testing.T) {
	t.Parallel()

	if cloneBoolPointer(nil) != nil {
		t.Fatal("cloneBoolPointer(nil) = non-nil, want nil")
	}

	value := true
	cloned := cloneBoolPointer(&value)
	if cloned == nil || *cloned != value {
		t.Fatalf("cloneBoolPointer(&value) = %#v, want %v", cloned, value)
	}
	if cloned == &value {
		t.Fatal("cloneBoolPointer(&value) returned original pointer")
	}
}

func TestLoadManifest_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name          string
		daemonVersion string
		fileName      string
		content       string
		wantErr       error
		wantField     string
	}{
		{
			name:          "missing name",
			daemonVersion: "0.6.0",
			fileName:      manifestTOMLFileName,
			content: `[extension]
version = "0.2.1"
min_agh_version = "0.5.0"
`,
			wantErr:   ErrManifestInvalid,
			wantField: "name",
		},
		{
			name:          "missing version",
			daemonVersion: "0.6.0",
			fileName:      manifestTOMLFileName,
			content: `[extension]
name = "pgvector-memory"
min_agh_version = "0.5.0"
`,
			wantErr:   ErrManifestInvalid,
			wantField: "version",
		},
		{
			name:          "invalid version semver",
			daemonVersion: "0.6.0",
			fileName:      manifestJSONFileName,
			content: `{
  "extension": {
    "name": "pgvector-memory",
    "version": "latest",
    "min_agh_version": "0.5.0"
  }
}
`,
			wantErr:   ErrManifestInvalid,
			wantField: "version",
		},
		{
			name:          "invalid capability name",
			daemonVersion: "0.6.0",
			fileName:      manifestJSONFileName,
			content: `{
  "extension": {
    "name": "pgvector-memory",
    "version": "0.2.1",
    "min_agh_version": "0.5.0"
  },
  "capabilities": {
    "provides": ["bad capability"]
  }
}
`,
			wantErr:   ErrManifestInvalid,
			wantField: "capabilities.provides[0]",
		},
		{
			name:          "incompatible minimum agh version",
			daemonVersion: "0.4.0",
			fileName:      manifestTOMLFileName,
			content:       validManifestTOML,
			wantErr:       ErrManifestIncompatible,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withDaemonVersion(t, tc.daemonVersion)

			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, tc.fileName), tc.content)

			_, err := LoadManifest(dir)
			if err == nil {
				t.Fatal("LoadManifest() error = nil, want non-nil")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("LoadManifest() error = %v, want %v", err, tc.wantErr)
			}

			if tc.wantField == "" {
				return
			}

			var validationErr *ManifestValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("LoadManifest() error = %T, want *ManifestValidationError", err)
			}
			if validationErr.Field != tc.wantField {
				t.Fatalf("validation field = %q, want %q", validationErr.Field, tc.wantField)
			}
		})
	}
}

func TestManifestValidateRejectsDaemonOnlyResourcePublishFamily(t *testing.T) {
	t.Parallel()

	manifest := expectedManifest()
	manifest.Resources.Publish = ResourceGrantRequest{
		Families: []string{"bridge_instances"},
		MaxScope: resources.ResourceScopeKindGlobal,
	}

	err := manifest.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}

	var validationErr *ManifestValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Validate() error type = %T, want *ManifestValidationError", err)
	}
	if got, want := validationErr.Field, "resources.publish"; got != want {
		t.Fatalf("Validate() field = %q, want %q", got, want)
	}
}

func TestManifestValidateRejectsInvalidResourcePublishScope(t *testing.T) {
	t.Parallel()

	manifest := expectedManifest()
	manifest.Resources.Publish = ResourceGrantRequest{
		Families: []string{"tools"},
		MaxScope: resources.ResourceScopeKind("session"),
	}

	err := manifest.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}

	var validationErr *ManifestValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Validate() error type = %T, want *ManifestValidationError", err)
	}
	if got, want := validationErr.Field, "resources.publish"; got != want {
		t.Fatalf("Validate() field = %q, want %q", got, want)
	}
}

func TestLoadManifest_PrefersTOMLWhenBothFilesExist(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, manifestTOMLFileName), validManifestTOML)
	writeFile(t, filepath.Join(dir, manifestJSONFileName), `{
  "extension": {
    "name": "json-fallback",
    "version": "0.2.1",
    "min_agh_version": "0.5.0"
  }
}`)

	manifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if manifest.Name != expectedManifest().Name {
		t.Fatalf("manifest.Name = %q, want %q", manifest.Name, expectedManifest().Name)
	}
}

func TestLoadManifest_ReturnsTypedNotFoundError(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("LoadManifest() error = nil, want ErrManifestNotFound")
	}
	if !errors.Is(err, ErrManifestNotFound) {
		t.Fatalf("LoadManifest() error = %v, want ErrManifestNotFound", err)
	}

	var notFoundErr *ManifestNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("LoadManifest() error = %T, want *ManifestNotFoundError", err)
	}
	if notFoundErr.Dir != dir {
		t.Fatalf("ManifestNotFoundError.Dir = %q, want %q", notFoundErr.Dir, dir)
	}
}

func TestLoadManifest_AcceptsUnknownTopLevelSections(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, manifestJSONFileName), `{
  "extension": {
    "name": "future-friendly",
    "version": "0.2.1",
    "min_agh_version": "0.5.0"
  },
  "future": {
    "mode": "enabled"
  }
}`)

	manifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if manifest.Name != "future-friendly" {
		t.Fatalf("manifest.Name = %q, want %q", manifest.Name, "future-friendly")
	}
}

func TestLoadManifest_RejectsConflictingRootAndWrappedValues(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, manifestJSONFileName), `{
  "name": "root-name",
  "extension": {
    "name": "wrapped-name",
    "version": "0.2.1",
    "min_agh_version": "0.5.0"
  }
}`)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("LoadManifest() error = nil, want conflict error")
	}
	if !errors.Is(err, ErrManifestInvalid) {
		t.Fatalf("LoadManifest() error = %v, want ErrManifestInvalid", err)
	}

	var validationErr *ManifestValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("LoadManifest() error = %T, want *ManifestValidationError", err)
	}
	if validationErr.Field != "name" {
		t.Fatalf("validation field = %q, want %q", validationErr.Field, "name")
	}
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name    string
		payload string
		want    Duration
		wantErr bool
	}{
		{
			name:    "string",
			payload: `"5s"`,
			want:    duration(5 * time.Second),
		},
		{
			name:    "nanoseconds",
			payload: `5000000000`,
			want:    duration(5 * time.Second),
		},
		{
			name:    "null",
			payload: `null`,
			want:    0,
		},
		{
			name:    "invalid",
			payload: `"nope"`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got Duration
			err := json.Unmarshal([]byte(tc.payload), &got)
			if tc.wantErr {
				if err == nil {
					t.Fatal("json.Unmarshal() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("duration = %v, want %v", time.Duration(got), time.Duration(tc.want))
			}
		})
	}
}

func TestParseSemanticVersion_PrereleaseComparison(t *testing.T) {
	alpha1, ok := parseSemanticVersion("1.2.3-alpha.1+build.5")
	if !ok {
		t.Fatal("parseSemanticVersion(alpha.1) = false, want true")
	}

	alpha2, ok := parseSemanticVersion("1.2.3-alpha.2")
	if !ok {
		t.Fatal("parseSemanticVersion(alpha.2) = false, want true")
	}

	release, ok := parseSemanticVersion("1.2.3")
	if !ok {
		t.Fatal("parseSemanticVersion(release) = false, want true")
	}

	if compareSemanticVersions(alpha1, alpha2) >= 0 {
		t.Fatalf("compareSemanticVersions(alpha1, alpha2) = %d, want < 0", compareSemanticVersions(alpha1, alpha2))
	}
	if compareSemanticVersions(release, alpha2) <= 0 {
		t.Fatalf("compareSemanticVersions(release, alpha2) = %d, want > 0", compareSemanticVersions(release, alpha2))
	}
}

func TestManifestValidate_AllowsWildcardSecurityCapability(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	manifest := expectedManifest()
	manifest.Security.Capabilities = []string{"*"}

	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestManifestValidate_RejectsInvalidActionName(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	manifest := expectedManifest()
	manifest.Actions.Requires = []string{"bad action"}

	err := manifest.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want ErrManifestInvalid")
	}
	if !errors.Is(err, ErrManifestInvalid) {
		t.Fatalf("Validate() error = %v, want ErrManifestInvalid", err)
	}

	var validationErr *ManifestValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Validate() error = %T, want *ManifestValidationError", err)
	}
	if validationErr.Field != "actions.requires[0]" {
		t.Fatalf("validation field = %q, want %q", validationErr.Field, "actions.requires[0]")
	}
}

func TestManifestValidate_RequiresBridgeMetadataForBridgeAdapters(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	t.Run("Should reject bridge adapters without platform metadata", func(t *testing.T) {
		manifest := expectedManifest()
		manifest.Capabilities.Provides = []string{extensionprotocol.CapabilityProvideBridgeAdapter}

		err := manifest.Validate()
		if err == nil {
			t.Fatal("Validate() error = nil, want ErrManifestInvalid")
		}
		if !errors.Is(err, ErrManifestInvalid) {
			t.Fatalf("Validate() error = %v, want ErrManifestInvalid", err)
		}

		var validationErr *ManifestValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("Validate() error = %T, want *ManifestValidationError", err)
		}
		if validationErr.Field != "bridge.platform" {
			t.Fatalf("validation field = %q, want %q", validationErr.Field, "bridge.platform")
		}
	})

	t.Run("Should reject bridge adapters without display name metadata", func(t *testing.T) {
		manifest := expectedManifest()
		manifest.Capabilities.Provides = []string{extensionprotocol.CapabilityProvideBridgeAdapter}
		manifest.Bridge.Platform = "telegram"

		err := manifest.Validate()
		if err == nil {
			t.Fatal("Validate() error = nil, want ErrManifestInvalid")
		}

		var validationErr *ManifestValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("Validate() error = %T, want *ManifestValidationError", err)
		}
		if validationErr.Field != "bridge.display_name" {
			t.Fatalf("validation field = %q, want %q", validationErr.Field, "bridge.display_name")
		}
	})

	t.Run("Should accept bridge adapters with complete bridge metadata", func(t *testing.T) {
		manifest := expectedManifest()
		manifest.Capabilities.Provides = []string{extensionprotocol.CapabilityProvideBridgeAdapter}
		manifest.Bridge.Platform = "telegram"
		manifest.Bridge.DisplayName = "Telegram"

		if err := manifest.Validate(); err != nil {
			t.Fatalf("Validate() with bridge metadata error = %v", err)
		}
	})
}

func TestManifestValidate_ValidatesBridgeSecretSlotsAndConfigSchemaHints(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	t.Run("Should reject bridge secret slots without names", func(t *testing.T) {
		manifest := expectedManifest()
		manifest.Capabilities.Provides = []string{extensionprotocol.CapabilityProvideBridgeAdapter}
		manifest.Bridge.Platform = "slack"
		manifest.Bridge.DisplayName = "Slack"
		manifest.Bridge.SecretSlots = []bridgepkg.BridgeSecretSlot{{Required: true}}

		err := manifest.Validate()
		if err == nil {
			t.Fatal("Validate() error = nil, want ErrManifestInvalid")
		}

		var validationErr *ManifestValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("Validate() error = %T, want *ManifestValidationError", err)
		}
		if got, want := validationErr.Field, "bridge.secret_slots[0]"; got != want {
			t.Fatalf("validation field = %q, want %q", got, want)
		}
	})

	t.Run("Should reject duplicate bridge secret slot names", func(t *testing.T) {
		manifest := expectedManifest()
		manifest.Capabilities.Provides = []string{extensionprotocol.CapabilityProvideBridgeAdapter}
		manifest.Bridge.Platform = "slack"
		manifest.Bridge.DisplayName = "Slack"
		manifest.Bridge.SecretSlots = []bridgepkg.BridgeSecretSlot{
			{Name: "bot_token", Required: true},
			{Name: " bot_token ", Required: true},
		}

		err := manifest.Validate()
		if err == nil {
			t.Fatal("Validate() error = nil, want ErrManifestInvalid")
		}

		var validationErr *ManifestValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("Validate() error = %T, want *ManifestValidationError", err)
		}
		if got, want := validationErr.Field, "bridge.secret_slots[1].name"; got != want {
			t.Fatalf("validation field = %q, want %q", got, want)
		}
	})

	t.Run("Should accept bridge secret slots and config schema hints", func(t *testing.T) {
		manifest := expectedManifest()
		manifest.Capabilities.Provides = []string{extensionprotocol.CapabilityProvideBridgeAdapter}
		manifest.Bridge.Platform = "slack"
		manifest.Bridge.DisplayName = "Slack"
		manifest.Bridge.SecretSlots = []bridgepkg.BridgeSecretSlot{
			{Name: "bot_token", Description: "Bot OAuth token", Required: true},
			{Name: "signing_secret", Description: "Request signing secret", Required: true},
		}
		manifest.Bridge.ConfigSchema = &bridgepkg.BridgeProviderConfigSchema{
			Schema:  "agh.bridge.slack",
			Version: "v1",
		}

		if err := manifest.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})
}

func TestManifestHelpers_ErrorFormattingAndDurationMethods(t *testing.T) {
	notFound := &ManifestNotFoundError{
		Dir:   "/tmp/ext",
		Paths: []string{"/tmp/ext/extension.toml", "/tmp/ext/extension.json"},
	}
	if got := notFound.Error(); got == "" {
		t.Fatal("ManifestNotFoundError.Error() returned empty string")
	}

	validationErr := &ManifestValidationError{
		Field:   "version",
		Value:   "latest",
		Message: "must be a semantic version",
	}
	if got := validationErr.Error(); got == "" {
		t.Fatal("ManifestValidationError.Error() returned empty string")
	}

	compatibilityErr := &ManifestCompatibilityError{
		CurrentVersion: "0.4.0",
		MinVersion:     "0.5.0",
	}
	if got := compatibilityErr.Error(); got == "" {
		t.Fatal("ManifestCompatibilityError.Error() returned empty string")
	}

	zero := duration(0)
	if !zero.IsZero() {
		t.Fatal("Duration.IsZero() = false, want true")
	}

	value := duration(5 * time.Second)
	if value.IsZero() {
		t.Fatal("Duration.IsZero() = true, want false")
	}
	if value.String() != "5s" {
		t.Fatalf("Duration.String() = %q, want %q", value.String(), "5s")
	}

	text, err := value.MarshalText()
	if err != nil {
		t.Fatalf("Duration.MarshalText() error = %v", err)
	}
	if string(text) != "5s" {
		t.Fatalf("Duration.MarshalText() = %q, want %q", string(text), "5s")
	}

	encoded, err := value.MarshalJSON()
	if err != nil {
		t.Fatalf("Duration.MarshalJSON() error = %v", err)
	}
	if string(encoded) != `"5s"` {
		t.Fatalf("Duration.MarshalJSON() = %s, want %s", string(encoded), `"5s"`)
	}
}

func TestLoadManifest_RejectsManifestDirectoryEntries(t *testing.T) {
	dir := t.TempDir()

	if err := os.Mkdir(filepath.Join(dir, manifestTOMLFileName), 0o700); err != nil {
		t.Fatalf("os.Mkdir(toml manifest dir): %v", err)
	}

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("LoadManifest() error = nil, want non-nil")
	}
}

func TestSemanticVersion_HelperValidation(t *testing.T) {
	if _, ok := parseSemanticVersion("1.2"); ok {
		t.Fatal("parseSemanticVersion(1.2) = true, want false")
	}
	if _, ok := parseSemanticVersion("1.2.3+build..5"); ok {
		t.Fatal("parseSemanticVersion(invalid build metadata) = true, want false")
	}

	if !validIdentifierPart("memory") {
		t.Fatal("validIdentifierPart(memory) = false, want true")
	}
	if validIdentifierPart("1memory") {
		t.Fatal("validIdentifierPart(1memory) = true, want false")
	}

	if !validIdentifierList("alpha.1", false) {
		t.Fatal("validIdentifierList(alpha.1) = false, want true")
	}
	if validIdentifierList("alpha..1", false) {
		t.Fatal("validIdentifierList(alpha..1) = true, want false")
	}

	if !validPrereleasePart("alpha-1") {
		t.Fatal("validPrereleasePart(alpha-1) = false, want true")
	}
	if validPrereleasePart("alpha!") {
		t.Fatal("validPrereleasePart(alpha!) = true, want false")
	}

	left, ok := parseSemanticVersion("1.2.3-alpha.beta")
	if !ok {
		t.Fatal("parseSemanticVersion(alpha.beta) = false, want true")
	}
	right, ok := parseSemanticVersion("1.2.3-alpha.1")
	if !ok {
		t.Fatal("parseSemanticVersion(alpha.1) = false, want true")
	}
	if compareSemanticVersions(left, right) <= 0 {
		t.Fatalf("compareSemanticVersions(alpha.beta, alpha.1) = %d, want > 0", compareSemanticVersions(left, right))
	}
}

func withDaemonVersion(t *testing.T, current string) {
	t.Helper()

	t.Cleanup(version.OverrideVersionForTesting(current))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q): %v", path, err)
	}
}

func duration(value time.Duration) Duration {
	return Duration(value)
}

func intPointer(value int) *int {
	return &value
}

func expectedManifest() Manifest {
	return Manifest{
		Name:          "pgvector-memory",
		Version:       "0.2.1",
		Description:   "PostgreSQL pgvector memory backend for AGH",
		MinAGHVersion: "0.5.0",
		Resources: ResourcesConfig{
			Skills: []string{"skills/"},
			Agents: []string{"agents/"},
			Hooks: []HookConfig{
				{
					Name:     "workspace-context",
					Event:    "prompt.post_assemble",
					Mode:     "sync",
					Priority: intPointer(20),
					Timeout:  duration(5 * time.Second),
					Matcher: HookMatcherConfig{
						WorkspaceRoot: "{{workspace_root}}",
						ToolName:      "write_file",
					},
					Executor: HookExecutorConfig{
						Kind:    "subprocess",
						Command: "node",
						Args:    []string{"dist/index.js", "--hook", "prompt_post_assemble"},
						Env: map[string]string{
							"NODE_ENV": "production",
						},
					},
				},
			},
			MCPServers: map[string]MCPServerConfig{
				"kubectl": {
					Command: "mcp-kubectl",
					Args:    []string{"--context", "production"},
					Env: map[string]string{
						"KUBECONFIG": "{{env:KUBECONFIG}}",
					},
				},
			},
		},
		Capabilities: CapabilitiesConfig{
			Provides: []string{"memory.backend"},
		},
		Actions: ActionsConfig{
			Requires: []string{"sessions/list", "sessions/events"},
		},
		Subprocess: SubprocessConfig{
			Command:             "agh-ext-pgvector",
			Args:                []string{"--config", "{{config_dir}}/pgvector.toml"},
			HealthCheckInterval: duration(30 * time.Second),
			ShutdownTimeout:     duration(10 * time.Second),
			Env: map[string]string{
				"PGVECTOR_URL": "{{env:PGVECTOR_URL}}",
			},
		},
		Security: SecurityConfig{
			Capabilities: []string{"memory.read", "memory.write", "session.read"},
		},
	}
}

const validManifestTOML = `[extension]
name = "pgvector-memory"
version = "0.2.1"
description = "PostgreSQL pgvector memory backend for AGH"
min_agh_version = "0.5.0"

[resources]
skills = ["skills/"]
agents = ["agents/"]

[[resources.hooks]]
name = "workspace-context"
event = "prompt.post_assemble"
mode = "sync"
priority = 20
timeout = "5s"
executor.kind = "subprocess"
executor.command = "node"
executor.args = ["dist/index.js", "--hook", "prompt_post_assemble"]
executor.env = { NODE_ENV = "production" }

[resources.hooks.matcher]
workspace_root = "{{workspace_root}}"
tool_name = "write_file"

[resources.mcp_servers.kubectl]
command = "mcp-kubectl"
args = ["--context", "production"]
env = { KUBECONFIG = "{{env:KUBECONFIG}}" }

[capabilities]
provides = ["memory.backend"]

[actions]
requires = ["sessions/list", "sessions/events"]

[subprocess]
command = "agh-ext-pgvector"
args = ["--config", "{{config_dir}}/pgvector.toml"]
health_check_interval = "30s"
shutdown_timeout = "10s"

[subprocess.env]
PGVECTOR_URL = "{{env:PGVECTOR_URL}}"

[security]
capabilities = ["memory.read", "memory.write", "session.read"]

[future]
mode = "enabled"
`

const validManifestJSON = `{
  "extension": {
    "name": "pgvector-memory",
    "version": "0.2.1",
    "description": "PostgreSQL pgvector memory backend for AGH",
    "min_agh_version": "0.5.0"
  },
  "resources": {
    "skills": ["skills/"],
    "agents": ["agents/"],
    "hooks": [
      {
        "name": "workspace-context",
        "event": "prompt.post_assemble",
        "mode": "sync",
        "priority": 20,
        "timeout": "5s",
        "matcher": {
          "workspace_root": "{{workspace_root}}",
          "tool_name": "write_file"
        },
        "executor": {
          "kind": "subprocess",
          "command": "node",
          "args": ["dist/index.js", "--hook", "prompt_post_assemble"],
          "env": {
            "NODE_ENV": "production"
          }
        }
      }
    ],
    "mcp_servers": {
      "kubectl": {
        "command": "mcp-kubectl",
        "args": ["--context", "production"],
        "env": {
          "KUBECONFIG": "{{env:KUBECONFIG}}"
        }
      }
    }
  },
  "capabilities": {
    "provides": ["memory.backend"]
  },
  "actions": {
    "requires": ["sessions/list", "sessions/events"]
  },
  "subprocess": {
    "command": "agh-ext-pgvector",
    "args": ["--config", "{{config_dir}}/pgvector.toml"],
    "health_check_interval": "30s",
    "shutdown_timeout": "10s",
    "env": {
      "PGVECTOR_URL": "{{env:PGVECTOR_URL}}"
    }
  },
  "security": {
    "capabilities": ["memory.read", "memory.write", "session.read"]
  },
  "future": {
    "mode": "enabled"
  }
}`
