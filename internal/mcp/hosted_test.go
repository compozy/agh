package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/tools"
)

func TestHostedServiceBindNonceLifecycle(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	executable := hostedTestExecutable(t, "agh")
	registry := &hostedRegistryStub{views: []tools.ToolView{hostedToolView("agh__hosted_echo")}}
	service := newHostedTestService(t, executable, registry, func() time.Time { return now })

	launch, err := service.Launch(t.Context(), HostedLaunchRequest{
		SessionID:   "sess-1",
		WorkspaceID: "ws-1",
		AgentName:   "codex",
	})
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	nonce := hostedLaunchNonce(t, launch.Args)
	peer := hostedTestPeer(executable)

	bind, err := service.Bind(t.Context(), HostedBindRequest{SessionID: "sess-1", Nonce: nonce}, peer)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if bind.Scope.SessionID != "sess-1" || bind.Scope.WorkspaceID != "ws-1" || bind.Scope.AgentName != "codex" {
		t.Fatalf("bind scope = %#v, want launch scope", bind.Scope)
	}

	_, err = service.Bind(t.Context(), HostedBindRequest{SessionID: "sess-1", Nonce: nonce}, peer)
	if !errors.Is(err, ErrHostedNonceInvalid) {
		t.Fatalf("second Bind() error = %v, want ErrHostedNonceInvalid", err)
	}
	if strings.Contains(err.Error(), nonce) {
		t.Fatalf("second Bind() leaked nonce in error: %q", err.Error())
	}

	expiring, err := service.Launch(t.Context(), HostedLaunchRequest{SessionID: "sess-expired"})
	if err != nil {
		t.Fatalf("Launch(expired) error = %v", err)
	}
	expiredNonce := hostedLaunchNonce(t, expiring.Args)
	now = now.Add(3 * time.Second)

	_, err = service.Bind(t.Context(), HostedBindRequest{SessionID: "sess-expired", Nonce: expiredNonce}, peer)
	if !errors.Is(err, ErrHostedNonceExpired) {
		t.Fatalf("expired Bind() error = %v, want ErrHostedNonceExpired", err)
	}
	if strings.Contains(err.Error(), expiredNonce) {
		t.Fatalf("expired Bind() leaked nonce in error: %q", err.Error())
	}
}

func TestHostedServiceValidatesPeerAndBinaryFailClosed(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	executable := hostedTestExecutable(t, "agh")
	otherExecutable := hostedTestExecutable(t, "other-agh")
	registry := &hostedRegistryStub{views: []tools.ToolView{hostedToolView("agh__hosted_echo")}}
	service := newHostedTestService(t, executable, registry, func() time.Time { return now })

	launch, err := service.Launch(t.Context(), HostedLaunchRequest{SessionID: "sess-1"})
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	nonce := hostedLaunchNonce(t, launch.Args)

	_, err = service.Bind(
		t.Context(),
		HostedBindRequest{SessionID: "sess-1", Nonce: nonce},
		PeerInfo{Supported: false, PID: 10, UID: os.Getuid(), ExecutablePath: executable},
	)
	if !errors.Is(err, ErrHostedPeerInvalid) {
		t.Fatalf("unsupported peer Bind() error = %v, want ErrHostedPeerInvalid", err)
	}

	_, err = service.Bind(
		t.Context(),
		HostedBindRequest{SessionID: "sess-1", Nonce: nonce},
		hostedTestPeer(otherExecutable),
	)
	if !errors.Is(err, ErrHostedBinaryInvalid) {
		t.Fatalf("wrong binary Bind() error = %v, want ErrHostedBinaryInvalid", err)
	}

	if _, err = service.Bind(
		t.Context(),
		HostedBindRequest{SessionID: "sess-1", Nonce: nonce},
		hostedTestPeer(executable),
	); err != nil {
		t.Fatalf("Bind(valid after failed validation) error = %v", err)
	}
}

