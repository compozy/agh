package session

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestValidatePermissionSubset(t *testing.T) {
	t.Parallel()

	parent := store.SessionPermissionPolicy{
		Tools:           []string{"edit", "read"},
		Skills:          []string{"go", "tests"},
		MCPServers:      []string{"filesystem"},
		WorkspacePaths:  []string{"/repo"},
		NetworkChannels: []string{"builders"},
		SandboxProfiles: []string{"default"},
	}

	tests := []struct {
		name    string
		child   store.SessionPermissionPolicy
		wantErr bool
	}{
		{
			name:  "exact",
			child: parent,
		},
		{
			name: "subset",
			child: store.SessionPermissionPolicy{
				Tools:           []string{"read"},
				Skills:          []string{"go"},
				MCPServers:      []string{"filesystem"},
				WorkspacePaths:  []string{"/repo"},
				NetworkChannels: []string{"builders"},
				SandboxProfiles: []string{"default"},
			},
		},
		{
			name: "superset rejected",
			child: store.SessionPermissionPolicy{
				Tools: []string{"edit", "shell"},
			},
			wantErr: true,
		},
		{
			name: "unknown atom rejected",
			child: store.SessionPermissionPolicy{
				MCPServers: []string{"unknown-server"},
			},
			wantErr: true,
		},
		{
			name: "blank atom rejected",
			child: store.SessionPermissionPolicy{
				Tools: []string{" "},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidatePermissionSubset(parent, tt.child)
			if tt.wantErr {
				if !errors.Is(err, ErrSpawnPermissionDenied) {
					t.Fatalf("ValidatePermissionSubset() error = %v, want %v", err, ErrSpawnPermissionDenied)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidatePermissionSubset() error = %v", err)
			}
		})
	}
}

func TestManagerSpawnCreatesChildWithDurableLineageAndNarrowPermissions(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	h := newHarness(t, WithNow(func() time.Time { return now }))
	parentPolicy := store.SessionPermissionPolicy{
		Tools:           []string{"edit", "read"},
		Skills:          []string{"go", "tests"},
		MCPServers:      []string{"filesystem"},
		WorkspacePaths:  []string{h.workspace},
		NetworkChannels: []string{"builders"},
		SandboxProfiles: []string{"default"},
	}
	parent := createSpawnParent(t, h, parentPolicy, store.SessionSpawnBudget{
		MaxChildren:           2,
		MaxDepth:              1,
		MaxActivePerWorkspace: 2,
	})
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), parent.ID)
	})

	child, err := h.manager.Spawn(testutil.Context(t), SpawnOpts{
		ParentSessionID:  parent.ID,
		AgentName:        "coder",
		Name:             "child worker",
		PromptOverlay:    "Focus only on tests.",
		TTL:              30 * time.Minute,
		AutoStopOnParent: true,
		PermissionPolicy: store.SessionPermissionPolicy{
			Tools:           []string{"read"},
			Skills:          []string{"go"},
			MCPServers:      []string{"filesystem"},
			WorkspacePaths:  []string{h.workspace},
			NetworkChannels: []string{"builders"},
			SandboxProfiles: []string{"default"},
		},
	})
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), child.ID)
	})

	info := child.Info()
	if info.Type != SessionTypeSpawned {
		t.Fatalf("child type = %q, want %q", info.Type, SessionTypeSpawned)
	}
	if info.Channel != parent.Info().Channel {
		t.Fatalf("child channel = %q, want inherited %q", info.Channel, parent.Info().Channel)
	}
	if info.Lineage == nil {
		t.Fatal("child lineage = nil, want durable lineage")
	}
	if info.Lineage.ParentSessionID != parent.ID ||
		info.Lineage.RootSessionID != parent.ID ||
		info.Lineage.SpawnDepth != 1 ||
		info.Lineage.SpawnRole != DefaultSpawnRole ||
		!info.Lineage.AutoStopOnParent {
		t.Fatalf("child lineage = %#v", info.Lineage)
	}
	wantTTL := now.Add(30 * time.Minute)
	if info.Lineage.TTLExpiresAt == nil || !info.Lineage.TTLExpiresAt.Equal(wantTTL) {
		t.Fatalf("child TTL = %#v, want %s", info.Lineage.TTLExpiresAt, wantTTL)
	}
	if got := info.Lineage.PermissionPolicy.Tools; len(got) != 1 || got[0] != "read" {
		t.Fatalf("child permission tools = %#v, want narrowed read", got)
	}
	meta := readMeta(t, child.MetaPath())
	if meta.Lineage == nil || meta.Lineage.ParentSessionID != parent.ID {
		t.Fatalf("persisted lineage = %#v, want parent %q", meta.Lineage, parent.ID)
	}
	if len(h.driver.startCalls) < 2 ||
		!strings.Contains(h.driver.startCalls[len(h.driver.startCalls)-1].SystemPrompt, "Focus only on tests.") {
		t.Fatalf("child prompt overlay was not appended to start prompt: %#v", h.driver.startCalls)
	}
}

