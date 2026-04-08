package skills

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestHookRunnerRunHooksReturnsEmptyForNoSkills(t *testing.T) {
	t.Parallel()

	runner, logs := newHookRunnerForTest()
	if got := runner.RunHooks(t.Context(), HookSessionCreated, nil, HookPayload{}); len(got) != 0 {
		t.Fatalf("RunHooks(nil) len = %d, want 0", len(got))
	}
	if got := runner.RunHooks(t.Context(), HookSessionCreated, []*Skill{}, HookPayload{}); len(got) != 0 {
		t.Fatalf("RunHooks(empty) len = %d, want 0", len(got))
	}
	if logs.Len() != 0 {
		t.Fatalf("logs = %q, want empty logs", logs.String())
	}
}

func TestHookRunnerRunHooksFiltersEventAndCapturesPayload(t *testing.T) {
	t.Parallel()

	runner, logs := newHookRunnerForTest()
	script := hookScriptPath(t)

	results := runner.RunHooks(t.Context(), HookSessionCreated, []*Skill{
		newSkillWithHook("created-skill", SourceUser, HookDecl{
			Event:   HookSessionCreated,
			Command: "/bin/sh",
			Args:    []string{script},
			Env: map[string]string{
				"HOOK_TEST_OUTPUT_MODE": "combined",
				"HOOK_TEST_CUSTOM_ENV":  "custom-value",
			},
		}),
		newSkillWithHook("stopped-skill", SourceUser, HookDecl{
			Event:   HookSessionStopped,
			Command: "/bin/sh",
			Args:    []string{script},
			Env: map[string]string{
				"HOOK_TEST_OUTPUT": "should-not-run",
			},
		}),
	}, HookPayload{
		SessionID: "session-123",
		AgentName: "codex",
		Workspace: "/tmp/workspace",
		Event:     "ignored-input-event",
	})

	if len(results) != 1 {
		t.Fatalf("RunHooks() len = %d, want 1", len(results))
	}

	result := results[0]
	if result.SkillName != "created-skill" {
		t.Fatalf("result.SkillName = %q, want %q", result.SkillName, "created-skill")
	}
	if result.Event != HookSessionCreated {
		t.Fatalf("result.Event = %q, want %q", result.Event, HookSessionCreated)
	}
	if result.Error != nil {
		t.Fatalf("result.Error = %v, want nil", result.Error)
	}
	if result.Duration <= 0 {
		t.Fatalf("result.Duration = %s, want > 0", result.Duration)
	}

	payloadJSON, envValue, ok := strings.Cut(result.Output, "|")
	if !ok {
		t.Fatalf("result.Output = %q, want payload|env", result.Output)
	}
	if envValue != "custom-value" {
		t.Fatalf("env value = %q, want %q", envValue, "custom-value")
	}

	var payload HookPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", payloadJSON, err)
	}

	if payload.SessionID != "session-123" {
		t.Fatalf("payload.SessionID = %q, want %q", payload.SessionID, "session-123")
	}
	if payload.AgentName != "codex" {
		t.Fatalf("payload.AgentName = %q, want %q", payload.AgentName, "codex")
	}
	if payload.Workspace != "/tmp/workspace" {
		t.Fatalf("payload.Workspace = %q, want %q", payload.Workspace, "/tmp/workspace")
	}
	if payload.Event != string(HookSessionCreated) {
		t.Fatalf("payload.Event = %q, want %q", payload.Event, HookSessionCreated)
	}
	if logs.Len() != 0 {
		t.Fatalf("logs = %q, want empty logs", logs.String())
	}
}

func TestHookRunnerRunHooksOrdersBySourceAndSkillName(t *testing.T) {
	t.Parallel()

	runner, logs := newHookRunnerForTest(aghconfig.SkillsConfig{
		AllowedMarketplaceMCP: []string{"marketplace-skill"},
	})
	script := hookScriptPath(t)

	results := runner.RunHooks(t.Context(), HookSessionCreated, []*Skill{
		newSkillWithHook("workspace-skill", SourceWorkspace, hookOutput(script, "workspace-skill")),
		newSkillWithHook("beta-user", SourceUser, hookOutput(script, "beta-user")),
		newSkillWithHook("bundled-skill", SourceBundled, hookOutput(script, "bundled-skill")),
		newSkillWithHook("marketplace-skill", SourceMarketplace, hookOutput(script, "marketplace-skill")),
		newSkillWithHook("additional-skill", SourceAdditional, hookOutput(script, "additional-skill")),
		newSkillWithHook("alpha-user", SourceUser, hookOutput(script, "alpha-user")),
	}, HookPayload{})

	got := make([]string, 0, len(results))
	for _, result := range results {
		got = append(got, result.Output)
		if result.Error != nil {
			t.Fatalf("result for %q error = %v, want nil", result.SkillName, result.Error)
		}
	}

	want := []string{
		"bundled-skill",
		"marketplace-skill",
		"alpha-user",
		"beta-user",
		"additional-skill",
		"workspace-skill",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("hook order = %#v, want %#v", got, want)
	}
	if logs.Len() != 0 {
		t.Fatalf("logs = %q, want empty logs", logs.String())
	}
}

