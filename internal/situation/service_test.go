package situation

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/soul"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestRenderPromptPreservesSectionOrderAndOmitsUnavailableSections(t *testing.T) {
	t.Parallel()

	payload := contract.AgentContextPayload{
		Self: contract.AgentIdentityPayload{
			SessionID: "sess-1",
			AgentName: "coder",
			Provider:  "codex",
		},
		Workspace: contract.AgentWorkspacePayload{
			ID:      "ws-1",
			Name:    "agh",
			RootDir: "/work/agh",
		},
		Session: contract.AgentSessionPayload{
			ID:        "sess-1",
			Type:      session.SessionTypeUser,
			State:     session.StateActive,
			CreatedAt: fixedTime(),
			UpdatedAt: fixedTime(),
		},
		Capabilities: contract.AgentCapabilitySectionPayload{
			Section: contract.AgentContextSectionMetaPayload{Limit: 2},
		},
		Limits: contract.AgentLimitsPayload{
			MaxChildren:         5,
			MaxSpawnDepth:       1,
			MaxActiveTaskLeases: 1,
			ContextSectionLimit: 2,
		},
		Provenance: contract.AgentContextProvenancePayload{
			GeneratedAt: fixedTime(),
			Source:      ProvenanceSource,
		},
	}

	rendered, err := RenderPrompt(&payload)
	if err != nil {
		t.Fatalf("RenderPrompt() error = %v", err)
	}

	wantOrder := []string{
		`"self"`,
		`"workspace"`,
		`"session"`,
		`"capabilities"`,
		`"limits"`,
		`"provenance"`,
	}
	assertOrder(t, rendered, wantOrder)
	for _, omitted := range []string{`"task"`, `"coordination_channel"`, `"inbox_summary"`, `"peer_roster"`} {
		if strings.Contains(rendered, omitted) {
			t.Fatalf("RenderPrompt() included unavailable section %s: %s", omitted, rendered)
		}
	}
}