func TestHostedServiceProjectionAndCallUseRegistryScope(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	executable := hostedTestExecutable(t, "agh")
	registry := &hostedRegistryStub{
		views: []tools.ToolView{
			hostedToolView("agh__zeta"),
			hostedToolView("agh__alpha"),
		},
		result: tools.ToolResult{Content: []tools.ToolContent{{Type: "text", Text: "ok"}}},
	}
	service := newHostedTestService(t, executable, registry, func() time.Time { return now })

	launch, err := service.Launch(t.Context(), HostedLaunchRequest{
		SessionID:   "sess-1",
		WorkspaceID: "ws-1",
		AgentName:   "codex",
	})
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	peer := hostedTestPeer(executable)
	bind, err := service.Bind(
		t.Context(),
		HostedBindRequest{SessionID: "sess-1", Nonce: hostedLaunchNonce(t, launch.Args)},
		peer,
	)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if got, want := hostedToolIDs(bind.Tools), []string{"agh__alpha", "agh__zeta"}; !slices.Equal(got, want) {
		t.Fatalf("bind tools = %#v, want sorted session projection %#v", got, want)
	}

	projection, err := service.Projection(t.Context(), bind.BindID, peer)
	if err != nil {
		t.Fatalf("Projection() error = %v", err)
	}
	if projection.Digest == "" || projection.Digest != bind.Digest {
		t.Fatalf(
			"projection digest = %q, bind digest = %q, want stable non-empty digest",
			projection.Digest,
			bind.Digest,
		)
	}

	_, err = service.Call(t.Context(), HostedCallRequest{
		BindID:        bind.BindID,
		ToolName:      "agh__alpha",
		ToolCallID:    "call-1",
		CorrelationID: "corr-1",
		Input:         json.RawMessage(`{"message":"hello","approval_token":"client-supplied"}`),
	}, peer)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	scope, call := registry.lastCall(t)
	if scope.SessionID != "sess-1" || scope.WorkspaceID != "ws-1" || scope.AgentName != "codex" {
		t.Fatalf("registry call scope = %#v, want hosted launch scope", scope)
	}
	if call.ToolID != "agh__alpha" || call.ToolCallID != "call-1" || call.CorrelationID != "corr-1" {
		t.Fatalf("registry call identity = %#v, want hosted call identity", call)
	}
	if call.ApprovalToken != "" {
		t.Fatalf("registry call ApprovalToken = %q, want empty", call.ApprovalToken)
	}
}

