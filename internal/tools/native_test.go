package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestNativeProviderDispatch(t *testing.T) {
	t.Parallel()

	t.Run("Should resolve and call native handlers through registry dispatch", func(t *testing.T) {
		t.Parallel()

		descriptor := validDescriptor()
		descriptor.ID = ToolIDSkillList
		descriptor.Backend.NativeName = "skill_list"
		called := false
		provider, err := NewNativeProvider(descriptor.Source, NativeTool{
			Descriptor: descriptor,
			Call: func(_ context.Context, scope Scope, req CallRequest) (ToolResult, error) {
				called = true
				if scope.SessionID != "sess-1" || scope.WorkspaceID != "ws-1" || scope.AgentName != "agent-1" {
					t.Fatalf("scope = %#v, want call request scope", scope)
				}
				if req.ToolID != descriptor.ID {
					t.Fatalf("req.ToolID = %q, want %q", req.ToolID, descriptor.ID)
				}
				return ToolResult{Structured: json.RawMessage(`{"ok":true}`)}, nil
			},
		})
		if err != nil {
			t.Fatalf("NewNativeProvider() error = %v", err)
		}
		registry, err := NewRegistry(WithProviders(provider))
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}

		result, err := registry.Call(t.Context(), Scope{
			SessionID:   "sess-1",
			WorkspaceID: "ws-1",
			AgentName:   "agent-1",
		}, CallRequest{ToolID: descriptor.ID})
		if err != nil {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want nil", err)
		}
		if !called {
			t.Fatal("native handler was not called")
		}
		if got, want := string(result.Structured), `{"ok":true}`; got != want {
			t.Fatalf("result.Structured = %s, want %s", got, want)
		}
	})

	t.Run("Should reject schema invalid input before native handler invocation", func(t *testing.T) {
		t.Parallel()

		descriptor := validDispatchDescriptor()
		called := false
		provider, err := NewNativeProvider(descriptor.Source, NativeTool{
			Descriptor: descriptor,
			Call: func(context.Context, Scope, CallRequest) (ToolResult, error) {
				called = true
				return ToolResult{}, nil
			},
		})
		if err != nil {
			t.Fatalf("NewNativeProvider() error = %v", err)
		}
		registry, err := NewRegistry(WithProviders(provider))
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}

		_, err = registry.Call(
			t.Context(),
			Scope{},
			CallRequest{ToolID: descriptor.ID, Input: json.RawMessage(`{"query":7}`)},
		)
		if !errors.Is(err, ErrToolInvalidInput) {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want ErrToolInvalidInput", err)
		}
		if called {
			t.Fatal("native handler was called for schema-invalid input")
		}
	})

	t.Run("Should surface unavailable dependencies before native handler invocation", func(t *testing.T) {
		t.Parallel()

		descriptor := validDescriptor()
		called := false
		provider, err := NewNativeProvider(descriptor.Source, NativeTool{
			Descriptor: descriptor,
			Availability: func(context.Context, Scope) Availability {
				return Unavailable(ReasonDependencyMissing)
			},
			Call: func(context.Context, Scope, CallRequest) (ToolResult, error) {
				called = true
				return ToolResult{}, nil
			},
		})
		if err != nil {
			t.Fatalf("NewNativeProvider() error = %v", err)
		}
		registry, err := NewRegistry(WithProviders(provider))
		if err != nil {
			t.Fatalf("NewRegistry() error = %v", err)
		}

		_, err = registry.Call(t.Context(), Scope{}, CallRequest{ToolID: descriptor.ID})
		if !errors.Is(err, ErrToolUnavailable) {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want ErrToolUnavailable", err)
		}
		if called {
			t.Fatal("native handler was called for unavailable dependency")
		}
	})
}

func TestNativeProviderValidation(t *testing.T) {
	t.Parallel()

	t.Run("Should reject native tools without handlers", func(t *testing.T) {
		t.Parallel()

		descriptor := validDescriptor()
		_, err := NewNativeProvider(descriptor.Source, NativeTool{Descriptor: descriptor})
		requireReason(t, err, ReasonHandlerMissing)
	})

	t.Run("Should reject native tools whose source differs from provider source", func(t *testing.T) {
		t.Parallel()

		descriptor := validDescriptor()
		source := descriptor.Source
		source.Owner = "other"
		_, err := NewNativeProvider(source, NativeTool{
			Descriptor: descriptor,
			Call: func(context.Context, Scope, CallRequest) (ToolResult, error) {
				return ToolResult{}, nil
			},
		})
		requireReason(t, err, ReasonSourceDisabled)
	})

	t.Run("Should reject duplicate native tool IDs", func(t *testing.T) {
		t.Parallel()

		descriptor := validDescriptor()
		nativeTool := NativeTool{
			Descriptor: descriptor,
			Call: func(context.Context, Scope, CallRequest) (ToolResult, error) {
				return ToolResult{}, nil
			},
		}
		_, err := NewNativeProvider(descriptor.Source, nativeTool, nativeTool)
		requireReason(t, err, ReasonConflictedID)
	})
}
