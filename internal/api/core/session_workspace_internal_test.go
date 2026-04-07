package core

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/pedronauck/agh/internal/session"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type workspaceGetterStub struct {
	get func(context.Context, string) (workspacepkg.Workspace, error)
}

func (s workspaceGetterStub) Get(ctx context.Context, ref string) (workspacepkg.Workspace, error) {
	return s.get(ctx, ref)
}

func TestSessionWorkspaceHelpers(t *testing.T) {
	t.Parallel()

	t.Run("validate create session request", func(t *testing.T) {
		t.Parallel()

		if err := validateCreateSessionRequest("core-test", "", ""); err == nil {
			t.Fatal("validateCreateSessionRequest() error = nil, want non-nil")
		}
		if err := validateCreateSessionRequest("core-test", "alpha", "/workspace"); err == nil {
			t.Fatal("validateCreateSessionRequest(mutually exclusive) error = nil, want non-nil")
		}
		if err := validateCreateSessionRequest("core-test", "", "relative"); err == nil {
			t.Fatal("validateCreateSessionRequest(relative path) error = nil, want non-nil")
		}
		if err := validateCreateSessionRequest("core-test", "alpha", ""); err != nil {
			t.Fatalf("validateCreateSessionRequest(workspace ref) error = %v", err)
		}
	})

	t.Run("lookup workspace id", func(t *testing.T) {
		t.Parallel()

		if _, err := lookupWorkspaceID(context.Background(), "core-test", nil, "alpha"); err == nil {
			t.Fatal("lookupWorkspaceID(nil resolver) error = nil, want non-nil")
		}

		id, err := lookupWorkspaceID(context.Background(), "core-test", workspaceGetterStub{
			get: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
				if ref != "alpha" {
					t.Fatalf("Get ref = %q, want alpha", ref)
				}
				return workspacepkg.Workspace{ID: "ws_alpha"}, nil
			},
		}, "alpha")
		if err != nil {
			t.Fatalf("lookupWorkspaceID() error = %v", err)
		}
		if id != "ws_alpha" {
			t.Fatalf("lookupWorkspaceID() = %q, want ws_alpha", id)
		}
	})

	t.Run("filter and trim helpers", func(t *testing.T) {
		t.Parallel()

		filtered := filterSessionInfosByWorkspaceIDInternal([]*session.SessionInfo{
			{ID: "sess-1", WorkspaceID: "ws_alpha"},
			nil,
			{ID: "sess-2", WorkspaceID: "ws_beta"},
		}, "ws_alpha")
		if len(filtered) != 1 || filtered[0].ID != "sess-1" {
			t.Fatalf("filterSessionInfosByWorkspaceIDInternal() = %#v", filtered)
		}

		trimmed := trimStringSliceInternal([]string{" one ", "", " two "})
		if len(trimmed) != 3 || trimmed[0] != "one" || trimmed[2] != "two" {
			t.Fatalf("trimStringSliceInternal() = %#v", trimmed)
		}
	})

	t.Run("path validators", func(t *testing.T) {
		t.Parallel()

		if err := validateAbsolutePathInternal("core-test", "path", ""); err == nil {
			t.Fatal("validateAbsolutePathInternal(empty) error = nil, want non-nil")
		}
		if err := validateAbsolutePathInternal("core-test", "path", "relative"); err == nil {
			t.Fatal("validateAbsolutePathInternal(relative) error = nil, want non-nil")
		}
		if err := validateAbsolutePathInternal("core-test", "path", "/workspace"); err != nil {
			t.Fatalf("validateAbsolutePathInternal(abs) error = %v", err)
		}

		if err := validateAbsolutePathsInternal("core-test", "paths", []string{"/workspace", "relative"}); err == nil {
			t.Fatal("validateAbsolutePathsInternal(relative entry) error = nil, want non-nil")
		}
		if err := validateAbsolutePathsInternal("core-test", "paths", []string{" /workspace ", ""}); err != nil {
			t.Fatalf("validateAbsolutePathsInternal(valid) error = %v", err)
		}
	})
}

func TestSessionWorkspaceStatusMappings(t *testing.T) {
	t.Parallel()

	if got := statusForWorkspaceError(workspacepkg.ErrWorkspaceNotFound); got != http.StatusNotFound {
		t.Fatalf("statusForWorkspaceError(not found) = %d, want %d", got, http.StatusNotFound)
	}
	if got := statusForWorkspaceError(workspacepkg.ErrWorkspaceRootMissing); got != http.StatusGone {
		t.Fatalf("statusForWorkspaceError(root missing) = %d, want %d", got, http.StatusGone)
	}
	if got := statusForWorkspaceError(workspacepkg.ErrWorkspaceHasSessions); got != http.StatusConflict {
		t.Fatalf("statusForWorkspaceError(has sessions) = %d, want %d", got, http.StatusConflict)
	}
	if got := statusForWorkspaceError(errors.New("boom")); got != http.StatusInternalServerError {
		t.Fatalf("statusForWorkspaceError(default) = %d, want %d", got, http.StatusInternalServerError)
	}

	if got := statusForSessionError(session.ErrSessionNotFound); got != http.StatusNotFound {
		t.Fatalf("statusForSessionError(session missing) = %d, want %d", got, http.StatusNotFound)
	}
	if got := statusForSessionError(os.ErrNotExist); got != http.StatusNotFound {
		t.Fatalf("statusForSessionError(os not exist) = %d, want %d", got, http.StatusNotFound)
	}
	if got := statusForSessionError(workspacepkg.ErrWorkspaceRootMissing); got != http.StatusGone {
		t.Fatalf("statusForSessionError(root missing) = %d, want %d", got, http.StatusGone)
	}
	if got := statusForSessionError(session.ErrSessionNotActive); got != http.StatusBadRequest {
		t.Fatalf("statusForSessionError(not active) = %d, want %d", got, http.StatusBadRequest)
	}
	if got := statusForSessionError(session.ErrPendingPermissionConflict); got != http.StatusConflict {
		t.Fatalf("statusForSessionError(conflict) = %d, want %d", got, http.StatusConflict)
	}
	if got := statusForSessionError(errors.New("boom")); got != http.StatusInternalServerError {
		t.Fatalf("statusForSessionError(default) = %d, want %d", got, http.StatusInternalServerError)
	}
}
