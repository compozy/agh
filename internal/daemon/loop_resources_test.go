package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	core "github.com/compozy/agh/internal/api/core"
	aghconfig "github.com/compozy/agh/internal/config"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/resources"
	"github.com/compozy/agh/internal/testutil"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestLoopProjectorShouldBuildAndApplyCatalogSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("Should build and apply a defensive loop catalog snapshot", func(t *testing.T) {
		t.Parallel()

		catalog := newResourceCatalog(looppkg.CloneResourceSpec)
		projector := newLoopProjector(catalog)
		if projector == nil {
			t.Fatal("newLoopProjector() = nil, want projector")
		}

		records := []resources.Record[looppkg.ResourceSpec]{
			{
				ID:      "loop-a",
				Version: 7,
				Scope:   resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
				Spec:    testLoopSpec(t, "loop-a", looppkg.SourceUser),
			},
		}
		plan, err := projector.Build(context.Background(), records)
		if err != nil {
			t.Fatalf("projector.Build() error = %v", err)
		}
		if got, want := plan.Kind(), looppkg.ResourceKind; got != want {
			t.Fatalf("plan.Kind() = %q, want %q", got, want)
		}
		if got, want := plan.Revision(), int64(7); got != want {
			t.Fatalf("plan.Revision() = %d, want %d", got, want)
		}
		if err := projector.Apply(context.Background(), plan); err != nil {
			t.Fatalf("projector.Apply() error = %v", err)
		}
		snapshot := catalog.Snapshot()
		if got, want := len(snapshot), 1; got != want {
			t.Fatalf("len(catalog.Snapshot()) = %d, want %d", got, want)
		}
		snapshot[0].Spec.Name = "mutated"
		if got, want := catalog.Snapshot()[0].Spec.Name, "loop-a"; got != want {
			t.Fatalf("catalog.Snapshot()[0].Spec.Name = %q, want %q", got, want)
		}
	})
}

