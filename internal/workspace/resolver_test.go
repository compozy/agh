package workspace

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestResolveRoutesByIdentifierType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       func(root string, ws Workspace) string
		assertCalls func(t *testing.T, store *mockWorkspaceStore)
	}{
		{
			name: "id",
			input: func(_ string, ws Workspace) string {
				return ws.ID
			},
			assertCalls: func(t *testing.T, store *mockWorkspaceStore) {
				t.Helper()
				if got := len(store.getWorkspaceCalls); got != 1 {
					t.Fatalf("GetWorkspace() calls = %d, want 1", got)
				}
				if got := len(store.getByNameCalls); got != 0 {
					t.Fatalf("GetWorkspaceByName() calls = %d, want 0", got)
				}
				if got := len(store.getByPathCalls); got != 0 {
					t.Fatalf("GetWorkspaceByPath() calls = %d, want 0", got)
				}
			},
		},
		{
			name: "name",
			input: func(_ string, ws Workspace) string {
				return ws.Name
			},
			assertCalls: func(t *testing.T, store *mockWorkspaceStore) {
				t.Helper()
				if got := len(store.getWorkspaceCalls); got != 0 {
					t.Fatalf("GetWorkspace() calls = %d, want 0", got)
				}
				if got := len(store.getByNameCalls); got != 1 {
					t.Fatalf("GetWorkspaceByName() calls = %d, want 1", got)
				}
				if got := len(store.getByPathCalls); got != 0 {
					t.Fatalf("GetWorkspaceByPath() calls = %d, want 0", got)
				}
			},
		},
		{
			name: "absolute path",
			input: func(root string, _ Workspace) string {
				return root
			},
			assertCalls: func(t *testing.T, store *mockWorkspaceStore) {
				t.Helper()
				if got := len(store.getWorkspaceCalls); got != 0 {
					t.Fatalf("GetWorkspace() calls = %d, want 0", got)
				}
				if got := len(store.getByNameCalls); got != 0 {
					t.Fatalf("GetWorkspaceByName() calls = %d, want 0", got)
				}
				if got := len(store.getByPathCalls); got != 1 {
					t.Fatalf("GetWorkspaceByPath() calls = %d, want 1", got)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			homePaths := newTestHomePaths(t)
			root := t.TempDir()
			ws := Workspace{ID: "ws_route", RootDir: mustCanonicalRoot(t, root), Name: "repo"}

			store := newMockWorkspaceStore(ws)
			loader := &countingConfigLoader{cfg: validConfig(homePaths)}
			resolver := newTestResolver(t, store,
				WithHomePaths(homePaths),
				WithConfigLoader(loader.Load),
			)

			resolved, err := resolver.Resolve(ctx, tt.input(root, ws))
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if resolved.ID != ws.ID {
				t.Fatalf("Resolve() ID = %q, want %q", resolved.ID, ws.ID)
			}

			tt.assertCalls(t, store)
		})
	}
}

func TestResolveOrRegisterExistingWorkspace(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := newTestHomePaths(t)
	root := t.TempDir()
	ws := Workspace{ID: "ws_existing", RootDir: mustCanonicalRoot(t, root), Name: "repo"}

	store := newMockWorkspaceStore(ws)
	loader := &countingConfigLoader{cfg: validConfig(homePaths)}
	resolver := newTestResolver(t, store,
		WithHomePaths(homePaths),
		WithConfigLoader(loader.Load),
	)

	resolved, err := resolver.ResolveOrRegister(ctx, root)
	if err != nil {
		t.Fatalf("ResolveOrRegister() error = %v", err)
	}
	if resolved.ID != ws.ID {
		t.Fatalf("ResolveOrRegister() ID = %q, want %q", resolved.ID, ws.ID)
	}
	if got := len(store.insertCalls); got != 0 {
		t.Fatalf("InsertWorkspace() calls = %d, want 0", got)
	}
}

