package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/testutil"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

var (
	errTriggerFailure  = errors.New("trigger failure")
	errProviderFailure = errors.New("provider failure")
)

func TestResourceCatalogProjectorBuildAndApply(t *testing.T) {
	t.Parallel()

	catalog := newResourceCatalog(cloneToolSpec)
	projector := newToolProjector(catalog)
	if projector == nil {
		t.Fatal("newToolProjector() = nil, want projector")
	}
	if got, want := projector.Kind(), toolspkg.ToolResourceKind; got != want {
		t.Fatalf("projector.Kind() = %q, want %q", got, want)
	}
	if got := projector.DependsOn(); got != nil {
		t.Fatalf("projector.DependsOn() = %#v, want nil", got)
	}

	records := []resources.Record[toolspkg.Tool]{{
		ID:      "lookup",
		Version: 3,
		Scope: resources.ResourceScope{
			Kind: resources.ResourceScopeKindGlobal,
		},
		Spec: toolspkg.Tool{
			Name:        "lookup",
			Description: "Search extension data",
			InputSchema: json.RawMessage(`{"type":"object"}`),
			Source:      toolspkg.ToolSourceExtension,
		},
	}}

	plan, err := projector.Build(context.Background(), records)
	if err != nil {
		t.Fatalf("projector.Build() error = %v", err)
	}
	if got, want := plan.Kind(), toolspkg.ToolResourceKind; got != want {
		t.Fatalf("plan.Kind() = %q, want %q", got, want)
	}
	if got, want := plan.Revision(), int64(3); got != want {
		t.Fatalf("plan.Revision() = %d, want %d", got, want)
	}
	if got, want := plan.OperationCount(), 1; got != want {
		t.Fatalf("plan.OperationCount() = %d, want %d", got, want)
	}

	if err := projector.Apply(context.Background(), plan); err != nil {
		t.Fatalf("projector.Apply() error = %v", err)
	}
	if got, want := catalog.Revision(), int64(3); got != want {
		t.Fatalf("catalog.Revision() = %d, want %d", got, want)
	}

	snapshot := catalog.Snapshot()
	if got, want := len(snapshot), 1; got != want {
		t.Fatalf("len(snapshot) = %d, want %d", got, want)
	}
	snapshot[0].Spec.Name = "mutated"
	if got, want := catalog.Snapshot()[0].Spec.Name, "lookup"; got != want {
		t.Fatalf("catalog.Snapshot()[0].Spec.Name = %q, want %q", got, want)
	}
}

