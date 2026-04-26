package session

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestCreateManualSessionProducesRootLineage(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sess := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), sess.ID)
	})

	info := sess.Info()
	if info.Lineage == nil {
		t.Fatal("session.Info().Lineage = nil, want root lineage")
	}
	if info.Lineage.ParentSessionID != "" ||
		info.Lineage.RootSessionID != sess.ID ||
		info.Lineage.SpawnDepth != 0 ||
		info.Lineage.AutoStopOnParent {
		t.Fatalf("root lineage = %#v, want no parent, own root, depth 0", info.Lineage)
	}

	meta := readMeta(t, sess.MetaPath())
	if meta.Lineage == nil {
		t.Fatal("meta.Lineage = nil, want persisted root lineage")
	}
	if got, want := meta.Lineage.RootSessionID, sess.ID; got != want {
		t.Fatalf("meta.Lineage.RootSessionID = %q, want %q", got, want)
	}
}

func TestCreateSpawnedAndCoordinatorSessionsValidateLineage(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	h := newHarness(t, WithNow(func() time.Time { return now }))
	parent := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), parent.ID)
	})

	childTTL := now.Add(45 * time.Minute)
	child, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "worker",
		Workspace: h.workspaceID,
		Type:      SessionTypeSpawned,
		Lineage: &store.SessionLineage{
			ParentSessionID:  parent.ID,
			RootSessionID:    parent.ID,
			SpawnDepth:       1,
			SpawnRole:        "worker",
			TTLExpiresAt:     &childTTL,
			AutoStopOnParent: true,
			SpawnBudget: store.SessionSpawnBudget{
				MaxChildren:           2,
				MaxDepth:              1,
				MaxActivePerWorkspace: 4,
			},
			PermissionPolicy: store.SessionPermissionPolicy{
				Tools:           []string{"edit", "read"},
				Skills:          []string{"go"},
				NetworkChannels: []string{"builders"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Create(spawned) error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), child.ID)
	})

	childInfo := child.Info()
	if childInfo.Type != SessionTypeSpawned {
		t.Fatalf("child type = %q, want %q", childInfo.Type, SessionTypeSpawned)
	}
	if childInfo.Lineage == nil {
		t.Fatal("child lineage = nil, want metadata")
	}
	if childInfo.Lineage.ParentSessionID != parent.ID ||
		childInfo.Lineage.RootSessionID != parent.ID ||
		childInfo.Lineage.SpawnDepth != 1 ||
		childInfo.Lineage.SpawnRole != "worker" ||
		!childInfo.Lineage.AutoStopOnParent {
		t.Fatalf("child lineage = %#v", childInfo.Lineage)
	}
	if childInfo.Lineage.TTLExpiresAt == nil || !childInfo.Lineage.TTLExpiresAt.Equal(childTTL) {
		t.Fatalf("child TTL = %#v, want %s", childInfo.Lineage.TTLExpiresAt, childTTL)
	}
	if childInfo.Lineage.SpawnBudget.TTLSeconds <= 0 {
		t.Fatalf("child budget TTLSeconds = %d, want derived positive value", childInfo.Lineage.SpawnBudget.TTLSeconds)
	}
	if got := childInfo.Lineage.PermissionPolicy.Tools; len(got) != 2 || got[0] != "edit" || got[1] != "read" {
		t.Fatalf("child policy tools = %#v, want normalized atoms", got)
	}

	coordinatorTTL := now.Add(2 * time.Hour)
	coordinator, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "coordinator",
		Workspace: h.workspaceID,
		Type:      SessionTypeCoordinator,
		Lineage: &store.SessionLineage{
			SpawnRole:    "coordinator",
			TTLExpiresAt: &coordinatorTTL,
			SpawnBudget:  store.SessionSpawnBudget{MaxChildren: 5, MaxDepth: 1},
		},
	})
	if err != nil {
		t.Fatalf("Create(coordinator) error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), coordinator.ID)
	})
	if coordinator.Info().Type != SessionTypeCoordinator ||
		coordinator.Info().Lineage == nil ||
		coordinator.Info().Lineage.RootSessionID != coordinator.ID ||
		coordinator.Info().Lineage.ParentSessionID != "" {
		t.Fatalf("coordinator info = %#v", coordinator.Info())
	}
}

func TestCreateRejectsInvalidLineage(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	h := newHarness(t, WithNow(func() time.Time { return now }))
	parent := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), parent.ID)
	})
	future := now.Add(time.Hour)
	expired := now.Add(-time.Second)

	tests := []struct {
		name    string
		opts    CreateOpts
		wantErr string
	}{
		{
			name: "invalid depth",
			opts: spawnedCreateOpts(parent.ID, &store.SessionLineage{
				ParentSessionID: parent.ID,
				RootSessionID:   parent.ID,
				SpawnDepth:      -1,
				TTLExpiresAt:    &future,
			}),
			wantErr: "spawn depth",
		},
		{
			name: "missing root",
			opts: spawnedCreateOpts(parent.ID, &store.SessionLineage{
				ParentSessionID: parent.ID,
				SpawnDepth:      1,
				TTLExpiresAt:    &future,
			}),
			wantErr: "root session id is required",
		},
		{
			name: "expired ttl",
			opts: spawnedCreateOpts(parent.ID, &store.SessionLineage{
				ParentSessionID: parent.ID,
				RootSessionID:   parent.ID,
				SpawnDepth:      1,
				TTLExpiresAt:    &expired,
			}),
			wantErr: "ttl deadline must be in the future",
		},
		{
			name: "malformed policy",
			opts: spawnedCreateOpts(parent.ID, &store.SessionLineage{
				ParentSessionID: parent.ID,
				RootSessionID:   parent.ID,
				SpawnDepth:      1,
				TTLExpiresAt:    &future,
				PermissionPolicy: store.SessionPermissionPolicy{
					Tools: []string{"edit", " "},
				},
			}),
			wantErr: "empty atom",
		},
		{
			name: "missing parent",
			opts: spawnedCreateOpts(parent.ID, &store.SessionLineage{
				ParentSessionID: "sess-missing",
				RootSessionID:   parent.ID,
				SpawnDepth:      1,
				TTLExpiresAt:    &future,
			}),
			wantErr: "validate parent lineage",
		},
		{
			name: "coordinator missing ttl",
			opts: CreateOpts{
				AgentName: "coder",
				Workspace: h.workspaceID,
				Type:      SessionTypeCoordinator,
				Lineage:   &store.SessionLineage{SpawnRole: "coordinator"},
			},
			wantErr: "require a ttl deadline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.manager.Create(testutil.Context(t), tt.opts)
			if err == nil {
				t.Fatalf("Create(%s) error = nil, want failure", tt.name)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Create(%s) error = %v, want substring %q", tt.name, err, tt.wantErr)
			}
			if !errors.Is(err, ErrSessionNotFound) && strings.Contains(tt.wantErr, "validate parent lineage") {
				t.Fatalf("Create(%s) error = %v, want ErrSessionNotFound wrapping", tt.name, err)
			}
		})
	}
}

func spawnedCreateOpts(rootID string, lineage *store.SessionLineage) CreateOpts {
	return CreateOpts{
		AgentName: "coder",
		Workspace: "ws-primary",
		Type:      SessionTypeSpawned,
		Lineage:   lineage,
		Name:      "child-of-" + rootID,
	}
}
