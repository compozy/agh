package udsapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/api/contract"
	mcppkg "github.com/compozy/agh/internal/mcp"
	"github.com/gin-gonic/gin"
)

func TestHostedMCPStreamErrorData(t *testing.T) {
	t.Parallel()

	t.Run("Should emit stable stream error without raw backend details", func(t *testing.T) {
		t.Parallel()

		payload := hostedMCPStreamErrorData(
			fmt.Errorf("bind failed for agh_claim_secret: %w", mcppkg.ErrHostedBindNotFound),
		)
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("json.Marshal(hosted MCP stream error) error = %v", err)
		}
		if strings.Contains(string(encoded), "agh_claim_secret") || strings.Contains(string(encoded), "bind failed") {
			t.Fatalf("hosted MCP stream error payload leaked backend detail: %s", encoded)
		}
		if payload.Error != "hosted_mcp_projection_failed" ||
			payload.Status != http.StatusForbidden ||
			payload.Message != http.StatusText(http.StatusForbidden) {
			t.Fatalf("hosted MCP stream error payload = %#v, want stable forbidden error", payload)
		}
	})
}

func TestHostedMCPJSONRouteErrors(t *testing.T) {
	t.Parallel()

	t.Run("Should expose stable caller-correctable errors without backend details", func(t *testing.T) {
		t.Parallel()

		router, peer := newHostedMCPRouteTestHarness(t)
		for _, tt := range []struct {
			name       string
			method     string
			path       string
			body       string
			statusCode int
			wantError  string
		}{
			{
				name:       "Should report missing session id on bind",
				method:     http.MethodPost,
				path:       "/api/internal/hosted-mcp/bind",
				body:       `{}`,
				statusCode: http.StatusBadRequest,
				wantError:  "hosted_mcp_session_required",
			},
			{
				name:       "Should report missing bind id on projection",
				method:     http.MethodGet,
				path:       "/api/internal/hosted-mcp/projection?bind_id=",
				statusCode: http.StatusBadRequest,
				wantError:  "hosted_mcp_bind_required",
			},
			{
				name:       "Should report missing bind id on tool call",
				method:     http.MethodPost,
				path:       "/api/internal/hosted-mcp/tools/call",
				body:       `{"tool_name":"shell.exec"}`,
				statusCode: http.StatusBadRequest,
				wantError:  "hosted_mcp_bind_required",
			},
			{
				name:       "Should report missing bind id on release",
				method:     http.MethodPost,
				path:       "/api/internal/hosted-mcp/release",
				body:       `{}`,
				statusCode: http.StatusBadRequest,
				wantError:  "hosted_mcp_bind_required",
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				recorder := httptest.NewRecorder()
				request := httptest.NewRequestWithContext(
					context.Background(),
					tt.method,
					tt.path,
					bytes.NewBufferString(tt.body),
				)
				request = request.WithContext(mcppkg.ContextWithPeerInfo(request.Context(), peer, nil))
				router.ServeHTTP(recorder, request)

				if recorder.Code != tt.statusCode {
					t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.statusCode, recorder.Body.String())
				}
				var payload contract.ErrorPayload
				if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
					t.Fatalf("Unmarshal(error payload) error = %v; body=%s", err, recorder.Body.String())
				}
				if !strings.Contains(payload.Error, tt.wantError) {
					t.Fatalf("error payload = %#v, want %q", payload, tt.wantError)
				}
				if strings.Contains(payload.Error, "hosted-mcp backend error") {
					t.Fatalf("error payload used opaque backend fallback for 4xx: %#v", payload)
				}
			})
		}
	})

	t.Run("Should keep unexpected backend errors redacted", func(t *testing.T) {
		t.Parallel()

		if got := hostedMCPSafeError().Error(); got != errHostedMCPBackend.Error() {
			t.Fatalf("hostedMCPSafeError().Error() = %q, want generic backend error", got)
		}
		if strings.Contains(hostedMCPSafeError().Error(), errors.New("database leaked agh_claim_secret").Error()) {
			t.Fatal("hostedMCPSafeError leaked backend details")
		}
	})
}

func newHostedMCPRouteTestHarness(t *testing.T) (*gin.Engine, mcppkg.PeerInfo) {
	t.Helper()

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}
	service, err := mcppkg.NewHostedService(mcppkg.HostedConfig{
		Enabled:        true,
		ExpectedBinary: executable,
	})
	if err != nil {
		t.Fatalf("NewHostedService() error = %v", err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerHostedMCPRoutes(router.Group("/api"), &Handlers{HostedMCP: service})
	peer := mcppkg.PeerInfo{
		PID:            os.Getpid(),
		UID:            os.Getuid(),
		GID:            os.Getgid(),
		ExecutablePath: executable,
		Supported:      true,
	}
	return router, peer
}
