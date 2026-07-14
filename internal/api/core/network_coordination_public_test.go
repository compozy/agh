package core_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/api/testutil"
	"github.com/compozy/agh/internal/network/participation"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func coordinationWorkspaceService() testutil.StubWorkspaceService {
	return testutil.StubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			return workspacepkg.Workspace{ID: ref, Name: ref, RootDir: "/workspace"}, nil
		},
	}
}

type stubCoordinationSettings struct {
	setting workspacepkg.CoordinationSetting
	setErr  error
	getErr  error
	calls   int
}

func (s *stubCoordinationSettings) Get(
	_ context.Context,
	workspaceID string,
) (workspacepkg.CoordinationSetting, error) {
	if s.getErr != nil {
		return workspacepkg.CoordinationSetting{}, s.getErr
	}
	out := s.setting
	if out.WorkspaceID == "" {
		out.WorkspaceID = workspaceID
	}
	return out, nil
}

func (s *stubCoordinationSettings) Set(
	_ context.Context,
	workspaceID string,
	enabled bool,
	actor string,
) (workspacepkg.CoordinationSetting, error) {
	s.calls++
	if s.setErr != nil {
		return workspacepkg.CoordinationSetting{}, s.setErr
	}
	s.setting = workspacepkg.CoordinationSetting{
		WorkspaceID: workspaceID,
		Enabled:     enabled,
		Revision:    s.setting.Revision + 1,
		UpdatedAt:   time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC),
		UpdatedBy:   actor,
	}
	return s.setting, nil
}

type stubCoordinationInvitations struct {
	row      workspacepkg.CoordinationInvitation
	getErr   error
	dismiss  error
	resetErr error
}

func (s *stubCoordinationInvitations) GetInvitation(
	_ context.Context,
	scopeKind string,
	scopeID string,
) (workspacepkg.CoordinationInvitation, error) {
	if s.getErr != nil {
		return workspacepkg.CoordinationInvitation{}, s.getErr
	}
	if s.row.ScopeKind == "" {
		return workspacepkg.CoordinationInvitation{
			ScopeKind: scopeKind,
			ScopeID:   scopeID,
			Dismissed: false,
		}, nil
	}
	return s.row, nil
}

func (s *stubCoordinationInvitations) DismissInvitation(
	_ context.Context,
	scopeKind string,
	scopeID string,
	actor string,
) (workspacepkg.CoordinationInvitation, error) {
	if s.dismiss != nil {
		return workspacepkg.CoordinationInvitation{}, s.dismiss
	}
	s.row = workspacepkg.CoordinationInvitation{
		ScopeKind:   scopeKind,
		ScopeID:     scopeID,
		Dismissed:   true,
		DismissedAt: time.Date(2026, 7, 14, 12, 1, 0, 0, time.UTC),
		DismissedBy: actor,
	}
	return s.row, nil
}

func (s *stubCoordinationInvitations) ResetInvitation(
	_ context.Context,
	_ string,
	_ string,
) error {
	if s.resetErr != nil {
		return s.resetErr
	}
	s.row = workspacepkg.CoordinationInvitation{}
	return nil
}

