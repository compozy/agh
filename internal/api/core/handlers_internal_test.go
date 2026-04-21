package core

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	bundlepkg "github.com/pedronauck/agh/internal/bundles"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
)

type sessionManagerStub struct {
	status func(context.Context, string) (*session.Info, error)
}

func (s sessionManagerStub) Create(context.Context, session.CreateOpts) (*session.Session, error) {
	return nil, nil
}

func (s sessionManagerStub) List() []*session.Info { return nil }

func (s sessionManagerStub) ListAll(context.Context) ([]*session.Info, error) { return nil, nil }

func (s sessionManagerStub) Status(ctx context.Context, id string) (*session.Info, error) {
	if s.status != nil {
		return s.status(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s sessionManagerStub) Events(context.Context, string, store.EventQuery) ([]store.SessionEvent, error) {
	return nil, nil
}

func (s sessionManagerStub) History(context.Context, string, store.EventQuery) ([]store.TurnHistory, error) {
	return nil, nil
}

func (s sessionManagerStub) Transcript(context.Context, string) ([]transcript.Message, error) {
	return nil, nil
}

func (s sessionManagerStub) Stop(context.Context, string) error { return nil }

func (s sessionManagerStub) StopWithCause(context.Context, string, session.StopCause, string) error {
	return nil
}

func (s sessionManagerStub) Resume(context.Context, string) (*session.Session, error) {
	return nil, nil
}

func (s sessionManagerStub) ClearConversation(context.Context, string) (*session.Session, error) {
	return nil, nil
}

func (s sessionManagerStub) Prompt(context.Context, string, string) (<-chan acp.AgentEvent, error) {
	ch := make(chan acp.AgentEvent)
	close(ch)
	return ch, nil
}

func (s sessionManagerStub) CancelPrompt(context.Context, string) error {
	return nil
}

func (s sessionManagerStub) ApprovePermission(context.Context, string, acp.ApproveRequest) error {
	return nil
}

type bundleServiceStub struct {
	networkSettingsFn func(context.Context) (bundlepkg.NetworkSettings, error)
}

type networkServiceStub struct {
	statusFn func(context.Context) (*network.Status, error)
}

func (s networkServiceStub) Send(context.Context, network.SendRequest) (string, error) {
	return "", nil
}

func (s networkServiceStub) ListPeers(context.Context, string) ([]network.PeerInfo, error) {
	return nil, nil
}

func (s networkServiceStub) ListChannels(context.Context) ([]network.ChannelInfo, error) {
	return nil, nil
}

func (s networkServiceStub) Status(ctx context.Context) (*network.Status, error) {
	if s.statusFn != nil {
		return s.statusFn(ctx)
	}
	return nil, nil
}

func (s networkServiceStub) Inbox(context.Context, string) ([]network.Envelope, error) {
	return nil, nil
}

func (s bundleServiceStub) Catalog(context.Context) ([]bundlepkg.CatalogEntry, error) {
	return nil, nil
}

func (s bundleServiceStub) PreviewActivation(
	context.Context,
	bundlepkg.ActivateRequest,
) (bundlepkg.ActivationPreview, error) {
	return bundlepkg.ActivationPreview{}, nil
}

func (s bundleServiceStub) Activate(context.Context, bundlepkg.ActivateRequest) (bundlepkg.ActivationPreview, error) {
	return bundlepkg.ActivationPreview{}, nil
}

func (s bundleServiceStub) ListActivations(context.Context) ([]bundlepkg.ActivationPreview, error) {
	return nil, nil
}

func (s bundleServiceStub) GetActivation(context.Context, string) (bundlepkg.ActivationPreview, error) {
	return bundlepkg.ActivationPreview{}, nil
}

func (s bundleServiceStub) UpdateActivation(
	context.Context,
	bundlepkg.UpdateActivationRequest,
) (bundlepkg.ActivationPreview, error) {
	return bundlepkg.ActivationPreview{}, nil
}

func (s bundleServiceStub) Deactivate(context.Context, string) error {
	return nil
}

func (s bundleServiceStub) NetworkSettings(ctx context.Context) (bundlepkg.NetworkSettings, error) {
	if s.networkSettingsFn != nil {
		return s.networkSettingsFn(ctx)
	}
	return bundlepkg.NetworkSettings{}, nil
}

func TestResolveUserHomeDir(t *testing.T) {
	t.Parallel()

	type result struct {
		got  string
		want string
		err  error
	}

	tests := []struct {
		name               string
		run                func(t *testing.T) result
		wantErrContains    string
		wantErrNotContains string
	}{
		{
			name: "ShouldPreferResolvedLookupValue",
			run: func(t *testing.T) result {
				t.Helper()

				want := filepath.Join(t.TempDir(), "user-home")
				homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), aghconfig.DirName))
				if err != nil {
					t.Fatalf("ResolveHomePathsFrom() error = %v", err)
				}

				got, resolveErr := resolveUserHomeDir(homePaths, func() (string, error) {
					return want, nil
				})
				return result{got: got, want: want, err: resolveErr}
			},
		},
		{
			name: "ShouldFallbackToCanonicalAGHHomeParentWhenLookupFails",
			run: func(t *testing.T) result {
				t.Helper()

				aghHome := filepath.Join(t.TempDir(), aghconfig.DirName)
				homePaths, err := aghconfig.ResolveHomePathsFrom(aghHome)
				if err != nil {
					t.Fatalf("ResolveHomePathsFrom() error = %v", err)
				}

				got, resolveErr := resolveUserHomeDir(homePaths, func() (string, error) {
					return "", errors.New("boom")
				})
				return result{got: got, want: filepath.Dir(homePaths.HomeDir), err: resolveErr}
			},
		},
		{
			name: "ShouldReturnRedactedErrorWhenResolvePathFailsWithoutFallback",
			run: func(t *testing.T) result {
				t.Helper()

				homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "agh-home"))
				if err != nil {
					t.Fatalf("ResolveHomePathsFrom() error = %v", err)
				}

				got, resolveErr := resolveUserHomeDirWithResolver(
					homePaths,
					func() (string, error) {
						return "secret-user-home", nil
					},
					func(string) (string, error) {
						return "", errors.New("boom")
					},
				)
				return result{got: got, want: "", err: resolveErr}
			},
			wantErrContains:    "resolve user home directory",
			wantErrNotContains: "secret-user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.run(t)
			if result.got != result.want {
				t.Fatalf("resolveUserHomeDir() = %q, want %q", result.got, result.want)
			}

			if tt.wantErrContains == "" {
				if result.err != nil {
					t.Fatalf("resolveUserHomeDir() error = %v, want nil", result.err)
				}
				return
			}

			if result.err == nil {
				t.Fatal("resolveUserHomeDir() error = nil, want non-nil")
			}
			if !strings.Contains(result.err.Error(), tt.wantErrContains) {
				t.Fatalf("resolveUserHomeDir() error = %q, want substring %q", result.err.Error(), tt.wantErrContains)
			}
			if tt.wantErrNotContains != "" && strings.Contains(result.err.Error(), tt.wantErrNotContains) {
				t.Fatalf(
					"resolveUserHomeDir() error = %q, should not include %q",
					result.err.Error(),
					tt.wantErrNotContains,
				)
			}
		})
	}
}