func TestResolveOrRegisterAutoRegisterDedupesNameAndPrefixesID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := newTestHomePaths(t)
	root := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll(root) error = %v", err)
	}

	store := newMockWorkspaceStore(
		Workspace{ID: "ws_taken", RootDir: filepath.Join(t.TempDir(), "other"), Name: "repo"},
	)
	loader := &countingConfigLoader{cfg: validConfig(homePaths)}
	resolver := newTestResolver(t, store,
		WithHomePaths(homePaths),
		WithConfigLoader(loader.Load),
		WithIDGenerator(func(_ string) string { return "ws_fixed" }),
	)

	resolved, err := resolver.ResolveOrRegister(ctx, root)
	if err != nil {
		t.Fatalf("ResolveOrRegister() error = %v", err)
	}

	if resolved.ID != "ws_fixed" {
		t.Fatalf("ResolveOrRegister() ID = %q, want %q", resolved.ID, "ws_fixed")
	}
	if resolved.Name != "repo-2" {
		t.Fatalf("ResolveOrRegister() Name = %q, want %q", resolved.Name, "repo-2")
	}
	if !strings.HasPrefix(resolved.ID, "ws_") {
		t.Fatalf("ResolveOrRegister() ID = %q, want ws_ prefix", resolved.ID)
	}
	if got := len(store.insertCalls); got != 1 {
		t.Fatalf("InsertWorkspace() calls = %d, want 1", got)
	}
	if got := store.insertCalls[0].Name; got != "repo-2" {
		t.Fatalf("InsertWorkspace() Name = %q, want %q", got, "repo-2")
	}
}

func TestResolverCRUDFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := newTestHomePaths(t)
	root := t.TempDir()
	additionalOne := t.TempDir()
	additionalTwo := t.TempDir()
	canonicalAdditionalOne := mustCanonicalRoot(t, additionalOne)
	canonicalAdditionalTwo := mustCanonicalRoot(t, additionalTwo)

	store := newMockWorkspaceStore()
	loader := &countingConfigLoader{cfg: validConfig(homePaths)}
	resolver := newTestResolver(t, store,
		WithHomePaths(homePaths),
		WithConfigLoader(loader.Load),
		WithIDGenerator(func(_ string) string { return "ws_manual" }),
	)

	registered, err := resolver.Register(ctx, RegisterOptions{
		RootDir:        root,
		Name:           "repo",
		AdditionalDirs: []string{additionalOne, root, additionalOne},
		DefaultAgent:   "workspace-agent",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if registered.ID != "ws_manual" {
		t.Fatalf("Register() ID = %q, want %q", registered.ID, "ws_manual")
	}
	if got, want := registered.AdditionalDirs, []string{canonicalAdditionalOne}; !slices.Equal(got, want) {
		t.Fatalf("Register() AdditionalDirs = %#v, want %#v", got, want)
	}

	gotByName, err := resolver.Get(ctx, "repo")
	if err != nil {
		t.Fatalf("Get(name) error = %v", err)
	}
	if gotByName.ID != registered.ID {
		t.Fatalf("Get(name) ID = %q, want %q", gotByName.ID, registered.ID)
	}

	listed, err := resolver.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != registered.ID {
		t.Fatalf("List() = %#v, want one workspace %q", listed, registered.ID)
	}

	newName := "repo-renamed"
	newDefaultAgent := ""
	newAdditionalDirs := []string{additionalTwo}
	if err := resolver.Update(ctx, registered.ID, UpdateOptions{
		Name:           &newName,
		DefaultAgent:   &newDefaultAgent,
		AdditionalDirs: &newAdditionalDirs,
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	resolved, err := resolver.Resolve(ctx, registered.ID)
	if err != nil {
		t.Fatalf("Resolve(updated) error = %v", err)
	}
	if resolved.Name != newName {
		t.Fatalf("Resolve(updated) Name = %q, want %q", resolved.Name, newName)
	}
	if got, want := resolved.AdditionalDirs, []string{canonicalAdditionalTwo}; !slices.Equal(got, want) {
		t.Fatalf("Resolve(updated) AdditionalDirs = %#v, want %#v", got, want)
	}
	if resolved.DefaultAgent != "" {
		t.Fatalf("Resolve(updated) DefaultAgent = %q, want empty", resolved.DefaultAgent)
	}
	if resolved.Config.Defaults.Agent != aghconfig.DefaultAgentName {
		t.Fatalf("Resolve(updated) Config.Defaults.Agent = %q, want %q", resolved.Config.Defaults.Agent, aghconfig.DefaultAgentName)
	}

	if err := resolver.Unregister(ctx, registered.ID); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}
	if _, err := resolver.Get(ctx, registered.ID); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("Get(after unregister) error = %v, want %v", err, ErrWorkspaceNotFound)
	}
}

func TestResolveCacheHitInvalidateAndEviction(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := newTestHomePaths(t)
	root := t.TempDir()
	workspaceConfig := filepath.Join(root, aghconfig.DirName, aghconfig.ConfigName)
	agentFile := filepath.Join(root, aghconfig.DirName, aghconfig.AgentsDirName, "coder", agentDefinitionFile)
	skillsDir := filepath.Join(root, aghconfig.DirName, aghconfig.SkillsDirName)
	skillOne := filepath.Join(skillsDir, "alpha")
	skillTwo := filepath.Join(skillsDir, "beta")

	writeFile(t, workspaceConfig, "[http]\nport = 4242\n")
	writeAgentDef(t, agentFile, "coder", "v1")
	writeSkill(t, skillOne)

	ws := Workspace{ID: "ws_cache", RootDir: root, Name: "repo"}
	store := newMockWorkspaceStore(ws)
	loader := &countingConfigLoader{cfg: validConfig(homePaths)}
	currentTime := time.Unix(1_700_000_000, 0).UTC()

	resolver := newTestResolver(t, store,
		WithHomePaths(homePaths),
		WithConfigLoader(loader.Load),
		WithNow(func() time.Time { return currentTime }),
		WithCacheTTL(10*time.Minute),
	)

	first, err := resolver.Resolve(ctx, ws.ID)
	if err != nil {
		t.Fatalf("Resolve(first) error = %v", err)
	}
	if got := loader.Calls(); got != 1 {
		t.Fatalf("config loader calls after first resolve = %d, want 1", got)
	}

	currentTime = currentTime.Add(1 * time.Minute)
	second, err := resolver.Resolve(ctx, ws.ID)
	if err != nil {
		t.Fatalf("Resolve(second) error = %v", err)
	}
	if got := loader.Calls(); got != 1 {
		t.Fatalf("config loader calls after cache hit = %d, want 1", got)
	}
	if !second.ResolvedAt.Equal(first.ResolvedAt) {
		t.Fatalf("ResolvedAt on cache hit = %v, want %v", second.ResolvedAt, first.ResolvedAt)
	}

	modTime := time.Unix(1_700_000_100, 0).UTC()
	writeFile(t, workspaceConfig, "[http]\nport = 4343\n")
	touchPath(t, workspaceConfig, modTime)
	currentTime = currentTime.Add(1 * time.Minute)
	if _, err := resolver.Resolve(ctx, ws.ID); err != nil {
		t.Fatalf("Resolve(after config change) error = %v", err)
	}
	if got := loader.Calls(); got != 2 {
		t.Fatalf("config loader calls after config invalidation = %d, want 2", got)
	}

	modTime = modTime.Add(1 * time.Minute)
	writeAgentDef(t, agentFile, "coder", "v2")
	touchPath(t, agentFile, modTime)
	currentTime = currentTime.Add(1 * time.Minute)
	afterAgent, err := resolver.Resolve(ctx, ws.ID)
	if err != nil {
		t.Fatalf("Resolve(after agent change) error = %v", err)
	}
	if got := loader.Calls(); got != 3 {
		t.Fatalf("config loader calls after agent invalidation = %d, want 3", got)
	}
	if got := agentModel(afterAgent.Agents, "coder"); got != "v2" {
		t.Fatalf("agent model after agent invalidation = %q, want %q", got, "v2")
	}

	writeSkill(t, skillTwo)
	touchPath(t, skillsDir, modTime.Add(1*time.Minute))
	currentTime = currentTime.Add(1 * time.Minute)
	afterSkill, err := resolver.Resolve(ctx, ws.ID)
	if err != nil {
		t.Fatalf("Resolve(after skill change) error = %v", err)
	}
	if got := loader.Calls(); got != 4 {
		t.Fatalf("config loader calls after skill invalidation = %d, want 4", got)
	}
	if got := skillNames(afterSkill.Skills); !slices.Equal(got, []string{"alpha", "beta"}) {
		t.Fatalf("skill names after skill invalidation = %#v, want %#v", got, []string{"alpha", "beta"})
	}

	resolver.Invalidate(ws.ID)
	currentTime = currentTime.Add(1 * time.Minute)
	if _, err := resolver.Resolve(ctx, ws.ID); err != nil {
		t.Fatalf("Resolve(after invalidate) error = %v", err)
	}
	if got := loader.Calls(); got != 5 {
		t.Fatalf("config loader calls after Invalidate = %d, want 5", got)
	}

	currentTime = currentTime.Add(11 * time.Minute)
	if _, err := resolver.Resolve(ctx, ws.ID); err != nil {
		t.Fatalf("Resolve(after TTL expiry) error = %v", err)
	}
	if got := loader.Calls(); got != 6 {
		t.Fatalf("config loader calls after TTL eviction = %d, want 6", got)
	}
}

func TestResolveMissingRootReturnsErrWorkspaceRootMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := newTestHomePaths(t)
	root := filepath.Join(t.TempDir(), "gone")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll(root) error = %v", err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("RemoveAll(root) error = %v", err)
	}

	store := newMockWorkspaceStore(Workspace{ID: "ws_missing", RootDir: root, Name: "repo"})
	loader := &countingConfigLoader{cfg: validConfig(homePaths)}
	resolver := newTestResolver(t, store,
		WithHomePaths(homePaths),
		WithConfigLoader(loader.Load),
	)

	if _, err := resolver.Resolve(ctx, "ws_missing"); !errors.Is(err, ErrWorkspaceRootMissing) {
		t.Fatalf("Resolve() error = %v, want %v", err, ErrWorkspaceRootMissing)
	}
}

