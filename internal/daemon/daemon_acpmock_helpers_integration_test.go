//go:build integration && !windows

package daemon

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	e2etest "github.com/pedronauck/agh/internal/testutil/e2e"
	"github.com/pedronauck/agh/internal/transcript"
)

func mockFixturePath(t testing.TB, name string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "testutil", "acpmock", "testdata", name)
}

func createFixtureBackedSession(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	agentName string,
	name string,
) aghcontract.SessionPayload {
	t.Helper()

	session, err := harness.CreateSession(ctx, aghcontract.CreateSessionRequest{
		AgentName:     agentName,
		Name:          name,
		WorkspacePath: harness.WorkspaceRoot,
	})
	if err != nil {
		t.Fatalf("CreateSession(%q) error = %v", agentName, err)
	}
	waitForRuntimeCondition(t, "fixture-backed session visible", 5*time.Second, func() bool {
		current, err := harness.GetSession(ctx, session.ID)
		return err == nil && current.ID == session.ID
	})
	return session
}

func joinTranscriptContent(messages []transcript.UIMessage) string {
	return transcript.JoinUIMessageText(messages)
}
