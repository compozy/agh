package observe

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/testutil"

	"github.com/pedronauck/agh/internal/store"
)

func TestReconciliationIndexesSessionDirNotInDB(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sessionDir := filepath.Join(h.home.SessionsDir, "sess-new")
	metaPath := store.SessionMetaFile(sessionDir)
	now := h.now.Add(30 * time.Minute)
	stopReason := store.StopUserCanceled

	if err := store.WriteSessionMeta(metaPath, store.SessionMeta{
		ID:          "sess-new",
		Name:        "New",
		AgentName:   "coder",
		Provider:    "claude",
		WorkspaceID: h.workspaceID,
		State:       "stopped",
		StopReason:  &stopReason,
		StopDetail:  "requested by API",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	result, err := h.observer.Reconcile(testutil.Context(t))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	sort.Strings(result.Indexed)
	if got, want := result.Indexed, []string{"sess-new"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("Indexed = %#v, want %#v", got, want)
	}

	sessions, err := h.observer.registry.ListSessions(testutil.Context(t), store.SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].State != "stopped" {
		t.Fatalf("sessions[0].State = %q, want stopped", sessions[0].State)
	}
	if sessions[0].StopReason != store.StopUserCanceled {
		t.Fatalf("sessions[0].StopReason = %q, want %q", sessions[0].StopReason, store.StopUserCanceled)
	}
	if sessions[0].StopDetail != "requested by API" {
		t.Fatalf("sessions[0].StopDetail = %q, want %q", sessions[0].StopDetail, "requested by API")
	}
	if sessions[0].Provider != "claude" {
		t.Fatalf("sessions[0].Provider = %q, want claude", sessions[0].Provider)
	}

	meta, err := store.ReadSessionMeta(metaPath)
	if err != nil {
		t.Fatalf("ReadSessionMeta() error = %v", err)
	}
	if meta.State != "stopped" {
		t.Fatalf("meta.State = %q, want stopped", meta.State)
	}
}

func TestReconciliationMarksMissingDirectoryAsOrphaned(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	now := h.now
	if err := h.observer.registry.RegisterSession(testutil.Context(t), store.SessionInfo{
		ID:          "sess-orphan",
		Name:        "Orphan",
		AgentName:   "coder",
		Provider:    "claude",
		WorkspaceID: h.workspaceID,
		State:       "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	result, err := h.observer.Reconcile(testutil.Context(t))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	sort.Strings(result.Orphaned)
	if got, want := result.Orphaned, []string{"sess-orphan"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("Orphaned = %#v, want %#v", got, want)
	}

	sessions, err := h.observer.registry.ListSessions(testutil.Context(t), store.SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].State != "orphaned" {
		t.Fatalf("sessions[0].State = %q, want orphaned", sessions[0].State)
	}
}

func TestReconciliationRepairsLegacyProviderBeforeIndexing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sessionDir := filepath.Join(h.home.SessionsDir, "sess-repair")
	metaPath := store.SessionMetaFile(sessionDir)
	now := h.now.Add(40 * time.Minute)

	if err := store.WriteSessionMeta(metaPath, store.SessionMeta{
		ID:          "sess-repair",
		Name:        "Repair",
		AgentName:   "coder",
		WorkspaceID: h.workspaceID,
		State:       "stopped",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	result, err := h.observer.Reconcile(testutil.Context(t))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if got, want := result.Indexed, []string{"sess-repair"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("Indexed = %#v, want %#v", got, want)
	}

	sessions, err := h.observer.registry.ListSessions(testutil.Context(t), store.SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if got, want := sessions[0].Provider, "claude"; got != want {
		t.Fatalf("sessions[0].Provider = %q, want %q", got, want)
	}

	meta, err := store.ReadSessionMeta(metaPath)
	if err != nil {
		t.Fatalf("ReadSessionMeta() error = %v", err)
	}
	if got, want := meta.Provider, "claude"; got != want {
		t.Fatalf("meta.Provider = %q, want %q", got, want)
	}
}

func TestReconciliationFailsWhenLegacyProviderRepairCannotResolveAgent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sessionDir := filepath.Join(h.home.SessionsDir, "sess-bad-repair")
	metaPath := store.SessionMetaFile(sessionDir)
	now := h.now.Add(50 * time.Minute)

	if err := store.WriteSessionMeta(metaPath, store.SessionMeta{
		ID:          "sess-bad-repair",
		Name:        "Bad Repair",
		AgentName:   "missing-agent",
		WorkspaceID: h.workspaceID,
		State:       "stopped",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	_, err := h.observer.Reconcile(testutil.Context(t))
	if err == nil {
		t.Fatal("Reconcile() error = nil, want legacy provider repair failure")
	}
	if !strings.Contains(err.Error(), "sess-bad-repair") {
		t.Fatalf("Reconcile() error = %q, want session id detail", err.Error())
	}
	if !strings.Contains(err.Error(), "missing-agent") {
		t.Fatalf("Reconcile() error = %q, want missing agent detail", err.Error())
	}

	sessions, err := h.observer.registry.ListSessions(testutil.Context(t), store.SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("len(sessions) = %d, want 0", len(sessions))
	}
}

func TestReconciliationSkipsLegacyStoppedSessionMetadata(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	validDir := filepath.Join(h.home.SessionsDir, "sess-valid")
	validMetaPath := store.SessionMetaFile(validDir)
	now := h.now.Add(45 * time.Minute)
	if err := store.WriteSessionMeta(validMetaPath, store.SessionMeta{
		ID:          "sess-valid",
		Name:        "Valid",
		AgentName:   "coder",
		Provider:    "claude",
		WorkspaceID: h.workspaceID,
		State:       "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("WriteSessionMeta(valid) error = %v", err)
	}

	legacyDir := filepath.Join(h.home.SessionsDir, "sess-legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(legacyDir) error = %v", err)
	}
	legacyMeta := `{
  "id": "sess-legacy",
  "name": "legacy-session",
  "goal": "Legacy prompt",
  "workspace": "` + h.workspace + `",
  "state": "stopped",
  "created_at": "2026-04-01T03:57:38.428414Z",
  "stopped_at": "2026-04-01T03:58:00.212132Z"
}
`
	if err := os.WriteFile(store.SessionMetaFile(legacyDir), []byte(legacyMeta), 0o644); err != nil {
		t.Fatalf("WriteFile(legacy meta) error = %v", err)
	}

	result, err := h.observer.Reconcile(testutil.Context(t))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	sort.Strings(result.Indexed)
	if got, want := result.Indexed, []string{"sess-valid"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("Indexed = %#v, want %#v", got, want)
	}

	sessions, err := h.observer.registry.ListSessions(testutil.Context(t), store.SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].ID != "sess-valid" {
		t.Fatalf("sessions[0].ID = %q, want %q", sessions[0].ID, "sess-valid")
	}
}

func TestReconciliationSkipsSessionMetadataMissingWorkspaceID(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	sessionDir := filepath.Join(h.home.SessionsDir, "sess-missing-workspace")
	meta := `{
  "id": "sess-missing-workspace",
  "name": "Missing Workspace",
  "agent_name": "coder",
  "state": "active",
  "created_at": "2026-04-03T18:30:00Z",
  "updated_at": "2026-04-03T18:30:00Z"
}
`
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(sessionDir) error = %v", err)
	}
	if err := os.WriteFile(store.SessionMetaFile(sessionDir), []byte(meta), 0o644); err != nil {
		t.Fatalf("WriteFile(meta) error = %v", err)
	}

	result, err := h.observer.Reconcile(testutil.Context(t))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if len(result.Indexed) != 0 {
		t.Fatalf("Indexed = %#v, want empty", result.Indexed)
	}
	if len(result.Orphaned) != 0 {
		t.Fatalf("Orphaned = %#v, want empty", result.Orphaned)
	}

	sessions, err := h.observer.registry.ListSessions(testutil.Context(t), store.SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("len(sessions) = %d, want 0", len(sessions))
	}
}
