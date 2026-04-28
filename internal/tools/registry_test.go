package tools

import (
	"context"
	"errors"
	"slices"
	"testing"
)

type registryTestProvider struct {
	source      SourceRef
	descriptors []Descriptor
	handles     map[ToolID]Handle
	resolveErr  map[ToolID]error
}

var _ Provider = registryTestProvider{}

func (p registryTestProvider) ID() SourceRef {
	return p.source
}

func (p registryTestProvider) List(_ context.Context, _ Scope) ([]Descriptor, error) {
	return append([]Descriptor(nil), p.descriptors...), nil
}

func (p registryTestProvider) Resolve(_ context.Context, _ Scope, id ToolID) (Handle, bool, error) {
	if err, ok := p.resolveErr[id]; ok {
		return nil, false, err
	}
	handle, ok := p.handles[id]
	return handle, ok, nil
}

type registryTestHandle struct {
	descriptor   Descriptor
	availability Availability
	result       ToolResult
	callErr      error
	call         func(context.Context, CallRequest) (ToolResult, error)
}

var _ Handle = (*registryTestHandle)(nil)

func (h *registryTestHandle) Descriptor() Descriptor {
	return h.descriptor
}

func (h *registryTestHandle) Availability(_ context.Context, _ Scope) Availability {
	return h.availability
}

func (h *registryTestHandle) Call(ctx context.Context, req CallRequest) (ToolResult, error) {
	if h.call != nil {
		return h.call(ctx, req)
	}
	if h.callErr != nil {
		return ToolResult{}, h.callErr
	}
	if len(h.result.Content) > 0 || len(h.result.Structured) > 0 || h.result.Preview != "" {
		return h.result, nil
	}
	return ToolResult{Content: []ToolContent{{Type: "text", Text: "ok"}}}, nil
}

type staticPolicyEvaluator struct {
	decision EffectiveToolDecision
}

var _ PolicyEvaluator = staticPolicyEvaluator{}

func (e staticPolicyEvaluator) Evaluate(_ context.Context, _ Scope, _ Descriptor) (EffectiveToolDecision, error) {
	return e.decision, nil
}

func TestRuntimeRegistryIndexingAndCollisions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("Should mark duplicate canonical IDs as conflicted and hide from session projection", func(t *testing.T) {
		t.Parallel()

		descriptor := validDescriptor()
		provider := providerWithDescriptors(SourceRef{Kind: SourceBuiltin, Owner: "daemon"}, descriptor, descriptor)
		registry, err := NewRegistry(WithProviders(provider))
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}
		operatorViews, err := registry.OperatorProjection(ctx, Scope{Operator: true})
		if err != nil {
			t.Fatalf("OperatorProjection() error = %v", err)
		}
		if got, want := len(operatorViews), 1; got != want {
			t.Fatalf("len(operatorViews) = %d, want %d", got, want)
		}
		if !operatorViews[0].Availability.Conflicted {
			t.Fatalf("operator view availability = %#v, want conflicted", operatorViews[0].Availability)
		}
		requireDecisionReason(t, operatorViews[0].Decision, ReasonConflictedID)
		sessionViews, err := registry.SessionProjection(ctx, Scope{})
		if err != nil {
			t.Fatalf("SessionProjection() error = %v", err)
		}
		if len(sessionViews) != 0 {
			t.Fatalf("sessionViews = %#v, want empty", sessionViews)
		}
	})

	t.Run("Should mark sanitized external name collisions with specific reason code", func(t *testing.T) {
		t.Parallel()

		first := mcpDescriptor("mcp__github__create_issue", "github", "create issue")
		second := mcpDescriptor("mcp__github__create_issue", "github", "create-issue")
		provider := providerWithDescriptors(first.Source, first, second)
		registry, err := NewRegistry(WithProviders(provider))
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}
		views, err := registry.OperatorProjection(ctx, Scope{Operator: true})
		if err != nil {
			t.Fatalf("OperatorProjection() error = %v", err)
		}
		if got, want := len(views), 1; got != want {
			t.Fatalf("len(views) = %d, want %d", got, want)
		}
		if !slices.Contains(views[0].Availability.ReasonCodes, ReasonConflictedSanitizedName) {
			t.Fatalf("availability reasons = %#v, want sanitized collision", views[0].Availability.ReasonCodes)
		}
	})

	t.Run("Should fail closed on over length canonical IDs", func(t *testing.T) {
		t.Parallel()

		descriptor := validDescriptor()
		descriptor.ID = ToolID("agh__" + longASCII(65))
		provider := providerWithDescriptors(SourceRef{Kind: SourceBuiltin, Owner: "daemon"}, descriptor)
		registry, err := NewRegistry(WithProviders(provider))
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}
		_, err = registry.OperatorProjection(ctx, Scope{Operator: true})
		requireReason(t, err, ReasonIDTooLong)
	})
}