func TestResolveSymlinkChangedUpdatesStoredRootDir(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := newTestHomePaths(t)
	parent := t.TempDir()
	targetOne := filepath.Join(parent, "target-one")
	targetTwo := filepath.Join(parent, "target-two")
	link := filepath.Join(parent, "workspace-link")

	if err := os.MkdirAll(targetOne, 0o755); err != nil {
		t.Fatalf("MkdirAll(targetOne) error = %v", err)
	}
	if err := os.MkdirAll(targetTwo, 0o755); err != nil {
		t.Fatalf("MkdirAll(targetTwo) error = %v", err)
	}
	createSymlink(t, targetOne, link)
	createSymlink(t, targetTwo, link)

	store := newMockWorkspaceStore(Workspace{ID: "ws_symlink", RootDir: link, Name: "repo"})
	loader := &countingConfigLoader{cfg: validConfig(homePaths)}
	resolver := newTestResolver(t, store,
		WithHomePaths(homePaths),
		WithConfigLoader(loader.Load),
	)

	resolved, err := resolver.Resolve(ctx, "ws_symlink")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	canonicalTargetTwo := mustCanonicalRoot(t, targetTwo)
	if resolved.RootDir != canonicalTargetTwo {
		t.Fatalf("Resolve() RootDir = %q, want %q", resolved.RootDir, canonicalTargetTwo)
	}
	if got := loader.LastRoot(); got != canonicalTargetTwo {
		t.Fatalf("loadConfig root = %q, want %q", got, canonicalTargetTwo)
	}
	if updated := store.mustWorkspace("ws_symlink"); updated.RootDir != canonicalTargetTwo {
		t.Fatalf("updated store root = %q, want %q", updated.RootDir, canonicalTargetTwo)
	}
}

func TestResolveLocalAgentOverridesGlobalByName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := newTestHomePaths(t)
	root := t.TempDir()

	writeAgentDef(t, filepath.Join(homePaths.AgentsDir, "coder", agentDefinitionFile), "coder", "global")
	writeAgentDef(t, filepath.Join(root, aghconfig.DirName, aghconfig.AgentsDirName, "coder", agentDefinitionFile), "coder", "local")
	writeAgentDef(t, filepath.Join(homePaths.AgentsDir, "reviewer", agentDefinitionFile), "reviewer", "review")

	store := newMockWorkspaceStore(Workspace{ID: "ws_agents", RootDir: root, Name: "repo"})
	loader := &countingConfigLoader{cfg: validConfig(homePaths)}
	resolver := newTestResolver(t, store,
		WithHomePaths(homePaths),
		WithConfigLoader(loader.Load),
	)

	resolved, err := resolver.Resolve(ctx, "ws_agents")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got := agentModel(resolved.Agents, "coder"); got != "local" {
		t.Fatalf("coder model = %q, want %q", got, "local")
	}
	if got := agentModel(resolved.Agents, "reviewer"); got != "review" {
		t.Fatalf("reviewer model = %q, want %q", got, "review")
	}
}

func TestResolveConfigFromRootOnly(t *testing.T) {
	ctx := context.Background()
	homePaths := newTestHomePaths(t)
	root := t.TempDir()
	additional := t.TempDir()
	t.Setenv("AGH_HOME", homePaths.HomeDir)

	writeFile(t, homePaths.ConfigFile, "[http]\nhost = \"localhost\"\nport = 2123\n")
	writeFile(t, filepath.Join(root, aghconfig.DirName, aghconfig.ConfigName), "[http]\nport = 4242\n")
	writeFile(t, filepath.Join(additional, aghconfig.DirName, aghconfig.ConfigName), "[http]\nport = 9999\n")

	store := newMockWorkspaceStore(Workspace{
		ID:             "ws_config",
		RootDir:        root,
		AdditionalDirs: []string{additional},
		Name:           "repo",
	})
	resolver := newTestResolver(t, store,
		WithHomePaths(homePaths),
	)

	resolved, err := resolver.Resolve(ctx, "ws_config")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got, want := resolved.Config.HTTP.Port, 4242; got != want {
		t.Fatalf("Resolve() HTTP.Port = %d, want %d", got, want)
	}
}