func TestToolMCPComparisonAndNilHelpers(t *testing.T) {
	t.Parallel()

	t.Run("ShouldHandleNilCatalogAndProjectorHelpers", func(t *testing.T) {
		t.Parallel()

		if got := newToolProjector(nil); got != nil {
			t.Fatalf("newToolProjector(nil) = %#v, want nil", got)
		}
		if got := newMCPServerProjector(nil); got != nil {
			t.Fatalf("newMCPServerProjector(nil) = %#v, want nil", got)
		}

		var nilCatalog *resourceCatalog[toolspkg.Tool]
		nilCatalog.Replace(9, []resources.Record[toolspkg.Tool]{{ID: "ignored"}})
		if got := nilCatalog.Snapshot(); got != nil {
			t.Fatalf("nilCatalog.Snapshot() = %#v, want nil", got)
		}
		if got := nilCatalog.Revision(); got != 0 {
			t.Fatalf("nilCatalog.Revision() = %d, want 0", got)
		}

		var nilPlan *resourceCatalogProjectionPlan[toolspkg.Tool]
		if got := nilPlan.Kind(); got != "" {
			t.Fatalf("nilPlan.Kind() = %q, want empty", got)
		}
		if got := nilPlan.Revision(); got != 0 {
			t.Fatalf("nilPlan.Revision() = %d, want 0", got)
		}
		if got := nilPlan.OperationCount(); got != 0 {
			t.Fatalf("nilPlan.OperationCount() = %d, want 0", got)
		}

		var nilProjector *resourceCatalogProjector[toolspkg.Tool]
		if got := nilProjector.Kind(); got != "" {
			t.Fatalf("nilProjector.Kind() = %q, want empty", got)
		}
		if got := nilProjector.DependsOn(); got != nil {
			t.Fatalf("nilProjector.DependsOn() = %#v, want nil", got)
		}
		if _, err := nilProjector.Build(context.Background(), nil); err == nil {
			t.Fatal("nilProjector.Build() error = nil, want non-nil")
		}
		if err := nilProjector.Apply(
			context.Background(),
			&resourceCatalogProjectionPlan[toolspkg.Tool]{},
		); err == nil {
			t.Fatal("nilProjector.Apply() error = nil, want non-nil")
		}
		if got := newToolMCPSourceSyncer(nil, nil, nil, nil, resources.MutationActor{}, nil, nil); got != nil {
			t.Fatalf("newToolMCPSourceSyncer(nil deps) = %#v, want nil", got)
		}
	})

	t.Run("ShouldCompareEncodedToolAndMCPResources", func(t *testing.T) {
		t.Parallel()

		toolCodec, err := toolspkg.NewResourceCodec()
		if err != nil {
			t.Fatalf("toolspkg.NewResourceCodec() error = %v", err)
		}
		mcpCodec, err := aghconfig.NewMCPServerResourceCodec()
		if err != nil {
			t.Fatalf("aghconfig.NewMCPServerResourceCodec() error = %v", err)
		}

		syncer := &toolMCPSourceSyncer{
			toolCodec: toolCodec,
			mcpCodec:  mcpCodec,
		}

		globalScope := resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
		workspaceScope := resources.ResourceScope{Kind: resources.ResourceScopeKindWorkspace, ID: "ws-1"}

		toolSpec := toolspkg.Tool{
			Name:        "lookup",
			Description: "Search extension data",
			InputSchema: json.RawMessage(`{"type":"object"}`),
			Source:      toolspkg.ToolSourceExtension,
		}
		toolEncoded, err := toolCodec.Encode(toolSpec)
		if err != nil {
			t.Fatalf("toolCodec.Encode() error = %v", err)
		}
		toolRecord := resources.Record[toolspkg.Tool]{
			Scope: globalScope,
			Spec:  toolSpec,
		}
		if !syncer.sameTool(toolRecord, globalScope, toolEncoded) {
			t.Fatal("sameTool() = false, want true for matching scope and spec")
		}
		if syncer.sameTool(toolRecord, workspaceScope, toolEncoded) {
			t.Fatal("sameTool() = true, want false for mismatched scope")
		}
		if syncer.sameTool(toolRecord, globalScope, []byte(`{"bad":true}`)) {
			t.Fatal("sameTool() = true, want false for mismatched encoding")
		}

		mcpSpec := aghconfig.MCPServer{
			Name:    "git",
			Command: "npx",
			Args:    []string{"@modelcontextprotocol/server-git"},
		}
		mcpEncoded, err := mcpCodec.Encode(mcpSpec)
		if err != nil {
			t.Fatalf("mcpCodec.Encode() error = %v", err)
		}
		mcpRecord := resources.Record[aghconfig.MCPServer]{
			Scope: globalScope,
			Spec:  mcpSpec,
		}
		if !syncer.sameMCPServer(mcpRecord, globalScope, mcpEncoded) {
			t.Fatal("sameMCPServer() = false, want true for matching scope and spec")
		}
		if syncer.sameMCPServer(mcpRecord, workspaceScope, mcpEncoded) {
			t.Fatal("sameMCPServer() = true, want false for mismatched scope")
		}
		if syncer.sameMCPServer(mcpRecord, globalScope, []byte(`{"bad":true}`)) {
			t.Fatal("sameMCPServer() = true, want false for mismatched encoding")
		}
	})

	t.Run("ShouldTreatPublisherHelpersAsNoopOrPassthrough", func(t *testing.T) {
		t.Parallel()

		var nilPublisher toolMCPPublisherFunc
		if err := nilPublisher.Sync(context.Background()); err != nil {
			t.Fatalf("nilPublisher.Sync() error = %v", err)
		}
		called := false
		publisher := toolMCPPublisherFunc(func(context.Context) error {
			called = true
			return nil
		})
		if err := publisher.Sync(context.Background()); err != nil {
			t.Fatalf("publisher.Sync() error = %v", err)
		}
		if !called {
			t.Fatal("publisher.Sync() did not invoke wrapped function")
		}
	})

	t.Run("ShouldBuildSyncerEvenWhenLoggerIsNil", func(t *testing.T) {
		t.Parallel()

		toolStore, toolCodec, mcpStore, mcpCodec := toolMCPSyncStores(t)
		syncerWithNilLogger := newToolMCPSourceSyncer(
			toolStore,
			toolCodec,
			mcpStore,
			mcpCodec,
			toolMCPSyncActor(),
			nil,
			nil,
		)
		if syncerWithNilLogger == nil {
			t.Fatal("newToolMCPSourceSyncer(nil logger) = nil, want syncer")
		}
		concreteSyncer, ok := syncerWithNilLogger.(*toolMCPSourceSyncer)
		if !ok {
			t.Fatalf("syncerWithNilLogger type = %T, want *toolMCPSourceSyncer", syncerWithNilLogger)
		}
		if err := concreteSyncer.Sync(context.Background()); err != nil {
			t.Fatalf("syncerWithNilLogger.Sync() error = %v", err)
		}
	})
}

