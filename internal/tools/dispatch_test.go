package tools

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
)

type recordingHookRunner struct {
	pre       func(context.Context, CallRequest) (CallRequest, EffectiveToolDecision, error)
	post      func(context.Context, CallRequest, ToolResult) (ToolResult, error)
	postError func(context.Context, CallRequest, error) error
}

var _ HookRunner = (*recordingHookRunner)(nil)

func (h *recordingHookRunner) PreCall(
	ctx context.Context,
	call CallRequest,
) (CallRequest, EffectiveToolDecision, error) {
	if h.pre != nil {
		return h.pre(ctx, call)
	}
	return call, hookAllowedDecision(), nil
}

func (h *recordingHookRunner) PostCall(
	ctx context.Context,
	call CallRequest,
	result ToolResult,
) (ToolResult, error) {
	if h.post != nil {
		return h.post(ctx, call, result)
	}
	return result, nil
}

func (h *recordingHookRunner) PostError(ctx context.Context, call CallRequest, err error) error {
	if h.postError != nil {
		return h.postError(ctx, call, err)
	}
	return nil
}

type recordingToolEventSink struct {
	mu     sync.Mutex
	events []ToolCallEvent
}

var _ ToolEventSink = (*recordingToolEventSink)(nil)

func (s *recordingToolEventSink) EmitToolEvent(_ context.Context, event ToolCallEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *recordingToolEventSink) kinds() []ToolCallEventKind {
	s.mu.Lock()
	defer s.mu.Unlock()
	kinds := make([]ToolCallEventKind, 0, len(s.events))
	for _, event := range s.events {
		kinds = append(kinds, event.Kind)
	}
	return kinds
}

func (s *recordingToolEventSink) snapshot() []ToolCallEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]ToolCallEvent(nil), s.events...)
}

func TestRuntimeRegistryDispatchValidationAndPolicy(t *testing.T) {
	t.Parallel()

	t.Run("Should reject invalid input before provider invocation", func(t *testing.T) {
		t.Parallel()

		descriptor := validDispatchDescriptor()
		called := false
		events := &recordingToolEventSink{}
		provider := dispatchProviderWithHandle(descriptor, &registryTestHandle{
			descriptor:   descriptor,
			availability: availableDispatchHandle(),
			call: func(context.Context, CallRequest) (ToolResult, error) {
				called = true
				return ToolResult{}, nil
			},
		})
		registry := mustDispatchRegistry(t, provider, WithToolEventSink(events))

		_, err := registry.Call(t.Context(), Scope{}, CallRequest{
			ToolID: descriptor.ID,
			Input:  json.RawMessage(`{"query":42}`),
		})
		if !errors.Is(err, ErrToolInvalidInput) {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want ErrToolInvalidInput", err)
		}
		if called {
			t.Fatal("provider handle was called for invalid input")
		}
		if got, want := events.kinds(), []ToolCallEventKind{ToolCallStarted, ToolCallFailed}; !slices.Equal(got, want) {
			t.Fatalf("event kinds = %#v, want %#v", got, want)
		}
	})

	t.Run("Should recheck policy before provider invocation and emit denial", func(t *testing.T) {
		t.Parallel()

		descriptor := validDispatchDescriptor()
		denyPattern, err := ParseToolPattern(descriptor.ID.String())
		if err != nil {
			t.Fatalf("ParseToolPattern() error = %v", err)
		}
		called := false
		events := &recordingToolEventSink{}
		provider := dispatchProviderWithHandle(descriptor, &registryTestHandle{
			descriptor:   descriptor,
			availability: availableDispatchHandle(),
			call: func(context.Context, CallRequest) (ToolResult, error) {
				called = true
				return ToolResult{}, nil
			},
		})
		registry := mustDispatchRegistry(
			t,
			provider,
			WithToolEventSink(events),
			WithPolicyInputs(PolicyInputs{
				SystemPermissionMode: PermissionModeApproveAll,
				DenyTools:            []ToolPattern{denyPattern},
			}, ToolsetCatalog{}),
		)

		_, err = registry.Call(
			t.Context(),
			Scope{},
			CallRequest{ToolID: descriptor.ID, Input: json.RawMessage(`{"query":"x"}`)},
		)
		if !errors.Is(err, ErrToolDenied) {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want ErrToolDenied", err)
		}
		if called {
			t.Fatal("provider handle was called for denied tool")
		}
		eventSnapshot := events.snapshot()
		if got, want := events.kinds(), []ToolCallEventKind{ToolCallDenied}; !slices.Equal(got, want) {
			t.Fatalf("event kinds = %#v, want %#v", got, want)
		}
		if !slices.Contains(eventSnapshot[0].ReasonCodes, ReasonPolicyDenied) {
			t.Fatalf("denial reasons = %#v, want policy_denied", eventSnapshot[0].ReasonCodes)
		}
	})

	t.Run("Should recheck availability before provider invocation and emit denial", func(t *testing.T) {
		t.Parallel()

		descriptor := validDispatchDescriptor()
		called := false
		events := &recordingToolEventSink{}
		provider := dispatchProviderWithHandle(descriptor, &registryTestHandle{
			descriptor: descriptor,
			availability: Availability{
				Registered:  true,
				Enabled:     true,
				Available:   false,
				Authorized:  true,
				Executable:  false,
				ReasonCodes: []ReasonCode{ReasonBackendUnhealthy},
			},
			call: func(context.Context, CallRequest) (ToolResult, error) {
				called = true
				return ToolResult{}, nil
			},
		})
		registry := mustDispatchRegistry(t, provider, WithToolEventSink(events))

		_, err := registry.Call(
			t.Context(),
			Scope{},
			CallRequest{ToolID: descriptor.ID, Input: json.RawMessage(`{"query":"x"}`)},
		)
		if !errors.Is(err, ErrToolUnavailable) {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want ErrToolUnavailable", err)
		}
		if called {
			t.Fatal("provider handle was called for unavailable tool")
		}
		if got, want := events.kinds(), []ToolCallEventKind{ToolCallDenied}; !slices.Equal(got, want) {
			t.Fatalf("event kinds = %#v, want %#v", got, want)
		}
	})
}