func TestBaseHandlersAccessorsAndSessionInfoHelpers(t *testing.T) {
	t.Parallel()

	var nilHandlers *BaseHandlers
	if got := nilHandlers.HTTPPortValue(); got != 0 {
		t.Fatalf("HTTPPortValue(nil) = %d, want 0", got)
	}
	if got := nilHandlers.StreamDoneChannel(); got != nil {
		t.Fatalf("StreamDoneChannel(nil) = %v, want nil", got)
	}

	done := make(chan struct{})
	calls := 0
	info := &session.Info{ID: "sess-1", WorkspaceID: "ws-alpha"}
	handlers := &BaseHandlers{
		Sessions: sessionManagerStub{
			status: func(_ context.Context, id string) (*session.Info, error) {
				calls++
				if id != "sess-1" {
					t.Fatalf("Status() id = %q, want sess-1", id)
				}
				return info, nil
			},
		},
	}
	handlers.SetHTTPPort(4510)
	handlers.SetStreamDone(done)

	if got := handlers.HTTPPortValue(); got != 4510 {
		t.Fatalf("HTTPPortValue() = %d, want 4510", got)
	}
	if got := handlers.StreamDoneChannel(); got != done {
		t.Fatalf("StreamDoneChannel() = %v, want %v", got, done)
	}
	if got := handlers.transportName(); got != "apicore" {
		t.Fatalf("transportName(default) = %q, want apicore", got)
	}

	handlers.TransportName = "uds-core"
	if got := handlers.transportName(); got != "uds-core" {
		t.Fatalf("transportName(custom) = %q, want uds-core", got)
	}

	eventInfo, err := handlers.sessionEventInfo(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("sessionEventInfo(disabled) error = %v", err)
	}
	if eventInfo != nil {
		t.Fatalf("sessionEventInfo(disabled) = %#v, want nil", eventInfo)
	}
	if calls != 0 {
		t.Fatalf("Status() called %d times with IncludeSessionWorkspaceInSSE disabled, want 0", calls)
	}

	handlers.IncludeSessionWorkspaceInSSE = true
	eventInfo, err = handlers.sessionEventInfo(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("sessionEventInfo(enabled) error = %v", err)
	}
	if eventInfo != info {
		t.Fatalf("sessionEventInfo(enabled) = %#v, want %#v", eventInfo, info)
	}

	streamInfo, err := handlers.streamSessionInfo(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("streamSessionInfo(enabled) error = %v", err)
	}
	if streamInfo != info {
		t.Fatalf("streamSessionInfo(enabled) = %#v, want %#v", streamInfo, info)
	}

	handlers.IncludeSessionWorkspaceInSSE = false
	streamInfo, err = handlers.streamSessionInfo(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("streamSessionInfo(disabled) error = %v", err)
	}
	if streamInfo != nil {
		t.Fatalf("streamSessionInfo(disabled) = %#v, want nil", streamInfo)
	}
	if calls != 3 {
		t.Fatalf("Status() calls = %d, want 3", calls)
	}

	handlers.Sessions = sessionManagerStub{
		status: func(context.Context, string) (*session.Info, error) {
			return nil, errors.New("boom")
		},
	}
	if _, err := handlers.streamSessionInfo(context.Background(), "sess-1"); err == nil {
		t.Fatal("streamSessionInfo(error) error = nil, want non-nil")
	}
}

