//go:build integration && !windows

package daemon

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	memcontract "github.com/pedronauck/agh/internal/memory/contract"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
	e2etest "github.com/pedronauck/agh/internal/testutil/e2e"
)

func TestDaemonE2EMemoryCatalogCLIHTTPParityAndLegacyPathIsolation(t *testing.T) {
	t.Parallel()

	harness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	legacyPath := filepath.Join(harness.WorkspaceRoot, ".compozy", "memory", "legacy-only.md")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(legacyPath), err)
	}
	if err := os.WriteFile(
		legacyPath,
		[]byte(
			memoryDocument("Legacy Decoy", "Legacy path should stay ignored", memcontract.TypeProject, "legacy decoy"),
		),
		0o644,
	); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", legacyPath, err)
	}

	writeMemoryViaCLI(
		t,
		ctx,
		harness,
		"",
		"prefs.md",
		memcontract.TypeUser,
		"User prefers concise answers",
		"Prefer concise answers with direct technical detail.",
		memcontract.ScopeGlobal,
	)
	writeMemoryViaCLI(
		t,
		ctx,
		harness,
		harness.WorkspaceRoot,
		"auth.md",
		memcontract.TypeProject,
		"Auth migration details",
		"Remember me: auth migration uses sessions and workspace-scoped recall.",
		memcontract.ScopeWorkspace,
	)
	writeMemoryViaCLI(
		t,
		ctx,
		harness,
		harness.WorkspaceRoot,
		"release-plan.md",
		memcontract.TypeProject,
		"Release plan details",
		"Release plan covers durable release checklist and observability ownership.",
		memcontract.ScopeWorkspace,
	)

	t.Run("Should return matching CLI and HTTP search results while ignoring legacy paths", func(t *testing.T) {
		var cliSearch aghcontract.MemorySearchResponse
		if err := harness.CLI.RunJSONInDir(
			ctx,
			harness.WorkspaceRoot,
			&cliSearch,
			"memory",
			"search",
			"auth sessions",
			"--workspace",
			harness.WorkspaceRoot,
			"-o",
			"json",
		); err != nil {
			t.Fatalf("CLI memory search error = %v", err)
		}
		if !containsSearchResult(cliSearch, "project_auth.md", memcontract.ScopeWorkspace) {
			t.Fatalf("CLI search results = %#v, want workspace project_auth.md hit", cliSearch)
		}
		if containsSearchResult(cliSearch, "legacy-only.md", memcontract.ScopeWorkspace) {
			t.Fatalf("CLI search results = %#v, want legacy path ignored", cliSearch)
		}

		var httpSearch aghcontract.MemorySearchResponse
		if err := harness.HTTPJSON(
			ctx,
			http.MethodPost,
			"/api/memory/search",
			memorySearchRequest("auth sessions", memcontract.ScopeWorkspace, harness.WorkspaceRoot),
			&httpSearch,
		); err != nil {
			t.Fatalf("HTTP memory search error = %v", err)
		}
		if !containsSearchResult(httpSearch, "project_auth.md", memcontract.ScopeWorkspace) {
			t.Fatalf("HTTP search results = %#v, want workspace project_auth.md hit", httpSearch)
		}
		if len(cliSearch.Results) == 0 || len(httpSearch.Results) == 0 {
			t.Fatalf("search results = cli:%#v http:%#v, want non-empty parity", cliSearch, httpSearch)
		}
		if got, want := httpSearch.Results[0].Memory.Filename, cliSearch.Results[0].Memory.Filename; got != want {
			t.Fatalf("HTTP top search filename = %q, want %q", got, want)
		}
		if got, want := httpSearch.Results[0].Memory.Scope, cliSearch.Results[0].Memory.Scope; got != want {
			t.Fatalf("HTTP top search scope = %q, want %q", got, want)
		}

		var cliLegacy aghcontract.MemorySearchResponse
		if err := harness.CLI.RunJSONInDir(
			ctx,
			harness.WorkspaceRoot,
			&cliLegacy,
			"memory",
			"search",
			"legacy decoy",
			"--workspace",
			harness.WorkspaceRoot,
			"-o",
			"json",
		); err != nil {
			t.Fatalf("CLI legacy search error = %v", err)
		}
		if len(cliLegacy.Results) != 0 {
			t.Fatalf("CLI legacy search results = %#v, want empty result set", cliLegacy)
		}

		var httpLegacy aghcontract.MemorySearchResponse
		if err := harness.HTTPJSON(
			ctx,
			http.MethodPost,
			"/api/memory/search",
			memorySearchRequest("legacy decoy", memcontract.ScopeWorkspace, harness.WorkspaceRoot),
			&httpLegacy,
		); err != nil {
			t.Fatalf("HTTP legacy search error = %v", err)
		}
		if len(httpLegacy.Results) != 0 {
			t.Fatalf("HTTP legacy search results = %#v, want empty result set", httpLegacy)
		}
	})

	t.Run("Should reindex through CLI and HTTP with matching counts", func(t *testing.T) {
		var cliReindex memcontract.ReindexResult
		if err := harness.CLI.RunJSONInDir(
			ctx,
			harness.WorkspaceRoot,
			&cliReindex,
			"memory",
			"reindex",
			"-o",
			"json",
		); err != nil {
			t.Fatalf("CLI memory reindex error = %v", err)
		}
		if got, want := cliReindex.IndexedFiles, 3; got != want {
			t.Fatalf("CLI reindex indexed_files = %d, want %d", got, want)
		}

		var httpReindex memcontract.ReindexResult
		if err := harness.HTTPJSON(
			ctx,
			http.MethodPost,
			"/api/memory/reindex",
			aghcontract.MemoryReindexV2Request{WorkspaceID: harness.WorkspaceRoot},
			&httpReindex,
		); err != nil {
			t.Fatalf("HTTP memory reindex error = %v", err)
		}
		if got, want := httpReindex.IndexedFiles, 3; got != want {
			t.Fatalf("HTTP reindex indexed_files = %d, want %d", got, want)
		}
	})

	t.Run("Should surface memory health and observe events", func(t *testing.T) {
		var health aghcontract.HealthResponse
		if err := harness.HTTPJSON(
			ctx,
			http.MethodGet,
			"/api/observe/health?workspace="+url.QueryEscape(harness.WorkspaceRoot),
			nil,
			&health,
		); err != nil {
			t.Fatalf("HTTP observe health error = %v", err)
		}
		if got, want := health.Memory.GlobalFiles, 1; got != want {
			t.Fatalf("health.Memory.GlobalFiles = %d, want %d", got, want)
		}
		if got, want := health.Memory.WorkspaceFiles, 2; got != want {
			t.Fatalf("health.Memory.WorkspaceFiles = %d, want %d", got, want)
		}
		if got, want := health.Memory.IndexedFiles, 3; got != want {
			t.Fatalf("health.Memory.IndexedFiles = %d, want %d", got, want)
		}
		if got := health.Memory.OrphanedFiles; got != 0 {
			t.Fatalf("health.Memory.OrphanedFiles = %d, want 0", got)
		}
		if health.Memory.LastReindex == nil {
			t.Fatalf("health.Memory.LastReindex = nil, want non-nil")
		}

		var reindexEvents aghcontract.ObserveEventsResponse
		if err := harness.UDSJSON(
			ctx,
			http.MethodGet,
			"/api/observe/events?type=memory.write.reindex&limit=10",
			nil,
			&reindexEvents,
		); err != nil {
			t.Fatalf("UDS observe reindex events error = %v", err)
		}
		if !containsObserveEventSummary(reindexEvents.Events, "memory.write.reindex", "indexed=3") {
			t.Fatalf("reindex observe events = %#v, want indexed=3 summary", reindexEvents.Events)
		}

		var searchEvents aghcontract.ObserveEventsResponse
		if err := harness.UDSJSON(
			ctx,
			http.MethodGet,
			"/api/observe/events?type=memory.recall.executed&limit=10",
			nil,
			&searchEvents,
		); err != nil {
			t.Fatalf("UDS observe search events error = %v", err)
		}
		if !containsObserveEventSummary(searchEvents.Events, "memory.recall.executed", `auth sessions`) {
			t.Fatalf("search observe events = %#v, want auth search summary", searchEvents.Events)
		}
		if !containsObserveEventSummary(searchEvents.Events, "memory.recall.executed", `legacy decoy`) {
			t.Fatalf("search observe events = %#v, want legacy search summary", searchEvents.Events)
		}
	})
}