func TestContextForSessionBoundsListsAndIncludesTaskChannelProvenance(t *testing.T) {
	t.Parallel()

	taskRecord := taskpkg.Task{
		ID:          "task-1",
		Identifier:  "AUTO-1",
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: "ws-1",
		Title:       "Implement context",
		Status:      taskpkg.TaskStatusInProgress,
		Priority:    taskpkg.PriorityHigh,
		Owner:       &taskpkg.Ownership{Kind: taskpkg.OwnerKindAgentSession, Ref: "sess-1"},
		UpdatedAt:   fixedTime().Add(time.Minute),
	}
	run := taskpkg.Run{
		ID:             "run-1",
		TaskID:         "task-1",
		Status:         taskpkg.TaskRunStatusRunning,
		Attempt:        1,
		ClaimedBy:      &taskpkg.ActorIdentity{Kind: taskpkg.ActorKindAgentSession, Ref: "sess-1"},
		SessionID:      "sess-1",
		NetworkChannel: "coord-channel",
		Metadata:       jsonRaw(t, `{"coordination_channel_id":"coord-1","workflow_id":"wf-1"}`),
		QueuedAt:       fixedTime(),
		StartedAt:      fixedTime().Add(time.Minute),
	}
	displayName := "Reviewer"
	service := NewService(Deps{
		Now:          fixedNow,
		SectionLimit: 2,
		WorkspaceResolver: workspaceResolverFunc(func(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
			return workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{ID: "ws-1", Name: "AGH", RootDir: "/work/agh"},
				Config:    aghconfig.Config{Defaults: aghconfig.DefaultsConfig{Provider: "codex"}},
			}, nil
		}),
		AgentResolver: agentResolverFunc(func(string, *workspacepkg.ResolvedWorkspace) (aghconfig.AgentDef, error) {
			return aghconfig.AgentDef{
				Name:     "coder",
				Provider: "codex",
				Model:    "gpt-test",
				Capabilities: &aghconfig.CapabilityCatalog{Capabilities: []aghconfig.CapabilityDef{
					{ID: "build", Summary: "Build code"},
					{ID: "review", Summary: "Review code"},
				}},
			}, nil
		}),
		SkillRegistry: skillRegistryFunc(
			func(context.Context, *workspacepkg.ResolvedWorkspace) ([]*skillspkg.Skill, error) {
				return []*skillspkg.Skill{
					{Meta: skillspkg.SkillMeta{Name: "alpha", Description: "Alpha skill"}, Enabled: true},
					{Meta: skillspkg.SkillMeta{Name: "beta", Description: "Beta skill"}, Enabled: true},
				}, nil
			},
		),
		TaskStore: taskStoreStub{
			tasks: map[string]taskpkg.Task{"task-1": taskRecord},
			runs:  []taskpkg.Run{run},
		},
		Network: networkStub{
			envelopes: []network.Envelope{
				coordinationEnvelope(t, "msg-3", "coord-channel", "third", fixedTime().Add(3*time.Minute)),
				coordinationEnvelope(t, "msg-2", "coord-channel", "second", fixedTime().Add(2*time.Minute)),
				coordinationEnvelope(t, "msg-1", "coord-channel", "first", fixedTime().Add(time.Minute)),
			},
			peers: []network.PeerInfo{
				{
					PeerID:  "peer-c",
					Channel: "coord-channel",
					PeerCard: network.PeerCard{
						PeerID:       "peer-c",
						Capabilities: []string{"test"},
					},
				},
				{
					PeerID:  "peer-a",
					Channel: "coord-channel",
					PeerCard: network.PeerCard{
						PeerID:       "peer-a",
						DisplayName:  &displayName,
						Capabilities: []string{"review"},
					},
				},
				{
					PeerID:  "peer-b",
					Channel: "coord-channel",
					PeerCard: network.PeerCard{
						PeerID:       "peer-b",
						Capabilities: []string{"build"},
					},
				},
			},
		},
		CoordinatorConfig: coordinatorResolverFunc(
			func(context.Context, string) (aghconfig.CoordinatorConfig, error) {
				return aghconfig.CoordinatorConfig{MaxChildren: 3}, nil
			},
		),
	})

	payload, err := service.ContextForSession(context.Background(), &session.Info{
		ID:          "sess-1",
		Name:        "Coding Session",
		AgentName:   "coder",
		Provider:    "codex",
		WorkspaceID: "ws-1",
		Workspace:   "/work/agh",
		Type:        session.SessionTypeUser,
		Lineage: &store.SessionLineage{
			ParentSessionID: "sess-parent",
			RootSessionID:   "sess-root",
			SpawnDepth:      1,
			SpawnRole:       "worker",
		},
		State:     session.StateActive,
		CreatedAt: fixedTime(),
		UpdatedAt: fixedTime(),
	})
	if err != nil {
		t.Fatalf("ContextForSession() error = %v", err)
	}

	if got, want := payload.Self.Model, "gpt-test"; got != want {
		t.Fatalf("Self.Model = %q, want %q", got, want)
	}
	if payload.Session.Lineage == nil ||
		payload.Session.Lineage.ParentSessionID != "sess-parent" ||
		payload.Session.Lineage.RootSessionID != "sess-root" ||
		payload.Session.Lineage.SpawnDepth != 1 {
		t.Fatalf("Session.Lineage = %#v, want spawned lineage projection", payload.Session.Lineage)
	}
	if payload.Task.Task == nil || payload.Task.Task.ID != "task-1" {
		t.Fatalf("Task section = %#v, want task-1", payload.Task)
	}
	if payload.Task.Lease == nil || payload.Task.Lease.CoordinationChannelID != "coord-1" {
		t.Fatalf("Task lease = %#v, want coord-1", payload.Task.Lease)
	}
	if !payload.CoordinationChannel.Available ||
		payload.CoordinationChannel.Channel == nil ||
		payload.CoordinationChannel.Channel.WorkflowID != "wf-1" {
		t.Fatalf("CoordinationChannel = %#v, want workflow-bound channel", payload.CoordinationChannel)
	}
	if got := payload.InboxSummary.Section; got.Limit != 2 || got.Returned != 2 || !got.Truncated {
		t.Fatalf("Inbox section = %#v, want truncated limit 2", got)
	}
	if got, want := payload.InboxSummary.UnreadCount, 3; got != want {
		t.Fatalf("UnreadCount = %d, want %d", got, want)
	}
	if got := payload.PeerRoster.Section; got.Limit != 2 || got.Returned != 2 || !got.Truncated {
		t.Fatalf("Peer section = %#v, want truncated limit 2", got)
	}
	if got := payload.Capabilities.Section; got.Limit != 2 || got.Returned != 2 || !got.Truncated {
		t.Fatalf("Capability section = %#v, want truncated limit 2", got)
	}
	if got, want := payload.Limits.MaxChildren, 3; got != want {
		t.Fatalf("Limits.MaxChildren = %d, want %d", got, want)
	}
	if got, want := payload.Provenance.Source, ProvenanceSource; got != want {
		t.Fatalf("Provenance.Source = %q, want %q", got, want)
	}

	rendered, err := RenderPrompt(&payload)
	if err != nil {
		t.Fatalf("RenderPrompt(context) error = %v", err)
	}
	assertOrder(t, rendered, []string{
		`"self"`,
		`"workspace"`,
		`"session"`,
		`"task"`,
		`"coordination_channel"`,
		`"inbox_summary"`,
		`"peer_roster"`,
		`"capabilities"`,
		`"limits"`,
		`"provenance"`,
	})
	if strings.Contains(rendered, "claim_token") {
		t.Fatalf("RenderPrompt() leaked raw claim token field: %s", rendered)
	}
}