func TestRuntimeRegistryProjections(t *testing.T) {
	t.Parallel()

	t.Run("Should keep operator diagnostics broader than session projection", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		skillView := validDescriptor()
		taskUpdate := validDescriptor()
		taskUpdate.ID = "agh__task_update"
		taskUpdate.ReadOnly = false
		taskUpdate.Risk = RiskMutating
		networkSend := validDescriptor()
		networkSend.ID = "agh__network_send"
		networkSend.ReadOnly = false
		networkSend.OpenWorld = true
		networkSend.Risk = RiskOpenWorld
		conflicted := mcpDescriptor("mcp__github__search", "github", "search")
		conflictedDuplicate := mcpDescriptor("mcp__github__search", "github", "Search")
		provider := providerWithDescriptors(
			SourceRef{Kind: SourceBuiltin, Owner: "daemon"},
			skillView,
			taskUpdate,
			networkSend,
			conflicted,
			conflictedDuplicate,
		)
		provider.handles[networkSend.ID] = &registryTestHandle{
			descriptor: networkSend,
			availability: Availability{
				Registered:  true,
				Enabled:     true,
				ReasonCodes: []ReasonCode{ReasonBackendUnhealthy},
			},
		}
		denyTaskPattern, err := ParseToolPattern("agh__task_*")
		if err != nil {
			t.Fatalf("ParseToolPattern() error = %v", err)
		}
		registry, err := NewRegistry(
			WithProviders(provider),
			WithPolicyInputs(PolicyInputs{
				SystemPermissionMode: PermissionModeApproveAll,
				ExternalDefault:      ExternalDefaultEnabled,
				DenyTools:            []ToolPattern{denyTaskPattern},
			}, ToolsetCatalog{}),
		)
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}

		operatorViews, err := registry.OperatorProjection(ctx, Scope{Operator: true})
		if err != nil {
			t.Fatalf("OperatorProjection() error = %v", err)
		}
		if got, want := len(operatorViews), 4; got != want {
			t.Fatalf("len(operatorViews) = %d, want %d", got, want)
		}
		requireViewReason(t, operatorViews, taskUpdate.ID, ReasonPolicyDenied)
		requireViewReason(t, operatorViews, networkSend.ID, ReasonBackendUnhealthy)
		requireViewReason(t, operatorViews, conflicted.ID, ReasonConflictedSanitizedName)

		sessionViews, err := registry.SessionProjection(ctx, Scope{})
		if err != nil {
			t.Fatalf("SessionProjection() error = %v", err)
		}
		if got, want := len(sessionViews), 1; got != want {
			t.Fatalf("len(sessionViews) = %d, want %d: %#v", got, want, sessionViews)
		}
		if sessionViews[0].Descriptor.ID != skillView.ID {
			t.Fatalf("session tool = %q, want %q", sessionViews[0].Descriptor.ID, skillView.ID)
		}
	})
}

func TestRuntimeRegistrySearchGetAndCustomEvaluator(t *testing.T) {
	t.Parallel()

	t.Run("Should search and get through a custom evaluator", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		skillView := validDescriptor()
		skillView.Tags = []string{"skills"}
		taskRead := validDescriptor()
		taskRead.ID = "agh__task_read"
		taskRead.DisplayTitle = "Read Task"
		taskRead.Description = "Read task details"
		taskRead.Tags = []string{"tasks"}
		provider := providerWithDescriptors(SourceRef{Kind: SourceBuiltin, Owner: "daemon"}, skillView, taskRead)
		registry, err := NewRegistry(
			WithProviders(provider),
			WithPolicyEvaluator(staticPolicyEvaluator{decision: EffectiveToolDecision{
				VisibleToOperator:    true,
				VisibleToSession:     true,
				Callable:             true,
				SystemPermissionMode: string(PermissionModeApproveAll),
			}}),
		)
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}

		operatorList, err := registry.List(ctx, Scope{Operator: true})
		if err != nil {
			t.Fatalf("List(operator) error = %v", err)
		}
		if got, want := len(operatorList), 2; got != want {
			t.Fatalf("len(operatorList) = %d, want %d", got, want)
		}
		searchResults, err := registry.Search(ctx, Scope{}, SearchQuery{Query: "task", Limit: 1})
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if got, want := len(searchResults), 1; got != want {
			t.Fatalf("len(searchResults) = %d, want %d", got, want)
		}
		if searchResults[0].Descriptor.ID != taskRead.ID {
			t.Fatalf("search result = %q, want %q", searchResults[0].Descriptor.ID, taskRead.ID)
		}
		got, err := registry.Get(ctx, Scope{}, skillView.ID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.Descriptor.ID != skillView.ID {
			t.Fatalf("Get() = %q, want %q", got.Descriptor.ID, skillView.ID)
		}
	})
}