func TestDaemonLoopAPIServiceShouldPublishWithServerManagedCASVersion(t *testing.T) {
	t.Parallel()

	t.Run("Should stamp the next version and reject stale publishes", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openDaemonTestGlobalDB(t)
		homePaths := testHomePaths(t)
		workspaceRoot := filepath.Join(t.TempDir(), "workspace")
		if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
			t.Fatalf("MkdirAll(workspaceRoot) error = %v", err)
		}
		now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
		if err := db.InsertWorkspace(ctx, workspacepkg.Workspace{
			ID:        "ws-1",
			Name:      "workspace-one",
			RootDir:   workspaceRoot,
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			t.Fatalf("InsertWorkspace() error = %v", err)
		}
		resolver, err := workspacepkg.NewResolver(
			db,
			workspacepkg.WithHomePaths(homePaths),
			workspacepkg.WithLogger(discardLogger()),
		)
		if err != nil {
			t.Fatalf("workspace.NewResolver() error = %v", err)
		}

		loopRoot := filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.LoopsDirName)
		catalog := newResourceCatalog(looppkg.CloneResourceSpec)
		initialSpec := testLoopSpec(t, "alpha", looppkg.SourceWorkspace)
		initialSpec.Dir = filepath.Join(loopRoot, "alpha")
		initialSpec.FilePath = filepath.Join(loopRoot, "alpha", "loop.yaml")
		catalog.Replace(1, []resources.Record[looppkg.ResourceSpec]{{
			ID:      "loop:ws-1:alpha",
			Version: 1,
			Scope:   resources.ResourceScope{Kind: resources.ResourceScopeKindWorkspace, ID: "ws-1"},
			Spec:    initialSpec,
		}})
		service := &daemonLoopAPIService{
			catalog:           catalog,
			publisher:         &loopAPITestPublisher{},
			workspaceResolver: resolver,
			now:               func() time.Time { return now },
		}

		expected := 1
		response, err := service.PatchLoop(ctx, "ws-1", "alpha", contract.PatchLoopRequest{
			ExpectedVersion: &expected,
			Definition:      loopAPITestDocument(t, "alpha", 1, "first publish"),
		})
		if err != nil {
			t.Fatalf("PatchLoop(first) error = %v", err)
		}
		if response.Loop.Version != 2 {
			t.Fatalf("PatchLoop(first) version = %d, want 2", response.Loop.Version)
		}
		published := readLoopDefinitionFile(t, filepath.Join(loopRoot, "alpha", "loop.yaml"))
		if published.Meta.Version != 2 {
			t.Fatalf("published meta.version = %d, want 2", published.Meta.Version)
		}

		_, err = service.PatchLoop(ctx, "ws-1", "alpha", contract.PatchLoopRequest{
			ExpectedVersion: &expected,
			Definition:      loopAPITestDocument(t, "alpha", 1, "stale publish"),
		})
		var conflict *core.LoopVersionConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("PatchLoop(stale) error = %v, want LoopVersionConflictError", err)
		}
		if conflict.CurrentVersion != 2 {
			t.Fatalf("PatchLoop(stale) current version = %d, want 2", conflict.CurrentVersion)
		}

		expected = 2
		_, err = service.PatchLoop(ctx, "ws-1", "alpha", contract.PatchLoopRequest{
			ExpectedVersion: &expected,
			Definition:      loopAPITestDocument(t, "alpha", 1, "mismatched client version"),
		})
		if !errors.Is(err, looppkg.ErrValidation) {
			t.Fatalf("PatchLoop(mismatched definition version) error = %v, want ErrValidation", err)
		}
	})

	t.Run("Should return created loops before asynchronous catalog projection runs", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openDaemonTestGlobalDB(t)
		homePaths := testHomePaths(t)
		workspaceRoot := filepath.Join(t.TempDir(), "workspace")
		if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
			t.Fatalf("MkdirAll(workspaceRoot) error = %v", err)
		}
		now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
		if err := db.InsertWorkspace(ctx, workspacepkg.Workspace{
			ID:        "ws-create",
			Name:      "workspace-create",
			RootDir:   workspaceRoot,
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			t.Fatalf("InsertWorkspace() error = %v", err)
		}
		resolver, err := workspacepkg.NewResolver(
			db,
			workspacepkg.WithHomePaths(homePaths),
			workspacepkg.WithLogger(discardLogger()),
		)
		if err != nil {
			t.Fatalf("workspace.NewResolver() error = %v", err)
		}

		catalog := newResourceCatalog(looppkg.CloneResourceSpec)
		service := &daemonLoopAPIService{
			catalog:           catalog,
			publisher:         &loopAPITestPublisher{},
			workspaceResolver: resolver,
			now:               func() time.Time { return now },
		}
		def := loopAPITestDocument(t, "alpha", 999, "created publish")

		response, err := service.CreateLoop(ctx, "ws-create", contract.CreateLoopRequest{Definition: &def})
		if err != nil {
			t.Fatalf("CreateLoop() error = %v", err)
		}
		if response.Loop.Name != "alpha" || response.Loop.Version != 1 {
			t.Fatalf("CreateLoop() payload = %#v, want alpha version 1", response.Loop)
		}

		getResponse, err := service.GetLoop(ctx, "ws-create", "alpha")
		if err != nil {
			t.Fatalf("GetLoop(created) error = %v", err)
		}
		if getResponse.Loop.Name != "alpha" || getResponse.Loop.Version != 1 {
			t.Fatalf("GetLoop(created) payload = %#v, want alpha version 1", getResponse.Loop)
		}

		_, err = service.CreateLoop(ctx, "ws-create", contract.CreateLoopRequest{Definition: &def})
		if !errors.Is(err, looppkg.ErrDefinitionExists) {
			t.Fatalf("CreateLoop(duplicate) error = %v, want ErrDefinitionExists", err)
		}
	})

	t.Run("Should classify missing fork sources as not found", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openDaemonTestGlobalDB(t)
		homePaths := testHomePaths(t)
		workspaceRoot := filepath.Join(t.TempDir(), "workspace")
		if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
			t.Fatalf("MkdirAll(workspaceRoot) error = %v", err)
		}
		now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
		if err := db.InsertWorkspace(ctx, workspacepkg.Workspace{
			ID:        "ws-fork",
			Name:      "workspace-fork",
			RootDir:   workspaceRoot,
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			t.Fatalf("InsertWorkspace() error = %v", err)
		}
		resolver, err := workspacepkg.NewResolver(
			db,
			workspacepkg.WithHomePaths(homePaths),
			workspacepkg.WithLogger(discardLogger()),
		)
		if err != nil {
			t.Fatalf("workspace.NewResolver() error = %v", err)
		}
		service := &daemonLoopAPIService{
			catalog:           newResourceCatalog(looppkg.CloneResourceSpec),
			publisher:         &loopAPITestPublisher{},
			workspaceResolver: resolver,
			now:               func() time.Time { return now },
		}

		_, err = service.CreateLoop(ctx, "ws-fork", contract.CreateLoopRequest{ForkFromName: "missing-loop"})
		if !errors.Is(err, looppkg.ErrDefinitionNotFound) {
			t.Fatalf("CreateLoop(fork missing) error = %v, want ErrDefinitionNotFound", err)
		}
	})

	t.Run("Should classify absent definitions as not found", func(t *testing.T) {
		t.Parallel()

		service := &daemonLoopAPIService{catalog: newResourceCatalog(looppkg.CloneResourceSpec)}
		if _, err := service.GetLoop(
			testutil.Context(t),
			"ws-1",
			"missing",
		); !errors.Is(err, looppkg.ErrDefinitionNotFound) {
			t.Fatalf("GetLoop(missing) error = %v, want ErrDefinitionNotFound", err)
		}
		if _, err := service.GetLoop(
			testutil.Context(t),
			"ws-1",
			"MyLoop",
		); !errors.Is(err, looppkg.ErrValidation) {
			t.Fatalf("GetLoop(malformed name) error = %v, want ErrValidation", err)
		}
	})
}