func TestDaemonE2EMemoryRecallUsesCatalogSynthesisWithoutMutatingStoredUserMessage(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	harness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  mockFixturePath(t, "memory_recall_fixture.json"),
			FixtureAgent: "memory-recall-agent",
			AgentName:    "memory-recall-agent",
		}},
	})
	registration, ok := harness.MockAgentRegistration("memory-recall-agent")
	if !ok {
		t.Fatal("MockAgentRegistration(memory-recall-agent) = missing, want present")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	writeMemoryViaUDS(
		t,
		ctx,
		harness,
		"auth.md",
		harness.WorkspaceRoot,
		"Remember me: auth migration uses sessions and workspace-scoped recall.",
	)

	indexPath := filepath.Join(harness.WorkspaceRoot, ".agh", "memory", "MEMORY.md")
	staleIndex := "- [Old](missing.md) - stale index entry\n"
	if err := os.WriteFile(indexPath, []byte(staleIndex), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", indexPath, err)
	}

	session := createFixtureBackedSession(t, ctx, harness, "memory-recall-agent", "memory-recall-session")
	if _, err := harness.PromptSession(ctx, session.ID, "remember me"); err != nil {
		t.Fatalf("PromptSession() error = %v", err)
	}

	eventsResp, err := harness.SessionEvents(ctx, session.ID)
	if err != nil {
		t.Fatalf("SessionEvents() error = %v", err)
	}
	events := decodeAgentEvents(t, eventsResp.Events)
	diagnostics, err := acpmock.ReadDiagnostics(registration.DiagnosticsPath)
	if err != nil {
		t.Fatalf("ReadDiagnostics(%q) error = %v", registration.DiagnosticsPath, err)
	}

	t.Run("Should persist the original user message without injected recall", func(t *testing.T) {
		if !containsAgentEvent(events, aghcontract.AgentEventPayload{
			Type: "agent_message",
			Text: "qa-memory acknowledged",
		}) {
			t.Fatalf("session events = %#v, want recall fixture assistant response", events)
		}
		userEvent, ok := firstAgentEventByType(events, "user_message")
		if !ok {
			t.Fatalf("session events = %#v, want stored user_message event", events)
		}
		if got, want := userEvent.Text, "remember me"; got != want {
			t.Fatalf("stored user_message text = %q, want %q", got, want)
		}
		if strings.Contains(userEvent.Text, "Relevant durable memory for this turn:") {
			t.Fatalf("stored user_message text = %q, want no injected recall block", userEvent.Text)
		}
	})

	t.Run("Should dispatch recalled context to the agent prompt", func(t *testing.T) {
		promptDiagnostics := acpmock.PromptDiagnostics(diagnostics)
		if got, want := len(promptDiagnostics), 1; got != want {
			t.Fatalf("len(promptDiagnostics) = %d, want %d; diagnostics=%#v", got, want, diagnostics)
		}
		prompt := promptDiagnostics[0].Prompt
		if !strings.Contains(prompt, "Relevant durable memory for this turn:") {
			t.Fatalf("mock prompt = %q, want recall preamble", prompt)
		}
		if !strings.Contains(prompt, "- auth [workspace]") {
			t.Fatalf("mock prompt = %q, want workspace recall heading", prompt)
		}
		if !strings.Contains(prompt, "auth migration uses sessions") {
			t.Fatalf("mock prompt = %q, want recalled auth snippet", prompt)
		}
		if !strings.Contains(prompt, "\n\nUser message:\nremember me") {
			t.Fatalf("mock prompt = %q, want raw user message suffix", prompt)
		}
		if strings.Contains(prompt, "missing.md") {
			t.Fatalf("mock prompt = %q, want stale MEMORY.md entry ignored", prompt)
		}
	})

	t.Run("Should leave the stale index file unchanged on disk", func(t *testing.T) {
		indexBytes, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v", indexPath, err)
		}
		if got := string(indexBytes); got != staleIndex {
			t.Fatalf("workspace MEMORY.md = %q, want unchanged stale index", got)
		}
	})
}

