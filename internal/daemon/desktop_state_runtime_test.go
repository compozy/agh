package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/compozy/agh/internal/clientstate"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/store/globaldb"
	"github.com/compozy/agh/internal/testutil"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestDesktopStateWorkspaceDeletionGate(t *testing.T) {
	t.Parallel()

	t.Run("Should purge state and close subscriptions before same-id recreation", func(t *testing.T) {
		t.Parallel()
		fixture := newDaemonDesktopStateFixture(t)
		sessionPreparation := &desktopStateSessionRemovalPreparation{}
		sessions := &desktopStateRemovalSessionManager{
			fakeSessionManager: &fakeSessionManager{},
			workspaceID:        fixture.workspace.ID,
			preparation:        sessionPreparation,
		}
		state := &bootState{
			desktopStateBootState: desktopStateBootState{
				desktopStateResolver: fixture.adapter,
				desktopState:         fixture.engine,
			},
			workspaceResolver: fixture.resolver,
		}
		if err := installWorkspaceRemovalPreparer(state, sessions); err != nil {
			t.Fatalf("installWorkspaceRemovalPreparer() error = %v", err)
		}

		ctx := testutil.Context(t)
		entries, err := fixture.engine.Apply(
			ctx,
			clientstate.WorkspaceID(fixture.workspace.ID),
			"os_shell",
			[]clientstate.Op{{Kind: clientstate.OpPut, Key: "desktop", Value: []byte(`{"v":1}`)}},
			clientstate.ApplyOptions{},
		)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		if len(entries) != 1 || entries[0].Rev != 1 || entries[0].Seq != 1 {
			t.Fatalf("Apply() entries = %#v, want initial rev=1 seq=1", entries)
		}
		subscription, err := fixture.engine.Watch(
			ctx,
			clientstate.WorkspaceID(fixture.workspace.ID),
			[]string{"os_shell"},
		)
		if err != nil {
			t.Fatalf("Watch() error = %v", err)
		}
		t.Cleanup(func() {
			if err := subscription.Close(); err != nil {
				t.Errorf("Subscription.Close() error = %v", err)
			}
		})

		if err := fixture.resolver.Unregister(ctx, fixture.workspace.ID); err != nil {
			t.Fatalf("Unregister() error = %v", err)
		}
		if sessionPreparation.commits != 1 || sessionPreparation.rollbacks != 0 {
			t.Fatalf(
				"session preparation = (commits=%d rollbacks=%d), want (1 0)",
				sessionPreparation.commits,
				sessionPreparation.rollbacks,
			)
		}
		select {
		case _, open := <-subscription.Events():
			if open {
				t.Fatal("subscription remained open after workspace purge")
			}
		case <-time.After(time.Second):
			t.Fatal("workspace purge did not close the subscription")
		}
		if _, err := fixture.engine.Get(
			ctx, clientstate.WorkspaceID(fixture.workspace.ID), "os_shell", "desktop",
		); !errors.Is(err, clientstate.ErrWorkspaceNotFound) {
			t.Fatalf("Get(deleted workspace) error = %v, want ErrWorkspaceNotFound", err)
		}

		if err := fixture.database.InsertWorkspace(ctx, fixture.workspace); err != nil {
			t.Fatalf("InsertWorkspace(same id) error = %v", err)
		}
		fixture.resolver.Invalidate(fixture.workspace.ID)
		if _, err := fixture.engine.Get(
			ctx, clientstate.WorkspaceID(fixture.workspace.ID), "os_shell", "desktop",
		); !errors.Is(err, clientstate.ErrNotFound) {
			t.Fatalf("Get(recreated workspace) error = %v, want empty ErrNotFound", err)
		}
		created, err := fixture.engine.Apply(
			ctx,
			clientstate.WorkspaceID(fixture.workspace.ID),
			"os_shell",
			[]clientstate.Op{{Kind: clientstate.OpPut, Key: "desktop", Value: []byte(`{"v":2}`)}},
			clientstate.ApplyOptions{},
		)
		if err != nil {
			t.Fatalf("Apply(recreated) error = %v", err)
		}
		if len(created) != 1 || created[0].Rev != 1 || created[0].Seq != 1 {
			t.Fatalf("recreated entry = %#v, want fresh rev=1 seq=1", created)
		}
	})

	t.Run("Should reopen the original generation when deletion preparation rolls back", func(t *testing.T) {
		t.Parallel()
		fixture := newDaemonDesktopStateFixture(t)
		ctx := testutil.Context(t)
		workspaceID := clientstate.WorkspaceID(fixture.workspace.ID)
		generation, err := fixture.adapter.ResolveWorkspace(ctx, workspaceID)
		if err != nil {
			t.Fatalf("ResolveWorkspace() error = %v", err)
		}
		preparation, err := fixture.adapter.prepareRemoval(fixture.workspace, fixture.engine)
		if err != nil {
			t.Fatalf("prepareRemoval() error = %v", err)
		}
		if _, err := fixture.adapter.ResolveWorkspace(
			ctx,
			workspaceID,
		); !errors.Is(err, clientstate.ErrWorkspaceNotFound) {
			t.Fatalf("ResolveWorkspace(deleting) error = %v, want ErrWorkspaceNotFound", err)
		}
		purgeGeneration, err := fixture.adapter.ResolveWorkspaceForPurge(ctx, workspaceID)
		if err != nil {
			t.Fatalf("ResolveWorkspaceForPurge() error = %v", err)
		}
		if purgeGeneration != generation {
			t.Fatalf("purge generation = %q, want %q", purgeGeneration, generation)
		}
		if err := preparation.Rollback(ctx); err != nil {
			t.Fatalf("Rollback() error = %v", err)
		}
		reopened, err := fixture.adapter.ResolveWorkspace(ctx, workspaceID)
		if err != nil {
			t.Fatalf("ResolveWorkspace(after rollback) error = %v", err)
		}
		if reopened != generation {
			t.Fatalf("generation after rollback = %q, want %q", reopened, generation)
		}
	})
}

