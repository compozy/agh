package session

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/testutil"
)

func TestManagerDelete(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "ShouldRemoveStoppedSessionFromHistory",
			run: func(t *testing.T) {
				h := newHarness(t)
				session := createSession(t, h)

				if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
					t.Fatalf("Stop() error = %v", err)
				}

				if _, err := os.Stat(session.SessionDir()); err != nil {
					t.Fatalf("Stat(session dir before delete) error = %v", err)
				}

				if err := h.manager.Delete(testutil.Context(t), session.ID); err != nil {
					t.Fatalf("Delete() error = %v", err)
				}

				if _, err := os.Stat(session.SessionDir()); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("Stat(session dir after delete) error = %v, want os.ErrNotExist", err)
				}
				if _, err := h.manager.Status(testutil.Context(t), session.ID); !errors.Is(err, ErrSessionNotFound) {
					t.Fatalf("Status(after delete) error = %v, want %v", err, ErrSessionNotFound)
				}

				infos, err := h.manager.ListAll(testutil.Context(t))
				if err != nil {
					t.Fatalf("ListAll() error = %v", err)
				}
				for _, info := range infos {
					if info != nil && info.ID == session.ID {
						t.Fatalf("ListAll() still returned deleted session %q", session.ID)
					}
				}
			},
		},
		{
			name: "ShouldStopActiveSessionBeforeRemovingArtifacts",
			run: func(t *testing.T) {
				h := newHarness(t)
				session := createSession(t, h)

				if got := h.driver.stopCalls; got != 0 {
					t.Fatalf("driver stop calls before delete = %d, want 0", got)
				}

				if err := h.manager.Delete(testutil.Context(t), session.ID); err != nil {
					t.Fatalf("Delete(active) error = %v", err)
				}

				if got := h.driver.stopCalls; got != 1 {
					t.Fatalf("driver stop calls after delete = %d, want 1", got)
				}
				if _, ok := h.manager.Get(session.ID); ok {
					t.Fatalf("Get(%q) after delete = found, want missing", session.ID)
				}
				if _, err := os.Stat(session.SessionDir()); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("Stat(session dir after delete) error = %v, want os.ErrNotExist", err)
				}
			},
		},
		{
			name: "ShouldWrapStopErrorsWithDeleteContext",
			run: func(t *testing.T) {
				h := newHarness(t)
				session := createSession(t, h)
				stopErr := errors.New("driver stop failed")
				h.driver.stopHook = func(*fakeProcess) error {
					return stopErr
				}

				err := h.manager.Delete(testutil.Context(t), session.ID)
				if !errors.Is(err, stopErr) {
					t.Fatalf("Delete() error = %v, want wrapped stop error", err)
				}
				if !strings.Contains(err.Error(), `session: stop "`) {
					t.Fatalf("Delete() error = %q, want stop context", err.Error())
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t)
		})
	}
}