func TestLoopSourceSyncerShouldProjectAndDeleteManagedRecords(t *testing.T) {
	t.Parallel()

	t.Run("Should write desired loops and delete stale managed loops", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		kernel, err := resources.NewKernel(db.DB())
		if err != nil {
			t.Fatalf("resources.NewKernel() error = %v", err)
		}
		codec, err := looppkg.NewResourceCodec()
		if err != nil {
			t.Fatalf("looppkg.NewResourceCodec() error = %v", err)
		}
		store, err := resources.NewStore[looppkg.ResourceSpec](kernel, codec)
		if err != nil {
			t.Fatalf("resources.NewStore() error = %v", err)
		}

		providerItems := []loopPublicationInput{
			{
				sourceKey: "test/global/loop-a",
				scope:     resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
				spec:      testLoopSpec(t, "loop-a", looppkg.SourceUser),
			},
		}
		syncer := newLoopSourceSyncer(
			store,
			codec,
			loopSyncActor(),
			discardLogger(),
			nil,
			func(context.Context) ([]loopPublicationInput, error) {
				return append([]loopPublicationInput(nil), providerItems...), nil
			},
		)

		if err := syncer.Sync(context.Background()); err != nil {
			t.Fatalf("Sync(first) error = %v", err)
		}
		records, err := store.List(
			context.Background(),
			loopSyncActor(),
			resources.ResourceFilter{Kind: looppkg.ResourceKind},
		)
		if err != nil {
			t.Fatalf("store.List(first) error = %v", err)
		}
		if got, want := len(records), 1; got != want {
			t.Fatalf("len(records first) = %d, want %d", got, want)
		}
		if got, want := records[0].Spec.Name, "loop-a"; got != want {
			t.Fatalf("records[0].Spec.Name = %q, want %q", got, want)
		}

		providerItems = nil
		if err := syncer.Sync(context.Background()); err != nil {
			t.Fatalf("Sync(second) error = %v", err)
		}
		records, err = store.List(
			context.Background(),
			loopSyncActor(),
			resources.ResourceFilter{Kind: looppkg.ResourceKind},
		)
		if err != nil {
			t.Fatalf("store.List(second) error = %v", err)
		}
		if got := len(records); got != 0 {
			t.Fatalf("len(records second) = %d, want 0", got)
		}
	})
}

func testLoopSpec(t *testing.T, name string, source looppkg.Source) looppkg.ResourceSpec {
	t.Helper()

	opts := looppkg.ResourceParseOptions{
		Source:   source,
		FilePath: "/tmp/" + name + "/loop.yaml",
	}
	spec, _, err := looppkg.ParseResource([]byte(testLoopYAML(name, "test")), opts)
	if err != nil {
		t.Fatalf("looppkg.ParseResource(%q) error = %v", name, err)
	}
	return spec
}

func testLoopYAML(name string, description string) string {
	return `apiVersion: agh.loop/v1
kind: Loop
meta:
  name: ` + name + `
  version: 1
  description: ` + description + `
  catalog:
    use_when: Test daemon projection
    keywords: [projection]
    category: test
concurrency: queue
inputs:
  target:
    type: string
    required: true
contract:
  goal: Test daemon projection
  definition_of_done: Projection works
  terminal_states: [done, failed]
  iteration_cap: 3
  no_progress:
    window: 2
    hash_fields: []
  budget:
    tokens: 0
    wall_clock_sec: 0
    on_exceeded: halt
start:
  - kind: cli
graph:
  nodes:
    - id: target_input
      class: source
      kind: input
      input_ref: target
    - id: normalize_target
      class: action
      kind: transform
      params:
        map:
          target:
            value: ok
  edges:
    - from: target_input
      to: normalize_target
`
}

type loopAPITestPublisher struct {
	calls int
}

func (p *loopAPITestPublisher) Sync(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("loop api test publisher: %w", err)
	}
	p.calls++
	return nil
}

func loopAPITestDefinition(t *testing.T, name string, version int, description string) dsl.Definition {
	t.Helper()

	raw := strings.Replace(testLoopYAML(name, description), "version: 1", fmt.Sprintf("version: %d", version), 1)
	def, err := dsl.Parse([]byte(raw))
	if err != nil {
		t.Fatalf("dsl.Parse(%q) error = %v", name, err)
	}
	def.Normalize()
	return def
}

func loopAPITestDocument(t *testing.T, name string, version int, description string) contract.LoopDefinitionDocument {
	t.Helper()

	document, err := contract.NewLoopDefinitionDocument(loopAPITestDefinition(t, name, version, description))
	if err != nil {
		t.Fatalf("NewLoopDefinitionDocument(%q) error = %v", name, err)
	}
	return document
}

func readLoopDefinitionFile(t *testing.T, path string) dsl.Definition {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	def, err := dsl.Parse(data)
	if err != nil {
		t.Fatalf("dsl.Parse(%q) error = %v", path, err)
	}
	return def
}
