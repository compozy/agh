package config

import (
	"testing"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

func TestToolConfigPathPolicy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		path   string
		denial PathDenial
		kind   ValueKind
	}{
		{
			name: "Should allow default agent mutation",
			path: "defaults.agent",
			kind: ConfigValueString,
		},
		{
			name: "Should allow runtime limit mutation",
			path: "limits.max_sessions",
			kind: ConfigValueInt,
		},
		{
			name: "Should allow soul enabled mutation",
			path: "agents.soul.enabled",
			kind: ConfigValueBool,
		},
		{
			name: "Should allow soul max body limit mutation",
			path: "agents.soul.max_body_bytes",
			kind: ConfigValueInt64,
		},
		{
			name: "Should allow soul context projection mutation",
			path: "agents.soul.context_projection_bytes",
			kind: ConfigValueInt64,
		},
		{
			name:   "Should reject daemon socket trust root",
			path:   "daemon.socket",
			denial: ConfigPathTrustForbidden,
		},
		{
			name:   "Should reject HTTP port trust root",
			path:   "http.port",
			denial: ConfigPathTrustForbidden,
		},
		{
			name:   "Should reject provider secret binding",
			path:   "providers.claude.credential_slots[0].secret_ref",
			denial: ConfigPathSecretForbidden,
		},
		{
			name:   "Should reject MCP auth secret path",
			path:   "mcp_servers[0].env.TOKEN",
			denial: ConfigPathSecretForbidden,
		},
		{
			name:   "Should reject sandbox runtime root trust path",
			path:   "sandboxes.default.runtime_root",
			denial: ConfigPathTrustForbidden,
		},
		{
			name:   "Should reject provider command trust root",
			path:   "providers.claude.command",
			denial: ConfigPathTrustForbidden,
		},
		{
			name:   "Should reject memory global dir trust root",
			path:   "memory.global_dir",
			denial: ConfigPathTrustForbidden,
		},
		{
			name:   "Should reject network port trust root",
			path:   "network.port",
			denial: ConfigPathTrustForbidden,
		},
		{
			name:   "Should reject tool policy trust root",
			path:   "tools.policy.external_default",
			denial: ConfigPathTrustForbidden,
		},
		{
			name:   "Should reject extension trust root",
			path:   "extensions.marketplace.registry",
			denial: ConfigPathTrustForbidden,
		},
		{
			name:   "Should reject hook declarations through config tools",
			path:   "hooks.declarations",
			denial: ConfigPathTrustForbidden,
		},
		{
			name:   "Should reject unknown mutable path",
			path:   "unknown.value",
			denial: ConfigPathForbidden,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path, err := ParseDottedConfigPath(tc.path)
			if err != nil {
				t.Fatalf("ParseDottedConfigPath() error = %v", err)
			}
			policy, err := ClassifyToolConfigPath(path)
			if err != nil {
				t.Fatalf("ClassifyToolConfigPath() error = %v", err)
			}
			if policy.Denial != tc.denial {
				t.Fatalf("PathPolicy.Denial = %q, want %q", policy.Denial, tc.denial)
			}
			if tc.denial == ConfigPathAllowed && policy.Kind != tc.kind {
				t.Fatalf("PathPolicy.Kind = %d, want %d", policy.Kind, tc.kind)
			}
		})
	}
}

func TestNormalizeToolConfigValue(t *testing.T) {
	t.Parallel()

	boolValue, err := NormalizeToolConfigValue(ConfigValueBool, "true")
	if err != nil {
		t.Fatalf("NormalizeToolConfigValue(bool) error = %v", err)
	}
	if boolValue != true {
		t.Fatalf("NormalizeToolConfigValue(bool) = %#v, want true", boolValue)
	}

	intValue, err := NormalizeToolConfigValue(ConfigValueInt, float64(7))
	if err != nil {
		t.Fatalf("NormalizeToolConfigValue(int) error = %v", err)
	}
	if intValue != 7 {
		t.Fatalf("NormalizeToolConfigValue(int) = %#v, want 7", intValue)
	}

	int64Value, err := NormalizeToolConfigValue(ConfigValueInt64, "922337203685477580")
	if err != nil {
		t.Fatalf("NormalizeToolConfigValue(int64) error = %v", err)
	}
	if int64Value != int64(922337203685477580) {
		t.Fatalf("NormalizeToolConfigValue(int64) = %#v, want int64", int64Value)
	}

	floatValue, err := NormalizeToolConfigValue(ConfigValueFloat, "1.25")
	if err != nil {
		t.Fatalf("NormalizeToolConfigValue(float) error = %v", err)
	}
	if floatValue != 1.25 {
		t.Fatalf("NormalizeToolConfigValue(float) = %#v, want 1.25", floatValue)
	}

	durationValue, err := NormalizeToolConfigValue(ConfigValueDuration, "5s")
	if err != nil {
		t.Fatalf("NormalizeToolConfigValue(duration) error = %v", err)
	}
	if durationValue != "5s" {
		t.Fatalf("NormalizeToolConfigValue(duration) = %#v, want 5s", durationValue)
	}

	value, err := NormalizeToolConfigValue(ConfigValueStringSlice, []any{"codex", "claude"})
	if err != nil {
		t.Fatalf("NormalizeToolConfigValue(string slice) error = %v", err)
	}
	values, ok := value.([]string)
	if !ok || len(values) != 2 || values[0] != "codex" || values[1] != "claude" {
		t.Fatalf("NormalizeToolConfigValue(string slice) = %#v, want two strings", value)
	}

	if _, err := NormalizeToolConfigValue(ConfigValueDuration, "not-a-duration"); err == nil {
		t.Fatal("NormalizeToolConfigValue(invalid duration) error = nil, want non-nil")
	}
	if _, err := NormalizeToolConfigValue(ConfigValueInt, float64(1.5)); err == nil {
		t.Fatal("NormalizeToolConfigValue(non-integral int) error = nil, want non-nil")
	}
}