func TestRuntimeRegistryDispatchHooksAndErrors(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve call context when pre-call hook patches input", func(t *testing.T) {
		t.Parallel()

		descriptor := validDispatchDescriptor()
		provider := dispatchProviderWithHandle(descriptor, &registryTestHandle{
			descriptor:   descriptor,
			availability: availableDispatchHandle(),
			call: func(_ context.Context, req CallRequest) (ToolResult, error) {
				if req.ToolCallID != "tool-call-1" || req.TurnID != "turn-1" || req.CorrelationID != "corr-1" {
					t.Fatalf("call context ids = %#v, want preserved ids", req)
				}
				if req.SessionID != "sess-1" || req.WorkspaceID != "ws-1" || req.AgentName != "codex" {
					t.Fatalf("call scope = %#v, want preserved scope", req)
				}
				if req.ApprovalToken != "approval-ref" || !slices.Contains(req.SensitiveInputFields, "token") {
					t.Fatalf("security context = %#v, want approval and sensitive field markers", req)
				}
				if string(req.Input) != `{"query":"patched"}` {
					t.Fatalf("CallRequest.Input = %s, want patched input", req.Input)
				}
				return ToolResult{Content: []ToolContent{{Type: "text", Text: "ok"}}}, nil
			},
		})
		hooks := &recordingHookRunner{
			pre: func(_ context.Context, _ CallRequest) (CallRequest, EffectiveToolDecision, error) {
				return CallRequest{Input: json.RawMessage(`{"query":"patched"}`)}, hookAllowedDecision(), nil
			},
		}
		registry := mustDispatchRegistry(t, provider, WithHookRunner(hooks))

		_, err := registry.Call(
			t.Context(),
			Scope{SessionID: "sess-1", WorkspaceID: "ws-1", AgentName: "codex"},
			CallRequest{
				ToolID:               descriptor.ID,
				ToolCallID:           "tool-call-1",
				TurnID:               "turn-1",
				CorrelationID:        "corr-1",
				Input:                json.RawMessage(`{"query":"original","token":"secret"}`),
				SensitiveInputFields: []string{"token"},
				ApprovalToken:        "approval-ref",
			},
		)
		if err != nil {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want nil", err)
		}
	})

	t.Run("Should let pre-call hook denial prevent provider invocation", func(t *testing.T) {
		t.Parallel()

		descriptor := validDispatchDescriptor()
		called := false
		events := &recordingToolEventSink{}
		provider := dispatchProviderWithHandle(descriptor, &registryTestHandle{
			descriptor:   descriptor,
			availability: availableDispatchHandle(),
			call: func(context.Context, CallRequest) (ToolResult, error) {
				called = true
				return ToolResult{}, nil
			},
		})
		hooks := &recordingHookRunner{
			pre: func(_ context.Context, call CallRequest) (CallRequest, EffectiveToolDecision, error) {
				if call.ToolID != descriptor.ID {
					t.Fatalf("PreCall ToolID = %q, want %q", call.ToolID, descriptor.ID)
				}
				return call, EffectiveToolDecision{
					Callable:   false,
					HookResult: policyResultDenied,
				}, nil
			},
		}
		registry := mustDispatchRegistry(t, provider, WithHookRunner(hooks), WithToolEventSink(events))

		_, err := registry.Call(
			t.Context(),
			Scope{},
			CallRequest{ToolID: descriptor.ID, Input: json.RawMessage(`{"query":"x"}`)},
		)
		if !errors.Is(err, ErrToolDenied) {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want ErrToolDenied", err)
		}
		if called {
			t.Fatal("provider handle was called after pre-call denial")
		}
		eventSnapshot := events.snapshot()
		if got, want := events.kinds(), []ToolCallEventKind{ToolCallStarted, ToolCallDenied}; !slices.Equal(got, want) {
			t.Fatalf("event kinds = %#v, want %#v", got, want)
		}
		if !slices.Contains(eventSnapshot[1].ReasonCodes, ReasonHookDenied) {
			t.Fatalf("denial reasons = %#v, want hook_denied", eventSnapshot[1].ReasonCodes)
		}
	})

	t.Run("Should run post-error hook with canonical tool ID on provider failure", func(t *testing.T) {
		t.Parallel()

		descriptor := validDispatchDescriptor()
		events := &recordingToolEventSink{}
		provider := dispatchProviderWithHandle(descriptor, &registryTestHandle{
			descriptor:   descriptor,
			availability: availableDispatchHandle(),
			callErr:      errors.New("provider exploded"),
		})
		postErrorCalled := false
		hooks := &recordingHookRunner{
			postError: func(_ context.Context, call CallRequest, err error) error {
				postErrorCalled = true
				if call.ToolID != descriptor.ID {
					t.Fatalf("PostError ToolID = %q, want %q", call.ToolID, descriptor.ID)
				}
				if !errors.Is(err, ErrToolBackendFailed) {
					t.Fatalf("PostError error = %v, want ErrToolBackendFailed", err)
				}
				return nil
			},
		}
		registry := mustDispatchRegistry(t, provider, WithHookRunner(hooks), WithToolEventSink(events))

		_, err := registry.Call(
			t.Context(),
			Scope{},
			CallRequest{ToolID: descriptor.ID, Input: json.RawMessage(`{"query":"x"}`)},
		)
		if !errors.Is(err, ErrToolBackendFailed) {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want ErrToolBackendFailed", err)
		}
		if !postErrorCalled {
			t.Fatal("post-error hook was not called")
		}
		if got, want := events.kinds(), []ToolCallEventKind{ToolCallStarted, ToolCallFailed}; !slices.Equal(got, want) {
			t.Fatalf("event kinds = %#v, want %#v", got, want)
		}
	})

	t.Run("Should normalize provider cancellation errors deterministically", func(t *testing.T) {
		t.Parallel()

		descriptor := validDispatchDescriptor()
		events := &recordingToolEventSink{}
		provider := dispatchProviderWithHandle(descriptor, &registryTestHandle{
			descriptor:   descriptor,
			availability: availableDispatchHandle(),
			callErr:      context.Canceled,
		})
		registry := mustDispatchRegistry(t, provider, WithToolEventSink(events))

		_, err := registry.Call(
			t.Context(),
			Scope{},
			CallRequest{ToolID: descriptor.ID, Input: json.RawMessage(`{"query":"x"}`)},
		)
		if !errors.Is(err, ErrToolCanceled) {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want ErrToolCanceled", err)
		}
		eventSnapshot := events.snapshot()
		if got, want := events.kinds(), []ToolCallEventKind{ToolCallStarted, ToolCallFailed}; !slices.Equal(got, want) {
			t.Fatalf("event kinds = %#v, want %#v", got, want)
		}
		if got, want := eventSnapshot[1].ErrorCode, ErrorCodeCanceled; got != want {
			t.Fatalf("failed event error code = %q, want %q", got, want)
		}
		if !slices.Contains(eventSnapshot[1].ReasonCodes, ReasonCallCanceled) {
			t.Fatalf("failed event reasons = %#v, want call_canceled", eventSnapshot[1].ReasonCodes)
		}
	})
}