func writeMemoryViaCLI(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	workdir string,
	filename string,
	memoryType memcontract.Type,
	description string,
	content string,
	scope memcontract.Scope,
) {
	t.Helper()

	var result aghcontract.MemoryMutationDecisionResponse
	args := []string{
		"memory",
		"write",
		"--type",
		string(memoryType),
		"--name",
		strings.TrimSuffix(filename, filepath.Ext(filename)),
		"--description",
		description,
		"--content",
		content,
		"--scope",
		string(scope),
		"-o",
		"json",
	}
	if scope == memcontract.ScopeWorkspace {
		args = append(args[:len(args)-2], "--workspace", workdir, "-o", "json")
	}
	if err := harness.CLI.RunJSONInDir(ctx, workdir, &result, args...); err != nil {
		t.Fatalf("CLI memory write %q error = %v", filename, err)
	}
	if !result.Applied {
		t.Fatalf("CLI memory write %q = %#v, want applied=true", filename, result)
	}
}

func writeMemoryViaUDS(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	filename string,
	workspace string,
	content string,
) {
	t.Helper()

	var response aghcontract.MemoryMutationDecisionResponse
	if err := harness.UDSJSON(
		ctx,
		http.MethodPost,
		"/api/memory",
		aghcontract.MemoryCreateRequest{
			Scope:       memcontract.ScopeWorkspace,
			WorkspaceID: workspace,
			Type:        memcontract.TypeProject,
			Name:        strings.TrimSuffix(filename, filepath.Ext(filename)),
			Content:     content,
		},
		&response,
	); err != nil {
		t.Fatalf("UDS memory write %q error = %v", filename, err)
	}
	if !response.Applied {
		t.Fatalf("UDS memory write %q = %#v, want applied=true", filename, response)
	}
}

func memorySearchRequest(
	query string,
	scope memcontract.Scope,
	workspace string,
) aghcontract.MemorySearchRequest {
	return aghcontract.MemorySearchRequest{
		QueryText:   query,
		Scope:       scope,
		WorkspaceID: workspace,
	}
}

func containsSearchResult(response aghcontract.MemorySearchResponse, filename string, scope memcontract.Scope) bool {
	for _, result := range response.Results {
		if result.Memory.Filename == filename && result.Memory.Scope == scope {
			return true
		}
	}
	return false
}

func containsObserveEventSummary(
	events []aghcontract.ObserveEventPayload,
	eventType string,
	summaryFragment string,
) bool {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		if strings.Contains(event.Summary, summaryFragment) {
			return true
		}
	}
	return false
}

func firstAgentEventByType(
	events []aghcontract.AgentEventPayload,
	eventType string,
) (aghcontract.AgentEventPayload, bool) {
	for _, event := range events {
		if event.Type == eventType {
			return event, true
		}
	}
	return aghcontract.AgentEventPayload{}, false
}