type daemonDesktopStateFixture struct {
	database  *globaldb.GlobalDB
	resolver  *workspacepkg.Resolver
	adapter   *desktopStateWorkspaceResolver
	engine    *clientstate.Engine
	workspace workspacepkg.Workspace
}

func newDaemonDesktopStateFixture(t *testing.T) daemonDesktopStateFixture {
	t.Helper()
	database := openDaemonTestGlobalDB(t)
	homePaths := testHomePaths(t)
	resolver, err := workspacepkg.NewResolver(
		database,
		workspacepkg.WithHomePaths(homePaths),
		workspacepkg.WithLogger(discardLogger()),
		workspacepkg.WithConfigLoader(func(rootDir string) (aghconfig.Config, error) {
			return aghconfig.LoadForHome(homePaths, aghconfig.WithWorkspaceRoot(rootDir))
		}),
	)
	if err != nil {
		t.Fatalf("workspace.NewResolver() error = %v", err)
	}
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", root, err)
	}
	workspace, err := resolver.Register(testutil.Context(t), workspacepkg.RegisterOptions{
		RootDir: root,
		Name:    "desktop-state",
	})
	if err != nil {
		t.Fatalf("workspace.Register() error = %v", err)
	}
	adapter, err := newDesktopStateWorkspaceResolver(resolver, discardLogger())
	if err != nil {
		t.Fatalf("newDesktopStateWorkspaceResolver() error = %v", err)
	}
	engine, err := clientstate.Open(
		clientstate.DatabasePath(t.TempDir()),
		adapter,
		clientstate.DefaultLimits(),
		clientstate.WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("clientstate.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := engine.Close(); err != nil {
			t.Errorf("Engine.Close() error = %v", err)
		}
	})
	return daemonDesktopStateFixture{
		database: database, resolver: resolver, adapter: adapter, engine: engine, workspace: workspace,
	}
}

type desktopStateRemovalSessionManager struct {
	*fakeSessionManager
	workspaceID string
	preparation workspacepkg.UnregisterPreparation
}

func (m *desktopStateRemovalSessionManager) PrepareWorkspaceRemoval(
	_ context.Context,
	workspaceID string,
) (workspacepkg.UnregisterPreparation, error) {
	if workspaceID != m.workspaceID {
		return nil, errors.New("unexpected workspace removal id")
	}
	return m.preparation, nil
}

type desktopStateSessionRemovalPreparation struct {
	commits   int
	rollbacks int
}

func (*desktopStateSessionRemovalPreparation) BeforeDelete(context.Context) error {
	return nil
}

func (p *desktopStateSessionRemovalPreparation) Commit(context.Context) error {
	p.commits++
	return nil
}

func (p *desktopStateSessionRemovalPreparation) Rollback(context.Context) error {
	p.rollbacks++
	return nil
}