func TestHookRunnerRunHooksBlocksMarketplaceHooksWithoutConsent(t *testing.T) {
	t.Parallel()

	runner, logs := newHookRunnerForTest()
	script := hookScriptPath(t)

	results := runner.RunHooks(t.Context(), HookSessionCreated, []*Skill{
		newSkillWithHook("marketplace-skill", SourceMarketplace, hookOutput(script, "marketplace-skill")),
		newSkillWithHook("user-skill", SourceUser, hookOutput(script, "user-skill")),
	}, HookPayload{})

	if len(results) != 1 {
		t.Fatalf("RunHooks() len = %d, want 1 allowed hook result", len(results))
	}
	if got, want := results[0].SkillName, "user-skill"; got != want {
		t.Fatalf("results[0].SkillName = %q, want %q", got, want)
	}
	if got, want := results[0].Output, "user-skill"; got != want {
		t.Fatalf("results[0].Output = %q, want %q", got, want)
	}

	output := logs.String()
	if !strings.Contains(output, "blocked hook") {
		t.Fatalf("logs = %q, want blocked hook warning", output)
	}
	if !strings.Contains(output, "skill_name=marketplace-skill") {
		t.Fatalf("logs = %q, want blocked marketplace skill name", output)
	}
}

func TestHookRunnerRunHooksFailsOpenOnHookError(t *testing.T) {
	t.Parallel()

	runner, logs := newHookRunnerForTest()
	script := hookScriptPath(t)

	results := runner.RunHooks(t.Context(), HookSessionCreated, []*Skill{
		newSkillWithHook("failing-skill", SourceUser, HookDecl{
			Event:   HookSessionCreated,
			Command: "/bin/sh",
			Args:    []string{script},
			Env: map[string]string{
				"HOOK_TEST_OUTPUT":    "before-exit",
				"HOOK_TEST_EXIT_CODE": "7",
				"HOOK_TEST_STDERR":    "hook failed",
			},
		}),
		newSkillWithHook("after-failure", SourceWorkspace, hookOutput(script, "after-failure")),
	}, HookPayload{})

	if len(results) != 2 {
		t.Fatalf("RunHooks() len = %d, want 2", len(results))
	}
	if results[0].Error == nil {
		t.Fatal("results[0].Error = nil, want hook failure")
	}
	if !strings.Contains(results[0].Output, "before-exit") {
		t.Fatalf("results[0].Output = %q, want captured stdout", results[0].Output)
	}
	if results[1].Error != nil {
		t.Fatalf("results[1].Error = %v, want nil", results[1].Error)
	}
	if results[1].Output != "after-failure" {
		t.Fatalf("results[1].Output = %q, want %q", results[1].Output, "after-failure")
	}

	output := logs.String()
	if !strings.Contains(output, "level=WARN") {
		t.Fatalf("logs = %q, want warn log", output)
	}
	if !strings.Contains(output, "skill_name=failing-skill") {
		t.Fatalf("logs = %q, want failing skill name", output)
	}
	if !strings.Contains(output, "event=on_session_created") {
		t.Fatalf("logs = %q, want event field", output)
	}
	if strings.Contains(output, "before-exit") {
		t.Fatalf("logs = %q, want stdout redacted from failure logs", output)
	}
	if strings.Contains(output, "hook failed") {
		t.Fatalf("logs = %q, want stderr redacted from failure logs", output)
	}
	if !strings.Contains(output, "redacted output") {
		t.Fatalf("logs = %q, want redacted output summary", output)
	}
}

