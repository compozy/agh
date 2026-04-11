package udsapi

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/pedronauck/agh/internal/api/contract"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
)

func TestListExtensionsHandlerReturnsExtensions(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlersWithExtensions(t, stubSessionManager{}, stubObserver{}, stubExtensionService{
		ListFn: func(context.Context) ([]contract.ExtensionPayload, error) {
			return []contract.ExtensionPayload{{
				Name:    "ext-a",
				Enabled: true,
				State:   "active",
			}}, nil
		},
	}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/extensions", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Extensions []contract.ExtensionPayload `json:"extensions"`
	}
	decodeJSONResponse(t, recorder, &response)
	if got, want := len(response.Extensions), 1; got != want {
		t.Fatalf("len(extensions) = %d, want %d", got, want)
	}
	if response.Extensions[0].Name != "ext-a" || !response.Extensions[0].Enabled {
		t.Fatalf("extensions[0] = %#v", response.Extensions[0])
	}
}

func TestInstallExtensionHandlerValidatesAndCreates(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlersWithExtensions(t, stubSessionManager{}, stubObserver{}, stubExtensionService{
		InstallFn: func(_ context.Context, req contract.InstallExtensionRequest) (contract.ExtensionPayload, error) {
			if req.Path != "/tmp/ext-a" || req.Checksum != "sha256:abc" {
				t.Fatalf("Install() req = %#v", req)
			}
			return contract.ExtensionPayload{Name: "ext-a", State: "installed"}, nil
		},
	}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/extensions", []byte(`{"path":" /tmp/ext-a ","checksum":" sha256:abc "}`))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response struct {
		Extension contract.ExtensionPayload `json:"extension"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Extension.Name != "ext-a" {
		t.Fatalf("extension = %#v", response.Extension)
	}

	missingPath := performRequest(t, engine, http.MethodPost, "/api/extensions", []byte(`{"checksum":"sha256:abc"}`))
	if missingPath.Code != http.StatusBadRequest {
		t.Fatalf("missing path status = %d, want %d; body=%s", missingPath.Code, http.StatusBadRequest, missingPath.Body.String())
	}

	missingChecksum := performRequest(t, engine, http.MethodPost, "/api/extensions", []byte(`{"path":"/tmp/ext-a"}`))
	if missingChecksum.Code != http.StatusBadRequest {
		t.Fatalf("missing checksum status = %d, want %d; body=%s", missingChecksum.Code, http.StatusBadRequest, missingChecksum.Body.String())
	}
}

func TestEnableDisableExtensionHandlers(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlersWithExtensions(t, stubSessionManager{}, stubObserver{}, stubExtensionService{
		EnableFn: func(_ context.Context, name string) (contract.ExtensionPayload, error) {
			if name != "ext-a" {
				t.Fatalf("Enable() name = %q, want ext-a", name)
			}
			return contract.ExtensionPayload{Name: name, Enabled: true, State: "active"}, nil
		},
		DisableFn: func(_ context.Context, name string) (contract.ExtensionPayload, error) {
			if name != "ext-a" {
				t.Fatalf("Disable() name = %q, want ext-a", name)
			}
			return contract.ExtensionPayload{Name: name, Enabled: false, State: "inactive"}, nil
		},
	}, homePaths)
	engine := newTestRouter(t, handlers)

	enableResp := performRequest(t, engine, http.MethodPost, "/api/extensions/%20ext-a%20/enable", nil)
	if enableResp.Code != http.StatusOK {
		t.Fatalf("enable status = %d, want %d; body=%s", enableResp.Code, http.StatusOK, enableResp.Body.String())
	}
	disableResp := performRequest(t, engine, http.MethodPost, "/api/extensions/%20ext-a%20/disable", nil)
	if disableResp.Code != http.StatusOK {
		t.Fatalf("disable status = %d, want %d; body=%s", disableResp.Code, http.StatusOK, disableResp.Body.String())
	}

	blankName := performRequest(t, engine, http.MethodPost, "/api/extensions/%20%20/enable", nil)
	if blankName.Code != http.StatusBadRequest {
		t.Fatalf("blank name status = %d, want %d; body=%s", blankName.Code, http.StatusBadRequest, blankName.Body.String())
	}
}

func TestExtensionStatusCodeMappingsAndApproveSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: http.StatusOK},
		{name: "not found", err: extensionpkg.ErrExtensionNotFound, want: http.StatusNotFound},
		{name: "checksum mismatch", err: extensionpkg.ErrExtensionChecksumMismatch, want: http.StatusBadRequest},
		{name: "invalid manifest", err: extensionpkg.ErrManifestInvalid, want: http.StatusBadRequest},
		{name: "incompatible manifest", err: extensionpkg.ErrManifestIncompatible, want: http.StatusBadRequest},
		{name: "manifest missing", err: extensionpkg.ErrManifestNotFound, want: http.StatusBadRequest},
		{name: "os missing", err: os.ErrNotExist, want: http.StatusBadRequest},
		{name: "default", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extensionStatusCode(tt.err); got != tt.want {
				t.Fatalf("extensionStatusCode(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}

	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))
	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", nil)
	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("approve status = %d, want %d; body=%s", recorder.Code, http.StatusNotImplemented, recorder.Body.String())
	}
}