func TestManagerSpawnRejectsPolicyViolations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		run     func(t *testing.T, h *harness, parent *Session) error
		wantErr error
	}{
		{
			name: "missing TTL",
			run: func(t *testing.T, h *harness, parent *Session) error {
				t.Helper()
				_, err := h.manager.Spawn(testutil.Context(t), SpawnOpts{
					ParentSessionID: parent.ID,
					AgentName:       "coder",
				})
				return err
			},
			wantErr: ErrSpawnValidation,
		},
		{
			name: "coordinator role",
			run: func(t *testing.T, h *harness, parent *Session) error {
				t.Helper()
				_, err := h.manager.Spawn(testutil.Context(t), SpawnOpts{
					ParentSessionID: parent.ID,
					AgentName:       "coder",
					SpawnRole:       "coordinator",
					TTL:             time.Minute,
				})
				return err
			},
			wantErr: ErrSpawnValidation,
		},
		{
			name: "permission widening",
			run: func(t *testing.T, h *harness, parent *Session) error {
				t.Helper()
				_, err := h.manager.Spawn(testutil.Context(t), SpawnOpts{
					ParentSessionID: parent.ID,
					AgentName:       "coder",
					TTL:             time.Minute,
					PermissionPolicy: store.SessionPermissionPolicy{
						Tools: []string{"shell"},
					},
				})
				return err
			},
			wantErr: ErrSpawnPermissionDenied,
		},
		{
			name: "cross workspace",
			run: func(t *testing.T, h *harness, parent *Session) error {
				t.Helper()
				_, err := h.manager.Spawn(testutil.Context(t), SpawnOpts{
					ParentSessionID: parent.ID,
					AgentName:       "coder",
					Workspace:       "ws-other",
					TTL:             time.Minute,
				})
				return err
			},
			wantErr: ErrSpawnPermissionDenied,
		},
		{
			name: "child cap",
			run: func(t *testing.T, h *harness, parent *Session) error {
				t.Helper()
				child, err := h.manager.Spawn(testutil.Context(t), SpawnOpts{
					ParentSessionID: parent.ID,
					AgentName:       "coder",
					TTL:             time.Minute,
				})
				if err != nil {
					return err
				}
				t.Cleanup(func() {
					_ = h.manager.Stop(testutil.Context(t), child.ID)
				})
				_, err = h.manager.Spawn(testutil.Context(t), SpawnOpts{
					ParentSessionID: parent.ID,
					AgentName:       "coder",
					TTL:             time.Minute,
				})
				return err
			},
			wantErr: ErrSpawnLimitExceeded,
		},
		{
			name: "max depth",
			run: func(t *testing.T, h *harness, parent *Session) error {
				t.Helper()
				child, err := h.manager.Spawn(testutil.Context(t), SpawnOpts{
					ParentSessionID: parent.ID,
					AgentName:       "coder",
					TTL:             time.Minute,
				})
				if err != nil {
					return err
				}
				t.Cleanup(func() {
					_ = h.manager.Stop(testutil.Context(t), child.ID)
				})
				_, err = h.manager.Spawn(testutil.Context(t), SpawnOpts{
					ParentSessionID: child.ID,
					AgentName:       "coder",
					TTL:             time.Minute,
				})
				return err
			},
			wantErr: ErrSpawnLimitExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			parent := createSpawnParent(t, h, store.SessionPermissionPolicy{
				Tools: []string{"read"},
			}, store.SessionSpawnBudget{MaxChildren: 1, MaxDepth: 1})
			t.Cleanup(func() {
				_ = h.manager.Stop(testutil.Context(t), parent.ID)
			})

			err := tt.run(t, h, parent)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Spawn() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestManagerSpawnHooksCarryLineageAndCannotWidenPermissions(t *testing.T) {
	t.Parallel()

	t.Run("payloads", func(t *testing.T) {
		t.Parallel()

		hooks := &recordingSessionSpawnHooks{}
		h := newHarness(t, WithHookSet(HookSet{Spawn: hooks}))
		parent := createSpawnParent(t, h, store.SessionPermissionPolicy{
			Tools: []string{"read"},
		}, store.SessionSpawnBudget{MaxChildren: 2, MaxDepth: 1})
		t.Cleanup(func() {
			_ = h.manager.Stop(testutil.Context(t), parent.ID)
		})

		child, err := h.manager.Spawn(testutil.Context(t), SpawnOpts{
			ParentSessionID:  parent.ID,
			AgentName:        "coder",
			TTL:              time.Minute,
			AutoStopOnParent: true,
			PermissionPolicy: store.SessionPermissionPolicy{Tools: []string{"read"}},
		})
		if err != nil {
			t.Fatalf("Spawn() error = %v", err)
		}
		t.Cleanup(func() {
			_ = h.manager.Stop(testutil.Context(t), child.ID)
		})

		if len(hooks.preCreate) != 1 || len(hooks.created) != 1 {
			t.Fatalf("hook counts pre=%d created=%d, want 1 each", len(hooks.preCreate), len(hooks.created))
		}
		pre := hooks.preCreate[0]
		if pre.ParentSessionID != parent.ID ||
			pre.RootSessionID != parent.ID ||
			pre.SpawnDepth != 1 ||
			pre.ChildSessionID != "" ||
			len(pre.ChildPermissions.Tools) != 1 ||
			pre.ChildPermissions.Tools[0] != "read" {
			t.Fatalf("pre-create payload = %#v, want parent/root/depth and narrowed permissions", pre)
		}
		created := hooks.created[0]
		if created.ParentSessionID != parent.ID ||
			created.RootSessionID != parent.ID ||
			created.ChildSessionID != child.ID ||
			created.SpawnDepth != 1 ||
			created.AgentName != "coder" {
			t.Fatalf("created payload = %#v, want durable child lineage", created)
		}
	})

	t.Run("widening patch rejected", func(t *testing.T) {
		t.Parallel()

		hooks := &recordingSessionSpawnHooks{
			preCreatePatch: func(payload hookspkg.SpawnPreCreatePayload) hookspkg.SpawnPreCreatePayload {
				payload.ChildPermissions.Tools = []string{"shell"}
				return payload
			},
		}
		h := newHarness(t, WithHookSet(HookSet{Spawn: hooks}))
		parent := createSpawnParent(t, h, store.SessionPermissionPolicy{
			Tools: []string{"read"},
		}, store.SessionSpawnBudget{MaxChildren: 2, MaxDepth: 1})
		t.Cleanup(func() {
			_ = h.manager.Stop(testutil.Context(t), parent.ID)
		})

		_, err := h.manager.Spawn(testutil.Context(t), SpawnOpts{
			ParentSessionID:  parent.ID,
			AgentName:        "coder",
			TTL:              time.Minute,
			AutoStopOnParent: true,
			PermissionPolicy: store.SessionPermissionPolicy{Tools: []string{"read"}},
		})
		if !errors.Is(err, ErrSpawnPermissionDenied) {
			t.Fatalf("Spawn() error = %v, want %v", err, ErrSpawnPermissionDenied)
		}
	})
}

func createSpawnParent(
	t *testing.T,
	h *harness,
	policy store.SessionPermissionPolicy,
	budget store.SessionSpawnBudget,
) *Session {
	t.Helper()

	parent, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "parent",
		Workspace: h.workspaceID,
		Type:      SessionTypeUser,
		Lineage: &store.SessionLineage{
			SpawnBudget:      budget,
			PermissionPolicy: policy,
		},
	})
	if err != nil {
		t.Fatalf("Create(parent) error = %v", err)
	}
	return parent
}