func TestToolMCPSourceSyncerHandlesNilReceiverAndTriggerFailures(t *testing.T) {
	t.Parallel()

	var nilSyncer *toolMCPSourceSyncer
	if err := nilSyncer.Sync(context.Background()); err != nil {
		t.Fatalf("nilSyncer.Sync() error = %v", err)
	}

	toolStore, toolCodec, mcpStore, mcpCodec := toolMCPSyncStores(t)
	syncer := newToolMCPSourceSyncer(
		toolStore,
		toolCodec,
		mcpStore,
		mcpCodec,
		toolMCPSyncActor(),
		discardLogger(),
		func(context.Context, resources.ResourceKind, resources.ReconcileReason) error {
			return errTriggerFailure
		},
		func(context.Context) (toolMCPDesiredResources, error) {
			return toolMCPDesiredResources{
				tools: []toolPublicationInput{{
					sourceKey: "test/tool/lookup",
					scope:     resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
					spec: toolspkg.Tool{
						Name:        "lookup",
						Description: "Search extension data",
						InputSchema: json.RawMessage(`{"type":"object"}`),
						Source:      toolspkg.ToolSourceExtension,
					},
				}},
			}, nil
		},
	)

	err := syncer.Sync(context.Background())
	if err == nil {
		t.Fatal("syncer.Sync() error = nil, want trigger failure")
	}
	if !errors.Is(err, errTriggerFailure) {
		t.Fatalf("syncer.Sync() error = %v, want %v", err, errTriggerFailure)
	}
}

func TestToolMCPSourceSyncerSyncPropagatesProviderFailure(t *testing.T) {
	t.Parallel()

	toolStore, toolCodec, mcpStore, mcpCodec := toolMCPSyncStores(t)
	syncer := newToolMCPSourceSyncer(
		toolStore,
		toolCodec,
		mcpStore,
		mcpCodec,
		toolMCPSyncActor(),
		discardLogger(),
		nil,
		func(context.Context) (toolMCPDesiredResources, error) {
			return toolMCPDesiredResources{}, errProviderFailure
		},
	)

	err := syncer.Sync(context.Background())
	if err == nil {
		t.Fatal("syncer.Sync() error = nil, want provider failure")
	}
	if !errors.Is(err, errProviderFailure) {
		t.Fatalf("syncer.Sync() error = %v, want %v", err, errProviderFailure)
	}
}

