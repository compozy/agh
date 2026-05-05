package session

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestBundledCoordinatorFallback(t *testing.T) {
	t.Run("Should create a coordinator session without a materialized agent definition", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		configureBundledCoordinatorFallbackWorkspace(t, h)

		created := createBundledCoordinatorSession(t, h)
		t.Cleanup(func() {
			if err := h.manager.Stop(testutil.Context(t), created.ID); err != nil {
				t.Fatalf("Stop(%q) error = %v", created.ID, err)
			}
		})

		if got, want := created.AgentName, aghconfig.DefaultCoordinatorAgentName; got != want {
			t.Fatalf("Create().AgentName = %q, want %q", got, want)
		}
		if got, want := created.Type, SessionTypeCoordinator; got != want {
			t.Fatalf("Create().Type = %q, want %q", got, want)
		}
		if got, want := len(h.driver.startCalls), 1; got != want {
			t.Fatalf("len(startCalls) = %d, want %d", got, want)
		}
		if got, want := h.driver.startCalls[0].AgentName, aghconfig.DefaultCoordinatorAgentName; got != want {
			t.Fatalf("startCalls[0].AgentName = %q, want %q", got, want)
		}
		if got := h.driver.startCalls[0].SystemPrompt; !strings.Contains(
			got,
			aghconfig.DefaultCoordinatorAgentDef().Prompt,
		) {
			t.Fatalf("startCalls[0].SystemPrompt = %q, want bundled coordinator prompt", got)
		}
	})

	t.Run("Should resume a coordinator session without a materialized agent definition", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		configureBundledCoordinatorFallbackWorkspace(t, h)

		ctx := testutil.Context(t)
		created := createBundledCoordinatorSession(t, h)
		if err := h.manager.Stop(ctx, created.ID); err != nil {
			t.Fatalf("Stop(%q) error = %v", created.ID, err)
		}

		resumed, err := h.manager.Resume(ctx, created.ID)
		if err != nil {
			t.Fatalf("Resume(%q) error = %v", created.ID, err)
		}
		t.Cleanup(func() {
			if err := h.manager.Stop(testutil.Context(t), resumed.ID); err != nil {
				t.Fatalf("Stop(%q) error = %v", resumed.ID, err)
			}
		})

		if got, want := resumed.AgentName, aghconfig.DefaultCoordinatorAgentName; got != want {
			t.Fatalf("Resume().AgentName = %q, want %q", got, want)
		}
		if got, want := len(h.driver.startCalls), 2; got != want {
			t.Fatalf("len(startCalls) after resume = %d, want %d", got, want)
		}
		if got := h.driver.startCalls[1].SystemPrompt; !strings.Contains(
			got,
			aghconfig.DefaultCoordinatorAgentDef().Prompt,
		) {
			t.Fatalf("startCalls[1].SystemPrompt = %q, want bundled coordinator prompt", got)
		}
	})

	t.Run(
		"Should validate stopped coordinator infrastructure without a materialized agent definition",
		func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			configureBundledCoordinatorFallbackWorkspace(t, h)

			meta := validResumeMeta(h, "sess-coordinator-fallback")
			meta.AgentName = aghconfig.DefaultCoordinatorAgentName
			meta.Provider = "claude"
			meta.SessionType = string(SessionTypeCoordinator)
			writeResumeEventStore(t, h.homePaths, meta.ID, []byte("not-empty"))

			errs := h.manager.validateInfrastructure(testutil.Context(t), meta)
			if len(errs) != 0 {
				t.Fatalf("validateInfrastructure() errors = %#v, want none", errs)
			}
		},
	)

	t.Run("Should repair a blank coordinator provider from the bundled fallback", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		configureBundledCoordinatorFallbackWorkspace(t, h)

		meta := validResumeMeta(h, "sess-coordinator-repair")
		meta.AgentName = aghconfig.DefaultCoordinatorAgentName
		meta.Provider = ""
		meta.SessionType = string(SessionTypeCoordinator)

		repaired, err := RepairLegacyProvider(
			testutil.Context(t),
			filepath.Join(t.TempDir(), "meta.json"),
			meta,
			LegacyProviderRepairOptions{
				Now:               h.manager.now,
				Logger:            h.manager.logger,
				WorkspaceResolver: h.resolver,
				AgentResolver:     h.manager.agentResolver,
			},
		)
		if err != nil {
			t.Fatalf("RepairLegacyProvider() error = %v", err)
		}
		if got, want := repaired.Provider, "claude"; got != want {
			t.Fatalf("RepairLegacyProvider().Provider = %q, want %q", got, want)
		}
	})
}

func createBundledCoordinatorSession(t *testing.T, h *harness) *Session {
	t.Helper()

	created, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: aghconfig.DefaultCoordinatorAgentName,
		Provider:  "claude",
		Name:      "bundled-coordinator",
		Workspace: h.workspaceID,
		Channel:   "coord-fallback-test",
		Lineage: &store.SessionLineage{
			SpawnRole: string(SessionTypeCoordinator),
			TTLExpiresAt: func() *time.Time {
				ttl := h.manager.now().UTC().Add(time.Hour)
				return &ttl
			}(),
		},
		Type: SessionTypeCoordinator,
	})
	if err != nil {
		t.Fatalf("Create(coordinator) error = %v", err)
	}
	return created
}

func configureBundledCoordinatorFallbackWorkspace(t *testing.T, h *harness) {
	t.Helper()

	resolved, err := h.resolver.Resolve(context.Background(), h.workspaceID)
	if err != nil {
		t.Fatalf("Resolve(%q) error = %v", h.workspaceID, err)
	}
	resolved.Config.Defaults.Provider = "claude"
	h.cfg.Defaults.Provider = "claude"
	h.resolver.upsert(&resolved)
	if _, err := resolveWorkspaceAgent(
		aghconfig.DefaultCoordinatorAgentName,
		&resolved,
	); !errors.Is(
		err,
		workspacepkg.ErrAgentNotAvailable,
	) {
		t.Fatalf("resolveWorkspaceAgent(coordinator) error = %v, want ErrAgentNotAvailable", err)
	}
}
