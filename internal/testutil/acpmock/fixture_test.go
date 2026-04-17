package acpmock

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestLoadFixtureParsesMultipleAgentsAndScenarioPrimitives(t *testing.T) {
	t.Parallel()

	t.Run("maps named agents and bridge responses", func(t *testing.T) {
		t.Parallel()

		fixture, err := LoadFixture(filepath.Join("testdata", "multi_agent_fixture.json"))
		if err != nil {
			t.Fatalf("LoadFixture() error = %v", err)
		}

		if got, want := len(fixture.Agents), 2; got != want {
			t.Fatalf("len(fixture.Agents) = %d, want %d", got, want)
		}

		alpha, err := fixture.Agent("alpha")
		if err != nil {
			t.Fatalf("fixture.Agent(alpha) error = %v", err)
		}
		turn, err := alpha.SelectTurn("hello alpha", 1)
		if err != nil {
			t.Fatalf("alpha.SelectTurn() error = %v", err)
		}
		if got, want := turn.Name, "alpha-hello"; got != want {
			t.Fatalf("turn.Name = %q, want %q", got, want)
		}
		if got, want := turn.Steps[0].Kind, StepKindAssistant; got != want {
			t.Fatalf("turn.Steps[0].Kind = %q, want %q", got, want)
		}
		if got, want := turn.Steps[1].Kind, StepKindBridgeContent; got != want {
			t.Fatalf("turn.Steps[1].Kind = %q, want %q", got, want)
		}
	})

	t.Run("maps permission and environment primitives", func(t *testing.T) {
		t.Parallel()

		fixture, err := LoadFixture(filepath.Join("testdata", "permission_env_fixture.json"))
		if err != nil {
			t.Fatalf("LoadFixture() error = %v", err)
		}

		approver, err := fixture.Agent("approver")
		if err != nil {
			t.Fatalf("fixture.Agent(approver) error = %v", err)
		}
		permissionTurn, err := approver.SelectTurn("request permission", 1)
		if err != nil {
			t.Fatalf("approver.SelectTurn() error = %v", err)
		}
		if got, want := permissionTurn.Steps[0].Kind, StepKindPermission; got != want {
			t.Fatalf("permission step kind = %q, want %q", got, want)
		}
		if got, want := permissionTurn.Steps[0].ExpectDecision, "allow-always"; got != want {
			t.Fatalf("permission ExpectDecision = %q, want %q", got, want)
		}

		runner, err := fixture.Agent("runner")
		if err != nil {
			t.Fatalf("fixture.Agent(runner) error = %v", err)
		}
		environmentTurn, err := runner.SelectTurn("run environment", 1)
		if err != nil {
			t.Fatalf("runner.SelectTurn() error = %v", err)
		}
		if got, want := environmentTurn.Steps[0].Kind, StepKindEnvironment; got != want {
			t.Fatalf("environment step kind = %q, want %q", got, want)
		}
		if got, want := environmentTurn.Steps[0].Command, "agh"; got != want {
			t.Fatalf("environment step command = %q, want %q", got, want)
		}
	})
}

