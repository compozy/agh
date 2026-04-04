package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestOpenGlobalDBCreatesSchemaAndEnablesWAL(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(t, globalDB.db, "sessions", "event_summaries", "token_stats", "permission_log")
	assertJournalModeWAL(t, globalDB.db)
	assertSynchronousNormal(t, globalDB.db)
}

func TestGlobalDBRegisterUpdateAndListSessions(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	createdAt := time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC)
	session := SessionInfo{
		ID:          "sess-global",
		Name:        "Alpha",
		AgentName:   "coder",
		Workspace:   "/tmp/workspace",
		SessionType: "dream",
		State:       "active",
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}

	if err := globalDB.RegisterSession(testContext(t), session); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	acpSessionID := "acp-123"
	if err := globalDB.UpdateSessionState(testContext(t), SessionStateUpdate{
		ID:           session.ID,
		State:        "stopped",
		ACPSessionID: &acpSessionID,
		UpdatedAt:    createdAt.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("UpdateSessionState() error = %v", err)
	}

	sessions, err := globalDB.ListSessions(testContext(t), SessionListQuery{State: "stopped"})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].State != "stopped" {
		t.Fatalf("sessions[0].State = %q, want stopped", sessions[0].State)
	}
	if sessions[0].SessionType != "dream" {
		t.Fatalf("sessions[0].SessionType = %q, want dream", sessions[0].SessionType)
	}
	if sessions[0].ACPSessionID == nil || *sessions[0].ACPSessionID != "acp-123" {
		t.Fatalf("sessions[0].ACPSessionID = %#v, want acp-123", sessions[0].ACPSessionID)
	}
}