func TestRuntimeRegistryDispatchResultLimitingAndRedaction(t *testing.T) {
	t.Parallel()

	t.Run("Should redact sensitive result fields before post-call hooks and events", func(t *testing.T) {
		t.Parallel()

		descriptor := validDispatchDescriptor()
		descriptor.MaxResultBytes = 4096
		events := &recordingToolEventSink{}
		provider := dispatchProviderWithHandle(descriptor, &registryTestHandle{
			descriptor:   descriptor,
			availability: availableDispatchHandle(),
			result: ToolResult{
				Content: []ToolContent{
					{
						Type: "json",
						Data: json.RawMessage(`{"access_token":"secret","visible":"ok"}`),
						Metadata: map[string]json.RawMessage{
							"refresh_token": json.RawMessage(`"secret"`),
							"safe":          json.RawMessage(`"ok"`),
						},
					},
				},
				Structured: json.RawMessage(`{"password":"secret","visible":"ok"}`),
				Metadata: map[string]json.RawMessage{
					"api_key": json.RawMessage(`"secret"`),
					"safe":    json.RawMessage(`"ok"`),
				},
			},
		})
		postCalled := false
		hooks := &recordingHookRunner{
			post: func(_ context.Context, call CallRequest, result ToolResult) (ToolResult, error) {
				postCalled = true
				if call.ToolID != descriptor.ID {
					t.Fatalf("PostCall ToolID = %q, want %q", call.ToolID, descriptor.ID)
				}
				data, err := json.Marshal(result)
				if err != nil {
					t.Fatalf("json.Marshal(result) error = %v", err)
				}
				if strings.Contains(string(data), `"secret"`) {
					t.Fatalf("post-call result leaked secret: %s", data)
				}
				if _, ok := result.Metadata["api_key"]; ok {
					t.Fatalf("result.Metadata contains api_key after redaction: %#v", result.Metadata)
				}
				if _, ok := result.Content[0].Metadata["refresh_token"]; ok {
					t.Fatalf("content metadata contains refresh_token after redaction: %#v", result.Content[0].Metadata)
				}
				if len(result.Redactions) == 0 {
					t.Fatal("result.Redactions is empty, want redaction metadata")
				}
				return result, nil
			},
		}
		registry := mustDispatchRegistry(
			t,
			provider,
			WithHookRunner(hooks),
			WithToolEventSink(events),
			WithSensitiveResultFields("api_key"),
		)

		result, err := registry.Call(t.Context(), Scope{}, CallRequest{
			ToolID:               descriptor.ID,
			Input:                json.RawMessage(`{"query":"x","token":"secret"}`),
			SensitiveInputFields: []string{"token"},
		})
		if err != nil {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want nil", err)
		}
		if !postCalled {
			t.Fatal("post-call hook was not called")
		}
		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("json.Marshal(result) error = %v", err)
		}
		if strings.Contains(string(data), `"secret"`) {
			t.Fatalf("result leaked secret: %s", data)
		}
		eventData, err := json.Marshal(events.snapshot())
		if err != nil {
			t.Fatalf("json.Marshal(events) error = %v", err)
		}
		if strings.Contains(string(eventData), `"secret"`) {
			t.Fatalf("events leaked secret: %s", eventData)
		}
		eventSnapshot := events.snapshot()
		if len(eventSnapshot[0].RedactedInputFields) == 0 {
			t.Fatalf(
				"started event redacted input fields = %#v, want token redaction",
				eventSnapshot[0].RedactedInputFields,
			)
		}
	})

	t.Run("Should truncate oversized results with deterministic metadata", func(t *testing.T) {
		t.Parallel()

		descriptor := validDispatchDescriptor()
		descriptor.MaxResultBytes = 320
		events := &recordingToolEventSink{}
		provider := dispatchProviderWithHandle(descriptor, &registryTestHandle{
			descriptor:   descriptor,
			availability: availableDispatchHandle(),
			result: ToolResult{
				Content: []ToolContent{{Type: "text", Text: strings.Repeat("x", 1024)}},
			},
		})
		registry := mustDispatchRegistry(t, provider, WithToolEventSink(events))

		result, err := registry.Call(
			t.Context(),
			Scope{},
			CallRequest{ToolID: descriptor.ID, Input: json.RawMessage(`{"query":"x"}`)},
		)
		if err != nil {
			t.Fatalf("RuntimeRegistry.Call() error = %v, want nil", err)
		}
		if !result.Truncated {
			t.Fatalf("result.Truncated = false, want true: %#v", result)
		}
		if _, ok := result.Metadata["truncated_from_bytes"]; !ok {
			t.Fatalf("result.Metadata = %#v, want truncated_from_bytes", result.Metadata)
		}
		if !slices.ContainsFunc(result.Redactions, func(redaction Redaction) bool {
			return redaction.Reason == ReasonResultBudgetExceeded
		}) {
			t.Fatalf("result.Redactions = %#v, want result budget redaction", result.Redactions)
		}
		if got, want := events.kinds(), []ToolCallEventKind{
			ToolCallStarted,
			ToolCallCompleted,
			ToolResultTruncated,
		}; !slices.Equal(got, want) {
			t.Fatalf("event kinds = %#v, want %#v", got, want)
		}
	})
}