func TestContextForSessionIncludesCompactSoulProjection(t *testing.T) {
	t.Parallel()

	t.Run("Should expose compact soul before task without full body", func(t *testing.T) {
		t.Parallel()

		snapshot := testSituationSoulSnapshot(t, "body-secret-marker")
		taskRecord := taskpkg.Task{
			ID:          "task-soul",
			Identifier:  "SOUL-1",
			Scope:       taskpkg.ScopeWorkspace,
			WorkspaceID: "ws-1",
			Title:       "Use soul context",
			Status:      taskpkg.TaskStatusInProgress,
			Priority:    taskpkg.PriorityMedium,
			Owner:       &taskpkg.Ownership{Kind: taskpkg.OwnerKindAgentSession, Ref: "sess-1"},
			UpdatedAt:   fixedTime().Add(time.Minute),
		}
		run := taskpkg.Run{
			ID:        "run-soul",
			TaskID:    taskRecord.ID,
			Status:    taskpkg.TaskRunStatusRunning,
			SessionID: "sess-1",
			QueuedAt:  fixedTime(),
			StartedAt: fixedTime().Add(time.Minute),
		}
		service := NewService(Deps{
			Now:          fixedNow,
			SectionLimit: 3,
			WorkspaceResolver: workspaceResolverFunc(
				func(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
					return workspacepkg.ResolvedWorkspace{
						Workspace: workspacepkg.Workspace{ID: "ws-1", Name: "AGH", RootDir: "/work/agh"},
						Config:    aghconfig.Config{Defaults: aghconfig.DefaultsConfig{Provider: "codex"}},
					}, nil
				},
			),
			AgentResolver: agentResolverFunc(func(string, *workspacepkg.ResolvedWorkspace) (aghconfig.AgentDef, error) {
				return aghconfig.AgentDef{Name: "coder", Provider: "codex", Model: "gpt-test"}, nil
			}),
			TaskStore: taskStoreStub{
				tasks: map[string]taskpkg.Task{taskRecord.ID: taskRecord},
				runs:  []taskpkg.Run{run},
			},
			SoulSnapshots: soulSnapshotStoreStub{
				snapshots: map[string]soul.Snapshot{snapshot.ID: snapshot},
			},
		})

		payload, err := service.ContextForSession(context.Background(), &session.Info{
			ID:             "sess-1",
			AgentName:      "coder",
			Provider:       "codex",
			WorkspaceID:    "ws-1",
			Workspace:      "/work/agh",
			Type:           session.SessionTypeUser,
			State:          session.StateActive,
			SoulSnapshotID: snapshot.ID,
			SoulDigest:     snapshot.Digest,
			CreatedAt:      fixedTime(),
			UpdatedAt:      fixedTime(),
		})
		if err != nil {
			t.Fatalf("ContextForSession() error = %v", err)
		}

		if !payload.Soul.Present || !payload.Soul.Active || !payload.Soul.Valid {
			t.Fatalf("Soul flags = %#v, want present active valid", payload.Soul)
		}
		if got, want := payload.Soul.SnapshotID, snapshot.ID; got != want {
			t.Fatalf("Soul.SnapshotID = %q, want %q", got, want)
		}
		if got, want := payload.Soul.Role, "Reviewer"; got != want {
			t.Fatalf("Soul.Role = %q, want %q", got, want)
		}
		if got, want := payload.Soul.Tone, []string{"direct"}; !slices.Equal(got, want) {
			t.Fatalf("Soul.Tone = %#v, want %#v", got, want)
		}
		if got, want := payload.Soul.Principles, []string{"protect correctness"}; !slices.Equal(got, want) {
			t.Fatalf("Soul.Principles = %#v, want %#v", got, want)
		}
		encodedSoul, err := json.Marshal(payload.Soul)
		if err != nil {
			t.Fatalf("json.Marshal(Soul) error = %v", err)
		}
		if strings.Contains(string(encodedSoul), "body-secret-marker") ||
			strings.Contains(string(encodedSoul), `"body":`) {
			t.Fatalf("Soul compact payload leaked full body data: %s", encodedSoul)
		}

		rendered, err := RenderPrompt(&payload)
		if err != nil {
			t.Fatalf("RenderPrompt(context) error = %v", err)
		}
		assertOrder(t, rendered, []string{
			`"self"`,
			`"workspace"`,
			`"session"`,
			`"soul"`,
			`"task"`,
			`"capabilities"`,
			`"limits"`,
			`"provenance"`,
		})
		if strings.Contains(rendered, "body-secret-marker") || strings.Contains(rendered, `"body":`) {
			t.Fatalf("RenderPrompt() leaked full soul body: %s", rendered)
		}
	})
}