func TestGlobalDBRegisterSessionDefaultsTypeToUser(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	session := SessionInfo{
		ID:        "sess-default-type",
		AgentName: "coder",
		Workspace: "/tmp/workspace",
		State:     "active",
		CreatedAt: time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
	}

	if err := globalDB.RegisterSession(testContext(t), session); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	sessions, err := globalDB.ListSessions(testContext(t), SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if got, want := sessions[0].SessionType, defaultSessionType; got != want {
		t.Fatalf("sessions[0].SessionType = %q, want %q", got, want)
	}
}

func TestGlobalDBWriteEventSummary(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-summary")

	if err := globalDB.WriteEventSummary(testContext(t), EventSummary{
		SessionID: "sess-summary",
		Type:      "agent_message",
		AgentName: "coder",
		Summary:   "assistant replied",
		Timestamp: time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("WriteEventSummary() error = %v", err)
	}

	summaries, err := globalDB.ListEventSummaries(testContext(t), EventSummaryQuery{SessionID: "sess-summary"})
	if err != nil {
		t.Fatalf("ListEventSummaries() error = %v", err)
	}
	if got, want := len(summaries), 1; got != want {
		t.Fatalf("len(summaries) = %d, want %d", got, want)
	}
	if summaries[0].Summary != "assistant replied" {
		t.Fatalf("summaries[0].Summary = %q, want %q", summaries[0].Summary, "assistant replied")
	}
}

func TestGlobalDBUpdateTokenStatsAggregation(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-stats")

	currency := "USD"
	inputA := int64(10)
	outputA := int64(20)
	totalA := int64(30)
	costA := 1.25
	if err := globalDB.UpdateTokenStats(testContext(t), TokenStatsUpdate{
		SessionID:    "sess-stats",
		AgentName:    "coder",
		InputTokens:  &inputA,
		OutputTokens: &outputA,
		TotalTokens:  &totalA,
		CostAmount:   &costA,
		CostCurrency: &currency,
		Turns:        1,
	}); err != nil {
		t.Fatalf("UpdateTokenStats() error = %v", err)
	}

	outputB := int64(5)
	totalB := int64(5)
	costB := 0.75
	if err := globalDB.UpdateTokenStats(testContext(t), TokenStatsUpdate{
		SessionID:    "sess-stats",
		AgentName:    "coder",
		OutputTokens: &outputB,
		TotalTokens:  &totalB,
		CostAmount:   &costB,
		CostCurrency: &currency,
		Turns:        1,
	}); err != nil {
		t.Fatalf("UpdateTokenStats() error = %v", err)
	}

	stats, err := globalDB.ListTokenStats(testContext(t), TokenStatsQuery{SessionID: "sess-stats"})
	if err != nil {
		t.Fatalf("ListTokenStats() error = %v", err)
	}
	if got, want := len(stats), 1; got != want {
		t.Fatalf("len(stats) = %d, want %d", got, want)
	}
	if stats[0].InputTokens == nil || *stats[0].InputTokens != 10 {
		t.Fatalf("InputTokens = %#v, want 10", stats[0].InputTokens)
	}
	if stats[0].OutputTokens == nil || *stats[0].OutputTokens != 25 {
		t.Fatalf("OutputTokens = %#v, want 25", stats[0].OutputTokens)
	}
	if stats[0].TotalTokens == nil || *stats[0].TotalTokens != 35 {
		t.Fatalf("TotalTokens = %#v, want 35", stats[0].TotalTokens)
	}
	if stats[0].TotalCost == nil || *stats[0].TotalCost != 2.0 {
		t.Fatalf("TotalCost = %#v, want 2.0", stats[0].TotalCost)
	}
	if stats[0].CostCurrency == nil || *stats[0].CostCurrency != "USD" {
		t.Fatalf("CostCurrency = %#v, want USD", stats[0].CostCurrency)
	}
	if stats[0].TurnCount != 2 {
		t.Fatalf("TurnCount = %d, want 2", stats[0].TurnCount)
	}
}

func TestGlobalDBUpdateTokenStatsKeepsPerAgentRows(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-multi-agent")

	input := int64(10)
	if err := globalDB.UpdateTokenStats(testContext(t), TokenStatsUpdate{
		SessionID:   "sess-multi-agent",
		AgentName:   "coder",
		InputTokens: &input,
	}); err != nil {
		t.Fatalf("UpdateTokenStats(coder) error = %v", err)
	}
	if err := globalDB.UpdateTokenStats(testContext(t), TokenStatsUpdate{
		SessionID:   "sess-multi-agent",
		AgentName:   "reviewer",
		InputTokens: &input,
	}); err != nil {
		t.Fatalf("UpdateTokenStats(reviewer) error = %v", err)
	}

	stats, err := globalDB.ListTokenStats(testContext(t), TokenStatsQuery{SessionID: "sess-multi-agent"})
	if err != nil {
		t.Fatalf("ListTokenStats() error = %v", err)
	}
	if got := len(stats); got != 2 {
		t.Fatalf("len(stats) = %d, want 2", got)
	}

	byAgent := make(map[string]TokenStats, len(stats))
	for _, stat := range stats {
		byAgent[stat.AgentName] = stat
	}
	if _, ok := byAgent["coder"]; !ok {
		t.Fatalf("missing coder stats: %#v", stats)
	}
	if _, ok := byAgent["reviewer"]; !ok {
		t.Fatalf("missing reviewer stats: %#v", stats)
	}
}

func TestGlobalDBUpdateSessionStateReturnsNotFoundForMissingSession(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	err := globalDB.UpdateSessionState(testContext(t), SessionStateUpdate{
		ID:    "missing",
		State: "stopped",
	})
	if err == nil || !strings.Contains(err.Error(), `session "missing" not found`) {
		t.Fatalf("UpdateSessionState(missing) error = %v, want missing session error", err)
	}
}

func TestGlobalDBWritePermissionLogEntry(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-perm")

	if err := globalDB.WritePermissionLog(testContext(t), PermissionLogEntry{
		SessionID:  "sess-perm",
		AgentName:  "coder",
		Action:     "bash",
		Resource:   "/tmp/project",
		Decision:   "allow",
		PolicyUsed: "approve-reads",
		Timestamp:  time.Date(2026, 4, 3, 15, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("WritePermissionLog() error = %v", err)
	}

	entries, err := globalDB.ListPermissionLog(testContext(t), PermissionLogQuery{SessionID: "sess-perm"})
	if err != nil {
		t.Fatalf("ListPermissionLog() error = %v", err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}
	if entries[0].Decision != "allow" || entries[0].PolicyUsed != "approve-reads" {
		t.Fatalf("entry = %#v, want allow/approve-reads", entries[0])
	}
}

func TestGlobalDBReconcileSessions(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-keep")
	registerSessionForGlobalTests(t, globalDB, "sess-orphan")

	onDisk := []SessionInfo{
		{
			ID:        "sess-keep",
			AgentName: "coder",
			Workspace: "/tmp/sess-keep",
			State:     "stopped",
			CreatedAt: time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
		},
		{
			ID:        "sess-new",
			AgentName: "reviewer",
			Workspace: "/tmp/sess-new",
			State:     "stopped",
			CreatedAt: time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
		},
	}

	result, err := globalDB.ReconcileSessions(testContext(t), onDisk)
	if err != nil {
		t.Fatalf("ReconcileSessions() error = %v", err)
	}
	sort.Strings(result.Indexed)
	sort.Strings(result.Orphaned)
	if !equalStringSlices(result.Indexed, []string{"sess-new"}) {
		t.Fatalf("Indexed = %#v, want %#v", result.Indexed, []string{"sess-new"})
	}
	if !equalStringSlices(result.Orphaned, []string{"sess-orphan"}) {
		t.Fatalf("Orphaned = %#v, want %#v", result.Orphaned, []string{"sess-orphan"})
	}

	sessions, err := globalDB.ListSessions(testContext(t), SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	stateByID := make(map[string]string, len(sessions))
	for _, session := range sessions {
		stateByID[session.ID] = session.State
	}
	if stateByID["sess-new"] != "stopped" {
		t.Fatalf("stateByID[sess-new] = %q, want stopped", stateByID["sess-new"])
	}
	if stateByID["sess-orphan"] != "orphaned" {
		t.Fatalf("stateByID[sess-orphan] = %q, want orphaned", stateByID["sess-orphan"])
	}
}

func TestGlobalDBRecoversFromCorruption(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, GlobalDatabaseName)
	if err := os.WriteFile(path, []byte("bad sqlite"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	globalDB, err := OpenGlobalDB(testContext(t), path)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := globalDB.Close(testContext(t)); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	assertTablesPresent(t, globalDB.db, "sessions", "event_summaries", "token_stats", "permission_log")

	matches, err := filepath.Glob(path + ".corrupt.*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if got, want := len(matches), 1; got != want {
		t.Fatalf("len(corrupt files) = %d, want %d (%v)", got, want, matches)
	}
}

func openTestGlobalDB(t *testing.T) *GlobalDB {
	t.Helper()

	globalDB, err := OpenGlobalDB(testContext(t), filepath.Join(t.TempDir(), GlobalDatabaseName))
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := globalDB.Close(testContext(t)); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return globalDB
}

func registerSessionForGlobalTests(t *testing.T, globalDB *GlobalDB, sessionID string) {
	t.Helper()

	now := time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC)
	if err := globalDB.RegisterSession(testContext(t), SessionInfo{
		ID:        sessionID,
		AgentName: "coder",
		Workspace: "/tmp/" + sessionID,
		State:     "active",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("RegisterSession(%q) error = %v", sessionID, err)
	}
}