type recordingSessionSpawnHooks struct {
	preCreate      []hookspkg.SpawnPreCreatePayload
	created        []hookspkg.SpawnCreatedPayload
	preCreatePatch func(hookspkg.SpawnPreCreatePayload) hookspkg.SpawnPreCreatePayload
}

func (h *recordingSessionSpawnHooks) DispatchSpawnPreCreate(
	_ context.Context,
	payload hookspkg.SpawnPreCreatePayload,
) (hookspkg.SpawnPreCreatePayload, error) {
	h.preCreate = append(h.preCreate, payload)
	if h.preCreatePatch != nil {
		return h.preCreatePatch(payload), nil
	}
	return payload, nil
}

func (h *recordingSessionSpawnHooks) DispatchSpawnCreated(
	_ context.Context,
	payload hookspkg.SpawnCreatedPayload,
) (hookspkg.SpawnCreatedPayload, error) {
	h.created = append(h.created, payload)
	return payload, nil
}

func (h *recordingSessionSpawnHooks) DispatchSpawnParentStopped(
	_ context.Context,
	payload hookspkg.SpawnParentStoppedPayload,
) (hookspkg.SpawnParentStoppedPayload, error) {
	return payload, nil
}

func (h *recordingSessionSpawnHooks) DispatchSpawnTTLExpired(
	_ context.Context,
	payload hookspkg.SpawnTTLExpiredPayload,
) (hookspkg.SpawnTTLExpiredPayload, error) {
	return payload, nil
}

func (h *recordingSessionSpawnHooks) DispatchSpawnReaped(
	_ context.Context,
	payload hookspkg.SpawnReapedPayload,
) (hookspkg.SpawnReapedPayload, error) {
	return payload, nil
}
