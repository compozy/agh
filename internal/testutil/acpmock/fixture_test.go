package acpmock

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/pedronauck/agh/internal/acp"
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
		turn, err := alpha.SelectTurn("hello alpha", 1, acp.PromptMeta{TurnSource: acp.PromptTurnSourceUser})
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

	t.Run("maps permission and sandbox primitives", func(t *testing.T) {
		t.Parallel()

		fixture, err := LoadFixture(filepath.Join("testdata", "permission_env_fixture.json"))
		if err != nil {
			t.Fatalf("LoadFixture() error = %v", err)
		}

		approver, err := fixture.Agent("approver")
		if err != nil {
			t.Fatalf("fixture.Agent(approver) error = %v", err)
		}
		permissionTurn, err := approver.SelectTurn(
			"request permission",
			1,
			acp.PromptMeta{TurnSource: acp.PromptTurnSourceUser},
		)
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
		sandboxTurn, err := runner.SelectTurn(
			"run sandbox",
			1,
			acp.PromptMeta{TurnSource: acp.PromptTurnSourceNetwork},
		)
		if err != nil {
			t.Fatalf("runner.SelectTurn() error = %v", err)
		}
		if got, want := sandboxTurn.Steps[0].Kind, StepKindSandbox; got != want {
			t.Fatalf("sandbox step kind = %q, want %q", got, want)
		}
		if got, want := sandboxTurn.Steps[0].Command, "agh"; got != want {
			t.Fatalf("sandbox step command = %q, want %q", got, want)
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
		AgentName:       " mock-alpha ",
		DriverPath:      "/tmp/mock driver/acpmock-driver",
		DiagnosticsPath: diagnosticsPath,
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got, want := registration.AgentName, "mock-alpha"; got != want {
		t.Fatalf("registration.AgentName = %q, want %q", got, want)
	}
	if got, want := registration.AgentDefPath, filepath.Join(
		homePaths.AgentsDir,
		"mock-alpha",
		"AGENT.md",
	); got != want {
		t.Fatalf("registration.AgentDefPath = %q, want %q", got, want)
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
			PromptMeta:  acp.PromptMeta{TurnSource: acp.PromptTurnSourceUser},
			TurnName:    "alpha-hello",
			Match: TurnMatch{
				TurnSource: acp.PromptTurnSourceUser,
				UserText:   "hello",
			},
		},
		{
			AgentName:   "beta",
			SessionID:   "sess-2",
			PromptIndex: 2,
			Prompt:      "hello beta",
			PromptMeta: acp.PromptMeta{
				TurnSource: acp.PromptTurnSourceNetwork,
				Network: &acp.PromptNetworkMeta{
					MessageID:   "msg-2",
					Kind:        "say",
					Surface:     "direct",
					DirectID:    "direct_0123456789abcdef0123456789abcdef",
					WorkID:      "work_patch_42",
					ReplyTo:     "msg-root",
					TraceID:     "trace_ops_patch_42",
					CausationID: "msg-root",
					Trust:       "untrusted",
				},
			},
			TurnName: "beta-hello",
			Match: TurnMatch{
				TurnSource: acp.PromptTurnSourceNetwork,
				Network: &TurnMatchNetwork{
					MessageID:   "msg-2",
					Kind:        "say",
					Surface:     "direct",
					DirectID:    "direct_0123456789abcdef0123456789abcdef",
					WorkID:      "work_patch_42",
					ReplyTo:     "msg-root",
					TraceID:     "trace_ops_patch_42",
					CausationID: "msg-root",
					Trust:       "untrusted",
				},
			},
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
			raw:  `{"version":1,"agents":[{"name":"alpha","provider":"claude","turns":[{"match":{"turn_source":"user","user_text":"hi"},"steps":[{"kind":"assistant","text":"hi"}]}]}]}`,
			want: "fixture version 1",
		},
		{
			name: "duplicate agent",
			raw:  `{"version":2,"agents":[{"name":"alpha","provider":"claude","turns":[{"match":{"turn_source":"user","user_text":"hi"},"steps":[{"kind":"assistant","text":"hi"}]}]},{"name":"alpha","provider":"claude","turns":[{"match":{"turn_source":"user","user_text":"hello"},"steps":[{"kind":"assistant","text":"hi"}]}]}]}`,
			want: "duplicate agent",
		},
		{
			name: "legacy matcher fields are rejected",
			raw:  `{"version":2,"agents":[{"name":"alpha","provider":"claude","turns":[{"match":{"equals":"a"},"steps":[{"kind":"assistant","text":"hi"}]}]}]}`,
			want: "unknown field",
		},
		{
			name: "invalid stop reason",
			raw:  `{"version":2,"agents":[{"name":"alpha","provider":"claude","turns":[{"match":{"turn_source":"user","user_text":"hi"},"stop_reason":"bad","steps":[{"kind":"assistant","text":"hi"}]}]}]}`,
			want: "stop_reason",
		},
		{
			name: "invalid permission decision",
			raw:  `{"version":2,"agents":[{"name":"alpha","provider":"claude","turns":[{"match":{"turn_source":"user","user_text":"hi"},"steps":[{"kind":"permission","tool_call_id":"perm-1","tool_kind":"edit","expect_decision":"maybe"}]}]}]}`,
			want: "expect_decision",
		},
		{
			name: "sandbox cwd must be absolute",
			raw:  `{"version":2,"agents":[{"name":"alpha","provider":"claude","turns":[{"match":{"turn_source":"user","user_text":"hi"},"steps":[{"kind":"sandbox_exec","command":"agh","cwd":"relative"}]}]}]}`,
			want: "cwd must be absolute",
		},
		{
			name: "driver control requires payload",
			raw:  `{"version":2,"agents":[{"name":"alpha","provider":"claude","turns":[{"match":{"turn_source":"user","user_text":"hi"},"steps":[{"kind":"driver_control"}]}]}]}`,
			want: "driver_control is required",
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
	trimmedFixture, err := ParseFixture(
		[]byte(
			`{"version":2,"agents":[{"name":" alpha ","provider":"claude","turns":[{"match":{"turn_source":"user","user_text":"hi"},"steps":[{"kind":"assistant","text":"hi"}]}]}]}`,
		),
	)
	if err != nil {
		t.Fatalf("ParseFixture(trimmed name) error = %v", err)
	}
	trimmedAgent, err := trimmedFixture.Agent("alpha")
	if err != nil {
		t.Fatalf("trimmedFixture.Agent(alpha) error = %v", err)
	}
	if got, want := trimmedAgent.Name, "alpha"; got != want {
		t.Fatalf("trimmedAgent.Name = %q, want %q", got, want)
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
		acp.PromptMeta{TurnSource: acp.PromptTurnSourceUser},
	); err == nil ||
		!strings.Contains(err.Error(), "no turn matched") {
		t.Fatalf("alpha.SelectTurn(missing) error = %v, want no-match error", err)
	}

	augmentedPrompt := strings.Join([]string{
		"<agh-situation-context>",
		`{"self":{"session_id":"sess_123","agent_name":"alpha"}}`,
		"</agh-situation-context>",
		"",
		"<current-available-skills>",
		`  <skill name="agh-network">Coordinate over AGH network surfaces.</skill>`,
		"</current-available-skills>",
		"",
		"The <current-available-skills> block above is the authoritative current skill state for this turn.",
		"If it differs from any earlier <available-skills> startup snapshot, trust the current block.",
		"Use `agh__skill_view` to load full instructions for any skill.",
		"Use `agh__skill_view` to read a specific skill resource file when the skill references one.",
		aghCurrentSkillsLastInstructionLine,
		"",
		"Relevant durable memory for this turn:",
		"- Global [user]",
		"  Snippet: remember the harness",
		"Use recalled memory only when it remains consistent with the current repository and runtime state.",
		"",
		"User message:",
		"hello alpha",
	}, "\n")
	turn, err := alpha.SelectTurn(
		augmentedPrompt,
		1,
		acp.PromptMeta{TurnSource: acp.PromptTurnSourceUser},
	)
	if err != nil {
		t.Fatalf("alpha.SelectTurn(augmented prompt) error = %v", err)
	}
	if got, want := turn.Name, "alpha-hello"; got != want {
		t.Fatalf("augmented turn.Name = %q, want %q", got, want)
	}

	networkFixture, err := LoadFixture(filepath.Join("testdata", "network_collaboration_fixture.json"))
	if err != nil {
		t.Fatalf("LoadFixture(network) error = %v", err)
	}
	ops, err := networkFixture.Agent("ops-coordinator")
	if err != nil {
		t.Fatalf("fixture.Agent(ops-coordinator) error = %v", err)
	}
	turn, err = ops.SelectTurn("", 1, acp.PromptMeta{
		TurnSource: acp.PromptTurnSourceNetwork,
		Network: &acp.PromptNetworkMeta{
			MessageID:   "msg_direct_01",
			Kind:        "say",
			Channel:     "builders",
			Surface:     "direct",
			From:        "patch-worker.sess",
			WorkID:      "work_patch_42",
			ReplyTo:     "msg_say_01",
			TraceID:     "trace_ops_patch_42",
			To:          "ops-coordinator.sess",
			CausationID: "msg_say_01",
			Trust:       "untrusted",
		},
	})
	if err != nil {
		t.Fatalf("ops.SelectTurn(network) error = %v", err)
	}
	if got, want := turn.Name, "accept-direct-request"; got != want {
		t.Fatalf("network turn.Name = %q, want %q", got, want)
	}

	capabilityCurator, err := networkFixture.Agent("capability-curator")
	if err != nil {
		t.Fatalf("fixture.Agent(capability-curator) error = %v", err)
	}
	turn, err = capabilityCurator.SelectTurn("", 1, acp.PromptMeta{
		TurnSource: acp.PromptTurnSourceNetwork,
		Network: &acp.PromptNetworkMeta{
			MessageID: "msg_capability_say_01",
			Kind:      "say",
			Channel:   "capabilities",
			Surface:   "thread",
			ThreadID:  "thread_capabilities_main",
			From:      "release-bot.sess",
			To:        "capability-curator.sess",
			Trust:     "untrusted",
		},
	})
	if err != nil {
		t.Fatalf("capabilityCurator.SelectTurn(network) error = %v", err)
	}
	if got, want := turn.Name, "observe-capability-request"; got != want {
		t.Fatalf("capability curator turn.Name = %q, want %q", got, want)
	}
}

func TestTurnMatchNetworkRequiresExactConversationMetadata(t *testing.T) {
	t.Parallel()

	directMatcher := TurnMatchNetwork{
		MessageID:   "msg_direct_01",
		Kind:        "say",
		Channel:     "builders",
		Surface:     "direct",
		DirectID:    "direct_0123456789abcdef0123456789abcdef",
		From:        "patch-worker.sess",
		To:          "ops-coordinator.sess",
		WorkID:      "work_patch_42",
		ReplyTo:     "msg_say_01",
		TraceID:     "trace_ops_patch_42",
		CausationID: "msg_say_01",
		Trust:       "untrusted",
	}
	directMeta := acp.PromptNetworkMeta{
		MessageID:   "msg_direct_01",
		Kind:        "say",
		Channel:     "builders",
		Surface:     "direct",
		DirectID:    "direct_0123456789abcdef0123456789abcdef",
		From:        "patch-worker.sess",
		To:          "ops-coordinator.sess",
		WorkID:      "work_patch_42",
		ReplyTo:     "msg_say_01",
		TraceID:     "trace_ops_patch_42",
		CausationID: "msg_say_01",
		Trust:       "untrusted",
	}
	if !directMatcher.matches(directMeta) {
		t.Fatal("direct matcher did not match exact final conversation metadata")
	}

	directCases := []struct {
		name string
		edit func(*acp.PromptNetworkMeta)
	}{
		{name: "surface", edit: func(meta *acp.PromptNetworkMeta) { meta.Surface = "thread" }},
		{name: "direct id", edit: func(meta *acp.PromptNetworkMeta) { meta.DirectID = "direct_wrong" }},
		{name: "work id", edit: func(meta *acp.PromptNetworkMeta) { meta.WorkID = "work_wrong" }},
		{name: "reply to", edit: func(meta *acp.PromptNetworkMeta) { meta.ReplyTo = "msg_wrong" }},
		{name: "trace id", edit: func(meta *acp.PromptNetworkMeta) { meta.TraceID = "trace_wrong" }},
		{name: "causation id", edit: func(meta *acp.PromptNetworkMeta) { meta.CausationID = "msg_wrong" }},
		{name: "trust", edit: func(meta *acp.PromptNetworkMeta) { meta.Trust = "verified" }},
	}
	for _, tc := range directCases {
		t.Run("direct rejects wrong "+tc.name, func(t *testing.T) {
			t.Parallel()

			meta := directMeta
			tc.edit(&meta)
			if directMatcher.matches(meta) {
				t.Fatalf("direct matcher matched wrong %s metadata: %#v", tc.name, meta)
			}
		})
	}

	threadMatcher := TurnMatchNetwork{
		MessageID: "msg_say_01",
		Kind:      "say",
		Channel:   "builders",
		Surface:   "thread",
		ThreadID:  "thread_builders_main",
		TraceID:   "trace_ops_patch_42",
		Trust:     "untrusted",
	}
	threadMeta := acp.PromptNetworkMeta{
		MessageID: "msg_say_01",
		Kind:      "say",
		Channel:   "builders",
		Surface:   "thread",
		ThreadID:  "thread_builders_main",
		TraceID:   "trace_ops_patch_42",
		Trust:     "untrusted",
	}
	if !threadMatcher.matches(threadMeta) {
		t.Fatal("thread matcher did not match exact final conversation metadata")
	}

	threadCases := []struct {
		name string
		edit func(*acp.PromptNetworkMeta)
	}{
		{name: "surface", edit: func(meta *acp.PromptNetworkMeta) { meta.Surface = "direct" }},
		{name: "thread id", edit: func(meta *acp.PromptNetworkMeta) { meta.ThreadID = "thread_wrong" }},
		{name: "trace id", edit: func(meta *acp.PromptNetworkMeta) { meta.TraceID = "trace_wrong" }},
		{name: "trust", edit: func(meta *acp.PromptNetworkMeta) { meta.Trust = "verified" }},
	}
	for _, tc := range threadCases {
		t.Run("thread rejects wrong "+tc.name, func(t *testing.T) {
			t.Parallel()

			meta := threadMeta
			tc.edit(&meta)
			if threadMatcher.matches(meta) {
				t.Fatalf("thread matcher matched wrong %s metadata: %#v", tc.name, meta)
			}
		})
	}
}

func TestResolveDriverPathHonorsExplicitAndEnvOverrides(t *testing.T) {
	if got, err := resolveDriverPath("/tmp/custom-driver"); err != nil || got != "/tmp/custom-driver" {
		t.Fatalf("resolveDriverPath(override) = %q, %v, want override path", got, err)
	}

	t.Setenv(driverBinaryEnvVar, "/env/acpmock-driver")
	if got, err := resolveDriverPath(""); err != nil || got != "/env/acpmock-driver" {
		t.Fatalf("resolveDriverPath(env) = %q, %v, want /env/acpmock-driver", got, err)
	}
}

func TestRegistrationHelperOverridesAndDiagnosticsErrors(t *testing.T) {
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

	t.Run("resolve diagnostics path rejects unsafe agent names", func(t *testing.T) {
		t.Parallel()

		homePaths := mockHomePaths(t)
		if _, err := resolveDiagnosticsPath(homePaths, "../alpha", ""); err == nil ||
			!strings.Contains(err.Error(), "invalid agent name") {
			t.Fatalf("resolveDiagnosticsPath(unsafe) error = %v, want invalid-agent-name error", err)
		}
	})

	t.Run("resolve diagnostics path requires logs directory without override", func(t *testing.T) {
		t.Parallel()

		homePaths := mockHomePaths(t)
		homePaths.LogsDir = " "
		if _, err := resolveDiagnosticsPath(homePaths, "alpha", ""); err == nil ||
			!strings.Contains(err.Error(), "logs directory is required") {
			t.Fatalf("resolveDiagnosticsPath(blank logs dir) error = %v, want logs-dir validation", err)
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
		if err == nil || !strings.Contains(err.Error(), "lookup fixture agent") {
			t.Fatalf("Register(missing fixture agent) error = %v, want lookup context", err)
		}
	})

	t.Run("register rejects unsafe runtime agent names", func(t *testing.T) {
		t.Parallel()

		homePaths := mockHomePaths(t)
		_, err := Register(homePaths, RegisterOptions{
			FixturePath:  filepath.Join("testdata", "multi_agent_fixture.json"),
			FixtureAgent: "alpha",
			AgentName:    "../escape",
			DriverPath:   "/tmp/mock-driver",
		})
		if err == nil || !strings.Contains(err.Error(), "validate runtime agent name") {
			t.Fatalf("Register(unsafe runtime agent name) error = %v, want runtime-agent validation", err)
		}
	})

	t.Run("register wraps diagnostics path failures", func(t *testing.T) {
		t.Parallel()

		homePaths := mockHomePaths(t)
		homePaths.LogsDir = ""
		_, err := Register(homePaths, RegisterOptions{
			FixturePath:  filepath.Join("testdata", "multi_agent_fixture.json"),
			FixtureAgent: "alpha",
			AgentName:    "alpha",
			DriverPath:   "/tmp/mock-driver",
		})
		if err == nil || !strings.Contains(err.Error(), "resolve diagnostics path") {
			t.Fatalf("Register(blank logs dir) error = %v, want diagnostics-path context", err)
		}
	})
}

func TestBuildDriverBinaryHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	repoRoot, err := repoRootFromCaller()
	if err != nil {
		t.Fatalf("repoRootFromCaller() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = buildDriverBinary(ctx, repoRoot, filepath.Join(t.TempDir(), driverBinaryName()))
	if err == nil {
		t.Fatal("buildDriverBinary(canceled) error = nil, want context cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("buildDriverBinary(canceled) error = %v, want context.Canceled", err)
	}
}

func TestDefaultDriverPathSharesConcurrentBuildResult(t *testing.T) {
	driverBinaryMu.Lock()
	cached := driverBinaryPath
	driverBinaryPath = ""
	driverBinaryMu.Unlock()
	t.Cleanup(func() {
		if strings.TrimSpace(cached) == "" {
			return
		}
		driverBinaryMu.Lock()
		driverBinaryPath = cached
		driverBinaryMu.Unlock()
	})

	const callers = 4
	type result struct {
		path string
		err  error
	}

	results := make(chan result, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Go(func() {
			path, err := DefaultDriverPath()
			results <- result{path: path, err: err}
		})
	}
	wg.Wait()
	close(results)

	var builtPath string
	for item := range results {
		if item.err != nil {
			t.Fatalf("DefaultDriverPath() error = %v", item.err)
		}
		if builtPath == "" {
			builtPath = item.path
			continue
		}
		if item.path != builtPath {
			t.Fatalf("DefaultDriverPath() path = %q, want shared cached path %q", item.path, builtPath)
		}
	}

	if builtPath == "" {
		t.Fatal("DefaultDriverPath() returned empty path for all callers")
	}
	if _, err := os.Stat(builtPath); err != nil {
		t.Fatalf("os.Stat(%q) error = %v", builtPath, err)
	}
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
		if (TurnMatch{TurnSource: acp.PromptTurnSourceUser}).Validate("match") != nil {
			t.Fatal("TurnMatch.Validate(user selector) error != nil, want nil")
		}
		if (TurnMatchNetwork{}).Validate("match.network") == nil {
			t.Fatal("TurnMatchNetwork.Validate(empty) error = nil, want non-nil")
		}
		if (DriverControlStep{Action: DriverControlWriteRawJSONRPC}).Validate("driver_control") == nil {
			t.Fatal("DriverControlStep.Validate(missing raw_jsonrpc) error = nil, want non-nil")
		}
		if (DriverControlStep{Action: DriverControlDisconnect, DelayMS: -1}).Validate("driver_control") == nil {
			t.Fatal("DriverControlStep.Validate(negative delay) error = nil, want non-nil")
		}
		if (DriverControlStep{Action: DriverControlBlockUntilCancel, Async: true}).Validate("driver_control") == nil {
			t.Fatal("DriverControlStep.Validate(async block_until_cancel) error = nil, want non-nil")
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
