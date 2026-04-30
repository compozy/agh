package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestShouldCanonicalizeMCPToolIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		server     string
		tool       string
		want       ToolID
		wantErr    bool
		wantReason ReasonCode
	}{
		{
			name:   "ShouldNormalizeSupportedRawNames",
			server: " Git-Hub ",
			tool:   "Search.Tool",
			want:   "mcp__git_hub__search_tool",
		},
		{
			name:       "ShouldRejectDigitStart",
			server:     "9github",
			tool:       "search",
			wantErr:    true,
			wantReason: ReasonIDInvalidFormat,
		},
		{
			name:       "ShouldRejectReservedSeparatorAfterNormalization",
			server:     "git--hub",
			tool:       "search",
			wantErr:    true,
			wantReason: ReasonIDReservedConflict,
		},
		{
			name:       "ShouldRejectUnsupportedCharacters",
			server:     "git hub",
			tool:       "search",
			wantErr:    true,
			wantReason: ReasonIDInvalidFormat,
		},
		{
			name:       "ShouldRejectOverLengthIDs",
			server:     "github",
			tool:       "tool_" + strings.Repeat("a", 64),
			wantErr:    true,
			wantReason: ReasonIDTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Canonicalize(tt.server, tt.tool)
			if tt.wantErr {
				requireReason(t, err, tt.wantReason)
				return
			}
			if err != nil {
				t.Fatalf("Canonicalize() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Canonicalize() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShouldMapMCPAuthStatusToRegistryReasons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status string
		want   ReasonCode
		mapped bool
	}{
		{status: "unconfigured", want: ReasonMCPAuthUnconfigured, mapped: true},
		{status: "needs_login", want: ReasonMCPAuthRequired, mapped: true},
		{status: "expired", want: ReasonMCPAuthExpired, mapped: true},
		{status: "invalid", want: ReasonMCPAuthInvalid, mapped: true},
		{status: "refresh_failed", want: ReasonMCPAuthRefreshFailed, mapped: true},
		{status: "authenticated", mapped: false},
		{status: "", mapped: false},
	}

	for _, tt := range tests {
		name := "Should Map " + tt.status
		if tt.status == "" {
			name = "Should Map Empty Status"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, mapped := MCPAuthStatusReason(MCPAuthStatus{Status: tt.status})
			if mapped != tt.mapped {
				t.Fatalf("MCPAuthStatusReason(%q) mapped = %v, want %v", tt.status, mapped, tt.mapped)
			}
			if got != tt.want {
				t.Fatalf("MCPAuthStatusReason(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestShouldExposeMCPProviderDescriptorsAndPreserveOutputSchema(t *testing.T) {
	t.Run("Should Expose MCP Provider Descriptors And Preserve Output Schema", func(t *testing.T) {
		t.Parallel()

		outputSchema := json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}}}`)
		executor := newFakeMCPExecutor([]MCPToolDescriptor{{
			RawName:      "lookup",
			Title:        "Lookup",
			Description:  "Lookup data",
			InputSchema:  json.RawMessage(`{"type":"object","properties":{}}`),
			OutputSchema: outputSchema,
			ReadOnly:     true,
		}})
		provider := newTestMCPProvider(t, executor, []SourceRef{{
			Kind:          SourceMCP,
			Owner:         "github",
			RawServerName: "github",
		}})

		descriptors, err := provider.List(context.Background(), Scope{Operator: true})
		if err != nil {
			t.Fatalf("provider.List() error = %v", err)
		}
		if got, want := len(descriptors), 1; got != want {
			t.Fatalf("len(provider.List()) = %d, want %d", got, want)
		}
		descriptor := descriptors[0]
		if got, want := descriptor.ID, ToolID("mcp__github__lookup"); got != want {
			t.Fatalf("descriptor.ID = %q, want %q", got, want)
		}
		if got, want := string(descriptor.OutputSchema), string(outputSchema); got != want {
			t.Fatalf("descriptor.OutputSchema = %s, want %s", got, want)
		}
		if got, want := descriptor.Source.RawToolName, "lookup"; got != want {
			t.Fatalf("descriptor.Source.RawToolName = %q, want %q", got, want)
		}
		if !descriptor.ReadOnly || descriptor.Risk != RiskRead || descriptor.OpenWorld {
			t.Fatalf("descriptor risk flags = %#v, want read-only local-safe flags", descriptor)
		}
	})
}

func TestShouldFailClosedOnMCPSanitizedNameCollisions(t *testing.T) {
	t.Run("Should Fail Closed On MCP Sanitized Name Collisions", func(t *testing.T) {
		t.Parallel()

		executor := newFakeMCPExecutor([]MCPToolDescriptor{
			{
				RawName:     "foo-bar",
				InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			},
			{
				RawName:     "foo.bar",
				InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
			},
		})
		provider := newTestMCPProvider(t, executor, []SourceRef{{
			Kind:          SourceMCP,
			Owner:         "github",
			RawServerName: "github",
		}})
		registry, err := NewRegistry(WithProviders(provider))
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}

		views, err := registry.OperatorProjection(context.Background(), Scope{Operator: true})
		if err != nil {
			t.Fatalf("registry.OperatorProjection() error = %v", err)
		}
		if got, want := len(views), 1; got != want {
			t.Fatalf("len(views) = %d, want %d", got, want)
		}
		if !views[0].Availability.Conflicted {
			t.Fatalf("views[0].Availability.Conflicted = false, want true")
		}
		requireDecisionReason(t, views[0].Decision, ReasonConflictedSanitizedName)
	})
}

func TestShouldBlockMCPCallsWhenAuthIsRequired(t *testing.T) {
	t.Run("Should Block MCP Calls When Auth Is Required", func(t *testing.T) {
		t.Parallel()

		executor := newFakeMCPExecutor([]MCPToolDescriptor{{
			RawName:     "lookup",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		}})
		executor.status = MCPAuthStatus{Status: "needs_login"}
		provider := newTestMCPProvider(t, executor, []SourceRef{{
			Kind:          SourceMCP,
			Owner:         "github",
			RawServerName: "github",
		}})
		registry := newMCPRegistry(t, provider)

		_, err := registry.Call(context.Background(), Scope{}, CallRequest{
			ToolID: "mcp__github__lookup",
			Input:  json.RawMessage(`{}`),
		})
		requireReason(t, err, ReasonMCPAuthRequired)
		if !errors.Is(err, ErrToolUnavailable) {
			t.Fatalf("registry.Call() error = %v, want ErrToolUnavailable", err)
		}
	})
}

func TestShouldSkipMCPSourceWhenAuthBlocksDiscovery(t *testing.T) {
	t.Run("Should Skip MCP Source When Auth Blocks Discovery", func(t *testing.T) {
		t.Parallel()

		executor := newFakeMCPExecutor([]MCPToolDescriptor{{
			RawName:     "lookup",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		}})
		executor.listErr = NewToolError(
			ErrorCodeUnavailable,
			"mcp__github__lookup",
			"mcp login required",
			ErrToolUnavailable,
			ReasonMCPAuthRequired,
		)
		provider := newTestMCPProvider(t, executor, []SourceRef{{
			Kind:          SourceMCP,
			Owner:         "github",
			RawServerName: "github",
		}})

		descriptors, err := provider.List(context.Background(), Scope{Operator: true})
		if err != nil {
			t.Fatalf("provider.List() error = %v", err)
		}
		if len(descriptors) != 0 {
			t.Fatalf("provider.List() descriptors = %#v, want empty while auth blocks discovery", descriptors)
		}
	})
}

func TestShouldCallMCPProviderThroughRegistry(t *testing.T) {
	t.Run("Should Call MCP Provider Through Registry", func(t *testing.T) {
		t.Parallel()

		executor := newFakeMCPExecutor([]MCPToolDescriptor{{
			RawName:     "lookup",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		}})
		executor.status = MCPAuthStatus{Status: "authenticated"}
		provider := newTestMCPProvider(t, executor, []SourceRef{{
			Kind:          SourceMCP,
			Owner:         "github",
			RawServerName: "github",
		}})
		registry := newMCPRegistry(t, provider)

		result, err := registry.Call(context.Background(), Scope{}, CallRequest{
			ToolID: "mcp__github__lookup",
			Input:  json.RawMessage(`{"query":"octo"}`),
		})
		if err != nil {
			t.Fatalf("registry.Call() error = %v", err)
		}
		if got, want := result.Preview, "called lookup"; got != want {
			t.Fatalf("result.Preview = %q, want %q", got, want)
		}
		if got, want := executor.lastCall().RawToolName, "lookup"; got != want {
			t.Fatalf("executor.lastCall().RawToolName = %q, want %q", got, want)
		}
	})
}

type fakeMCPExecutor struct {
	mu      sync.Mutex
	tools   []MCPToolDescriptor
	status  MCPAuthStatus
	listErr error
	calls   []MCPToolCallRequest
}

func newFakeMCPExecutor(tools []MCPToolDescriptor) *fakeMCPExecutor {
	return &fakeMCPExecutor{
		tools:  tools,
		status: MCPAuthStatus{Status: "unconfigured"},
	}
}

func (f *fakeMCPExecutor) ListTools(context.Context, SourceRef) ([]MCPToolDescriptor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]MCPToolDescriptor, len(f.tools))
	for i := range f.tools {
		out[i] = f.tools[i]
		out[i].InputSchema = cloneRawMessage(f.tools[i].InputSchema)
		out[i].OutputSchema = cloneRawMessage(f.tools[i].OutputSchema)
	}
	return out, nil
}

func (f *fakeMCPExecutor) CallTool(
	_ context.Context,
	_ SourceRef,
	req MCPToolCallRequest,
) (ToolResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, req)
	return ToolResult{
		Content: []ToolContent{{
			Type: "text",
			Text: "called " + req.RawToolName,
		}},
		Preview: "called " + req.RawToolName,
	}, nil
}

func (f *fakeMCPExecutor) Status(context.Context, SourceRef) (MCPAuthStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status, nil
}

func (f *fakeMCPExecutor) lastCall() MCPToolCallRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return MCPToolCallRequest{}
	}
	return f.calls[len(f.calls)-1]
}

func newTestMCPProvider(t *testing.T, executor *fakeMCPExecutor, sources []SourceRef) *MCPProvider {
	t.Helper()

	provider, err := NewMCPProvider(
		MCPSourceListerFunc(func(context.Context) ([]SourceRef, error) {
			return sources, nil
		}),
		executor,
		executor,
	)
	if err != nil {
		t.Fatalf("NewMCPProvider() error = %v", err)
	}
	return provider
}

func newMCPRegistry(t *testing.T, provider Provider) *RuntimeRegistry {
	t.Helper()

	registry, err := NewRegistry(
		WithProviders(provider),
		WithPolicyInputs(PolicyInputs{
			SystemPermissionMode: PermissionModeApproveAll,
			ExternalDefault:      ExternalDefaultEnabled,
			ApprovalAvailable:    true,
		}, ToolsetCatalog{}),
	)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	return registry
}
