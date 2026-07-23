// Suite: Window-manager HTTP/UDS transport parity.
// Invariant: WM-TRANSPORT-001 exposes identical status and JSON bodies across
// both public transports for reads, mutations, clients, layouts, and errors.
// Owning layer: public route adapters. Canonical suite: this file.
// Boundary IN: transport registration and shared core-handler composition.
// Boundary OUT: semantic command behavior belongs to internal/windowmanager.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/api/core"
	"github.com/compozy/agh/internal/api/udsapi"
	"github.com/compozy/agh/internal/windowmanager"
	"github.com/gin-gonic/gin"
)

func TestWindowManagerHTTPUDSParity(t *testing.T) {
	t.Run("Should return identical public status and bodies through both transports", func(t *testing.T) {
		t.Parallel()
		httpManager := newWindowManagerParityService(t)
		udsManager := newWindowManagerParityService(t)
		httpHandlers := &Handlers{
			BaseHandlers: core.NewBaseHandlers(&core.BaseHandlerConfig{
				TransportName: "httpapi",
				WindowManager: httpManager,
			}),
			boundHost: "127.0.0.1",
		}
		udsHandlers := &udsapi.Handlers{
			BaseHandlers: core.NewBaseHandlers(&core.BaseHandlerConfig{
				TransportName: "udsapi",
				WindowManager: udsManager,
			}),
		}
		httpRouter := gin.New()
		udsRouter := gin.New()
		RegisterRoutes(httpRouter, httpHandlers)
		udsapi.RegisterRoutes(udsRouter, udsHandlers)
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Second)
			defer cancel()
			if err := httpHandlers.ShutdownWindowManagerStreams(ctx); err != nil {
				t.Errorf("HTTP ShutdownWindowManagerStreams() error = %v", err)
			}
			if err := udsHandlers.ShutdownWindowManagerStreams(ctx); err != nil {
				t.Errorf("UDS ShutdownWindowManagerStreams() error = %v", err)
			}
		})

		basePath := "/api/workspaces/workspace-a/window-manager"
		assertWindowManagerTransportParity(
			t,
			httpRouter,
			udsRouter,
			http.MethodGet,
			basePath,
			"",
			http.StatusOK,
		)

		command := windowManagerParityCreateDesktopBody()
		assertWindowManagerTransportParity(
			t, httpRouter, udsRouter, http.MethodPost, basePath+"/preview", command, http.StatusOK,
		)
		assertWindowManagerTransportParity(
			t, httpRouter, udsRouter, http.MethodPost, basePath+"/commands", command, http.StatusOK,
		)
		registered := assertWindowManagerTransportParity(
			t,
			httpRouter,
			udsRouter,
			http.MethodPost,
			basePath+"/clients",
			`{"workspace_id":"workspace-a","client_id":"client-a"}`,
			http.StatusCreated,
		)
		var registeredClient contract.WindowManagerClientView
		if err := json.Unmarshal(registered.Body.Bytes(), &registeredClient); err != nil {
			t.Fatalf("decode registered client: %v", err)
		}
		if registeredClient.PresentationRevision != 1 {
			t.Fatalf("registered presentation revision = %d, want 1", registeredClient.PresentationRevision)
		}
		assertWindowManagerTransportParity(
			t,
			httpRouter,
			udsRouter,
			http.MethodGet,
			basePath+"/clients",
			"",
			http.StatusOK,
		)

		layoutResponse := assertWindowManagerTransportParity(
			t, httpRouter, udsRouter, http.MethodGet, basePath+"/layout", "", http.StatusOK,
		)
		var document contract.WindowManagerLayoutDocument
		if err := json.Unmarshal(layoutResponse.Body.Bytes(), &document); err != nil {
			t.Fatalf("decode exported layout: %v", err)
		}
		validation := marshalWindowManagerParityJSON(t, contract.WindowManagerLayoutValidationRequest{
			WorkspaceID: "workspace-a", Document: document,
		})
		assertWindowManagerTransportParity(
			t, httpRouter, udsRouter, http.MethodPost, basePath+"/layout/validate", validation, http.StatusOK,
		)
		revision := contract.WindowManagerRevision(1)
		replacement := marshalWindowManagerParityJSON(t, contract.WindowManagerLayoutReplaceRequest{
			WorkspaceID: "workspace-a", ExpectedRevision: &revision,
			Actor: contract.WindowManagerActor{Kind: "test", ID: "actor"}, Origin: "parity-test",
			Document: document,
		})
		assertWindowManagerTransportParity(
			t, httpRouter, udsRouter, http.MethodPut, basePath+"/layout", replacement, http.StatusOK,
		)

		invalid := strings.Replace(command, `"desktop.create"`, `"desktop.unknown"`, 1)
		assertWindowManagerTransportParity(
			t, httpRouter, udsRouter, http.MethodPost, basePath+"/commands", invalid, http.StatusUnprocessableEntity,
		)
		assertWindowManagerTransportParity(
			t,
			httpRouter,
			udsRouter,
			http.MethodDelete,
			basePath+"/clients/client-a",
			"",
			http.StatusNoContent,
		)
		assertWindowManagerTransportParity(
			t,
			httpRouter,
			udsRouter,
			http.MethodGet,
			"/api/workspaces/missing/window-manager",
			"",
			http.StatusNotFound,
		)
	})
}

