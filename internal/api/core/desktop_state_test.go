package core

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/clientstate"
	"github.com/gin-gonic/gin"
)

func TestDesktopStateHandlers(t *testing.T) {
	t.Parallel()

	t.Run("Should return one canonical desktop-state entry (UT-022)", func(t *testing.T) {
		t.Parallel()
		router, engine := newDesktopStateHandlerFixture(t, clientstate.DefaultLimits())
		seedDesktopState(t, engine, "desktop", `{"v":1,"focusedId":"tasks"}`)

		response := performDesktopStateRequest(t, router, http.MethodGet, desktopStateItemURL("desktop"), nil)
		assertDesktopStateStatus(t, response, http.StatusOK)
		var entry contract.DesktopStateEntry
		decodeDesktopStateResponse(t, response, &entry)
		if entry.Key != "desktop" || entry.Rev != 1 || entry.Seq != 1 || entry.Deleted {
			t.Fatalf("entry = %#v, want live desktop rev=1 seq=1", entry)
		}
		if got := entry.Value["focusedId"]; got != "tasks" {
			t.Fatalf("value.focusedId = %#v, want tasks", got)
		}
		if entry.UpdatedAt.IsZero() {
			t.Fatal("updated_at is zero")
		}
	})

	t.Run("Should return the stable not-found body for an absent key (UT-023)", func(t *testing.T) {
		t.Parallel()
		router, _ := newDesktopStateHandlerFixture(t, clientstate.DefaultLimits())

		response := performDesktopStateRequest(t, router, http.MethodGet, desktopStateItemURL("missing"), nil)
		assertDesktopStateError(t, response, http.StatusNotFound, contract.DesktopStateErrorNotFound, "missing")
	})

	t.Run("Should increment the revision when replacing a value (UT-024)", func(t *testing.T) {
		t.Parallel()
		router, _ := newDesktopStateHandlerFixture(t, clientstate.DefaultLimits())

		first := performDesktopStateRequest(
			t, router, http.MethodPut, desktopStateItemURL("desktop"),
			map[string]any{"value": map[string]any{"v": 1}},
		)
		assertDesktopStateStatus(t, first, http.StatusOK)
		var firstEntry contract.DesktopStateEntry
		decodeDesktopStateResponse(t, first, &firstEntry)

		second := performDesktopStateRequest(
			t, router, http.MethodPut, desktopStateItemURL("desktop"),
			map[string]any{"value": map[string]any{"v": 2}, "if_rev": firstEntry.Rev},
		)
		assertDesktopStateStatus(t, second, http.StatusOK)
		var secondEntry contract.DesktopStateEntry
		decodeDesktopStateResponse(t, second, &secondEntry)
		if secondEntry.Rev != firstEntry.Rev+1 || secondEntry.Seq != firstEntry.Seq+1 {
			t.Fatalf("second entry = %#v, want rev and seq incremented from %#v", secondEntry, firstEntry)
		}
	})

	t.Run("Should reject a stale put revision (UT-025)", func(t *testing.T) {
		t.Parallel()
		router, engine := newDesktopStateHandlerFixture(t, clientstate.DefaultLimits())
		seedDesktopState(t, engine, "desktop", `{"v":1}`)

		response := performDesktopStateRequest(
			t, router, http.MethodPut, desktopStateItemURL("desktop"),
			map[string]any{"value": map[string]any{"v": 2}, "if_rev": 7},
		)
		assertDesktopStateError(
			t, response, http.StatusConflict, contract.DesktopStateErrorRevConflict, "desktop",
		)
	})

	t.Run("Should reject an oversized put value (UT-026)", func(t *testing.T) {
		t.Parallel()
		router, _ := newDesktopStateHandlerFixture(t, clientstate.Limits{
			MaxValueBytes: 16, MaxKeysPerWorkspace: 32,
		})

		response := performDesktopStateRequest(
			t, router, http.MethodPut, desktopStateItemURL("desktop"),
			map[string]any{"value": map[string]any{"payload": "this value is too large"}},
		)
		assertDesktopStateError(
			t, response, http.StatusRequestEntityTooLarge, contract.DesktopStateErrorValueTooLarge, "desktop",
		)
	})

	t.Run("Should map quota and key validation failures (UT-027)", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name       string
			key        string
			limits     clientstate.Limits
			seed       bool
			wantStatus int
			wantCode   contract.DesktopStateErrorCode
		}{
			{
				name: "quota", key: "second", seed: true,
				limits:     clientstate.Limits{MaxValueBytes: 1024, MaxKeysPerWorkspace: 1},
				wantStatus: http.StatusUnprocessableEntity,
				wantCode:   contract.DesktopStateErrorKeyQuota,
			},
			{
				name: "invalid key", key: "bad$key",
				limits:     clientstate.DefaultLimits(),
				wantStatus: http.StatusUnprocessableEntity,
				wantCode:   contract.DesktopStateErrorInvalidKey,
			},
		}
		for _, testCase := range cases {
			t.Run("Should reject "+testCase.name, func(t *testing.T) {
				t.Parallel()
				router, engine := newDesktopStateHandlerFixture(t, testCase.limits)
				if testCase.seed {
					seedDesktopState(t, engine, "first", `{"v":1}`)
				}
				response := performDesktopStateRequest(
					t, router, http.MethodPut, desktopStateItemURL(testCase.key),
					map[string]any{"value": map[string]any{"v": 1}},
				)
				assertDesktopStateError(t, response, testCase.wantStatus, testCase.wantCode, testCase.key)
			})
		}
	})

	t.Run("Should delete a live value and return an empty body (UT-028)", func(t *testing.T) {
		t.Parallel()
		router, engine := newDesktopStateHandlerFixture(t, clientstate.DefaultLimits())
		seedDesktopState(t, engine, "desktop", `{"v":1}`)

		response := performDesktopStateRequest(t, router, http.MethodDelete, desktopStateItemURL("desktop"), nil)
		assertDesktopStateStatus(t, response, http.StatusNoContent)
		if response.Body.Len() != 0 {
			t.Fatalf("DELETE body = %q, want empty", response.Body.String())
		}
	})

	t.Run("Should reject delete for an absent value (UT-028)", func(t *testing.T) {
		t.Parallel()
		router, _ := newDesktopStateHandlerFixture(t, clientstate.DefaultLimits())

		response := performDesktopStateRequest(t, router, http.MethodDelete, desktopStateItemURL("missing"), nil)
		assertDesktopStateError(t, response, http.StatusNotFound, contract.DesktopStateErrorNotFound, "missing")
	})

	t.Run("Should reject a stale delete revision (UT-028)", func(t *testing.T) {
		t.Parallel()
		router, engine := newDesktopStateHandlerFixture(t, clientstate.DefaultLimits())
		seedDesktopState(t, engine, "desktop", `{"v":1}`)

		response := performDesktopStateRequest(
			t, router, http.MethodDelete, desktopStateItemURL("desktop")+"?if_rev=2", nil,
		)
		assertDesktopStateError(
			t, response, http.StatusConflict, contract.DesktopStateErrorRevConflict, "desktop",
		)
	})

	t.Run("Should list a key-sorted snapshot and its sequence fence (UT-029)", func(t *testing.T) {
		t.Parallel()
		router, engine := newDesktopStateHandlerFixture(t, clientstate.DefaultLimits())
		seedDesktopState(t, engine, "z:last", `{"v":1}`)
		seedDesktopState(t, engine, "a:first", `{"v":1}`)

		response := performDesktopStateRequest(t, router, http.MethodGet, desktopStateCollectionURL(), nil)
		assertDesktopStateStatus(t, response, http.StatusOK)
		var payload contract.DesktopStateListResponse
		decodeDesktopStateResponse(t, response, &payload)
		if payload.AsOfSeq != 2 || len(payload.Entries) != 2 {
			t.Fatalf("list payload = %#v, want fence 2 with two entries", payload)
		}
		if payload.Entries[0].Key != "a:first" || payload.Entries[1].Key != "z:last" {
			t.Fatalf("entry order = [%s %s], want [a:first z:last]", payload.Entries[0].Key, payload.Entries[1].Key)
		}
	})

	t.Run("Should apply a batch atomically and reject invalid values", func(t *testing.T) {
		t.Parallel()
		router, engine := newDesktopStateHandlerFixture(t, clientstate.DefaultLimits())
		response := performDesktopStateRequest(
			t, router, http.MethodPost, desktopStateCollectionURL()+"/apply",
			map[string]any{"ops": []any{
				map[string]any{"kind": "put", "key": "desktop", "value": map[string]any{"v": 1}},
				map[string]any{"kind": "put", "key": "win:tasks", "value": map[string]any{"v": 1}},
			}},
		)
		assertDesktopStateStatus(t, response, http.StatusOK)
		var payload contract.DesktopStateApplyResponse
		decodeDesktopStateResponse(t, response, &payload)
		if len(payload.Results) != 2 {
			t.Fatalf("results = %#v, want two entries", payload.Results)
		}

		invalid := performDesktopStateRequest(
			t, router, http.MethodPost, desktopStateCollectionURL()+"/apply",
			map[string]any{"ops": []any{
				map[string]any{"kind": "put", "key": "third", "value": nil},
			}},
		)
		assertDesktopStateError(
			t, invalid, http.StatusUnprocessableEntity, contract.DesktopStateErrorInvalidValue, "third",
		)
		entries, err := engine.List(context.Background(), "w1", desktopStateDomain)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("stored entries = %#v, want invalid batch to leave two entries", entries)
		}
	})

	t.Run("Should reject every route for an unknown workspace (UT-076)", func(t *testing.T) {
		t.Parallel()
		router, _ := newDesktopStateHandlerFixture(t, clientstate.DefaultLimits())

		response := performDesktopStateRequest(
			t, router, http.MethodPut, "/api/workspaces/missing/desktop-state/desktop",
			map[string]any{"value": map[string]any{"v": 1}},
		)
		assertDesktopStateError(
			t, response, http.StatusNotFound, contract.DesktopStateErrorWorkspace, "desktop",
		)
	})
}