func providerWithDescriptors(source SourceRef, descriptors ...Descriptor) registryTestProvider {
	handles := make(map[ToolID]Handle, len(descriptors))
	for _, descriptor := range descriptors {
		handles[descriptor.ID] = &registryTestHandle{
			descriptor: descriptor,
			availability: Availability{
				Registered:  true,
				Enabled:     true,
				Available:   true,
				Authorized:  true,
				Executable:  true,
				ReasonCodes: nil,
			},
		}
	}
	return registryTestProvider{
		source:      source,
		descriptors: descriptors,
		handles:     handles,
		resolveErr:  map[ToolID]error{},
	}
}

func mcpDescriptor(id ToolID, owner string, rawName string) Descriptor {
	descriptor := validDescriptor()
	descriptor.ID = id
	descriptor.DisplayTitle = rawName
	descriptor.Backend = BackendRef{
		Kind:      BackendMCP,
		MCPServer: owner,
		MCPTool:   rawName,
	}
	descriptor.Source = SourceRef{
		Kind:          SourceMCP,
		Owner:         owner,
		RawServerName: owner,
		RawToolName:   rawName,
	}
	descriptor.Visibility = VisibilityModel
	descriptor.Risk = RiskRead
	descriptor.ReadOnly = true
	descriptor.Destructive = false
	descriptor.OpenWorld = false
	return descriptor
}

func requireViewReason(t *testing.T, views []ToolView, id ToolID, reason ReasonCode) {
	t.Helper()

	for i := range views {
		if views[i].Descriptor.ID != id {
			continue
		}
		requireDecisionReason(t, views[i].Decision, reason)
		return
	}
	t.Fatalf("view %q not found in %#v", id, views)
}

func longASCII(length int) string {
	return string(makeFilledBytes(length, 'a'))
}

func makeFilledBytes(length int, value byte) []byte {
	if length < 0 {
		return nil
	}
	data := make([]byte, length)
	for i := range data {
		data[i] = value
	}
	return data
}

func TestRuntimeRegistryCallDispatchesRegisteredTool(t *testing.T) {
	t.Parallel()

	t.Run("Should call provider handle after policy passes", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		descriptor := validDescriptor()
		called := false
		provider := providerWithDescriptors(SourceRef{Kind: SourceBuiltin, Owner: "daemon"}, descriptor)
		provider.handles[descriptor.ID] = &registryTestHandle{
			descriptor:   descriptor,
			availability: provider.handles[descriptor.ID].Availability(ctx, Scope{}),
			call: func(_ context.Context, req CallRequest) (ToolResult, error) {
				called = true
				if req.ToolID != descriptor.ID {
					t.Fatalf("CallRequest.ToolID = %q, want %q", req.ToolID, descriptor.ID)
				}
				return ToolResult{Content: []ToolContent{{Type: "text", Text: "ok"}}}, nil
			},
		}
		registry, err := NewRegistry(
			WithProviders(provider),
		)
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}
		result, err := registry.Call(ctx, Scope{}, CallRequest{ToolID: descriptor.ID})
		if err != nil {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want nil", err)
		}
		if !called {
			t.Fatal("provider handle was not called")
		}
		if len(result.Content) != 1 || result.Content[0].Text != "ok" {
			t.Fatalf("result = %#v, want ok text content", result)
		}
	})
}

func TestRuntimeRegistryCallReturnsPolicyDenialsBeforeDispatch(t *testing.T) {
	t.Parallel()

	t.Run("Should return denied instead of not found for registered denied tools", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		descriptor := validDescriptor()
		denyPattern, err := ParseToolPattern(descriptor.ID.String())
		if err != nil {
			t.Fatalf("ParseToolPattern() error = %v", err)
		}
		registry, err := NewRegistry(
			WithProviders(providerWithDescriptors(SourceRef{Kind: SourceBuiltin, Owner: "daemon"}, descriptor)),
			WithPolicyInputs(PolicyInputs{
				SystemPermissionMode: PermissionModeApproveAll,
				DenyTools:            []ToolPattern{denyPattern},
			}, ToolsetCatalog{}),
		)
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}
		_, err = registry.Call(ctx, Scope{}, CallRequest{ToolID: descriptor.ID})
		if !errors.Is(err, ErrToolDenied) {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want ErrToolDenied", err)
		}
	})
}
