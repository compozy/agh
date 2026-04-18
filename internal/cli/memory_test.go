package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/testutil"

	"github.com/pedronauck/agh/internal/memory"
	"github.com/spf13/cobra"
)

func TestMemoryListCommandFormatsAndScope(t *testing.T) {
	t.Parallel()

	var seenScope memory.Scope
	deps := newTestDeps(t, &stubClient{
		listMemoryFn: func(_ context.Context, scope memory.Scope, workspace string) ([]MemoryHeaderRecord, error) {
			seenScope = scope
			if scope != memory.ScopeGlobal {
				t.Fatalf("scope = %q, want global", scope)
			}
			if workspace != "" {
				t.Fatalf("workspace = %q, want empty", workspace)
			}
			return []MemoryHeaderRecord{{
				Filename:    "prefs.md",
				Name:        "Prefs",
				Description: "saved preference",
				Type:        memory.MemoryTypeUser,
			}}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "memory", "list", "--scope", "global")
	if err != nil {
		t.Fatalf("memory list error = %v", err)
	}
	if seenScope != memory.ScopeGlobal {
		t.Fatalf("seenScope = %q, want global", seenScope)
	}
	if !strings.Contains(stdout, "Memories") || !strings.Contains(stdout, "prefs.md") {
		t.Fatalf("stdout = %q, want rendered list", stdout)
	}
}

func TestMemoryReadCommandOutputsContent(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		listMemoryFn: func(_ context.Context, scope memory.Scope, _ string) ([]MemoryHeaderRecord, error) {
			if scope == memory.ScopeGlobal {
				return []MemoryHeaderRecord{{Filename: "prefs.md", Type: memory.MemoryTypeUser}}, nil
			}
			return nil, nil
		},
		readMemoryFn: func(_ context.Context, filename string, scope memory.Scope, workspace string) (MemoryReadRecord, error) {
			if filename != "prefs.md" || scope != memory.ScopeGlobal {
				t.Fatalf("ReadMemory args = %q %q %q", filename, scope, workspace)
			}
			return MemoryReadRecord{Content: "stored memory body"}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "memory", "read", "prefs.md")
	if err != nil {
		t.Fatalf("memory read error = %v", err)
	}
	if strings.TrimSpace(stdout) != "stored memory body" {
		t.Fatalf("stdout = %q, want raw content", stdout)
	}
}

func TestMemorySearchAndReindexCommands(t *testing.T) {
	t.Parallel()

	var searchQuery MemorySearchQuery
	var searchText string
	var reindexReq MemoryReindexRequest
	deps := newTestDeps(t, &stubClient{
		searchMemoryFn: func(_ context.Context, query string, opts MemorySearchQuery) ([]MemorySearchRecord, error) {
			searchText = query
			searchQuery = opts
			return []MemorySearchRecord{{
				Filename:  "auth.md",
				Name:      "Auth Rewrite",
				Scope:     memory.ScopeWorkspace,
				Score:     4.2,
				Snippet:   "Auth migration uses sessions",
				Workspace: "/workspace/project",
			}}, nil
		},
		reindexMemoryFn: func(_ context.Context, request MemoryReindexRequest) (MemoryReindexRecord, error) {
			reindexReq = request
			return MemoryReindexRecord{
				IndexedFiles: 2,
				Workspace:    "/workspace/project",
				CompletedAt:  fixedTestNow,
			}, nil
		},
	})

	searchOut, _, err := executeRootCommand(t, deps, "memory", "search", "auth", "rewrite")
	if err != nil {
		t.Fatalf("memory search error = %v", err)
	}
	if searchText != "auth rewrite" || searchQuery.Workspace != "/workspace/project" {
		t.Fatalf("search call = query:%q opts:%#v", searchText, searchQuery)
	}
	if !strings.Contains(searchOut, "Auth Rewrite") || !strings.Contains(searchOut, "auth.md") {
		t.Fatalf("search output = %q", searchOut)
	}

	reindexOut, _, err := executeRootCommand(t, deps, "memory", "reindex")
	if err != nil {
		t.Fatalf("memory reindex error = %v", err)
	}
	if reindexReq.Workspace != "/workspace/project" {
		t.Fatalf("reindex request = %#v, want workspace", reindexReq)
	}
	if !strings.Contains(reindexOut, "Indexed Files") || !strings.Contains(reindexOut, "2") {
		t.Fatalf("reindex output = %q", reindexOut)
	}
}

func TestMemoryWriteCommandBuildsDocumentAndUsesContentFlag(t *testing.T) {
	t.Parallel()

	var seenRequest MemoryWriteRequest
	deps := newTestDeps(t, &stubClient{
		writeMemoryFn: func(_ context.Context, filename string, request MemoryWriteRequest) (MemoryMutationRecord, error) {
			if filename != "prefs.md" {
				t.Fatalf("filename = %q, want prefs.md", filename)
			}
			seenRequest = request
			return MemoryMutationRecord{OK: true}, nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"memory",
		"write",
		"prefs.md",
		"--type",
		"user",
		"--description",
		"remember this",
		"--content",
		"body text",
		"-o",
		"json",
	)
	if err != nil {
		t.Fatalf("memory write error = %v", err)
	}
	if seenRequest.Scope != "global" || seenRequest.Workspace != "" {
		t.Fatalf("request scope/workspace = %#v", seenRequest)
	}
	if !strings.Contains(seenRequest.Content, "type: user") ||
		!strings.Contains(seenRequest.Content, "description: remember this") ||
		!strings.Contains(seenRequest.Content, "body text") {
		t.Fatalf("request content = %q", seenRequest.Content)
	}

	var payload memoryMutationView
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal(write output) error = %v; stdout=%s", err, stdout)
	}
	if payload.Status != "written" {
		t.Fatalf("payload = %#v, want written status", payload)
	}

	var workspaceRequest MemoryWriteRequest
	cmd := newRootCommand(newTestDeps(t, &stubClient{
		writeMemoryFn: func(_ context.Context, filename string, request MemoryWriteRequest) (MemoryMutationRecord, error) {
			if filename != "project.md" {
				t.Fatalf("filename = %q, want project.md", filename)
			}
			workspaceRequest = request
			return MemoryMutationRecord{OK: true}, nil
		},
	}))
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.SetOut(&stdoutBuf)
	cmd.SetErr(&stderrBuf)
	cmd.SetIn(strings.NewReader("stdin body"))
	cmd.SetArgs(
		[]string{"memory", "write", "project.md", "--type", "project", "--description", "project memory", "-o", "json"},
	)
	if err := cmd.ExecuteContext(testutil.Context(t)); err != nil {
		t.Fatalf("memory write from stdin error = %v; stderr=%s", err, stderrBuf.String())
	}
	if workspaceRequest.Scope != "workspace" || workspaceRequest.Workspace != "/workspace/project" {
		t.Fatalf("workspaceRequest = %#v, want workspace scope", workspaceRequest)
	}
}

func TestMemoryDeleteAndConsolidateCommands(t *testing.T) {
	t.Parallel()

	var deleted bool
	var consolidated bool
	deps := newTestDeps(t, &stubClient{
		listMemoryFn: func(_ context.Context, scope memory.Scope, _ string) ([]MemoryHeaderRecord, error) {
			switch scope {
			case memory.ScopeGlobal:
				return nil, nil
			case memory.ScopeWorkspace:
				return []MemoryHeaderRecord{{Filename: "project.md", Type: memory.MemoryTypeProject}}, nil
			default:
				return nil, nil
			}
		},
		deleteMemoryFn: func(_ context.Context, filename string, scope memory.Scope, workspace string) (MemoryMutationRecord, error) {
			deleted = true
			if filename != "project.md" || scope != memory.ScopeWorkspace || workspace != "/workspace/project" {
				t.Fatalf("DeleteMemory args = %q %q %q", filename, scope, workspace)
			}
			return MemoryMutationRecord{OK: true}, nil
		},
		consolidateMemoryFn: func(_ context.Context, workspace string) (MemoryConsolidateRecord, error) {
			consolidated = true
			if workspace != "/workspace/project" {
				t.Fatalf("workspace = %q, want /workspace/project", workspace)
			}
			return MemoryConsolidateRecord{Triggered: false, Reason: "gates not satisfied"}, nil
		},
	})

	deleteOut, _, err := executeRootCommand(t, deps, "memory", "delete", "project.md")
	if err != nil {
		t.Fatalf("memory delete error = %v", err)
	}
	if !deleted || !strings.Contains(deleteOut, "deleted") {
		t.Fatalf("delete output = %q, deleted=%v", deleteOut, deleted)
	}

	consolidateOut, _, err := executeRootCommand(t, deps, "memory", "consolidate")
	if err != nil {
		t.Fatalf("memory consolidate error = %v", err)
	}
	if !consolidated || !strings.Contains(consolidateOut, "gates not satisfied") {
		t.Fatalf("consolidate output = %q, consolidated=%v", consolidateOut, consolidated)
	}
}

func TestMemoryJSONOutputForListAndRead(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		listMemoryFn: func(_ context.Context, scope memory.Scope, _ string) ([]MemoryHeaderRecord, error) {
			switch scope {
			case memory.ScopeGlobal:
				return []MemoryHeaderRecord{{Filename: "prefs.md", Name: "Prefs", Type: memory.MemoryTypeUser}}, nil
			case memory.ScopeWorkspace:
				return nil, nil
			default:
				return nil, nil
			}
		},
		readMemoryFn: func(context.Context, string, memory.Scope, string) (MemoryReadRecord, error) {
			return MemoryReadRecord{Content: "memory content"}, nil
		},
	})

	listOut, _, err := executeRootCommand(t, deps, "memory", "list", "-o", "json")
	if err != nil {
		t.Fatalf("memory list json error = %v", err)
	}
	var listPayload []memoryListItem
	if err := json.Unmarshal([]byte(listOut), &listPayload); err != nil {
		t.Fatalf("json.Unmarshal(list) error = %v; out=%s", err, listOut)
	}
	if len(listPayload) != 1 || listPayload[0].Filename != "prefs.md" {
		t.Fatalf("list payload = %#v", listPayload)
	}

	readOut, _, err := executeRootCommand(t, deps, "memory", "read", "prefs.md", "-o", "json")
	if err != nil {
		t.Fatalf("memory read json error = %v", err)
	}
	var readPayload memoryReadView
	if err := json.Unmarshal([]byte(readOut), &readPayload); err != nil {
		t.Fatalf("json.Unmarshal(read) error = %v; out=%s", err, readOut)
	}
	if readPayload.Content != "memory content" {
		t.Fatalf("read payload = %#v", readPayload)
	}
}

func TestMemoryHelperLocationResolutionAndSorting(t *testing.T) {
	t.Parallel()

	recent := fixedTestNow.Add(-time.Minute)
	older := fixedTestNow.Add(-time.Hour)
	var seenWorkspace string
	client := &stubClient{
		listMemoryFn: func(_ context.Context, scope memory.Scope, workspace string) ([]MemoryHeaderRecord, error) {
			switch scope {
			case memory.ScopeGlobal:
				return []MemoryHeaderRecord{
					{Filename: "shared.md", Name: "Shared", Type: memory.MemoryTypeUser, ModTime: older},
				}, nil
			case memory.ScopeWorkspace:
				seenWorkspace = workspace
				return []MemoryHeaderRecord{
					{Filename: "project.md", Name: "Project", Type: memory.MemoryTypeProject, ModTime: recent},
					{
						Filename: "shared.md",
						Name:     "Shared",
						Type:     memory.MemoryTypeProject,
						ModTime:  recent.Add(-time.Minute),
					},
				}, nil
			default:
				return nil, nil
			}
		},
	}
	deps := newTestDeps(t, client)

	locations, err := listMemoryLocations(context.Background(), client, deps, "")
	if err != nil {
		t.Fatalf("listMemoryLocations() error = %v", err)
	}
	if len(locations) != 3 {
		t.Fatalf("locations len = %d, want 3", len(locations))
	}
	if locations[0].Header.Filename != "project.md" || locations[0].Scope != memory.ScopeWorkspace {
		t.Fatalf("locations[0] = %#v, want most recent workspace memory", locations[0])
	}
	if seenWorkspace != "/workspace/project" {
		t.Fatalf("workspace = %q, want /workspace/project", seenWorkspace)
	}

	location, err := resolveMemoryLocation(context.Background(), client, deps, "", "project.md")
	if err != nil {
		t.Fatalf("resolveMemoryLocation(project.md) error = %v", err)
	}
	if location.Scope != memory.ScopeWorkspace || location.Workspace != "/workspace/project" {
		t.Fatalf("location = %#v, want workspace resolution", location)
	}

	_, err = resolveMemoryLocation(context.Background(), client, deps, "", "shared.md")
	if err == nil || !strings.Contains(err.Error(), "--scope explicitly") {
		t.Fatalf("resolveMemoryLocation(shared.md) error = %v, want ambiguous scope error", err)
	}

	_, err = resolveMemoryLocation(context.Background(), client, deps, "", "missing.md")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("resolveMemoryLocation(missing.md) error = %v, want os.ErrNotExist", err)
	}
}

