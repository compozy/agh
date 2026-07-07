package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	toolspkg "github.com/compozy/agh/internal/tools"
)

func TestLoopToolSchemaSource(t *testing.T) {
	t.Parallel()

	t.Run("Should expose registry descriptor schemas as loop snapshots", func(t *testing.T) {
		t.Parallel()

		descriptor := loopToolSchemaDescriptor(t)
		source := newLoopToolSchemaSource(context.Background(), loopToolSchemaRegistry{
			views: map[toolspkg.ToolID]toolspkg.ToolView{
				descriptor.ID: {Descriptor: descriptor},
			},
		})

		snapshot, ok := source.Snapshot(descriptor.ID.String())
		if !ok {
			t.Fatalf("Snapshot(%q) ok = false, want true", descriptor.ID)
		}
		if got, want := snapshot.ToolID, descriptor.ID.String(); got != want {
			t.Fatalf("snapshot.ToolID = %q, want %q", got, want)
		}
		if !json.Valid(snapshot.InputSchema) {
			t.Fatalf("snapshot.InputSchema = %s, want valid JSON", snapshot.InputSchema)
		}
		if !json.Valid(snapshot.OutputSchema) {
			t.Fatalf("snapshot.OutputSchema = %s, want valid JSON", snapshot.OutputSchema)
		}
		if got, want := snapshot.InputSchemaDigest, descriptor.InputSchemaDigest; got != want {
			t.Fatalf("snapshot.InputSchemaDigest = %q, want %q", got, want)
		}
		if got, want := snapshot.OutputSchemaDigest, descriptor.OutputSchemaDigest; got != want {
			t.Fatalf("snapshot.OutputSchemaDigest = %q, want %q", got, want)
		}

		snapshot.InputSchema[0] = '['
		next, ok := source.Snapshot(descriptor.ID.String())
		if !ok {
			t.Fatalf("Snapshot(%q) after mutation ok = false, want true", descriptor.ID)
		}
		if got, want := next.InputSchema[0], byte('{'); got != want {
			t.Fatalf("next.InputSchema[0] = %q, want %q", got, want)
		}
	})

	t.Run("Should reject invalid or unknown tool identifiers", func(t *testing.T) {
		t.Parallel()

		source := newLoopToolSchemaSource(
			context.Background(),
			loopToolSchemaRegistry{views: map[toolspkg.ToolID]toolspkg.ToolView{}},
		)
		if _, ok := source.Snapshot("not a valid tool id"); ok {
			t.Fatal("Snapshot(invalid id) ok = true, want false")
		}
		if _, ok := source.Snapshot("ext__dev_cycle__missing"); ok {
			t.Fatal("Snapshot(unknown id) ok = true, want false")
		}
	})

	t.Run("Should pass caller context to registry lookups", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		descriptor := loopToolSchemaDescriptor(t)
		source := newLoopToolSchemaSource(ctx, loopToolSchemaRegistry{
			views: map[toolspkg.ToolID]toolspkg.ToolView{
				descriptor.ID: {Descriptor: descriptor},
			},
		})

		if _, ok := source.Snapshot(descriptor.ID.String()); ok {
			t.Fatalf("Snapshot(%q) ok = true, want false after canceled context", descriptor.ID)
		}
	})

	t.Run("Should disable schema source when registry is unavailable", func(t *testing.T) {
		t.Parallel()

		if source := newLoopToolSchemaSource(context.Background(), nil); source != nil {
			t.Fatalf("newLoopToolSchemaSource(nil) = %T, want nil", source)
		}
	})
}

type loopToolSchemaRegistry struct {
	views map[toolspkg.ToolID]toolspkg.ToolView
}

func (r loopToolSchemaRegistry) List(context.Context, toolspkg.Scope) ([]toolspkg.ToolView, error) {
	views := make([]toolspkg.ToolView, 0, len(r.views))
	for id := range r.views {
		views = append(views, r.views[id])
	}
	return views, nil
}

func (r loopToolSchemaRegistry) Search(
	context.Context,
	toolspkg.Scope,
	toolspkg.SearchQuery,
) ([]toolspkg.ToolView, error) {
	return r.List(context.Background(), toolspkg.Scope{})
}

func (r loopToolSchemaRegistry) Get(
	ctx context.Context,
	_ toolspkg.Scope,
	id toolspkg.ToolID,
) (toolspkg.ToolView, error) {
	if err := ctx.Err(); err != nil {
		return toolspkg.ToolView{}, err
	}
	view, ok := r.views[id]
	if !ok {
		return toolspkg.ToolView{}, errors.New("tool not found")
	}
	return view, nil
}

func (r loopToolSchemaRegistry) Call(
	context.Context,
	toolspkg.Scope,
	toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	return toolspkg.ToolResult{}, errors.New("unexpected tool call")
}

func loopToolSchemaDescriptor(t *testing.T) toolspkg.Descriptor {
	t.Helper()

	descriptor := toolspkg.Descriptor{
		ID:           "ext__dev_cycle__git_push",
		DisplayTitle: "Git Push",
		Description:  "Push the current branch to a remote.",
		InputSchema:  json.RawMessage(`{"type":"object","properties":{"remote":{"type":"string"}}}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"pushed":{"type":"boolean"}}}`),
		Backend: toolspkg.BackendRef{
			Kind:       toolspkg.BackendNativeGo,
			NativeName: "git_push",
		},
		Source: toolspkg.SourceRef{
			Kind:  toolspkg.SourceExtension,
			Owner: "dev-cycle",
		},
		Visibility:      toolspkg.VisibilityModel,
		Risk:            toolspkg.RiskMutating,
		ReadOnly:        false,
		ConcurrencySafe: false,
		MaxResultBytes:  4096,
	}
	withDigests, err := toolspkg.DescriptorWithSchemaDigests(descriptor)
	if err != nil {
		t.Fatalf("DescriptorWithSchemaDigests() error = %v", err)
	}
	return withDigests
}