func validDispatchDescriptor() Descriptor {
	descriptor := validDescriptor()
	descriptor.ID = "agh__dispatch_probe"
	descriptor.Backend.NativeName = "dispatch_probe"
	descriptor.InputSchema = json.RawMessage(`{
		"type":"object",
		"required":["query"],
		"properties":{"query":{"type":"string"},"token":{"type":"string"}},
		"additionalProperties":false
	}`)
	return descriptor
}

func hookAllowedDecision() EffectiveToolDecision {
	return EffectiveToolDecision{
		Callable:   true,
		HookResult: policyResultAllowed,
	}
}

func availableDispatchHandle() Availability {
	return Availability{
		Registered: true,
		Enabled:    true,
		Available:  true,
		Authorized: true,
		Executable: true,
	}
}

func dispatchProviderWithHandle(descriptor Descriptor, handle *registryTestHandle) registryTestProvider {
	return registryTestProvider{
		source:      SourceRef{Kind: SourceBuiltin, Owner: "daemon"},
		descriptors: []Descriptor{descriptor},
		handles:     map[ToolID]Handle{descriptor.ID: handle},
		resolveErr:  map[ToolID]error{},
	}
}

func mustDispatchRegistry(t *testing.T, provider registryTestProvider, opts ...RegistryOption) *RuntimeRegistry {
	t.Helper()

	options := append([]RegistryOption{WithProviders(provider)}, opts...)
	registry, err := NewRegistry(options...)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	return registry
}
