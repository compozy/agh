package session

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/testutil"
)

func TestManagerDelete(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "Should remove stopped session from history and catalog",
			run: func(t *testing.T) {
				catalog := newRecordingSessionCatalog()
				h := newHarness(t, WithSessionCatalog(catalog))
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
				if _, ok := catalog.get(session.ID); ok {
					t.Fatalf("catalog still returned deleted session %q", session.ID)
				}
			},
		},
		{
			name: "Should stop active session before removing catalog and artifacts",
			run: func(t *testing.T) {
				catalog := newRecordingSessionCatalog()
				catalog.requireExistingUpdates()
				h := newHarness(t, WithSessionCatalog(catalog))
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
				if _, ok := catalog.get(session.ID); ok {
					t.Fatalf("catalog still returned deleted active session %q", session.ID)
				}
			},
		},
		{
			name: "Should finish artifact cleanup when catalog was already deleted",
			run: func(t *testing.T) {
				catalog := newRecordingSessionCatalog()
				h := newHarness(t, WithSessionCatalog(catalog))
				session := createSession(t, h)

				if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
					t.Fatalf("Stop() error = %v", err)
				}
				if err := catalog.DeleteSession(testutil.Context(t), session.ID); err != nil {
					t.Fatalf("DeleteSession() setup error = %v", err)
				}

				if err := h.manager.Delete(testutil.Context(t), session.ID); err != nil {
					t.Fatalf("Delete() retry error = %v", err)
				}
				if _, err := os.Stat(session.SessionDir()); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("Stat(session dir after retry) error = %v, want os.ErrNotExist", err)
				}
			},
		},
		{
			name: "Should preserve artifacts when catalog deletion fails",
			run: func(t *testing.T) {
				catalogErr := errors.New("catalog unavailable")
				catalog := newRecordingSessionCatalog()
				catalog.setDeleteErr(catalogErr)
				h := newHarness(t, WithSessionCatalog(catalog))
				session := createSession(t, h)

				err := h.manager.Delete(testutil.Context(t), session.ID)
				if !errors.Is(err, catalogErr) {
					t.Fatalf("Delete() error = %v, want wrapped catalog error", err)
				}
				if _, statErr := os.Stat(session.SessionDir()); statErr != nil {
					t.Fatalf("Stat(session dir after catalog failure) error = %v", statErr)
				}
				if _, ok := catalog.get(session.ID); !ok {
					t.Fatalf("catalog lost session %q after failed deletion", session.ID)
				}
			},
		},
		{
			name: "Should return session not found when catalog and artifacts are absent",
			run: func(t *testing.T) {
				catalog := newRecordingSessionCatalog()
				h := newHarness(t, WithSessionCatalog(catalog))

				err := h.manager.Delete(testutil.Context(t), "sess-missing-delete")
				if !errors.Is(err, ErrSessionNotFound) {
					t.Fatalf("Delete(missing) error = %v, want ErrSessionNotFound", err)
				}
			},
		},
		{
			name: "Should ignore concurrent stop races that report session not found",
			run: func(t *testing.T) {
				called := false

				err := stopSessionBeforeDelete(
					testutil.Context(t),
					"sess-race",
					func(context.Context, string, StopCause, string) error {
						called = true
						return ErrSessionNotFound
					},
				)
				if err != nil {
					t.Fatalf("stopSessionBeforeDelete() error = %v, want nil", err)
				}
				if !called {
					t.Fatal("stopSessionBeforeDelete() did not call the stop function")
				}
			},
		},
		{
			name: "Should wrap stop errors with delete context",
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
			t.Parallel()
			tc.run(t)
		})
	}
}