type desktopStateHandlerResolver struct{}

func (desktopStateHandlerResolver) ResolveWorkspace(
	_ context.Context,
	workspace clientstate.WorkspaceID,
) (clientstate.WorkspaceGeneration, error) {
	if workspace != "w1" && workspace != "w2" {
		return "", clientstate.ErrWorkspaceNotFound
	}
	return clientstate.WorkspaceGeneration(string(workspace) + "-generation"), nil
}

func (desktopStateHandlerResolver) ResolveWorkspaceForPurge(
	_ context.Context,
	_ clientstate.WorkspaceID,
) (clientstate.WorkspaceGeneration, error) {
	return "", clientstate.ErrWorkspaceNotFound
}

func newDesktopStateHandlerFixture(
	t *testing.T,
	limits clientstate.Limits,
) (*gin.Engine, *clientstate.Engine) {
	t.Helper()
	engine, err := clientstate.Open(
		filepath.Join(t.TempDir(), clientstate.DatabaseName),
		desktopStateHandlerResolver{},
		limits,
		clientstate.WithClock(func() time.Time {
			return time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
		}),
	)
	if err != nil {
		t.Fatalf("clientstate.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := engine.Close(); err != nil {
			t.Errorf("Engine.Close() error = %v", err)
		}
	})
	handlers := NewBaseHandlers(&BaseHandlerConfig{DesktopState: engine})
	router := gin.New()
	collection := "/api/workspaces/:workspace_id/desktop-state"
	router.GET(collection, handlers.ListDesktopState)
	router.POST(collection+"/apply", handlers.ApplyDesktopState)
	router.GET(collection+"/:key", handlers.GetDesktopState)
	router.PUT(collection+"/:key", handlers.PutDesktopState)
	router.DELETE(collection+"/:key", handlers.DeleteDesktopState)
	return router, engine
}