func TestToolMCPSourceSyncerReplacesCanonicalSnapshot(t *testing.T) {
	t.Parallel()

	toolStore, toolCodec, mcpStore, mcpCodec := toolMCPSyncStores(t)
	desired := toolMCPDesiredResources{
		tools: []toolPublicationInput{{
			sourceKey: "test/tool/lookup",
			scope:     resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
			spec: toolspkg.Tool{
				Name:        "lookup",
				Description: "Search extension data",
				InputSchema: json.RawMessage(`{"type":"object"}`),
				Source:      toolspkg.ToolSourceExtension,
			},
		}},
		mcpServers: []mcpServerPublicationInput{{
			sourceKey: "test/mcp/git",
			scope:     resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
			spec: aghconfig.MCPServer{
				Name:    "git",
				Command: "npx",
				Args:    []string{"@modelcontextprotocol/server-git"},
			},
		}},
	}
	triggered := make(map[resources.ResourceKind]int)
	syncer := newToolMCPSourceSyncer(
		toolStore,
		toolCodec,
		mcpStore,
		mcpCodec,
		toolMCPSyncActor(),
		discardLogger(),
		func(_ context.Context, kind resources.ResourceKind, _ resources.ReconcileReason) error {
			triggered[kind]++
			return nil
		},
		func(context.Context) (toolMCPDesiredResources, error) {
			return desired, nil
		},
	)

	if err := syncer.Sync(context.Background()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	assertToolMCPStoreCounts(t, toolStore, mcpStore, 1, 1)
	if triggered[toolspkg.ToolResourceKind] != 1 || triggered[aghconfig.MCPServerResourceKind] != 1 {
		t.Fatalf("triggered = %#v, want one trigger per resource kind", triggered)
	}

	if err := syncer.Sync(context.Background()); err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}
	if triggered[toolspkg.ToolResourceKind] != 1 || triggered[aghconfig.MCPServerResourceKind] != 1 {
		t.Fatalf("triggered after no-op = %#v, want no additional triggers", triggered)
	}

	desired.tools = nil
	desired.mcpServers[0].spec.Command = "node"
	if err := syncer.Sync(context.Background()); err != nil {
		t.Fatalf("third Sync() error = %v", err)
	}
	assertToolMCPStoreCounts(t, toolStore, mcpStore, 0, 1)
	if triggered[toolspkg.ToolResourceKind] != 2 || triggered[aghconfig.MCPServerResourceKind] != 2 {
		t.Fatalf("triggered after replacement = %#v, want delete/update triggers", triggered)
	}
}

func TestNewToolMCPPublisherFallsBackToNoopWithoutResourceRuntime(t *testing.T) {
	t.Parallel()

	daemon := &Daemon{}

	publisher, err := daemon.newToolMCPPublisher(nil, nil)
	if err != nil {
		t.Fatalf("newToolMCPPublisher(nil state) error = %v", err)
	}
	if publisher == nil {
		t.Fatal("newToolMCPPublisher(nil state) = nil, want no-op publisher")
	}
	if err := publisher.Sync(context.Background()); err != nil {
		t.Fatalf("publisher.Sync(nil state) error = %v", err)
	}

	publisher, err = daemon.newToolMCPPublisher(&bootState{}, nil)
	if err != nil {
		t.Fatalf("newToolMCPPublisher(empty state) error = %v", err)
	}
	if publisher == nil {
		t.Fatal("newToolMCPPublisher(empty state) = nil, want no-op publisher")
	}
	if err := publisher.Sync(context.Background()); err != nil {
		t.Fatalf("publisher.Sync(empty state) error = %v", err)
	}
}

func TestNewToolMCPPublisherBuildsSyncerWhenResourceRuntimeIsReady(t *testing.T) {
	t.Parallel()

	db := openDaemonTestGlobalDB(t)
	kernel, err := resources.NewKernel(db.DB())
	if err != nil {
		t.Fatalf("resources.NewKernel() error = %v", err)
	}

	toolCodec, err := toolspkg.NewResourceCodec()
	if err != nil {
		t.Fatalf("toolspkg.NewResourceCodec() error = %v", err)
	}
	mcpCodec, err := aghconfig.NewMCPServerResourceCodec()
	if err != nil {
		t.Fatalf("aghconfig.NewMCPServerResourceCodec() error = %v", err)
	}

	codecs := resources.NewCodecRegistry()
	if err := resources.RegisterCodec(codecs, toolCodec); err != nil {
		t.Fatalf("RegisterCodec(tool) error = %v", err)
	}
	if err := resources.RegisterCodec(codecs, mcpCodec); err != nil {
		t.Fatalf("RegisterCodec(mcp) error = %v", err)
	}

	daemon := &Daemon{getenv: func(string) string { return "" }}
	publisher, err := daemon.newToolMCPPublisher(&bootState{
		logger:         discardLogger(),
		resourceKernel: kernel,
		resourceCodecs: codecs,
	}, nil)
	if err != nil {
		t.Fatalf("newToolMCPPublisher(ready state) error = %v", err)
	}
	if publisher == nil {
		t.Fatal("newToolMCPPublisher(ready state) = nil, want syncer")
	}
	if err := publisher.Sync(context.Background()); err != nil {
		t.Fatalf("publisher.Sync(ready state) error = %v", err)
	}
}