func TestNewResolverValidatesDependenciesAndDefaults(t *testing.T) {
	t.Parallel()

	if _, err := NewResolver(nil); err == nil {
		t.Fatal("NewResolver(nil) error = nil, want non-nil")
	}

	store := newMockWorkspaceStore()
	if _, err := NewResolver(store, WithConfigLoader(nil)); err == nil {
		t.Fatal("NewResolver(..., WithConfigLoader(nil)) error = nil, want non-nil")
	}

	resolver, err := NewResolver(store,
		WithLogger(nil),
		WithNow(nil),
		WithCacheTTL(0),
		WithIDGenerator(nil),
	)
	if err != nil {
		t.Fatalf("NewResolver(defaulted options) error = %v", err)
	}
	if resolver.logger == nil {
		t.Fatal("NewResolver() logger = nil, want default logger")
	}
	if resolver.now == nil {
		t.Fatal("NewResolver() now = nil, want default clock")
	}
	if resolver.cacheTTL != defaultCacheTTL {
		t.Fatalf("NewResolver() cacheTTL = %s, want %s", resolver.cacheTTL, defaultCacheTTL)
	}
	if resolver.idGenerator == nil {
		t.Fatal("NewResolver() idGenerator = nil, want default generator")
	}
}

func TestRegisterRollsBackWhenResolveFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := newTestHomePaths(t)
	root := t.TempDir()
	store := newMockWorkspaceStore()

	resolver := newTestResolver(t, store,
		WithHomePaths(homePaths),
		WithConfigLoader(func(string) (aghconfig.Config, error) {
			return aghconfig.Config{}, errors.New("boom")
		}),
		WithIDGenerator(func(_ string) string { return "ws_fail" }),
	)

	if _, err := resolver.Register(ctx, RegisterOptions{RootDir: root, Name: "repo"}); err == nil {
		t.Fatal("Register() error = nil, want non-nil")
	}
	if got := len(store.deleteCalls); got != 1 {
		t.Fatalf("DeleteWorkspace() calls = %d, want 1", got)
	}
	if _, err := store.GetWorkspace(ctx, "ws_fail"); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("GetWorkspace(rolled back) error = %v, want %v", err, ErrWorkspaceNotFound)
	}
}

func TestResolveOrRegisterReturnsConcurrentWinnerWhenPathTaken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := newTestHomePaths(t)
	root := t.TempDir()
	existing := Workspace{ID: "ws_existing", RootDir: mustCanonicalRoot(t, root), Name: "repo"}
	store := &concurrentPathStore{existing: existing}

	resolver := newTestResolver(t, store,
		WithHomePaths(homePaths),
		WithConfigLoader((&countingConfigLoader{cfg: validConfig(homePaths)}).Load),
		WithIDGenerator(func(_ string) string { return "ws_new" }),
	)

	resolved, err := resolver.ResolveOrRegister(ctx, root)
	if err != nil {
		t.Fatalf("ResolveOrRegister() error = %v", err)
	}
	if resolved.ID != existing.ID {
		t.Fatalf("ResolveOrRegister() ID = %q, want %q", resolved.ID, existing.ID)
	}
	if store.getByPathCalls != 2 {
		t.Fatalf("GetWorkspaceByPath() calls = %d, want 2", store.getByPathCalls)
	}
}

func TestListReturnsClonedWorkspaces(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := newMockWorkspaceStore(Workspace{
		ID:             "ws_list",
		RootDir:        mustCanonicalRoot(t, root),
		Name:           "repo",
		AdditionalDirs: []string{"one"},
	})
	resolver := newTestResolver(t, store)

	listed, err := resolver.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	listed[0].Name = "mutated"
	listed[0].AdditionalDirs[0] = "changed"

	stored := store.mustWorkspace("ws_list")
	if stored.Name != "repo" {
		t.Fatalf("store name after List() mutation = %q, want %q", stored.Name, "repo")
	}
	if stored.AdditionalDirs[0] != "one" {
		t.Fatalf("store AdditionalDirs after List() mutation = %#v, want %#v", stored.AdditionalDirs, []string{"one"})
	}
}