func seedDesktopState(t *testing.T, engine *clientstate.Engine, key string, value string) clientstate.Entry {
	t.Helper()
	entries, err := engine.Apply(context.Background(), "w1", desktopStateDomain, []clientstate.Op{{
		Kind: clientstate.OpPut, Key: key, Value: []byte(value),
	}}, clientstate.ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply(%s) error = %v", key, err)
	}
	if len(entries) != 1 {
		t.Fatalf("Apply(%s) returned %d entries, want 1", key, len(entries))
	}
	return entries[0]
}

func performDesktopStateRequest(
	t *testing.T,
	router http.Handler,
	method string,
	path string,
	body any,
) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal(request) error = %v", err)
		}
		reader = bytes.NewReader(encoded)
	}
	request := httptest.NewRequestWithContext(t.Context(), method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func desktopStateCollectionURL() string {
	return "/api/workspaces/w1/desktop-state"
}

func desktopStateItemURL(key string) string {
	return desktopStateCollectionURL() + "/" + key
}

func assertDesktopStateStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, want, response.Body.String())
	}
}

func assertDesktopStateError(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantStatus int,
	wantCode contract.DesktopStateErrorCode,
	wantKey string,
) {
	t.Helper()
	assertDesktopStateStatus(t, response, wantStatus)
	var payload contract.DesktopStateErrorPayload
	decodeDesktopStateResponse(t, response, &payload)
	if payload.Code != wantCode || payload.Error != string(wantCode) || payload.Key != wantKey {
		t.Fatalf("error payload = %#v, want code/error=%q key=%q", payload, wantCode, wantKey)
	}
}

func decodeDesktopStateResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, response.Body.String())
	}
}
