package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/memory"
	memcontract "github.com/compozy/agh/internal/memory/contract"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/testutil"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestDaemonMemoryProposalSinkTargetStore(t *testing.T) {
	t.Parallel()

	t.Run("Should normalize workspace-root candidates to the stable workspace identity", func(t *testing.T) {
		t.Parallel()

		baseDir := t.TempDir()
		workspaceRoot := filepath.Join(baseDir, "workspace")
		if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}
		identity, err := workspacepkg.EnsureIdentity(testutil.Context(t), workspaceRoot)
		if err != nil {
			t.Fatalf("EnsureIdentity() error = %v", err)
		}
		sink := daemonMemoryProposalSink{
			base: memory.NewStore(
				filepath.Join(baseDir, "global", "memory"),
				memory.WithCatalogDatabasePath(filepath.Join(baseDir, "agh.db")),
			),
		}
		candidate := memcontract.Candidate{
			WorkspaceID: "ws-registration",
			Scope:       memcontract.ScopeWorkspace,
			Frontmatter: memcontract.Header{
				Scope: memcontract.ScopeWorkspace,
				Type:  memcontract.TypeProject,
			},
			Metadata: map[string]string{
				"workspace_root": workspaceRoot,
			},
		}

		_, normalized, err := sink.targetStore(testutil.Context(t), candidate)
		if err != nil {
			t.Fatalf("targetStore() error = %v", err)
		}
		if normalized.WorkspaceID != identity.WorkspaceID {
			t.Fatalf(
				"candidate workspace id = %q, want stable identity %q",
				normalized.WorkspaceID,
				identity.WorkspaceID,
			)
		}
	})
}

func TestCollectMemoryExtractorOutput(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve streamed JSONL chunks without synthetic newlines", func(t *testing.T) {
		t.Parallel()

		events := make(chan acp.AgentEvent, 4)
		events <- acp.AgentEvent{Type: acp.EventTypeAgentMessage, Text: "```jsonl\n{\"type\":\"reference\",\"scope"}
		events <- acp.AgentEvent{
			Type: acp.EventTypeAgentMessage,
			Text: "\":\"workspace\",\"agent_tier\":\"workspace\",\"content\":\"Channel marketing already exists.\",\"evidence\":\"seq=52\"",
		}
		events <- acp.AgentEvent{
			Type: acp.EventTypeAgentMessage,
			Text: ",\"entity\":\"ws-test\",\"attribute\":\"network_channel_marketing_exists\"}\n```",
		}
		close(events)

		output, err := collectMemoryExtractorOutput(testutil.Context(t), events)
		if err != nil {
			t.Fatalf("collectMemoryExtractorOutput() error = %v", err)
		}
		if strings.Contains(output, "scope\n") {
			t.Fatalf("output = %q, want no synthetic newline inside streamed JSON key", output)
		}

		turn := memcontract.TurnRecord{
			SessionID:       "sess-parent",
			RootSessionID:   "sess-parent",
			AgentID:         "cto",
			ActorKind:       "agent_root",
			WorkspaceID:     "ws-test",
			SinceMessageSeq: 32,
			UntilMessageSeq: 52,
			Snapshot: memcontract.TranscriptSnapshot{
				Messages: []memcontract.TranscriptMessage{{
					Sequence: 52,
					Role:     "assistant",
					Content:  "Channel marketing exists.",
					At:       time.Date(2026, 5, 26, 21, 1, 52, 0, time.UTC),
				}},
			},
			Trigger: memcontract.TriggerPostMessage,
		}
		candidates, err := parseMemoryExtractorCandidates(
			output,
			turn,
			"/workspace/test",
			time.Date(2026, 5, 26, 21, 2, 0, 0, time.UTC),
		)
		if err != nil {
			t.Fatalf("parseMemoryExtractorCandidates() error = %v", err)
		}
		if len(candidates) != 1 {
			t.Fatalf("candidates = %#v, want one parsed candidate", candidates)
		}
		candidate := candidates[0]
		if candidate.Scope != memcontract.ScopeWorkspace || candidate.Frontmatter.Type != memcontract.TypeReference {
			t.Fatalf(
				"candidate scope/type = %s/%s, want workspace/reference",
				candidate.Scope,
				candidate.Frontmatter.Type,
			)
		}
		if !strings.Contains(candidate.Content, "Channel marketing already exists") {
			t.Fatalf("candidate content = %q, want streamed JSON content", candidate.Content)
		}
		if candidate.Metadata["workspace_root"] != "/workspace/test" {
			t.Fatalf("candidate metadata = %#v, want workspace_root", candidate.Metadata)
		}
	})
}

func TestForkedMemoryExtractor(t *testing.T) {
	t.Parallel()

	t.Run("Should pass the configured model to the extractor child spawn", func(t *testing.T) {
		t.Parallel()

		sessions := &recordingMemoryExtractorSessions{}
		extractor := &forkedMemoryExtractor{
			sessions:     sessions,
			defaultAgent: "memory-agent",
			model:        "claude-haiku-memory",
			deadline:     time.Second,
			now: func() time.Time {
				return time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
			},
		}
		turn := memcontract.TurnRecord{
			SessionID:       "sess-parent",
			RootSessionID:   "sess-parent",
			AgentID:         "",
			ActorKind:       "agent_root",
			WorkspaceID:     "ws-test",
			SinceMessageSeq: 1,
			UntilMessageSeq: 1,
			Snapshot: memcontract.TranscriptSnapshot{
				Messages: []memcontract.TranscriptMessage{{
					Sequence: 1,
					Role:     "assistant",
					Content:  "Pedro prefers concise updates.",
					At:       time.Date(2026, 5, 27, 9, 59, 0, 0, time.UTC),
				}},
			},
			Trigger: memcontract.TriggerPostMessage,
		}

		candidates, err := extractor.Extract(testutil.Context(t), turn)
		if err != nil {
			t.Fatalf("Extract() error = %v", err)
		}
		if len(candidates) != 0 {
			t.Fatalf("candidates = %#v, want none for empty child output", candidates)
		}
		if sessions.spawnOpts.AgentName != "memory-agent" {
			t.Fatalf("spawn agent = %q, want default memory agent", sessions.spawnOpts.AgentName)
		}
		if sessions.spawnOpts.Model != "claude-haiku-memory" {
			t.Fatalf("spawn model = %q, want configured extractor model", sessions.spawnOpts.Model)
		}
		if sessions.stoppedID != "sess-memory-child" {
			t.Fatalf("stopped child = %q, want spawned extractor child stopped", sessions.stoppedID)
		}
	})
}

type recordingMemoryExtractorSessions struct {
	spawnOpts session.SpawnOpts
	stoppedID string
}

func (s *recordingMemoryExtractorSessions) Spawn(
	_ context.Context,
	opts session.SpawnOpts,
) (*session.Session, error) {
	s.spawnOpts = opts
	return &session.Session{ID: "sess-memory-child"}, nil
}

func (s *recordingMemoryExtractorSessions) PromptSynthetic(
	_ context.Context,
	_ string,
	_ session.SyntheticPromptOpts,
) (<-chan acp.AgentEvent, error) {
	events := make(chan acp.AgentEvent)
	close(events)
	return events, nil
}

func (s *recordingMemoryExtractorSessions) StopWithCause(
	_ context.Context,
	id string,
	_ session.StopCause,
	_ string,
) error {
	s.stoppedID = id
	return nil
}
