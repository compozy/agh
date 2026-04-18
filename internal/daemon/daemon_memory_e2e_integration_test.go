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

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/memory"
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
		[]byte(memoryDocument("Legacy Decoy", "Legacy path should stay ignored", memory.MemoryTypeProject, "legacy decoy")),
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
		memory.MemoryTypeUser,
		"User prefers concise answers",
		"Prefer concise answers with direct technical detail.",
		memory.ScopeGlobal,
	)
	writeMemoryViaCLI(
		t,
		ctx,
		harness,
		harness.WorkspaceRoot,
		"auth.md",
		memory.MemoryTypeProject,
		"Auth migration details",
		"Remember me: auth migration uses sessions and workspace-scoped recall.",
		memory.ScopeWorkspace,
	)
	writeMemoryViaCLI(
		t,
		ctx,
		harness,
		harness.WorkspaceRoot,
		"release-plan.md",
		memory.MemoryTypeProject,
		"Release plan details",
		"Release plan covers regression gates and observability checks.",
		memory.ScopeWorkspace,
	)

	var cliSearch []memory.SearchResult
	if err := harness.CLI.RunJSONInDir(
		ctx,
		harness.WorkspaceRoot,
		&cliSearch,
		"memory",
		"search",
		"auth sessions",
		"-o",
		"json",
	); err != nil {
		t.Fatalf("CLI memory search error = %v", err)
	}
	if !containsSearchResult(cliSearch, "auth.md", memory.ScopeWorkspace) {
		t.Fatalf("CLI search results = %#v, want workspace auth.md hit", cliSearch)
	}
	if containsSearchResult(cliSearch, "legacy-only.md", memory.ScopeWorkspace) {
		t.Fatalf("CLI search results = %#v, want legacy path ignored", cliSearch)
	}

	var httpSearch []memory.SearchResult
	if err := harness.HTTPJSON(ctx, http.MethodGet, memorySearchPath("auth sessions", harness.WorkspaceRoot), nil, &httpSearch); err != nil {
		t.Fatalf("HTTP memory search error = %v", err)
	}
	if !containsSearchResult(httpSearch, "auth.md", memory.ScopeWorkspace) {
		t.Fatalf("HTTP search results = %#v, want workspace auth.md hit", httpSearch)
	}
	if len(cliSearch) == 0 || len(httpSearch) == 0 {
		t.Fatalf("search results = cli:%#v http:%#v, want non-empty parity", cliSearch, httpSearch)
	}
	if got, want := httpSearch[0].Filename, cliSearch[0].Filename; got != want {
		t.Fatalf("HTTP top search filename = %q, want %q", got, want)
	}
	if got, want := httpSearch[0].Scope, cliSearch[0].Scope; got != want {
		t.Fatalf("HTTP top search scope = %q, want %q", got, want)
	}

	var cliLegacy []memory.SearchResult
	if err := harness.CLI.RunJSONInDir(
		ctx,
		harness.WorkspaceRoot,
		&cliLegacy,
		"memory",
		"search",
		"legacy decoy",
		"-o",
		"json",
	); err != nil {
		t.Fatalf("CLI legacy search error = %v", err)
	}
	if len(cliLegacy) != 0 {
		t.Fatalf("CLI legacy search results = %#v, want empty result set", cliLegacy)
	}

	var httpLegacy []memory.SearchResult
	if err := harness.HTTPJSON(ctx, http.MethodGet, memorySearchPath("legacy decoy", harness.WorkspaceRoot), nil, &httpLegacy); err != nil {
		t.Fatalf("HTTP legacy search error = %v", err)
	}
	if len(httpLegacy) != 0 {
		t.Fatalf("HTTP legacy search results = %#v, want empty result set", httpLegacy)
	}

	var cliReindex memory.ReindexResult
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

	var httpReindex memory.ReindexResult
	if err := harness.HTTPJSON(
		ctx,
		http.MethodPost,
		"/api/memory/reindex",
		aghcontract.MemoryReindexRequest{Workspace: harness.WorkspaceRoot},
		&httpReindex,
	); err != nil {
		t.Fatalf("HTTP memory reindex error = %v", err)
	}
	if got, want := httpReindex.IndexedFiles, 3; got != want {
		t.Fatalf("HTTP reindex indexed_files = %d, want %d", got, want)
	}

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
		"/api/observe/events?type=memory.reindex&limit=10",
		nil,
		&reindexEvents,
	); err != nil {
		t.Fatalf("UDS observe reindex events error = %v", err)
	}
	if !containsObserveEventSummary(reindexEvents.Events, "memory.reindex", "indexed=3") {
		t.Fatalf("reindex observe events = %#v, want indexed=3 summary", reindexEvents.Events)
	}

	var searchEvents aghcontract.ObserveEventsResponse
	if err := harness.UDSJSON(
		ctx,
		http.MethodGet,
		"/api/observe/events?type=memory.search&limit=10",
		nil,
		&searchEvents,
	); err != nil {
		t.Fatalf("UDS observe search events error = %v", err)
	}
	if !containsObserveEventSummary(searchEvents.Events, "memory.search", `auth sessions`) {
		t.Fatalf("search observe events = %#v, want auth search summary", searchEvents.Events)
	}
	if !containsObserveEventSummary(searchEvents.Events, "memory.search", `legacy decoy`) {
		t.Fatalf("search observe events = %#v, want legacy search summary", searchEvents.Events)
	}
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
		memoryDocument(
			"Auth",
			"Workspace auth migration details",
			memory.MemoryTypeProject,
			"Remember me: auth migration uses sessions and workspace-scoped recall.",
		),
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

	diagnostics, err := acpmock.ReadDiagnostics(registration.DiagnosticsPath)
	if err != nil {
		t.Fatalf("ReadDiagnostics(%q) error = %v", registration.DiagnosticsPath, err)
	}
	if got, want := len(diagnostics), 1; got != want {
		t.Fatalf("len(diagnostics) = %d, want %d; diagnostics=%#v", got, want, diagnostics)
	}
	prompt := diagnostics[0].Prompt
	if !strings.Contains(prompt, "Relevant durable memory for this turn:") {
		t.Fatalf("mock prompt = %q, want recall preamble", prompt)
	}
	if !strings.Contains(prompt, "- Auth [workspace]") {
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

	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", indexPath, err)
	}
	if got := string(indexBytes); got != staleIndex {
		t.Fatalf("workspace MEMORY.md = %q, want unchanged stale index", got)
	}
}