func TestNewToolMCPPublisherReturnsCodecResolutionErrors(t *testing.T) {
	t.Parallel()

	db := openDaemonTestGlobalDB(t)
	kernel, err := resources.NewKernel(db.DB())
	if err != nil {
		t.Fatalf("resources.NewKernel() error = %v", err)
	}

	daemon := &Daemon{}
	emptyCodecs := resources.NewCodecRegistry()
	_, err = daemon.newToolMCPPublisher(&bootState{
		logger:         discardLogger(),
		resourceKernel: kernel,
		resourceCodecs: emptyCodecs,
	}, nil)
	if err == nil {
		t.Fatal("newToolMCPPublisher(empty codecs) error = nil, want tool codec failure")
	}
	if !errors.Is(err, resources.ErrCodecNotFound) {
		t.Fatalf("newToolMCPPublisher(empty codecs) error = %v, want %v", err, resources.ErrCodecNotFound)
	}

	toolCodec, err := toolspkg.NewResourceCodec()
	if err != nil {
		t.Fatalf("toolspkg.NewResourceCodec() error = %v", err)
	}
	toolOnlyCodecs := resources.NewCodecRegistry()
	if err := resources.RegisterCodec(toolOnlyCodecs, toolCodec); err != nil {
		t.Fatalf("RegisterCodec(tool) error = %v", err)
	}

	_, err = daemon.newToolMCPPublisher(&bootState{
		logger:         discardLogger(),
		resourceKernel: kernel,
		resourceCodecs: toolOnlyCodecs,
	}, nil)
	if err == nil {
		t.Fatal("newToolMCPPublisher(tool-only codecs) error = nil, want mcp codec failure")
	}
	if !errors.Is(err, resources.ErrCodecNotFound) {
		t.Fatalf("newToolMCPPublisher(tool-only codecs) error = %v, want %v", err, resources.ErrCodecNotFound)
	}
}

func TestValidateAndEncodeToolAndMCPServer(t *testing.T) {
	t.Parallel()

	toolCodec, err := toolspkg.NewResourceCodec()
	if err != nil {
		t.Fatalf("toolspkg.NewResourceCodec() error = %v", err)
	}

	toolScope := resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}
	toolSpec, toolEncoded, err := validateAndEncodeTool(context.Background(), toolCodec, toolScope, toolspkg.Tool{
		Name:        " lookup ",
		Description: " Search extension data ",
		InputSchema: json.RawMessage(`{"required":["query"],"type":"object"}`),
		Source:      toolspkg.ToolSourceExtension,
	})
	if err != nil {
		t.Fatalf("validateAndEncodeTool(valid) error = %v", err)
	}
	if got, want := toolSpec.Name, "lookup"; got != want {
		t.Fatalf("toolSpec.Name = %q, want %q", got, want)
	}
	var toolPayload struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Source      string `json:"source"`
		ReadOnly    bool   `json:"read_only"`
	}
	if err := json.Unmarshal(toolEncoded, &toolPayload); err != nil {
		t.Fatalf("json.Unmarshal(toolEncoded) error = %v", err)
	}
	if got, want := toolPayload.Name, "lookup"; got != want {
		t.Fatalf("toolPayload.Name = %#v, want %#v", got, want)
	}
	if got, want := toolPayload.Description, "Search extension data"; got != want {
		t.Fatalf("toolPayload.Description = %#v, want %#v", got, want)
	}
	if got, want := toolPayload.Source, "extension"; got != want {
		t.Fatalf("toolPayload.Source = %#v, want %#v", got, want)
	}
	if got, want := toolPayload.ReadOnly, false; got != want {
		t.Fatalf("toolPayload.ReadOnly = %#v, want %#v", got, want)
	}

	_, _, err = validateAndEncodeTool(context.Background(), toolCodec, toolScope, toolspkg.Tool{
		Name:   " ",
		Source: toolspkg.ToolSourceExtension,
	})
	if err == nil {
		t.Fatal("validateAndEncodeTool(invalid) error = nil, want validation failure")
	}

	mcpCodec, err := aghconfig.NewMCPServerResourceCodec()
	if err != nil {
		t.Fatalf("aghconfig.NewMCPServerResourceCodec() error = %v", err)
	}

	mcpSpec, mcpEncoded, err := validateAndEncodeMCPServer(
		context.Background(),
		mcpCodec,
		toolScope,
		aghconfig.MCPServer{
			Name:    " git ",
			Command: " npx ",
			Args:    []string{" --stdio "},
			Env: map[string]string{
				" TOKEN ": " secret ",
			},
		},
	)
	if err != nil {
		t.Fatalf("validateAndEncodeMCPServer(valid) error = %v", err)
	}
	if got, want := mcpSpec.Name, "git"; got != want {
		t.Fatalf("mcpSpec.Name = %q, want %q", got, want)
	}
	var mcpPayload struct {
		Name    string
		Command string
	}
	if err := json.Unmarshal(mcpEncoded, &mcpPayload); err != nil {
		t.Fatalf("json.Unmarshal(mcpEncoded) error = %v", err)
	}
	if got, want := mcpPayload.Name, "git"; got != want {
		t.Fatalf("mcpPayload.Name = %#v, want %#v", got, want)
	}
	if got, want := mcpPayload.Command, "npx"; got != want {
		t.Fatalf("mcpPayload.Command = %#v, want %#v", got, want)
	}

	_, _, err = validateAndEncodeMCPServer(context.Background(), mcpCodec, toolScope, aghconfig.MCPServer{Name: "git"})
	if err == nil {
		t.Fatal("validateAndEncodeMCPServer(invalid) error = nil, want validation failure")
	}
}