func TestRedactedConfigMapEntriesAndDiff(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	cfg := DefaultWithHome(homePaths)
	cfg.Defaults.Agent = "planner"
	cfg.Sandboxes["dev"] = SandboxProfile{
		Backend: "local",
		Env: map[string]string{
			"TOKEN": "secret",
		},
	}

	configMap := RedactedConfigMap(&cfg)
	entries := FlattenConfigEntries(configMap)
	agent, ok := EntryByPath(entries, "defaults.agent")
	if !ok || agent.Value != "planner" {
		t.Fatalf("EntryByPath(defaults.agent) = %#v/%v, want planner", agent, ok)
	}
	soulEnabled, ok := EntryByPath(entries, "agents.soul.enabled")
	if !ok || soulEnabled.Value != true {
		t.Fatalf("EntryByPath(agents.soul.enabled) = %#v/%v, want true", soulEnabled, ok)
	}
	soulMaxBody, ok := EntryByPath(entries, "agents.soul.max_body_bytes")
	if !ok || soulMaxBody.Value != int64(32768) {
		t.Fatalf("EntryByPath(agents.soul.max_body_bytes) = %#v/%v, want 32768", soulMaxBody, ok)
	}
	env, ok := EntryByPath(entries, "sandboxes.dev.env.TOKEN")
	if !ok || env.Value != RedactedValue() || !env.Redacted {
		t.Fatalf("EntryByPath(env) = %#v/%v, want redacted env", env, ok)
	}

	before := FlattenConfigEntries(RedactedConfigMap(&Config{Defaults: DefaultsConfig{Agent: DefaultAgentName}}))
	diff := DiffConfigEntries(before, entries)
	if len(diff) == 0 {
		t.Fatal("DiffConfigEntries() returned no differences")
	}
	found := false
	for _, entry := range diff {
		if entry.Path == "defaults.agent" && entry.After == "planner" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("DiffConfigEntries() = %#v, want defaults.agent change", diff)
	}
}

func TestConfigOverlayHookDeclarationsAndValues(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	target, err := ResolveConfigWriteTarget(homePaths, "", WriteScopeGlobal)
	if err != nil {
		t.Fatalf("ResolveConfigWriteTarget() error = %v", err)
	}

	enabled := true
	readOnly := true
	decl := hookspkg.HookDecl{
		Name:        "tool-audit",
		Event:       hookspkg.HookToolPreCall,
		Source:      hookspkg.HookSourceConfig,
		Mode:        hookspkg.HookModeSync,
		Required:    true,
		Priority:    42,
		PrioritySet: true,
		Timeout:     2 * time.Second,
		Enabled:     &enabled,
		Matcher: hookspkg.HookMatcher{
			AgentName:    "general",
			ToolReadOnly: &readOnly,
		},
		Command: "/bin/echo",
		Args:    []string{"audit"},
		Env: map[string]string{
			"PHASE": "test",
		},
	}
	if _, err := EditConfigOverlay(homePaths, "", target, func(editor *OverlayEditor) error {
		return editor.UpsertArrayTableItem(
			[]string{"hooks", "declarations"},
			"name",
			decl.Name,
			HookDeclarationOverlayValues(decl),
		)
	}); err != nil {
		t.Fatalf("EditConfigOverlay(hook) error = %v", err)
	}

	decls, err := OverlayHookDeclarations(target)
	if err != nil {
		t.Fatalf("OverlayHookDeclarations() error = %v", err)
	}
	if len(decls) != 1 {
		t.Fatalf("len(OverlayHookDeclarations()) = %d, want 1", len(decls))
	}
	got := decls[0]
	if got.Name != decl.Name ||
		got.Event != decl.Event ||
		got.Mode != decl.Mode ||
		!got.Required ||
		got.Priority != decl.Priority ||
		got.Timeout != decl.Timeout ||
		got.Command != decl.Command ||
		got.Env["PHASE"] != "test" ||
		got.Enabled == nil ||
		!*got.Enabled ||
		got.Matcher.ToolReadOnly == nil ||
		!*got.Matcher.ToolReadOnly {
		t.Fatalf("OverlayHookDeclarations() = %#v, want round-tripped hook", got)
	}
}