func TestContextForSessionMissingOptionalServicesOmitsUnavailableSections(t *testing.T) {
	t.Parallel()

	service := NewService(Deps{
		Now:          fixedNow,
		SectionLimit: 3,
	})

	payload, err := service.ContextForSession(context.Background(), &session.Info{
		ID:          "sess-1",
		AgentName:   "coder",
		Provider:    "codex",
		WorkspaceID: "ws-1",
		Workspace:   "/work/agh",
		Type:        session.SessionTypeUser,
		State:       session.StateActive,
		CreatedAt:   fixedTime(),
		UpdatedAt:   fixedTime(),
	})
	if err != nil {
		t.Fatalf("ContextForSession() error = %v", err)
	}
	if payload.Task.Available {
		t.Fatalf("Task.Available = true, want false without task store")
	}
	if payload.CoordinationChannel.Available {
		t.Fatalf("CoordinationChannel.Available = true, want false without task store")
	}
	if payload.InboxSummary.Section.Limit != 0 {
		t.Fatalf("Inbox section = %#v, want omitted without network", payload.InboxSummary.Section)
	}
	if payload.PeerRoster.Section.Limit != 0 {
		t.Fatalf("Peer section = %#v, want omitted without network", payload.PeerRoster.Section)
	}

	rendered, err := RenderPrompt(&payload)
	if err != nil {
		t.Fatalf("RenderPrompt() error = %v", err)
	}
	for _, omitted := range []string{`"task"`, `"coordination_channel"`, `"inbox_summary"`, `"peer_roster"`} {
		if strings.Contains(rendered, omitted) {
			t.Fatalf("RenderPrompt() included unavailable section %s: %s", omitted, rendered)
		}
	}
}