func newWindowManagerParityService(t *testing.T) *windowmanager.Manager {
	t.Helper()
	manager, err := windowmanager.NewService(
		windowmanager.NewMemoryRepository(),
		windowmanager.NewMemoryWorkspaceResolver("workspace-a"),
		nil,
		windowmanager.DefaultConfig(),
		windowmanager.WithClock(func() time.Time { return time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC) }),
		windowmanager.WithIDGenerator(func(kind string) (string, error) { return kind + "-parity", nil }),
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Errorf("Manager.Close() error = %v", err)
		}
	})
	return manager
}

func windowManagerParityCreateDesktopBody() string {
	return `{"workspace_id":"workspace-a","command_id":"desktop.create","expected_revision":0,` +
		`"actor":{"kind":"test","id":"actor"},"origin":"parity-test",` +
		`"payload":{"desktop_id":"desktop-second","name":"Second","purpose":"standard"}}`
}

func assertWindowManagerTransportParity(
	t *testing.T,
	httpRouter http.Handler,
	udsRouter http.Handler,
	method string,
	path string,
	body string,
	expectedStatus int,
) *httptest.ResponseRecorder {
	t.Helper()
	httpResponse := performWindowManagerParityRequest(t, httpRouter, method, path, body)
	udsResponse := performWindowManagerParityRequest(t, udsRouter, method, path, body)
	if httpResponse.Code != expectedStatus || udsResponse.Code != expectedStatus {
		t.Fatalf(
			"%s %s status: HTTP=%d UDS=%d want=%d; HTTP body=%s UDS body=%s",
			method,
			path,
			httpResponse.Code,
			udsResponse.Code,
			expectedStatus,
			httpResponse.Body.String(),
			udsResponse.Body.String(),
		)
	}
	if expectedStatus == http.StatusNoContent {
		if httpResponse.Body.Len() != 0 || udsResponse.Body.Len() != 0 {
			t.Fatalf("%s %s no-content bodies: HTTP=%q UDS=%q", method, path, httpResponse.Body, udsResponse.Body)
		}
		return httpResponse
	}
	var httpBody any
	if err := json.Unmarshal(httpResponse.Body.Bytes(), &httpBody); err != nil {
		t.Fatalf("decode HTTP body for %s %s: %v", method, path, err)
	}
	var udsBody any
	if err := json.Unmarshal(udsResponse.Body.Bytes(), &udsBody); err != nil {
		t.Fatalf("decode UDS body for %s %s: %v", method, path, err)
	}
	if !reflect.DeepEqual(httpBody, udsBody) {
		t.Fatalf("%s %s body mismatch: HTTP=%s UDS=%s", method, path, httpResponse.Body, udsResponse.Body)
	}
	return httpResponse
}

func performWindowManagerParityRequest(
	t *testing.T,
	router http.Handler,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func marshalWindowManagerParityJSON(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return string(payload)
}