func TestMemoryHelperContentScopeAndFormatting(t *testing.T) {
	t.Parallel()

	makeCmd := func(stdin string, args ...string) *cobra.Command {
		cmd := &cobra.Command{Use: "memory-test"}
		cmd.Flags().String("content", "", "")
		cmd.SetIn(strings.NewReader(stdin))
		if err := cmd.Flags().Parse(args); err != nil {
			t.Fatalf("Flags().Parse() error = %v", err)
		}
		return cmd
	}

	flagCmd := makeCmd("", "--content", "flag body")
	flagContent, err := resolveMemoryWriteContent(flagCmd, "flag body")
	if err != nil || flagContent != "flag body" {
		t.Fatalf("resolveMemoryWriteContent(flag) = %q, %v", flagContent, err)
	}

	stdinCmd := makeCmd("stdin body")
	stdinContent, err := resolveMemoryWriteContent(stdinCmd, "")
	if err != nil || stdinContent != "stdin body" {
		t.Fatalf("resolveMemoryWriteContent(stdin) = %q, %v", stdinContent, err)
	}

	bothCmd := makeCmd("stdin body", "--content", "flag body")
	if _, err := resolveMemoryWriteContent(bothCmd, "flag body"); err == nil {
		t.Fatal("resolveMemoryWriteContent(both) error = nil, want non-nil")
	}

	emptyCmd := makeCmd("")
	if _, err := resolveMemoryWriteContent(emptyCmd, ""); err == nil {
		t.Fatal("resolveMemoryWriteContent(empty) error = nil, want non-nil")
	}

	if content, err := readOptionalCommandInput(nil); err != nil || content != "" {
		t.Fatalf("readOptionalCommandInput(nil) = %q, %v", content, err)
	}

	scope, err := resolveCLIMemoryWriteScope("", memory.MemoryTypeProject)
	if err != nil || scope != memory.ScopeWorkspace {
		t.Fatalf("resolveCLIMemoryWriteScope(project) = %q, %v", scope, err)
	}
	if _, err := resolveCLIMemoryWriteScope("bogus", memory.MemoryTypeUser); err == nil {
		t.Fatal("resolveCLIMemoryWriteScope(bogus) error = nil, want non-nil")
	}
	if _, err := parseMemoryType("bogus"); err == nil {
		t.Fatal("parseMemoryType(bogus) error = nil, want non-nil")
	}

	document, err := formatMemoryDocument("my.project_notes.md", memory.MemoryTypeProject, "desc", "body")
	if err != nil {
		t.Fatalf("formatMemoryDocument() error = %v", err)
	}
	if !strings.Contains(document, "name: My Project Notes") || !strings.Contains(document, "description: desc") ||
		!strings.Contains(document, "body") {
		t.Fatalf("document = %q, want formatted frontmatter and body", document)
	}
	if _, err := formatMemoryDocument("", memory.MemoryTypeUser, "desc", "body"); err == nil {
		t.Fatal("formatMemoryDocument(empty filename) error = nil, want non-nil")
	}
	if memoryNameFromFilename("release_notes.v2.md") != "Release Notes V2" {
		t.Fatalf("memoryNameFromFilename() = %q", memoryNameFromFilename("release_notes.v2.md"))
	}
}