func TestPromptStartupSectionIncludesStartupIdentity(t *testing.T) {
	t.Parallel()

	service := NewService(Deps{Now: fixedNow, SectionLimit: 4})
	workspace := &workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{ID: "ws-1", Name: "AGH", RootDir: "/work/agh"},
		Config:    aghconfig.Config{Defaults: aghconfig.DefaultsConfig{Provider: "codex"}},
	}

	rendered, err := service.PromptStartupSection(
		context.Background(),
		session.StartupPromptContext{
			SessionID:   "sess-start",
			SessionName: "Startup",
			AgentName:   "coder",
			Provider:    "codex",
			WorkspaceID: "ws-1",
			Workspace:   "/work/agh",
			SessionType: session.SessionTypeUser,
			CreatedAt:   fixedTime(),
			UpdatedAt:   fixedTime(),
		},
		aghconfig.AgentDef{Name: "coder", Provider: "codex", Model: "gpt-test"},
		workspace,
	)
	if err != nil {
		t.Fatalf("PromptStartupSection() error = %v", err)
	}
	for _, want := range []string{
		`"session_id":"sess-start"`,
		`"agent_name":"coder"`,
		`"model":"gpt-test"`,
		`"workspace"`,
		`"provenance"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("PromptStartupSection() = %s, want substring %q", rendered, want)
		}
	}
}

func TestAugmentPrefixesFreshSituationWithoutRewritingMessage(t *testing.T) {
	t.Parallel()

	service := NewService(Deps{Now: fixedNow, SectionLimit: 2})
	augmented, err := service.Augment(
		context.Background(),
		&session.Session{
			ID:        "sess-1",
			AgentName: "coder",
			Provider:  "codex",
			Type:      session.SessionTypeUser,
			State:     session.StateActive,
			CreatedAt: fixedTime(),
			UpdatedAt: fixedTime(),
		},
		"original prompt",
	)
	if err != nil {
		t.Fatalf("Augment() error = %v", err)
	}
	if !strings.HasPrefix(augmented, promptContextOpen) {
		t.Fatalf("Augment() = %q, want situation context prefix", augmented)
	}
	if !strings.HasSuffix(augmented, "original prompt") {
		t.Fatalf("Augment() = %q, want original message suffix", augmented)
	}
	if got := strings.Count(augmented, promptContextOpen); got != 1 {
		t.Fatalf("situation context occurrences = %d, want 1", got)
	}
}

func TestPromptSectionAndHelperBranches(t *testing.T) {
	t.Parallel()

	service := NewService(Deps{Now: fixedNow})
	rendered, err := service.PromptSection(context.Background(), &workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{ID: "ws-1", RootDir: "/work/agh", Name: "AGH"},
	})
	if err != nil {
		t.Fatalf("PromptSection() error = %v", err)
	}
	if !strings.Contains(rendered, `"workspace"`) || !strings.Contains(rendered, `"limits"`) {
		t.Fatalf("PromptSection() = %s, want workspace and limits sections", rendered)
	}

	if got, err := service.Augment(context.Background(), nil, "message"); err != nil || got != "message" {
		t.Fatalf("Augment(nil session) = %q, %v; want original message", got, err)
	}
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.ContextForSession(canceledCtx, &session.Info{ID: "sess-1"}); err == nil {
		t.Fatal("ContextForSession(canceled ctx) error = nil, want validation error")
	}
	if !isContextError(context.Canceled) {
		t.Fatal("isContextError(context.Canceled) = false, want true")
	}
	if !isContextError(context.DeadlineExceeded) {
		t.Fatal("isContextError(context.DeadlineExceeded) = false, want true")
	}
}

func TestSelectionPreviewAndBoundingHelpers(t *testing.T) {
	t.Parallel()

	queuedAt := fixedTime()
	runs := []taskpkg.Run{
		{ID: "done", Status: taskpkg.TaskRunStatusCompleted, QueuedAt: queuedAt.Add(4 * time.Minute)},
		{ID: "queued", Status: taskpkg.TaskRunStatusQueued, QueuedAt: queuedAt.Add(3 * time.Minute)},
		{ID: "claimed", Status: taskpkg.TaskRunStatusClaimed, QueuedAt: queuedAt.Add(2 * time.Minute)},
		{ID: "starting", Status: taskpkg.TaskRunStatusStarting, QueuedAt: queuedAt.Add(time.Minute)},
		{ID: "running", Status: taskpkg.TaskRunStatusRunning, StartedAt: queuedAt.Add(5 * time.Minute)},
	}
	selected, ok := selectActiveRun(runs)
	if !ok || selected.ID != "running" {
		t.Fatalf("selectActiveRun() = %#v, %v; want running", selected, ok)
	}
	if _, ok := selectActiveRun([]taskpkg.Run{{ID: "done", Status: taskpkg.TaskRunStatusCompleted}}); ok {
		t.Fatal("selectActiveRun(terminal) ok = true, want false")
	}
	if got := activeRunRank(taskpkg.TaskRunStatusCanceled); got != -1 {
		t.Fatalf("activeRunRank(canceled) = %d, want -1", got)
	}
	if got := runActivityTime(taskpkg.Run{QueuedAt: queuedAt, StartedAt: queuedAt.Add(time.Minute)}); !got.Equal(
		queuedAt.Add(time.Minute),
	) {
		t.Fatalf("runActivityTime() = %s, want latest start", got)
	}

	direct := envelopeWithBody(t, network.KindSay, network.SayBody{Text: "direct message"})
	if got, want := envelopePreview(direct), "direct message"; got != want {
		t.Fatalf("envelopePreview(direct) = %q, want %q", got, want)
	}
	trace := envelopeWithBody(t, network.KindTrace, network.TraceBody{
		State:   network.WorkStateWorking,
		Message: "trace message",
	})
	if got, want := envelopePreview(trace), "trace message"; got != want {
		t.Fatalf("envelopePreview(trace) = %q, want %q", got, want)
	}
	capability := envelopeWithBody(t, network.KindCapability, network.CapabilityBody{
		Capability: capabilityPayload(t, "cap", "capability summary", "done"),
	})
	if got, want := envelopePreview(capability), "capability summary"; got != want {
		t.Fatalf("envelopePreview(capability) = %q, want %q", got, want)
	}
	detail := "receipt detail"
	receipt := envelopeWithBody(t, network.KindReceipt, network.ReceiptBody{
		ForID:  "msg-1",
		Status: network.ReceiptStatusAccepted,
		Detail: &detail,
	})
	if got, want := envelopePreview(receipt), "receipt detail"; got != want {
		t.Fatalf("envelopePreview(receipt) = %q, want %q", got, want)
	}
	greet := envelopeWithBody(t, network.KindGreet, network.GreetBody{
		PeerCard: network.PeerCard{
			PeerID:              "peer",
			ProfilesSupported:   []string{network.ProtocolV0},
			Capabilities:        []string{},
			ArtifactsSupported:  []string{},
			TrustModesSupported: []string{},
		},
		Summary: "hello peer",
	})
	if got, want := envelopePreview(greet), "hello peer"; got != want {
		t.Fatalf("envelopePreview(greet) = %q, want %q", got, want)
	}
	if got := envelopePreview(network.Envelope{Kind: network.KindSay, Body: json.RawMessage(`{`)}); got != "" {
		t.Fatalf("envelopePreview(invalid) = %q, want empty", got)
	}
	if got := envelopeTimestamp(network.Envelope{}); !got.IsZero() {
		t.Fatalf("envelopeTimestamp(zero) = %s, want zero", got)
	}

	long := strings.Repeat("x", inboxPreviewLimit+10)
	if got := truncateRunes(long, 12); utf8.RuneCountInString(got) != 12 || !strings.HasSuffix(got, "...") {
		t.Fatalf("truncateRunes() = %q, want 12-rune ellipsized value", got)
	}
	if got, want := truncateRunes("abcdef", 2), ".."; got != want {
		t.Fatalf("truncateRunes(short limit) = %q, want %q", got, want)
	}
}

func TestCoordinationMetadataAndPeerHelpers(t *testing.T) {
	t.Parallel()

	rawMetadata := jsonRaw(t, `{
		"task_id":"task-1",
		"run_id":"run-1",
		"coordination_channel_id":"coord-1",
		"message_kind":"status",
		"correlation_id":"corr-1"
	}`)
	direct := network.Envelope{
		Ext: network.ExtensionMap{
			"task_id":                 jsonRaw(t, `"task-1"`),
			"run_id":                  jsonRaw(t, `"run-1"`),
			"coordination_channel_id": jsonRaw(t, `"coord-1"`),
			"message_kind":            jsonRaw(t, `"status"`),
			"correlation_id":          jsonRaw(t, `"corr-1"`),
		},
	}
	if metadata, ok := coordinationMetadataFromEnvelope(direct); !ok || metadata.TaskID != "task-1" {
		t.Fatalf("coordinationMetadataFromEnvelope(direct) = %#v, %v; want task metadata", metadata, ok)
	}
	nested := network.Envelope{Ext: network.ExtensionMap{"metadata": rawMetadata}}
	if metadata, ok := coordinationMetadataFromEnvelope(nested); !ok || metadata.CorrelationID != "corr-1" {
		t.Fatalf("coordinationMetadataFromEnvelope(nested) = %#v, %v; want nested metadata", metadata, ok)
	}
	if _, ok := coordinationMetadataFromEnvelope(network.Envelope{}); ok {
		t.Fatal("coordinationMetadataFromEnvelope(empty) ok = true, want false")
	}
	if _, ok := decodeCoordinationMetadata(json.RawMessage(`{"claim_token":"raw"}`)); ok {
		t.Fatal("decodeCoordinationMetadata(raw claim token) ok = true, want false")
	}

	selfSession := "sess-self"
	display := "Peer A"
	roster := peerRoster([]network.PeerInfo{
		{
			SessionID: &selfSession,
			PeerID:    "self",
			Channel:   "coord",
		},
		{
			PeerID:  "peer-a",
			Channel: "coord",
			PeerCard: network.PeerCard{
				PeerID:       "peer-a",
				DisplayName:  &display,
				Capabilities: []string{"review", "review"},
			},
			CapabilityCatalogKnown: true,
			CapabilityCatalog: []session.NetworkPeerCapability{
				{ID: "build"},
				{ID: "review"},
			},
		},
	}, selfSession, 4)
	if got, want := len(roster.Peers), 1; got != want {
		t.Fatalf("peerRoster() peers = %d, want %d", got, want)
	}
	if got, want := roster.Peers[0].Capabilities, []string{"build", "review"}; !slices.Equal(got, want) {
		t.Fatalf("peer capabilities = %#v, want %#v", got, want)
	}
}

func assertOrder(t *testing.T, value string, tokens []string) {
	t.Helper()

	last := -1
	for _, token := range tokens {
		index := strings.Index(value, token)
		if index < 0 {
			t.Fatalf("missing token %q in %s", token, value)
		}
		if index <= last {
			t.Fatalf("token %q appeared out of order in %s", token, value)
		}
		last = index
	}
}

func fixedNow() time.Time {
	return fixedTime()
}

func fixedTime() time.Time {
	return time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
}

func jsonRaw(t *testing.T, value string) json.RawMessage {
	t.Helper()

	if !json.Valid([]byte(value)) {
		t.Fatalf("invalid JSON fixture: %s", value)
	}
	return json.RawMessage(value)
}

func coordinationEnvelope(
	t *testing.T,
	id string,
	channel string,
	text string,
	timestamp time.Time,
) network.Envelope {
	t.Helper()

	body, err := json.Marshal(network.SayBody{Text: text})
	if err != nil {
		t.Fatalf("marshal say body: %v", err)
	}
	metadata, err := json.Marshal(contract.CoordinationMessageMetadataPayload{
		TaskID:                "task-1",
		RunID:                 "run-1",
		CoordinationChannelID: "coord-1",
		MessageKind:           contract.CoordinationMessageStatus,
		CorrelationID:         id + "-corr",
	})
	if err != nil {
		t.Fatalf("marshal coordination metadata: %v", err)
	}
	return network.Envelope{
		Protocol: network.ProtocolV0,
		ID:       id,
		Kind:     network.KindSay,
		Channel:  channel,
		From:     "peer-a",
		TS:       timestamp.Unix(),
		Body:     body,
		Ext:      network.ExtensionMap{"coordination": metadata},
	}
}

func envelopeWithBody(t *testing.T, kind network.Kind, bodyValue network.Body) network.Envelope {
	t.Helper()

	body, err := json.Marshal(bodyValue)
	if err != nil {
		t.Fatalf("marshal %s body: %v", kind, err)
	}
	return network.Envelope{
		Protocol: network.ProtocolV0,
		ID:       "preview",
		Kind:     kind,
		Channel:  "coord",
		From:     "peer",
		TS:       fixedTime().Unix(),
		Body:     body,
	}
}

func capabilityPayload(
	t *testing.T,
	id string,
	summary string,
	outcome string,
) network.CapabilityEnvelopePayload {
	t.Helper()

	digest, err := aghconfig.CanonicalCapabilityDigest(aghconfig.CapabilityDef{
		ID:      id,
		Summary: summary,
		Outcome: outcome,
	})
	if err != nil {
		t.Fatalf("CanonicalCapabilityDigest() error = %v", err)
	}
	return network.CapabilityEnvelopePayload{
		ID:      id,
		Summary: summary,
		Outcome: outcome,
		Digest:  digest,
	}
}

type workspaceResolverFunc func(context.Context, string) (workspacepkg.ResolvedWorkspace, error)

func (fn workspaceResolverFunc) Resolve(
	ctx context.Context,
	idOrPath string,
) (workspacepkg.ResolvedWorkspace, error) {
	return fn(ctx, idOrPath)
}

type agentResolverFunc func(string, *workspacepkg.ResolvedWorkspace) (aghconfig.AgentDef, error)

func (fn agentResolverFunc) ResolveAgent(
	name string,
	workspace *workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, error) {
	return fn(name, workspace)
}

type skillRegistryFunc func(context.Context, *workspacepkg.ResolvedWorkspace) ([]*skillspkg.Skill, error)

func (fn skillRegistryFunc) ForWorkspace(
	ctx context.Context,
	workspace *workspacepkg.ResolvedWorkspace,
) ([]*skillspkg.Skill, error) {
	return fn(ctx, workspace)
}

func (fn skillRegistryFunc) ForAgent(
	ctx context.Context,
	workspace *workspacepkg.ResolvedWorkspace,
	_ string,
) ([]*skillspkg.Skill, error) {
	return fn(ctx, workspace)
}

type coordinatorResolverFunc func(context.Context, string) (aghconfig.CoordinatorConfig, error)

func (fn coordinatorResolverFunc) ResolveCoordinatorConfig(
	ctx context.Context,
	workspaceID string,
) (aghconfig.CoordinatorConfig, error) {
	return fn(ctx, workspaceID)
}

type taskStoreStub struct {
	tasks map[string]taskpkg.Task
	runs  []taskpkg.Run
}

func (s taskStoreStub) GetTask(_ context.Context, id string) (taskpkg.Task, error) {
	taskRecord, ok := s.tasks[id]
	if !ok {
		return taskpkg.Task{}, errors.New("missing task")
	}
	return taskRecord, nil
}

func (s taskStoreStub) ListTaskRuns(_ context.Context, query taskpkg.RunQuery) ([]taskpkg.Run, error) {
	runs := make([]taskpkg.Run, 0, len(s.runs))
	for _, run := range s.runs {
		if strings.TrimSpace(query.SessionID) != "" &&
			strings.TrimSpace(run.SessionID) != strings.TrimSpace(query.SessionID) {
			continue
		}
		runs = append(runs, run)
	}
	return runs, nil
}

type soulSnapshotStoreStub struct {
	snapshots map[string]soul.Snapshot
}

func (s soulSnapshotStoreStub) GetSoulSnapshot(_ context.Context, id string) (soul.Snapshot, error) {
	snapshot, ok := s.snapshots[strings.TrimSpace(id)]
	if !ok {
		return soul.Snapshot{}, soul.ErrSnapshotNotFound
	}
	return snapshot, nil
}

type networkStub struct {
	envelopes []network.Envelope
	peers     []network.PeerInfo
}

func (s networkStub) Inbox(_ context.Context, _ string) ([]network.Envelope, error) {
	return slices.Clone(s.envelopes), nil
}

func (s networkStub) ListPeers(_ context.Context, _ string) ([]network.PeerInfo, error) {
	return slices.Clone(s.peers), nil
}

func testSituationSoulSnapshot(t *testing.T, body string) soul.Snapshot {
	t.Helper()

	cfg := aghconfig.DefaultSoulConfig()
	resolved, err := soul.Parse(context.Background(), soul.ParseRequest{
		SourcePath:    "/work/agh/.agh/agents/coder/SOUL.md",
		WorkspaceRoot: "/work/agh",
		Content: []byte(strings.Join([]string{
			"---",
			"role: Reviewer",
			"tone:",
			"  - direct",
			"principles:",
			"  - protect correctness",
			"---",
			body,
		}, "\n")),
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("soul.Parse() error = %v", err)
	}
	provenance, err := soul.NewConfigProvenance(cfg, "test")
	if err != nil {
		t.Fatalf("NewConfigProvenance() error = %v", err)
	}
	snapshot, err := soul.SnapshotFromResolved(
		"soul-situation",
		"ws-1",
		"coder",
		&resolved,
		provenance,
		fixedTime(),
	)
	if err != nil {
		t.Fatalf("SnapshotFromResolved() error = %v", err)
	}
	return snapshot
}