func TestCloneConfigProducesDeepCopy(t *testing.T) {
	t.Parallel()

	original := aghconfig.Config{
		Providers: map[string]aghconfig.ProviderConfig{
			"claude": {
				Command:      "claude",
				DefaultModel: "sonnet",
				APIKeyEnv:    "ANTHROPIC_API_KEY",
				MCPServers: []aghconfig.MCPServer{
					{
						Name:    "github",
						Command: "npx",
						Args:    []string{"-y"},
						Env:     map[string]string{"TOKEN": "one"},
					},
				},
			},
		},
		Skills: aghconfig.SkillsConfig{
			Enabled:        true,
			DisabledSkills: []string{"alpha"},
			PollInterval:   time.Second,
		},
	}

	cloned := cloneConfig(original)
	cloned.Providers["claude"] = aghconfig.ProviderConfig{}
	cloned.Skills.DisabledSkills[0] = "beta"

	provider := original.Providers["claude"]
	if provider.Command != "claude" || provider.MCPServers[0].Env["TOKEN"] != "one" {
		t.Fatalf("original provider mutated: %#v", provider)
	}
	if got, want := original.Skills.DisabledSkills, []string{"alpha"}; !slices.Equal(got, want) {
		t.Fatalf("original Skills.DisabledSkills = %#v, want %#v", got, want)
	}
}

func TestWorkspaceHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("errorType", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			err  error
			want string
		}{
			{err: ErrWorkspaceNotFound, want: "workspace_not_found"},
			{err: ErrWorkspaceRootMissing, want: "workspace_root_missing"},
			{err: ErrWorkspaceNameTaken, want: "workspace_name_taken"},
			{err: ErrWorkspacePathTaken, want: "workspace_path_taken"},
			{err: context.Canceled, want: "context_canceled"},
			{err: context.DeadlineExceeded, want: "context_deadline_exceeded"},
			{err: errors.New("other"), want: "error"},
			{err: nil, want: ""},
		}

		for _, tt := range tests {
			if got := errorType(tt.err); got != tt.want {
				t.Fatalf("errorType(%v) = %q, want %q", tt.err, got, tt.want)
			}
		}
	})

	t.Run("checkContext", func(t *testing.T) {
		t.Parallel()

		if err := checkContext(nilTestContext()); err == nil {
			t.Fatal("checkContext(nil) error = nil, want non-nil")
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := checkContext(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("checkContext(cancelled) error = %v, want %v", err, context.Canceled)
		}

		if err := checkContext(context.Background()); err != nil {
			t.Fatalf("checkContext(background) error = %v, want nil", err)
		}
	})

	t.Run("canonicalRoot", func(t *testing.T) {
		t.Parallel()

		if _, err := canonicalRoot(""); err == nil {
			t.Fatal("canonicalRoot(\"\") error = nil, want non-nil")
		}
		if _, err := canonicalRoot(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, ErrWorkspaceRootMissing) {
			t.Fatalf("canonicalRoot(missing) error = %v, want %v", err, ErrWorkspaceRootMissing)
		}

		filePath := filepath.Join(t.TempDir(), "file.txt")
		writeFile(t, filePath, "not-a-dir")
		if _, err := canonicalRoot(filePath); err == nil {
			t.Fatal("canonicalRoot(file) error = nil, want non-nil")
		}
	})

	t.Run("snapshots and overrides", func(t *testing.T) {
		t.Parallel()

		snapshots := make(map[string]fileSnapshot)
		if err := addSnapshotIfExists("", snapshots); err != nil {
			t.Fatalf("addSnapshotIfExists(\"\") error = %v", err)
		}
		if err := addSnapshotIfExists(filepath.Join(t.TempDir(), "missing"), snapshots); err != nil {
			t.Fatalf("addSnapshotIfExists(missing) error = %v", err)
		}
		if len(snapshots) != 0 {
			t.Fatalf("snapshots for missing path = %#v, want empty", snapshots)
		}

		cfg := aghconfig.Config{Defaults: aghconfig.DefaultsConfig{Agent: aghconfig.DefaultAgentName}}
		applyDefaultAgentOverride(&cfg, "")
		if cfg.Defaults.Agent != aghconfig.DefaultAgentName {
			t.Fatalf("Defaults.Agent after empty override = %q, want %q", cfg.Defaults.Agent, aghconfig.DefaultAgentName)
		}
		applyDefaultAgentOverride(&cfg, "workspace-agent")
		if cfg.Defaults.Agent != "workspace-agent" {
			t.Fatalf("Defaults.Agent after override = %q, want %q", cfg.Defaults.Agent, "workspace-agent")
		}

		left := map[string]fileSnapshot{"a": {modTime: time.Unix(1, 0), size: 1}}
		right := map[string]fileSnapshot{"a": {modTime: time.Unix(1, 0), size: 1}}
		if !snapshotsEqual(left, right) {
			t.Fatal("snapshotsEqual() = false, want true")
		}
		right["a"] = fileSnapshot{modTime: time.Unix(2, 0), size: 1}
		if snapshotsEqual(left, right) {
			t.Fatal("snapshotsEqual() = true, want false")
		}

		if got := cloneStringMap(nil); got != nil {
			t.Fatalf("cloneStringMap(nil) = %#v, want nil", got)
		}
	})

	t.Run("generateID", func(t *testing.T) {
		t.Parallel()

		if got := generateID("ws"); !strings.HasPrefix(got, "ws_") {
			t.Fatalf("generateID(ws) = %q, want ws_ prefix", got)
		}
		if got := generateID(""); got == "" {
			t.Fatal("generateID(\"\") = empty, want non-empty")
		}
	})
}