type cliMemoryMutationRecord struct {
	Filename string       `json:"filename"`
	Scope    memory.Scope `json:"scope"`
	Status   string       `json:"status"`
}

func writeMemoryViaCLI(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	workdir string,
	filename string,
	memoryType memory.Type,
	description string,
	content string,
	scope memory.Scope,
) {
	t.Helper()

	var result cliMemoryMutationRecord
	if err := harness.CLI.RunJSONInDir(
		ctx,
		workdir,
		&result,
		"memory",
		"write",
		filename,
		"--type",
		string(memoryType),
		"--description",
		description,
		"--content",
		content,
		"--scope",
		string(scope),
		"-o",
		"json",
	); err != nil {
		t.Fatalf("CLI memory write %q error = %v", filename, err)
	}
	if got, want := result.Status, "written"; got != want {
		t.Fatalf("CLI memory write %q status = %q, want %q", filename, got, want)
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

	var response aghcontract.MemoryMutationResponse
	if err := harness.UDSJSON(
		ctx,
		http.MethodPut,
		"/api/memory/"+url.PathEscape(filename),
		aghcontract.MemoryWriteRequest{
			Scope:     string(memory.ScopeWorkspace),
			Workspace: workspace,
			Content:   content,
		},
		&response,
	); err != nil {
		t.Fatalf("UDS memory write %q error = %v", filename, err)
	}
	if !response.OK {
		t.Fatalf("UDS memory write %q = %#v, want ok=true", filename, response)
	}
}

func memoryDocument(name string, description string, memoryType memory.Type, body string) string {
	return strings.TrimSpace(strings.Join([]string{
		"---",
		"name: " + name,
		"description: " + description,
		"type: " + string(memoryType),
		"---",
		"",
		body,
	}, "\n")) + "\n"
}

func memorySearchPath(query string, workspace string) string {
	values := url.Values{}
	values.Set("q", query)
	if strings.TrimSpace(workspace) != "" {
		values.Set("workspace", workspace)
	}
	return "/api/memory/search?" + values.Encode()
}

func containsSearchResult(results []memory.SearchResult, filename string, scope memory.Scope) bool {
	for _, result := range results {
		if result.Filename == filename && result.Scope == scope {
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