func TestRegisterRendersValidatedAgentDefinition(t *testing.T) {
	t.Parallel()

	homePaths := mockHomePaths(t)
	diagnosticsPath := filepath.Join(t.TempDir(), "diag", "alpha.jsonl")

	registration, err := Register(homePaths, RegisterOptions{
		FixturePath:     filepath.Join("testdata", "multi_agent_fixture.json"),
		FixtureAgent:    "alpha",
		AgentName:       "mock-alpha",
		DriverPath:      "/tmp/mock driver/acpmock-driver",
		DiagnosticsPath: diagnosticsPath,
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	loaded, err := aghconfig.LoadAgentDefFile(registration.AgentDefPath)
	if err != nil {
		t.Fatalf("LoadAgentDefFile(%q) error = %v", registration.AgentDefPath, err)
	}
	if got, want := loaded.Name, "mock-alpha"; got != want {
		t.Fatalf("loaded.Name = %q, want %q", got, want)
	}
	if got, want := loaded.Provider, "claude"; got != want {
		t.Fatalf("loaded.Provider = %q, want %q", got, want)
	}
	if got, want := loaded.Command, registration.Command; got != want {
		t.Fatalf("loaded.Command = %q, want %q", got, want)
	}

	cfg := aghconfig.DefaultWithHome(homePaths)
	resolved, err := cfg.ResolveAgent(loaded)
	if err != nil {
		t.Fatalf("ResolveAgent() error = %v", err)
	}
	if got, want := resolved.Provider, "claude"; got != want {
		t.Fatalf("resolved.Provider = %q, want %q", got, want)
	}
	if got, want := resolved.Command, registration.Command; got != want {
		t.Fatalf("resolved.Command = %q, want %q", got, want)
	}
}

func TestReadDiagnosticsParsesJSONLines(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "diag.jsonl")
	want := []DiagnosticsRecord{
		{
			AgentName:   "alpha",
			SessionID:   "sess-1",
			PromptIndex: 1,
			Prompt:      "hello",
			TurnName:    "alpha-hello",
		},
		{
			AgentName:   "beta",
			SessionID:   "sess-2",
			PromptIndex: 2,
			Prompt:      "hello beta",
			TurnName:    "beta-hello",
		},
	}

	data, err := json.Marshal(want[0])
	if err != nil {
		t.Fatalf("json.Marshal(first) error = %v", err)
	}
	second, err := json.Marshal(want[1])
	if err != nil {
		t.Fatalf("json.Marshal(second) error = %v", err)
	}
	if err := os.WriteFile(path, append(append(data, '\n'), append(second, '\n')...), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}

	got, err := ReadDiagnostics(path)
	if err != nil {
		t.Fatalf("ReadDiagnostics() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadDiagnostics() = %#v, want %#v", got, want)
	}
}

func TestLoadFixtureAndParseFixtureValidationErrors(t *testing.T) {
	t.Parallel()

	t.Run("load fixture requires path", func(t *testing.T) {
		t.Parallel()

		if _, err := LoadFixture(" "); err == nil || !strings.Contains(err.Error(), "fixture path is required") {
			t.Fatalf("LoadFixture(empty) error = %v, want required-path error", err)
		}
	})

	t.Run("load fixture reports read error", func(t *testing.T) {
		t.Parallel()

		if _, err := LoadFixture(filepath.Join(t.TempDir(), "missing.json")); err == nil ||
			!strings.Contains(err.Error(), "read fixture") {
			t.Fatalf("LoadFixture(missing) error = %v, want read error", err)
		}
	})

	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "invalid version",
			raw:  `{"version":2,"agents":[{"name":"alpha","provider":"claude","turns":[{"steps":[{"kind":"assistant","text":"hi"}]}]}]}`,
			want: "fixture version 2",
		},
		{
			name: "duplicate agent",
			raw:  `{"version":1,"agents":[{"name":"alpha","provider":"claude","turns":[{"steps":[{"kind":"assistant","text":"hi"}]}]},{"name":"alpha","provider":"claude","turns":[{"steps":[{"kind":"assistant","text":"hi"}]}]}]}`,
			want: "duplicate agent",
		},
		{
			name: "turn match cannot set equals and contains",
			raw:  `{"version":1,"agents":[{"name":"alpha","provider":"claude","turns":[{"match":{"equals":"a","contains":"b"},"steps":[{"kind":"assistant","text":"hi"}]}]}]}`,
			want: "cannot set both equals and contains",
		},
		{
			name: "invalid stop reason",
			raw:  `{"version":1,"agents":[{"name":"alpha","provider":"claude","turns":[{"stop_reason":"bad","steps":[{"kind":"assistant","text":"hi"}]}]}]}`,
			want: "stop_reason",
		},
		{
			name: "invalid permission decision",
			raw:  `{"version":1,"agents":[{"name":"alpha","provider":"claude","turns":[{"steps":[{"kind":"permission","tool_call_id":"perm-1","tool_kind":"edit","expect_decision":"maybe"}]}]}]}`,
			want: "expect_decision",
		},
		{
			name: "environment cwd must be absolute",
			raw:  `{"version":1,"agents":[{"name":"alpha","provider":"claude","turns":[{"steps":[{"kind":"environment_exec","command":"agh","cwd":"relative"}]}]}]}`,
			want: "cwd must be absolute",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := ParseFixture([]byte(tc.raw)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ParseFixture(%s) error = %v, want substring %q", tc.name, err, tc.want)
			}
		})
	}
}

func TestFixtureLookupAndHelperErrors(t *testing.T) {
	t.Parallel()

	fixture, err := LoadFixture(filepath.Join("testdata", "multi_agent_fixture.json"))
	if err != nil {
		t.Fatalf("LoadFixture() error = %v", err)
	}

	if _, err := fixture.Agent(""); err == nil || !strings.Contains(err.Error(), "fixture agent name is required") {
		t.Fatalf("fixture.Agent(empty) error = %v, want required-name error", err)
	}
	if _, err := fixture.Agent("missing"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("fixture.Agent(missing) error = %v, want not-found error", err)
	}

	alpha, err := fixture.Agent("alpha")
	if err != nil {
		t.Fatalf("fixture.Agent(alpha) error = %v", err)
	}
	if _, err := alpha.SelectTurn("hello alpha", 0); err == nil || !strings.Contains(err.Error(), "must be >= 1") {
		t.Fatalf("alpha.SelectTurn(occurrence=0) error = %v, want validation error", err)
	}
	if _, err := alpha.SelectTurn(
		"different prompt",
		1,
	); err == nil ||
		!strings.Contains(err.Error(), "no turn matched") {
		t.Fatalf("alpha.SelectTurn(missing) error = %v, want no-match error", err)
	}
}