type mockWorkspaceStore struct {
	mu sync.Mutex

	workspaces map[string]Workspace

	insertCalls       []Workspace
	updateCalls       []Workspace
	deleteCalls       []string
	getWorkspaceCalls []string
	getByPathCalls    []string
	getByNameCalls    []string
	listCalls         int
}

func newMockWorkspaceStore(workspaces ...Workspace) *mockWorkspaceStore {
	store := &mockWorkspaceStore{
		workspaces: make(map[string]Workspace, len(workspaces)),
	}
	for _, ws := range workspaces {
		store.workspaces[ws.ID] = cloneWorkspace(ws)
	}
	return store
}

func (m *mockWorkspaceStore) InsertWorkspace(_ context.Context, ws Workspace) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.insertCalls = append(m.insertCalls, cloneWorkspace(ws))
	for _, existing := range m.workspaces {
		switch {
		case existing.RootDir == ws.RootDir:
			return ErrWorkspacePathTaken
		case existing.Name == ws.Name:
			return ErrWorkspaceNameTaken
		}
	}
	m.workspaces[ws.ID] = cloneWorkspace(ws)
	return nil
}

func (m *mockWorkspaceStore) UpdateWorkspace(_ context.Context, ws Workspace) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateCalls = append(m.updateCalls, cloneWorkspace(ws))
	if _, ok := m.workspaces[ws.ID]; !ok {
		return ErrWorkspaceNotFound
	}
	for _, existing := range m.workspaces {
		if existing.ID == ws.ID {
			continue
		}
		switch {
		case existing.RootDir == ws.RootDir:
			return ErrWorkspacePathTaken
		case existing.Name == ws.Name:
			return ErrWorkspaceNameTaken
		}
	}
	m.workspaces[ws.ID] = cloneWorkspace(ws)
	return nil
}

func (m *mockWorkspaceStore) DeleteWorkspace(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCalls = append(m.deleteCalls, id)
	if _, ok := m.workspaces[id]; !ok {
		return ErrWorkspaceNotFound
	}
	delete(m.workspaces, id)
	return nil
}

func (m *mockWorkspaceStore) GetWorkspace(_ context.Context, id string) (Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getWorkspaceCalls = append(m.getWorkspaceCalls, id)
	ws, ok := m.workspaces[id]
	if !ok {
		return Workspace{}, ErrWorkspaceNotFound
	}
	return cloneWorkspace(ws), nil
}

func (m *mockWorkspaceStore) GetWorkspaceByPath(_ context.Context, rootDir string) (Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getByPathCalls = append(m.getByPathCalls, rootDir)
	for _, ws := range m.workspaces {
		if ws.RootDir == rootDir {
			return cloneWorkspace(ws), nil
		}
	}
	return Workspace{}, ErrWorkspaceNotFound
}

func (m *mockWorkspaceStore) GetWorkspaceByName(_ context.Context, name string) (Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getByNameCalls = append(m.getByNameCalls, name)
	for _, ws := range m.workspaces {
		if ws.Name == name {
			return cloneWorkspace(ws), nil
		}
	}
	return Workspace{}, ErrWorkspaceNotFound
}