func TestNetworkCoordinationHandlers(t *testing.T) {
	t.Parallel()

	t.Run("Should return coordination provenance and invitation state on GET", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			coordinationWorkspaceService(),
			nil,
			nil,
		)
		settings := &stubCoordinationSettings{
			setting: workspacepkg.CoordinationSetting{
				WorkspaceID: "ws-alpha",
				Enabled:     true,
				Revision:    3,
				UpdatedAt:   time.Date(2026, 7, 14, 11, 0, 0, 0, time.UTC),
				UpdatedBy:   "operator",
			},
		}
		invitations := &stubCoordinationInvitations{
			row: workspacepkg.CoordinationInvitation{
				ScopeKind:   workspacepkg.InvitationScopeWorkspace,
				ScopeID:     "ws-alpha",
				Dismissed:   true,
				DismissedAt: time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC),
				DismissedBy: "operator",
			},
		}
		fixture.Handlers.CoordinationSettings = settings
		fixture.Handlers.CoordinationInvitations = invitations

		resp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/workspaces/ws-alpha/network-coordination",
			nil,
		)
		if resp.Code != http.StatusOK {
			t.Fatalf("GET coordination status = %d body=%s", resp.Code, resp.Body.String())
		}
		var body contract.NetworkCoordinationResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode coordination response: %v", err)
		}
		if !body.Coordination.Enabled || body.Coordination.Revision != 3 {
			t.Fatalf("coordination = %#v, want enabled revision=3", body.Coordination)
		}
		if body.Coordination.Invitation == nil || !body.Coordination.Invitation.Dismissed {
			t.Fatalf("invitation = %#v, want dismissed", body.Coordination.Invitation)
		}
	})

	t.Run("Should PUT enable and return updated revision", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			coordinationWorkspaceService(),
			nil,
			nil,
		)
		settings := &stubCoordinationSettings{}
		fixture.Handlers.CoordinationSettings = settings
		fixture.Handlers.CoordinationInvitations = &stubCoordinationInvitations{}

		resp := performRequest(
			t,
			fixture.Engine,
			http.MethodPut,
			"/workspaces/ws-alpha/network-coordination",
			[]byte(`{"enabled":true}`),
		)
		if resp.Code != http.StatusOK {
			t.Fatalf("PUT coordination status = %d body=%s", resp.Code, resp.Body.String())
		}
		var body contract.NetworkCoordinationResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode coordination response: %v", err)
		}
		if !body.Coordination.Enabled || body.Coordination.Revision != 1 {
			t.Fatalf("coordination = %#v, want enabled revision=1", body.Coordination)
		}
		if settings.calls != 1 {
			t.Fatalf("Set calls = %d, want 1", settings.calls)
		}
	})

	t.Run("Should reject PUT while network participation is unavailable with body", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			coordinationWorkspaceService(),
			nil,
			nil,
		)
		fixture.Handlers.CoordinationSettings = &stubCoordinationSettings{
			setErr: participation.ErrUnavailable,
		}
		fixture.Handlers.CoordinationInvitations = &stubCoordinationInvitations{}

		resp := performRequest(
			t,
			fixture.Engine,
			http.MethodPut,
			"/workspaces/ws-alpha/network-coordination",
			[]byte(`{"enabled":true}`),
		)
		if resp.Code != http.StatusConflict {
			t.Fatalf("PUT unavailable status = %d body=%s", resp.Code, resp.Body.String())
		}
		if !strings.Contains(resp.Body.String(), "unavailable") &&
			!strings.Contains(resp.Body.String(), participation.ErrUnavailable.Error()) {
			t.Fatalf("body = %s, want unavailable diagnostic", resp.Body.String())
		}
	})

	t.Run("Should reject unknown fields on coordination PUT", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			coordinationWorkspaceService(),
			nil,
			nil,
		)
		fixture.Handlers.CoordinationSettings = &stubCoordinationSettings{}
		fixture.Handlers.CoordinationInvitations = &stubCoordinationInvitations{}

		resp := performRequest(
			t,
			fixture.Engine,
			http.MethodPut,
			"/workspaces/ws-alpha/network-coordination",
			[]byte(`{"enabled":true,"network_channel":"builders"}`),
		)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("PUT unknown field status = %d body=%s", resp.Code, resp.Body.String())
		}
		body := resp.Body.String()
		if !strings.Contains(body, "unknown_field") || !strings.Contains(body, "network_channel") {
			t.Fatalf("body = %s, want unknown_field with network_channel named", body)
		}
	})

	t.Run("Should dismiss invitation and return daemon-backed state", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			coordinationWorkspaceService(),
			nil,
			nil,
		)
		invitations := &stubCoordinationInvitations{}
		fixture.Handlers.CoordinationSettings = &stubCoordinationSettings{}
		fixture.Handlers.CoordinationInvitations = invitations

		resp := performRequest(
			t,
			fixture.Engine,
			http.MethodPut,
			"/workspaces/ws-alpha/network-coordination/invitation",
			[]byte(`{"scope":"workspace","dismissed":true}`),
		)
		if resp.Code != http.StatusOK {
			t.Fatalf("PUT invitation status = %d body=%s", resp.Code, resp.Body.String())
		}
		var body contract.NetworkCoordinationResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode invitation response: %v", err)
		}
		if body.Coordination.Invitation == nil || !body.Coordination.Invitation.Dismissed {
			t.Fatalf("invitation = %#v, want dismissed", body.Coordination.Invitation)
		}
		if !invitations.row.Dismissed {
			t.Fatal("stub invitation row was not dismissed")
		}
	})

	t.Run("Should map workspace not found on coordination GET", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)
		fixture.Handlers.CoordinationSettings = &stubCoordinationSettings{
			getErr: workspacepkg.ErrWorkspaceNotFound,
		}

		resp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/workspaces/missing/network-coordination",
			nil,
		)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("GET missing workspace status = %d body=%s", resp.Code, resp.Body.String())
		}
	})
}