func TestRegistrationHelperOverridesAndDiagnosticsErrors(t *testing.T) {
	t.Run("resolve driver path honors override and env override", func(t *testing.T) {
		if got, err := resolveDriverPath("/tmp/custom-driver"); err != nil || got != "/tmp/custom-driver" {
			t.Fatalf("resolveDriverPath(override) = %q, %v, want override path", got, err)
		}

		t.Setenv(driverBinaryEnvVar, "/env/acpmock-driver")
		if got, err := resolveDriverPath(""); err != nil || got != "/env/acpmock-driver" {
			t.Fatalf("resolveDriverPath(env) = %q, %v, want /env/acpmock-driver", got, err)
		}
	})

	t.Run("default diagnostics path uses logs directory", func(t *testing.T) {
		t.Parallel()

		homePaths := mockHomePaths(t)
		got, err := resolveDiagnosticsPath(homePaths, "alpha", "")
		if err != nil {
			t.Fatalf("resolveDiagnosticsPath() error = %v", err)
		}
		want := filepath.Join(homePaths.LogsDir, "acpmock", "alpha.jsonl")
		if got != want {
			t.Fatalf("resolveDiagnosticsPath() = %q, want %q", got, want)
		}
	})

	t.Run("render agent def uses default prompt", func(t *testing.T) {
		t.Parallel()

		content := renderAgentDef("mock-alpha", AgentFixture{Provider: "claude"}, "node driver.js")
		if !strings.Contains(content, "You are mock-alpha.") {
			t.Fatalf("renderAgentDef() = %q, want default prompt", content)
		}
	})

	t.Run("read diagnostics reports invalid json", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "bad.jsonl")
		if err := os.WriteFile(path, []byte("not-json\n"), 0o600); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", path, err)
		}
		if _, err := ReadDiagnostics(path); err == nil || !strings.Contains(err.Error(), "decode diagnostics line 1") {
			t.Fatalf("ReadDiagnostics(invalid) error = %v, want decode error", err)
		}
	})

	t.Run("register requires agents dir", func(t *testing.T) {
		t.Parallel()

		if _, err := Register(aghconfig.HomePaths{}, RegisterOptions{}); err == nil ||
			!strings.Contains(err.Error(), "agents directory is required") {
			t.Fatalf("Register(empty home) error = %v, want agents-dir validation", err)
		}
	})

	t.Run("register reports missing fixture agent", func(t *testing.T) {
		t.Parallel()

		homePaths := mockHomePaths(t)
		_, err := Register(homePaths, RegisterOptions{
			FixturePath:  filepath.Join("testdata", "multi_agent_fixture.json"),
			FixtureAgent: "missing",
			DriverPath:   "/tmp/mock-driver",
		})
		if err == nil || !strings.Contains(err.Error(), "fixture agent") {
			t.Fatalf("Register(missing fixture agent) error = %v, want fixture-agent error", err)
		}
	})
}

func TestValidationAndDriverHelpers(t *testing.T) {
	t.Parallel()

	t.Run("validation helpers reject invalid values", func(t *testing.T) {
		t.Parallel()

		if err := validateToolKind("tool_kind", "bogus"); err == nil {
			t.Fatal("validateToolKind(invalid) error = nil, want non-nil")
		}
		if err := validateToolStatus("status", "bogus"); err == nil {
			t.Fatal("validateToolStatus(invalid) error = nil, want non-nil")
		}
		if err := validatePermissionDecision("decision", "bogus"); err == nil {
			t.Fatal("validatePermissionDecision(invalid) error = nil, want non-nil")
		}
		if (TurnMatch{Occurrence: -1}).Validate("match") == nil {
			t.Fatal("TurnMatch.Validate(negative occurrence) error = nil, want non-nil")
		}
		if (TurnFixture{}).Validate("turn") == nil {
			t.Fatal("TurnFixture.Validate(no steps) error = nil, want non-nil")
		}
		if hasTextPayload("", nil) {
			t.Fatal("hasTextPayload(empty) = true, want false")
		}
	})

	t.Run("driver helpers resolve defaults and omit empty diagnostics", func(t *testing.T) {
		t.Parallel()

		driverPath, err := DefaultDriverPath()
		if err != nil {
			t.Fatalf("DefaultDriverPath() error = %v", err)
		}
		if got, err := resolveDriverPath(""); err != nil || got != driverPath {
			t.Fatalf("resolveDriverPath(default) = %q, %v, want %q", got, err, driverPath)
		}

		command := BuildCommand("/tmp/driver", "/tmp/fixture.json", "alpha", "")
		if strings.Contains(command, "--diagnostics") {
			t.Fatalf("BuildCommand() = %q, want no diagnostics flag", command)
		}
	})

	t.Run("read diagnostics reports missing file", func(t *testing.T) {
		t.Parallel()

		if _, err := ReadDiagnostics(filepath.Join(t.TempDir(), "missing.jsonl")); err == nil ||
			!strings.Contains(err.Error(), "open diagnostics") {
			t.Fatalf("ReadDiagnostics(missing) error = %v, want open error", err)
		}
	})
}

func mockHomePaths(t testing.TB) aghconfig.HomePaths {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	return homePaths
}