func (m *mockWorkspaceStore) ListWorkspaces(_ context.Context) ([]Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.listCalls++
	workspaces := make([]Workspace, 0, len(m.workspaces))
	for _, ws := range m.workspaces {
		workspaces = append(workspaces, cloneWorkspace(ws))
	}
	slices.SortFunc(workspaces, func(left, right Workspace) int {
		if compare := strings.Compare(left.Name, right.Name); compare != 0 {
			return compare
		}
		return strings.Compare(left.ID, right.ID)
	})
	return workspaces, nil
}

func (m *mockWorkspaceStore) mustWorkspace(id string) Workspace {
	m.mu.Lock()
	defer m.mu.Unlock()

	ws, ok := m.workspaces[id]
	if !ok {
		panic("workspace not found: " + id)
	}
	return cloneWorkspace(ws)
}

type countingConfigLoader struct {
	mu    sync.Mutex
	cfg   aghconfig.Config
	calls int
	roots []string
}

type concurrentPathStore struct {
	existing       Workspace
	getByPathCalls int
}

func (s *concurrentPathStore) InsertWorkspace(context.Context, Workspace) error {
	return ErrWorkspacePathTaken
}

func (s *concurrentPathStore) UpdateWorkspace(context.Context, Workspace) error {
	return nil
}

func (s *concurrentPathStore) DeleteWorkspace(context.Context, string) error {
	return nil
}

func (s *concurrentPathStore) GetWorkspace(_ context.Context, id string) (Workspace, error) {
	if id != s.existing.ID {
		return Workspace{}, ErrWorkspaceNotFound
	}
	return cloneWorkspace(s.existing), nil
}

func (s *concurrentPathStore) GetWorkspaceByPath(_ context.Context, rootDir string) (Workspace, error) {
	s.getByPathCalls++
	if s.getByPathCalls == 1 {
		return Workspace{}, ErrWorkspaceNotFound
	}
	if rootDir != s.existing.RootDir {
		return Workspace{}, ErrWorkspaceNotFound
	}
	return cloneWorkspace(s.existing), nil
}

func (s *concurrentPathStore) GetWorkspaceByName(context.Context, string) (Workspace, error) {
	return Workspace{}, ErrWorkspaceNotFound
}

func (s *concurrentPathStore) ListWorkspaces(context.Context) ([]Workspace, error) {
	return nil, nil
}

func (l *countingConfigLoader) Load(root string) (aghconfig.Config, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.calls++
	l.roots = append(l.roots, root)
	return cloneConfig(l.cfg), nil
}

func (l *countingConfigLoader) Calls() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.calls
}

func (l *countingConfigLoader) LastRoot() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.roots) == 0 {
		return ""
	}
	return l.roots[len(l.roots)-1]
}

func newTestResolver(t *testing.T, store WorkspaceStore, opts ...Option) *Resolver {
	t.Helper()

	opts = append([]Option{WithLogger(discardLogger())}, opts...)
	resolver, err := NewResolver(store, opts...)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}
	return resolver
}

func newTestHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	return homePaths
}

func mustCanonicalRoot(t *testing.T, path string) string {
	t.Helper()

	root, err := canonicalRoot(path)
	if err != nil {
		t.Fatalf("canonicalRoot(%q) error = %v", path, err)
	}
	return root
}

func validConfig(homePaths aghconfig.HomePaths) aghconfig.Config {
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Port = 2123
	return cfg
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func writeAgentDef(t *testing.T, path string, name string, model string) {
	t.Helper()
	writeFile(t, path, strings.Join([]string{
		"---",
		"name: " + name,
		"provider: claude",
		"model: " + model,
		"---",
		"",
		"Prompt for " + name + ".",
		"",
	}, "\n"))
}

func writeSkill(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, skillDefinitionFile), strings.Join([]string{
		"---",
		"name: " + filepath.Base(dir),
		"description: test skill",
		"---",
		"",
		"Skill body.",
		"",
	}, "\n"))
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func touchPath(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("Chtimes(%q) error = %v", path, err)
	}
}

func createSymlink(t *testing.T, target string, link string) {
	t.Helper()
	if err := os.Remove(link); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Remove(%q) error = %v", link, err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("Symlink(%q -> %q) error = %v", link, target, err)
	}
}

func agentModel(agents []aghconfig.AgentDef, name string) string {
	for _, agent := range agents {
		if agent.Name == name {
			return agent.Model
		}
	}
	return ""
}

func skillNames(skills []SkillPath) []string {
	if len(skills) == 0 {
		return nil
	}

	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		names = append(names, filepath.Base(skill.Dir))
	}
	return names
}

func nilTestContext() context.Context {
	return nil
}