func TestHostedServiceReleaseAndFailureBranches(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	executable := hostedTestExecutable(t, "agh")
	registry := &hostedRegistryStub{views: []tools.ToolView{hostedToolView("agh__hosted_echo")}}
	peer := hostedTestPeer(executable)

	t.Run("Should reject disabled service and invalid construction", func(t *testing.T) {
		t.Parallel()

		disabled, err := NewHostedService(HostedConfig{ExpectedBinary: executable})
		if err != nil {
			t.Fatalf("NewHostedService(disabled) error = %v", err)
		}
		if _, err := disabled.Launch(
			t.Context(),
			HostedLaunchRequest{SessionID: "sess-disabled"},
		); !errors.Is(
			err,
			ErrHostedDisabled,
		) {
			t.Fatalf("Launch(disabled) error = %v, want ErrHostedDisabled", err)
		}
		if _, err := NewHostedService(HostedConfig{Enabled: true}); err == nil {
			t.Fatal("NewHostedService(blank expected binary) error = nil, want error")
		}
	})

	t.Run("Should cancel unbound launch records", func(t *testing.T) {
		t.Parallel()

		service := newHostedTestService(t, executable, registry, func() time.Time { return now })
		launch, err := service.Launch(t.Context(), HostedLaunchRequest{SessionID: "sess-cancel"})
		if err != nil {
			t.Fatalf("Launch() error = %v", err)
		}
		nonce := hostedLaunchNonce(t, launch.Args)
		service.CancelLaunch("sess-cancel")
		_, err = service.Bind(t.Context(), HostedBindRequest{SessionID: "sess-cancel", Nonce: nonce}, peer)
		if !errors.Is(err, ErrHostedNonceInvalid) {
			t.Fatalf("Bind(after CancelLaunch) error = %v, want ErrHostedNonceInvalid", err)
		}
	})

	t.Run("Should release bind and session records", func(t *testing.T) {
		t.Parallel()

		service := newHostedTestService(t, executable, registry, func() time.Time { return now })
		first := hostedTestBind(t, service, "sess-release-bind", peer)
		service.ReleaseBind(first.BindID)
		if _, err := service.Projection(t.Context(), first.BindID, peer); !errors.Is(err, ErrHostedBindNotFound) {
			t.Fatalf("Projection(after ReleaseBind) error = %v, want ErrHostedBindNotFound", err)
		}

		second := hostedTestBind(t, service, "sess-release-session", peer)
		service.ReleaseSession("sess-release-session")
		if _, err := service.Projection(t.Context(), second.BindID, peer); !errors.Is(err, ErrHostedBindNotFound) {
			t.Fatalf("Projection(after ReleaseSession) error = %v, want ErrHostedBindNotFound", err)
		}
	})

	t.Run("Should fail closed when registry is unavailable", func(t *testing.T) {
		t.Parallel()

		service := newHostedTestService(t, executable, nil, func() time.Time { return now })
		launch, err := service.Launch(t.Context(), HostedLaunchRequest{SessionID: "sess-no-registry"})
		if err != nil {
			t.Fatalf("Launch() error = %v", err)
		}
		_, err = service.Bind(
			t.Context(),
			HostedBindRequest{SessionID: "sess-no-registry", Nonce: hostedLaunchNonce(t, launch.Args)},
			peer,
		)
		if !errors.Is(err, ErrHostedRegistryRequired) {
			t.Fatalf("Bind(no registry) error = %v, want ErrHostedRegistryRequired", err)
		}
	})

	t.Run("Should reject missing bind and invalid tool names", func(t *testing.T) {
		t.Parallel()

		service := newHostedTestService(t, executable, registry, func() time.Time { return now })
		if _, err := service.Projection(t.Context(), "", peer); !errors.Is(err, ErrHostedBindRequired) {
			t.Fatalf("Projection(blank bind) error = %v, want ErrHostedBindRequired", err)
		}
		bind := hostedTestBind(t, service, "sess-invalid-call", peer)
		if _, err := service.Call(t.Context(), HostedCallRequest{
			BindID:   bind.BindID,
			ToolName: "not valid",
		}, peer); err == nil {
			t.Fatal("Call(invalid tool name) error = nil, want validation error")
		}
	})

	t.Run("Should default empty call input to object", func(t *testing.T) {
		t.Parallel()

		localRegistry := &hostedRegistryStub{views: []tools.ToolView{hostedToolView("agh__hosted_echo")}}
		service := newHostedTestService(t, executable, localRegistry, func() time.Time { return now })
		bind := hostedTestBind(t, service, "sess-empty-input", peer)
		if _, err := service.Call(t.Context(), HostedCallRequest{
			BindID:   bind.BindID,
			ToolName: "agh__hosted_echo",
		}, peer); err != nil {
			t.Fatalf("Call(empty input) error = %v", err)
		}
		_, call := localRegistry.lastCall(t)
		if string(call.Input) != `{}` {
			t.Fatalf("Call input = %s, want {}", call.Input)
		}
	})
}

