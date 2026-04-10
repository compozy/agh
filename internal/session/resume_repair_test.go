package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestClassifyPreviousStop(t *testing.T) {
	t.Parallel()

	existingReason := store.StopUserCanceled

	testCases := []struct {
		name        string
		meta        store.SessionMeta
		wantChanged bool
		wantReason  *store.StopReason
		wantDetail  string
	}{
		{
			name:        "active session classified as crashed",
			meta:        store.SessionMeta{State: string(StateActive)},
			wantChanged: true,
			wantReason:  stopReasonPointer(store.StopAgentCrashed),
			wantDetail:  "daemon crashed while session active",
		},
		{
			name:        "stopping session classified as crashed",
			meta:        store.SessionMeta{State: string(StateStopping)},
			wantChanged: true,
			wantReason:  stopReasonPointer(store.StopAgentCrashed),
			wantDetail:  "stop did not complete",
		},
		{
			name:        "starting session classified as error",
			meta:        store.SessionMeta{State: string(StateStarting)},
			wantChanged: true,
			wantReason:  stopReasonPointer(store.StopError),
			wantDetail:  "start did not complete",
		},
		{
			name: "stopped session preserves existing reason",
			meta: store.SessionMeta{
				State:      string(StateStopped),
				StopReason: &existingReason,
				StopDetail: "requested by user",
			},
			wantChanged: false,
			wantReason:  &existingReason,
			wantDetail:  "requested by user",
		},
		{
			name:        "stopped session with no reason remains unchanged",
			meta:        store.SessionMeta{State: string(StateStopped)},
			wantChanged: false,
			wantReason:  nil,
			wantDetail:  "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotMeta, gotChanged := classifyPreviousStop(tc.meta)
			if gotChanged != tc.wantChanged {
				t.Fatalf("classifyPreviousStop() changed = %v, want %v", gotChanged, tc.wantChanged)
			}
			assertOptionalStopReasonEqual(t, gotMeta.StopReason, tc.wantReason)
			if gotMeta.StopDetail != tc.wantDetail {
				t.Fatalf("classifyPreviousStop() detail = %q, want %q", gotMeta.StopDetail, tc.wantDetail)
			}
		})
	}
}

func TestValidateInfrastructure(t *testing.T) {
	t.Parallel()

	t.Run("valid infrastructure", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		meta := validResumeMeta(h, "sess-valid")
		writeResumeEventStore(t, h.homePaths, meta.ID, []byte("not-empty"))

		errs := h.manager.validateInfrastructure(testutil.Context(t), meta)
		if len(errs) != 0 {
			t.Fatalf("validateInfrastructure() errors = %#v, want none", errs)
		}
	})

	t.Run("missing workspace directory", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		missingWorkspace := filepath.Join(t.TempDir(), "missing-workspace")
		h.resolver.upsert(workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      h.workspaceID,
				RootDir: missingWorkspace,
				Name:    h.workspaceName,
			},
			Config: h.cfg,
			Agents: []aghconfig.AgentDef{{
				Name:     "coder",
				Provider: "claude",
				Prompt:   "You are a coding assistant.",
			}},
		})
		meta := validResumeMeta(h, "sess-missing-workspace")
		writeResumeEventStore(t, h.homePaths, meta.ID, []byte("not-empty"))

		errs := h.manager.validateInfrastructure(testutil.Context(t), meta)
		assertErrorContains(t, errs, missingWorkspace)
	})

	t.Run("unresolvable agent", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		meta := validResumeMeta(h, "sess-missing-agent")
		meta.AgentName = "missing-agent"
		writeResumeEventStore(t, h.homePaths, meta.ID, []byte("not-empty"))

		errs := h.manager.validateInfrastructure(testutil.Context(t), meta)
		assertErrorContains(t, errs, "missing-agent")
	})

	t.Run("missing event store", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		meta := validResumeMeta(h, "sess-missing-store")

		errs := h.manager.validateInfrastructure(testutil.Context(t), meta)
		assertErrorContains(t, errs, store.SessionDBFile(filepath.Join(h.homePaths.SessionsDir, meta.ID)))
	})

	t.Run("empty event store", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		meta := validResumeMeta(h, "sess-empty-store")
		writeResumeEventStore(t, h.homePaths, meta.ID, nil)

		errs := h.manager.validateInfrastructure(testutil.Context(t), meta)
		assertErrorContains(t, errs, "file is empty")
	})

	t.Run("invalid meta fields", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		meta := validResumeMeta(h, "")
		writeResumeEventStore(t, h.homePaths, "ignored", []byte("not-empty"))

		errs := h.manager.validateInfrastructure(testutil.Context(t), meta)
		assertErrorContains(t, errs, "session id")
	})

	t.Run("collects multiple failures", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		missingWorkspace := filepath.Join(t.TempDir(), "missing-workspace")
		h.resolver.upsert(workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      h.workspaceID,
				RootDir: missingWorkspace,
				Name:    h.workspaceName,
			},
			Config: h.cfg,
			Agents: []aghconfig.AgentDef{{
				Name:     "coder",
				Provider: "claude",
				Prompt:   "You are a coding assistant.",
			}},
		})
		meta := validResumeMeta(h, "sess-multi")
		meta.AgentName = "missing-agent"
		writeResumeEventStore(t, h.homePaths, meta.ID, nil)

		errs := h.manager.validateInfrastructure(testutil.Context(t), meta)
		if got, want := len(errs), 3; got != want {
			t.Fatalf("len(validateInfrastructure() errors) = %d, want %d (%#v)", got, want, errs)
		}
		assertErrorContains(t, errs, missingWorkspace)
		assertErrorContains(t, errs, "missing-agent")
		assertErrorContains(t, errs, "file is empty")
	})
}

func validResumeMeta(h *harness, sessionID string) store.SessionMeta {
	return store.SessionMeta{
		ID:          sessionID,
		Name:        "resume-session",
		AgentName:   "coder",
		WorkspaceID: h.workspaceID,
		SessionType: string(SessionTypeUser),
		State:       string(StateStopped),
	}
}

func writeResumeEventStore(t *testing.T, homePaths aghconfig.HomePaths, sessionID string, contents []byte) string {
	t.Helper()

	sessionDir := filepath.Join(homePaths.SessionsDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", sessionDir, err)
	}

	dbPath := store.SessionDBFile(sessionDir)
	if err := os.WriteFile(dbPath, contents, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", dbPath, err)
	}
	return dbPath
}

func assertOptionalStopReasonEqual(t *testing.T, got *store.StopReason, want *store.StopReason) {
	t.Helper()

	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("stop reason = %v, want %v", got, want)
	case *got != *want:
		t.Fatalf("stop reason = %q, want %q", *got, *want)
	}
}

func assertErrorContains(t *testing.T, errs []error, fragment string) {
	t.Helper()

	for _, err := range errs {
		if err != nil && strings.Contains(err.Error(), fragment) {
			return
		}
	}
	t.Fatalf("errors %#v do not contain %q", errs, fragment)
}