func TestNetworkStatusPayloadWrapsBundleSettingsErrors(t *testing.T) {
	t.Parallel()

	settingsErr := errors.New("settings boom")
	handlers := &BaseHandlers{
		Config: aghconfig.Config{
			Network: aghconfig.NetworkConfig{Enabled: true},
		},
		Network: networkServiceStub{
			statusFn: func(context.Context) (*network.Status, error) {
				return &network.Status{}, nil
			},
		},
		Bundles: bundleServiceStub{
			networkSettingsFn: func(context.Context) (bundlepkg.NetworkSettings, error) {
				return bundlepkg.NetworkSettings{}, settingsErr
			},
		},
	}

	_, err := handlers.networkStatusPayload(context.Background())
	if !errors.Is(err, settingsErr) {
		t.Fatalf("networkStatusPayload() error = %v, want wrapped settings error", err)
	}
	if !strings.Contains(err.Error(), "api: load bundle network settings") {
		t.Fatalf("networkStatusPayload() error = %q, want bundle settings context", err.Error())
	}
}

func TestBundleHandlersRejectNilReceiverWithoutPanicking(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var nilHandlers *BaseHandlers
	testCases := []struct {
		name   string
		invoke func(*gin.Context)
	}{
		{
			name: "ShouldRejectCatalogRequests",
			invoke: func(ctx *gin.Context) {
				nilHandlers.ListBundleCatalog(ctx)
			},
		},
		{
			name: "ShouldRejectPreviewRequests",
			invoke: func(ctx *gin.Context) {
				nilHandlers.PreviewBundleActivation(ctx)
			},
		},
		{
			name: "ShouldRejectActivationListRequests",
			invoke: func(ctx *gin.Context) {
				nilHandlers.ListBundleActivations(ctx)
			},
		},
		{
			name: "ShouldRejectActivationGetRequests",
			invoke: func(ctx *gin.Context) {
				nilHandlers.GetBundleActivation(ctx)
			},
		},
		{
			name: "ShouldRejectActivationUpdateRequests",
			invoke: func(ctx *gin.Context) {
				nilHandlers.UpdateBundleActivation(ctx)
			},
		},
		{
			name: "ShouldRejectActivationDeleteRequests",
			invoke: func(ctx *gin.Context) {
				nilHandlers.DeleteBundleActivation(ctx)
			},
		},
		{
			name: "ShouldRejectNetworkSettingsRequests",
			invoke: func(ctx *gin.Context) {
				nilHandlers.BundleNetworkSettings(ctx)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				"/",
				http.NoBody,
			)
			ctx.Params = gin.Params{{Key: "id", Value: "act-1"}}

			tc.invoke(ctx)

			if got, want := recorder.Code, http.StatusServiceUnavailable; got != want {
				t.Fatalf("status = %d, want %d", got, want)
			}

			var payload contract.ErrorPayload
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal(error payload) error = %v", err)
			}
			if got, want := payload.Error, "api: bundle service is not configured"; got != want {
				t.Fatalf("error payload = %q, want %q", got, want)
			}
		})
	}
}