func toolMCPSyncStores(
	t *testing.T,
) (
	resources.Store[toolspkg.Tool],
	resources.KindCodec[toolspkg.Tool],
	resources.Store[aghconfig.MCPServer],
	resources.KindCodec[aghconfig.MCPServer],
) {
	t.Helper()

	db := openDaemonTestGlobalDB(t)
	kernel, err := resources.NewKernel(db.DB())
	if err != nil {
		t.Fatalf("resources.NewKernel() error = %v", err)
	}

	toolCodec, err := toolspkg.NewResourceCodec()
	if err != nil {
		t.Fatalf("toolspkg.NewResourceCodec() error = %v", err)
	}
	toolStore, err := resources.NewStore(kernel, toolCodec)
	if err != nil {
		t.Fatalf("resources.NewStore(tool) error = %v", err)
	}

	mcpCodec, err := aghconfig.NewMCPServerResourceCodec()
	if err != nil {
		t.Fatalf("aghconfig.NewMCPServerResourceCodec() error = %v", err)
	}
	mcpStore, err := resources.NewStore(kernel, mcpCodec)
	if err != nil {
		t.Fatalf("resources.NewStore(mcp) error = %v", err)
	}

	return toolStore, toolCodec, mcpStore, mcpCodec
}

func assertToolMCPStoreCounts(
	t *testing.T,
	toolStore resources.Store[toolspkg.Tool],
	mcpStore resources.Store[aghconfig.MCPServer],
	wantTools int,
	wantMCPServers int,
) {
	t.Helper()

	source := toolMCPSyncActor().Source
	tools, err := toolStore.List(testutil.Context(t), toolMCPSyncActor(), resources.ResourceFilter{Source: &source})
	if err != nil {
		t.Fatalf("toolStore.List() error = %v", err)
	}
	if got := len(tools); got != wantTools {
		t.Fatalf("len(toolStore.List()) = %d, want %d", got, wantTools)
	}

	mcpServers, err := mcpStore.List(testutil.Context(t), toolMCPSyncActor(), resources.ResourceFilter{Source: &source})
	if err != nil {
		t.Fatalf("mcpStore.List() error = %v", err)
	}
	if got := len(mcpServers); got != wantMCPServers {
		t.Fatalf("len(mcpStore.List()) = %d, want %d", got, wantMCPServers)
	}
}