func TestMemoryBundleHelpers(t *testing.T) {
	t.Parallel()

	listBundle := memoryListBundle([]memoryLocation{{
		Scope: memory.ScopeGlobal,
		Header: MemoryHeaderRecord{
			Filename:    "prefs.md",
			Name:        "Prefs",
			Type:        memory.MemoryTypeUser,
			Description: "saved preference",
			ModTime:     fixedTestNow.Add(-time.Minute),
		},
	}}, func() time.Time { return fixedTestNow })
	listHuman, err := listBundle.human()
	if err != nil {
		t.Fatalf("listBundle.human() error = %v", err)
	}
	listToon, err := listBundle.toon()
	if err != nil {
		t.Fatalf("listBundle.toon() error = %v", err)
	}
	if !strings.Contains(listHuman, "prefs.md") || !strings.Contains(listToon, "prefs.md") {
		t.Fatalf("list outputs missing memory: human=%q toon=%q", listHuman, listToon)
	}

	readBundle := memoryReadBundle(memoryReadView{
		Filename: "prefs.md",
		Scope:    memory.ScopeGlobal,
		Content:  "memory body\n",
	})
	readHuman, err := readBundle.human()
	if err != nil {
		t.Fatalf("readBundle.human() error = %v", err)
	}
	readToon, err := readBundle.toon()
	if err != nil {
		t.Fatalf("readBundle.toon() error = %v", err)
	}
	if readHuman != "memory body" || !strings.Contains(readToon, "prefs.md") {
		t.Fatalf("read outputs = %q / %q", readHuman, readToon)
	}

	mutationBundle := memoryMutationBundle(memoryMutationView{
		Filename: "prefs.md",
		Scope:    memory.ScopeWorkspace,
		Type:     memory.MemoryTypeProject,
		Status:   boolStatus(true),
		Reason:   "queued",
	})
	mutationHuman, err := mutationBundle.human()
	if err != nil {
		t.Fatalf("mutationBundle.human() error = %v", err)
	}
	mutationToon, err := mutationBundle.toon()
	if err != nil {
		t.Fatalf("mutationBundle.toon() error = %v", err)
	}
	if !strings.Contains(mutationHuman, "queued") || !strings.Contains(mutationToon, "triggered") {
		t.Fatalf("mutation outputs = %q / %q", mutationHuman, mutationToon)
	}

	if titleCaseWord("mEMORY") != "Memory" {
		t.Fatalf("titleCaseWord() = %q, want Memory", titleCaseWord("mEMORY"))
	}
	if boolStatus(false) != "not-triggered" {
		t.Fatalf("boolStatus(false) = %q, want not-triggered", boolStatus(false))
	}
}
