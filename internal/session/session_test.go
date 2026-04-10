package session

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
)

func TestSessionStateTransitions(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	session := &Session{
		ID:        "sess-1",
		AgentName: "coder",
		Workspace: t.TempDir(),
		State:     StateStarting,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := session.activate(now.Add(time.Second)); err != nil {
		t.Fatalf("activate() error = %v", err)
	}
	if got := session.Info().State; got != StateActive {
		t.Fatalf("activate() state = %q, want %q", got, StateActive)
	}

	if err := session.beginStopping(now.Add(2 * time.Second)); err != nil {
		t.Fatalf("beginStopping() error = %v", err)
	}
	if got := session.Info().State; got != StateStopping {
		t.Fatalf("beginStopping() state = %q, want %q", got, StateStopping)
	}

	if err := session.markStopped(now.Add(3 * time.Second)); err != nil {
		t.Fatalf("markStopped() error = %v", err)
	}
	info := session.Info()
	if info.State != StateStopped {
		t.Fatalf("markStopped() state = %q, want %q", info.State, StateStopped)
	}
	if !info.UpdatedAt.Equal(now.Add(3 * time.Second)) {
		t.Fatalf("UpdatedAt = %s, want %s", info.UpdatedAt, now.Add(3*time.Second))
	}
}

func TestSessionInvalidTransitionRejected(t *testing.T) {
	t.Parallel()

	session := &Session{
		ID:        "sess-1",
		AgentName: "coder",
		Workspace: t.TempDir(),
		State:     StateStopped,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err := session.activate(time.Now().UTC())
	if err == nil {
		t.Fatal("activate() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("activate() error = %v, want ErrInvalidStateTransition", err)
	}
}

func TestSessionInfoCopiesCapabilities(t *testing.T) {
	t.Parallel()

	session := &Session{
		ID:           "sess-1",
		AgentName:    "coder",
		Workspace:    t.TempDir(),
		State:        StateActive,
		ACPSessionID: "acp-1",
		ACPCaps: acp.ACPCaps{
			SupportsLoadSession: true,
			SupportedModes:      []string{"chat"},
			SupportedModels:     []string{"gpt"},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	info := session.Info()
	info.ACPCaps.SupportedModes[0] = "mutated"
	info.ACPCaps.SupportedModels[0] = "mutated"

	latest := session.Info()
	if latest.ACPCaps.SupportedModes[0] != "chat" {
		t.Fatalf("SupportedModes mutated through Info() copy: %#v", latest.ACPCaps.SupportedModes)
	}
	if latest.ACPCaps.SupportedModels[0] != "gpt" {
		t.Fatalf("SupportedModels mutated through Info() copy: %#v", latest.ACPCaps.SupportedModels)
	}
}

func TestSessionInfoAndMetaIncludeStopFields(t *testing.T) {
	t.Run("Should include stop fields in info and metadata", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
		session := &Session{
			ID:          "sess-stop",
			Name:        "Stopped Session",
			AgentName:   "coder",
			WorkspaceID: "ws-stop",
			Workspace:   t.TempDir(),
			Type:        SessionTypeSystem,
			State:       StateStopped,
			stopCause:   CauseShutdown,
			stopReason:  store.StopShutdown,
			stopDetail:  "daemon shutdown",
			CreatedAt:   now,
			UpdatedAt:   now.Add(time.Minute),
		}

		info := session.Info()
		if info.StopReason != store.StopShutdown {
			t.Fatalf("Info().StopReason = %q, want %q", info.StopReason, store.StopShutdown)
		}
		if info.StopDetail != "daemon shutdown" {
			t.Fatalf("Info().StopDetail = %q, want %q", info.StopDetail, "daemon shutdown")
		}

		meta := session.Meta()
		if meta.StopReason == nil {
			t.Fatal("Meta().StopReason = nil, want non-nil")
		}
		if *meta.StopReason != store.StopShutdown {
			t.Fatalf("Meta().StopReason = %q, want %q", *meta.StopReason, store.StopShutdown)
		}
		if meta.StopDetail != "daemon shutdown" {
			t.Fatalf("Meta().StopDetail = %q, want %q", meta.StopDetail, "daemon shutdown")
		}
	})
}

func TestBeginPromptSetupReturnsErrSessionNotActive(t *testing.T) {
	t.Parallel()

	session := &Session{
		ID:        "sess-1",
		AgentName: "coder",
		Workspace: t.TempDir(),
		State:     StateStopped,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err := session.beginPromptSetup()
	if !errors.Is(err, ErrSessionNotActive) {
		t.Fatalf("beginPromptSetup() error = %v, want ErrSessionNotActive", err)
	}
	if !strings.Contains(err.Error(), "sess-1") {
		t.Fatalf("beginPromptSetup() error = %v, want session id context", err)
	}
}

func TestNormalizeSessionTypeDefaultsToUser(t *testing.T) {
	t.Parallel()

	if got := normalizeSessionType(""); got != SessionTypeUser {
		t.Fatalf("normalizeSessionType(\"\") = %q, want %q", got, SessionTypeUser)
	}
	if got := normalizeSessionType(" dream "); got != SessionTypeDream {
		t.Fatalf("normalizeSessionType(\" dream \") = %q, want %q", got, SessionTypeDream)
	}
	if got := normalizeSessionType("unknown"); got != SessionTypeUser {
		t.Fatalf("normalizeSessionType(\"unknown\") = %q, want %q", got, SessionTypeUser)
	}
}