func newHostedTestService(
	t *testing.T,
	executable string,
	registry tools.Registry,
	now func() time.Time,
) *HostedService {
	t.Helper()

	counter := byte(1)
	service, err := NewHostedService(HostedConfig{
		Enabled:        true,
		BindNonceTTL:   2 * time.Second,
		ExpectedBinary: executable,
		Registry: func() tools.Registry {
			return registry
		},
		Now: now,
		NonceReader: func(dst []byte) error {
			for i := range dst {
				dst[i] = counter
				counter++
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewHostedService() error = %v", err)
	}
	return service
}

func hostedTestBind(
	t *testing.T,
	service *HostedService,
	sessionID string,
	peer PeerInfo,
) HostedBindResponse {
	t.Helper()

	launch, err := service.Launch(t.Context(), HostedLaunchRequest{SessionID: sessionID})
	if err != nil {
		t.Fatalf("Launch(%q) error = %v", sessionID, err)
	}
	bind, err := service.Bind(
		t.Context(),
		HostedBindRequest{SessionID: sessionID, Nonce: hostedLaunchNonce(t, launch.Args)},
		peer,
	)
	if err != nil {
		t.Fatalf("Bind(%q) error = %v", sessionID, err)
	}
	return bind
}

func hostedTestExecutable(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", path, err)
	}
	return resolved
}

func hostedTestPeer(executable string) PeerInfo {
	return PeerInfo{
		Supported:      true,
		PID:            12345,
		UID:            os.Getuid(),
		ExecutablePath: executable,
	}
}

func hostedLaunchNonce(t *testing.T, args []string) string {
	t.Helper()

	for i := range args {
		if args[i] == "--bind-nonce" && i+1 < len(args) {
			return args[i+1]
		}
	}
	t.Fatalf("launch args = %#v, want --bind-nonce", args)
	return ""
}

func hostedToolIDs(views []tools.ToolView) []string {
	ids := make([]string, 0, len(views))
	for i := range views {
		ids = append(ids, views[i].Descriptor.ID.String())
	}
	return ids
}

func hostedToolView(id tools.ToolID) tools.ToolView {
	return tools.ToolView{
		Descriptor: tools.Descriptor{
			ID:           id,
			Backend:      tools.BackendRef{Kind: tools.BackendNativeGo, NativeName: id.String()},
			DisplayTitle: "Hosted " + id.String(),
			Description:  "Hosted test tool",
			InputSchema: json.RawMessage(
				`{"type":"object","properties":{"message":{"type":"string"}},"additionalProperties":false}`,
			),
			OutputSchema: json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}}}`),
			Source:       tools.SourceRef{Kind: tools.SourceBuiltin, Owner: "daemon"},
			Visibility:   tools.VisibilityModel,
			Risk:         tools.RiskRead,
			ReadOnly:     true,
		},
		Availability: tools.Availability{
			Registered: true,
			Enabled:    true,
			Available:  true,
			Authorized: true,
			Executable: true,
		},
		Decision: tools.EffectiveToolDecision{
			VisibleToOperator:    true,
			VisibleToSession:     true,
			Callable:             true,
			RegistryPolicyResult: "allowed",
		},
	}
}

type hostedRegistryStub struct {
	mu     sync.Mutex
	views  []tools.ToolView
	scopes []tools.Scope
	calls  []tools.CallRequest
	result tools.ToolResult
	err    error
}

var _ tools.Registry = (*hostedRegistryStub)(nil)

func (r *hostedRegistryStub) List(_ context.Context, scope tools.Scope) ([]tools.ToolView, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scopes = append(r.scopes, scope)
	if r.err != nil {
		return nil, r.err
	}
	out := make([]tools.ToolView, len(r.views))
	copy(out, r.views)
	return out, nil
}

func (r *hostedRegistryStub) Search(
	ctx context.Context,
	scope tools.Scope,
	_ tools.SearchQuery,
) ([]tools.ToolView, error) {
	return r.List(ctx, scope)
}

func (r *hostedRegistryStub) Get(ctx context.Context, scope tools.Scope, id tools.ToolID) (tools.ToolView, error) {
	views, err := r.List(ctx, scope)
	if err != nil {
		return tools.ToolView{}, err
	}
	for i := range views {
		if views[i].Descriptor.ID == id {
			return views[i], nil
		}
	}
	return tools.ToolView{}, tools.ErrToolNotFound
}

func (r *hostedRegistryStub) Call(
	_ context.Context,
	scope tools.Scope,
	req tools.CallRequest,
) (tools.ToolResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scopes = append(r.scopes, scope)
	r.calls = append(r.calls, req)
	if r.err != nil {
		return tools.ToolResult{}, r.err
	}
	if len(r.result.Content) > 0 || len(r.result.Structured) > 0 || r.result.Preview != "" {
		return r.result, nil
	}
	return tools.ToolResult{Content: []tools.ToolContent{{Type: "text", Text: "ok"}}}, nil
}

func (r *hostedRegistryStub) lastCall(t *testing.T) (tools.Scope, tools.CallRequest) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		t.Fatal("registry Call was not invoked")
	}
	return r.scopes[len(r.scopes)-1], r.calls[len(r.calls)-1]
}