func TestHookRunnerRunHooksTimesOut(t *testing.T) {
	t.Parallel()

	runner, logs := newHookRunnerForTest()
	script := hookScriptPath(t)

	results := runner.RunHooks(t.Context(), HookSessionCreated, []*Skill{
		newSkillWithHook("timeout-skill", SourceUser, HookDecl{
			Event:   HookSessionCreated,
			Command: "/bin/sh",
			Args:    []string{script},
			Timeout: 250 * time.Millisecond,
			Env: map[string]string{
				"HOOK_TEST_BUSY_LOOP": "1",
			},
		}),
	}, HookPayload{})

	if len(results) != 1 {
		t.Fatalf("RunHooks() len = %d, want 1", len(results))
	}
	if results[0].Error == nil {
		t.Fatal("results[0].Error = nil, want timeout error")
	}
	if !strings.Contains(results[0].Error.Error(), "timed out") {
		t.Fatalf("results[0].Error = %v, want timeout message", results[0].Error)
	}

	output := logs.String()
	if !strings.Contains(output, "level=WARN") {
		t.Fatalf("logs = %q, want warn log", output)
	}
	if !strings.Contains(output, "skill_name=timeout-skill") {
		t.Fatalf("logs = %q, want timeout skill name", output)
	}
}

func TestHookRunnerRunHooksDoesNotInheritAmbientEnvironment(t *testing.T) {
	t.Setenv("HOOK_TEST_AMBIENT_SECRET", "ambient-secret")

	runner, logs := newHookRunnerForTest()
	results := runner.RunHooks(t.Context(), HookSessionCreated, []*Skill{
		newSkillWithHook("ambient-env-skill", SourceUser, HookDecl{
			Event:   HookSessionCreated,
			Command: "/bin/sh",
			Args:    []string{"-c", `printf '%s' "${HOOK_TEST_AMBIENT_SECRET:-}"`},
		}),
	}, HookPayload{})

	if len(results) != 1 {
		t.Fatalf("RunHooks() len = %d, want 1", len(results))
	}
	if got := results[0].Output; got != "" {
		t.Fatalf("results[0].Output = %q, want ambient secret to be absent", got)
	}
	if logs.Len() != 0 {
		t.Fatalf("logs = %q, want empty logs", logs.String())
	}
}

func TestHookRunnerRunHooksCapsCapturedOutput(t *testing.T) {
	t.Parallel()

	runner, logs := newHookRunnerForTest()
	script := hookScriptPath(t)
	outputValue := strings.Repeat("x", hookCaptureLimitBytes+128)

	results := runner.RunHooks(t.Context(), HookSessionCreated, []*Skill{
		newSkillWithHook("chatty-skill", SourceUser, HookDecl{
			Event:   HookSessionCreated,
			Command: "/bin/sh",
			Args:    []string{script},
			Env: map[string]string{
				"HOOK_TEST_OUTPUT":    outputValue,
				"HOOK_TEST_STDERR":    outputValue,
				"HOOK_TEST_EXIT_CODE": "7",
			},
		}),
	}, HookPayload{})

	if len(results) != 1 {
		t.Fatalf("RunHooks() len = %d, want 1", len(results))
	}
	if !strings.Contains(results[0].Output, hookCaptureTruncateNote) {
		t.Fatalf("results[0].Output = %q, want truncation marker", results[0].Output)
	}

	output := logs.String()
	if strings.Contains(output, outputValue[:64]) {
		t.Fatalf("logs = %q, want large hook output redacted", output)
	}
	if !strings.Contains(output, "redacted output") {
		t.Fatalf("logs = %q, want redacted output summary", output)
	}
}

func newHookRunnerForTest(cfgs ...aghconfig.SkillsConfig) (*HookRunner, *bytes.Buffer) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	cfg := aghconfig.SkillsConfig{}
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	return NewHookRunner(cfg, logger), &logs
}

func newSkillWithHook(name string, source SkillSource, hook HookDecl) *Skill {
	return &Skill{
		Meta: SkillMeta{
			Name:        name,
			Description: "test skill",
		},
		Source: source,
		Hooks:  []HookDecl{hook},
	}
}

func hookOutput(scriptPath string, output string) HookDecl {
	return HookDecl{
		Event:   HookSessionCreated,
		Command: "/bin/sh",
		Args:    []string{scriptPath},
		Env: map[string]string{
			"HOOK_TEST_OUTPUT": output,
		},
	}
}

func hookScriptPath(t *testing.T) string {
	t.Helper()

	path, err := filepath.Abs("testdata/hooks/driver.sh")
	if err != nil {
		t.Fatalf("filepath.Abs(): %v", err)
	}

	return path
}
