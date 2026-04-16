//go:build integration

package extensionpkg_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/gin-gonic/gin"
	"github.com/kballard/go-shellquote"
	"github.com/pedronauck/agh/internal/cli"
	aghconfig "github.com/pedronauck/agh/internal/config"
	daemonpkg "github.com/pedronauck/agh/internal/daemon"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	referenceACPHelperEnvKey     = "AGH_TEST_REFERENCE_ACP_HELPER"
	referencePromptLogPathEnvKey = "AGH_TEST_REFERENCE_PROMPT_LOG_PATH"
)

type referenceHandshakeMarker struct {
	Request  subprocess.InitializeRequest  `json:"request"`
	Response subprocess.InitializeResponse `json:"response"`
	PID      int                           `json:"pid"`
}

type referenceHostCallMarker struct {
	SessionCount int               `json:"session_count"`
	Sessions     []json.RawMessage `json:"sessions"`
	Error        string            `json:"error,omitempty"`
	PID          int               `json:"pid"`
}

type referenceCapabilityMarker struct {
	Denied  bool           `json:"denied"`
	Message string         `json:"message,omitempty"`
	Code    int            `json:"code,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

type referencePromptLogEntry struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
}

type referenceHarness struct {
	client       cli.DaemonClient
	daemonCancel context.CancelFunc
	daemonErrCh  chan error
	homePaths    aghconfig.HomePaths
	logBuffer    *bytes.Buffer
	repoRoot     string
	workspace    workspacepkg.ResolvedWorkspace

	promptLogPath string

	secretHandshakePath string
	secretHostCallPath  string
	secretStartsPath    string
	secretCrashOncePath string
	secretShutdownPath  string

	promptHandshakePath  string
	promptHostCallPath   string
	promptCapabilityPath string
	promptShutdownPath   string
}

type referenceACPAgent struct{}

func TestReferenceExtensionACPHelperProcess(t *testing.T) {
	if os.Getenv(referenceACPHelperEnvKey) != "1" {
		return
	}

	conn := acpsdk.NewAgentSideConnection(referenceACPAgent{}, os.Stdout, os.Stdin)
	<-conn.Done()
	os.Exit(0)
}

func TestReferenceExtensionsEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference extension integration uses Unix domain sockets and shell command quoting")
	}

	repoRoot := referenceRepoRoot(t)
	buildReferenceArtifacts(t, repoRoot)

	harness := newReferenceHarness(t, repoRoot)

	secret := harness.installExtension(t, "sdk/examples/secret-guard")
	if secret.Name != "secret-guard" {
		t.Fatalf("secret install name = %q, want secret-guard", secret.Name)
	}
	if secret.State != "active" || secret.Health != "healthy" {
		t.Fatalf("secret install status = %#v, want active/healthy", secret)
	}

	promptEnhancer := harness.installExtension(t, "sdk/examples/prompt-enhancer")
	if promptEnhancer.Name != "prompt-enhancer" {
		t.Fatalf("prompt install name = %q, want prompt-enhancer", promptEnhancer.Name)
	}
	if promptEnhancer.State != "active" || promptEnhancer.Health != "healthy" {
		t.Fatalf("prompt install status = %#v, want active/healthy", promptEnhancer)
	}

	secretPID := waitForStartedPID(t, harness.secretStartsPath, 2, 10*time.Second)
	secretHandshake := waitForHandshakeMarker(t, harness.secretHandshakePath, secretPID, 10*time.Second)
	if secretHandshake.Request.ProtocolVersion != "1" {
		t.Fatalf("secret handshake protocol = %q, want 1", secretHandshake.Request.ProtocolVersion)
	}
	if secretHandshake.PID <= 0 {
		t.Fatalf("secret handshake pid = %d, want > 0", secretHandshake.PID)
	}
	if got := secretHandshake.Request.Capabilities.GrantedActions; len(got) != 1 || got[0] != "sessions/list" {
		t.Fatalf("secret granted actions = %#v, want sessions/list", got)
	}
	if got := secretHandshake.Request.Capabilities.GrantedSecurity; len(got) != 1 || got[0] != "session.read" {
		t.Fatalf("secret granted security = %#v, want session.read", got)
	}
	if got := secretHandshake.Response.SupportedHookEvents; len(got) != 1 || got[0] != string(hookspkg.HookInputPreSubmit) {
		t.Fatalf("secret supported hook events = %#v, want input.pre_submit", got)
	}

	promptHandshake := waitForJSONFile[referenceHandshakeMarker](t, harness.promptHandshakePath, 10*time.Second)
	if promptHandshake.Request.ProtocolVersion != "1" {
		t.Fatalf("prompt handshake protocol = %q, want 1", promptHandshake.Request.ProtocolVersion)
	}
	if promptHandshake.PID <= 0 {
		t.Fatalf("prompt handshake pid = %d, want > 0", promptHandshake.PID)
	}
	if got := promptHandshake.Request.Capabilities.GrantedActions; len(got) != 1 || got[0] != "sessions/list" {
		t.Fatalf("prompt granted actions = %#v, want sessions/list", got)
	}
	if got := promptHandshake.Request.Capabilities.GrantedSecurity; len(got) != 1 || got[0] != "session.read" {
		t.Fatalf("prompt granted security = %#v, want session.read", got)
	}
	if got := promptHandshake.Response.SupportedHookEvents; len(got) != 1 || got[0] != string(hookspkg.HookPromptPostAssemble) {
		t.Fatalf("prompt supported hook events = %#v, want prompt.post_assemble", got)
	}

	secretHostCall := waitForHostCallMarker(t, harness.secretHostCallPath, secretPID, 10*time.Second)
	if secretHostCall.Error != "" {
		t.Fatalf("secret host call error = %q, want empty", secretHostCall.Error)
	}

	promptHostCall := waitForHostCallMarker(t, harness.promptHostCallPath, promptHandshake.PID, 10*time.Second)
	if promptHostCall.Error != "" {
		t.Fatalf("prompt host call error = %q, want empty", promptHostCall.Error)
	}

	capabilityMarker := waitForJSONFile[referenceCapabilityMarker](t, harness.promptCapabilityPath, 10*time.Second)
	if !capabilityMarker.Denied {
		t.Fatalf("capability marker = %#v, want denied=true", capabilityMarker)
	}
	if capabilityMarker.Code != -32001 {
		t.Fatalf("capability code = %d, want -32001", capabilityMarker.Code)
	}
	if got := strings.TrimSpace(fmt.Sprint(capabilityMarker.Data["method"])); got != "sessions/create" {
		t.Fatalf("capability denied method = %q, want sessions/create", got)
	}

	hooks := harness.hookCatalog(t)
	if !hookCatalogContains(hooks, "secret-guard-hook", string(hookspkg.HookInputPreSubmit)) {
		t.Fatalf("hook catalog = %#v, want secret-guard input hook", hooks)
	}
	if !hookCatalogContains(hooks, "workspace-context", string(hookspkg.HookPromptPostAssemble)) {
		t.Fatalf("hook catalog = %#v, want prompt-enhancer prompt hook", hooks)
	}

	session := harness.createSession(t)
	if session.WorkspaceID == "" {
		t.Fatal("create session workspace id = empty, want resolved workspace")
	}

	if _, err := harness.promptSession(t, session.ID, "Summarize the current workspace."); err != nil {
		t.Fatalf("safe PromptSession() error = %v", err)
	}

	promptEntries := harness.waitForPromptEntries(t, 1)
	firstPrompt := promptEntries[0].Text
	if !containsFragmentsInOrder(
		firstPrompt,
		"[Workspace: "+harness.workspace.RootDir+"]",
		"You are a coding assistant.",
		"User request:",
		"Summarize the current workspace.",
	) {
		t.Fatalf("first prompt = %q, want workspace prefix plus user request", firstPrompt)
	}

	if _, err := harness.promptSession(t, session.ID, "please keep sk-abc123 safe"); err == nil {
		t.Fatal("secret PromptSession() error = nil, want denied hook error")
	} else if !strings.Contains(err.Error(), "input.pre_submit") && !strings.Contains(strings.ToLower(err.Error()), "denied") {
		t.Fatalf("secret PromptSession() error = %v, want hook denial", err)
	}

	harness.ensurePromptEntryCount(t, 1, 500*time.Millisecond)

	runs := harness.hookRuns(t, session.ID, 32)
	if !hookRunContains(runs, string(hookspkg.HookInputPreSubmit), "denied", "sk-") {
		t.Fatalf("hook runs = %#v, want denied input.pre_submit patch", runs)
	}

	secretBefore, err := harness.extensionStatus("secret-guard")
	if err != nil {
		t.Fatalf("ExtensionStatus(secret-guard) error = %v", err)
	}
	if secretBefore.PID <= 0 {
		t.Fatalf("secret-guard pid = %d, want active process pid", secretBefore.PID)
	}
	startsBefore := len(waitForNonEmptyFileLines(t, harness.secretStartsPath, 10*time.Second))
	if err := syscall.Kill(secretBefore.PID, syscall.SIGKILL); err != nil {
		t.Fatalf("syscall.Kill(%d, SIGKILL) error = %v", secretBefore.PID, err)
	}
	waitForCondition(t, 15*time.Second, "secret-guard restart after crash", func() bool {
		lines, err := readFileLines(harness.secretStartsPath)
		if err != nil {
			return false
		}
		status, statusErr := harness.extensionStatus("secret-guard")
		return statusErr == nil &&
			status.State == "active" &&
			status.Health == "healthy" &&
			status.PID > 0 &&
			status.PID != secretBefore.PID &&
			len(lines) >= startsBefore+1
	})

	harness.shutdown(t)

	secretShutdownLines := waitForNonEmptyFileLines(t, harness.secretShutdownPath, 10*time.Second)
	if len(secretShutdownLines) == 0 {
		t.Fatal("secret shutdown marker = empty, want at least one shutdown line")
	}
	promptShutdownLines := waitForNonEmptyFileLines(t, harness.promptShutdownPath, 10*time.Second)
	if len(promptShutdownLines) == 0 {
		t.Fatal("prompt shutdown marker = empty, want at least one shutdown line")
	}
}

func newReferenceHarness(t *testing.T, repoRoot string) *referenceHarness {
	t.Helper()

	homePaths := referenceHomePaths(t)
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	markersDir := filepath.Join(t.TempDir(), "markers")
	harness := &referenceHarness{
		homePaths: homePaths,
		logBuffer: &bytes.Buffer{},
		repoRoot:  repoRoot,

		promptLogPath: filepath.Join(markersDir, "acp-prompts.jsonl"),

		secretHandshakePath: filepath.Join(markersDir, "secret-handshake.json"),
		secretHostCallPath:  filepath.Join(markersDir, "secret-host-call.json"),
		secretStartsPath:    filepath.Join(markersDir, "secret-starts.log"),
		secretCrashOncePath: filepath.Join(markersDir, "secret-crash-once.json"),
		secretShutdownPath:  filepath.Join(markersDir, "secret-shutdown.log"),

		promptHandshakePath:  filepath.Join(markersDir, "prompt-handshake.json"),
		promptHostCallPath:   filepath.Join(markersDir, "prompt-host-call.json"),
		promptCapabilityPath: filepath.Join(markersDir, "prompt-capability.json"),
		promptShutdownPath:   filepath.Join(markersDir, "prompt-shutdown.log"),
	}

	t.Setenv(referencePromptLogPathEnvKey, harness.promptLogPath)
	t.Setenv("GIN_MODE", "release")
	gin.SetMode(gin.ReleaseMode)
	t.Setenv("AGH_SECRET_GUARD_HANDSHAKE_PATH", harness.secretHandshakePath)
	t.Setenv("AGH_SECRET_GUARD_HOST_CALL_PATH", harness.secretHostCallPath)
	t.Setenv("AGH_SECRET_GUARD_STARTS_PATH", harness.secretStartsPath)
	t.Setenv("AGH_SECRET_GUARD_CRASH_ONCE_PATH", "")
	t.Setenv("AGH_SECRET_GUARD_SHUTDOWN_PATH", harness.secretShutdownPath)
	t.Setenv("AGH_PROMPT_ENHANCER_HANDSHAKE_PATH", harness.promptHandshakePath)
	t.Setenv("AGH_PROMPT_ENHANCER_HOST_CALL_PATH", harness.promptHostCallPath)
	t.Setenv("AGH_PROMPT_ENHANCER_CAPABILITY_PATH", harness.promptCapabilityPath)
	t.Setenv("AGH_PROMPT_ENHANCER_SHUTDOWN_PATH", harness.promptShutdownPath)

	cfg := referenceConfig(t, homePaths)
	referenceWriteAgentDef(t, homePaths, "coder", referenceACPHelperCommand(t))
	harness.workspace = referenceSeedWorkspace(t, homePaths, cfg, filepath.Join(t.TempDir(), "workspace"))

	logger := slog.New(slog.NewTextHandler(harness.logBuffer, nil))
	daemon, err := daemonpkg.New(
		daemonpkg.WithHomePaths(homePaths),
		daemonpkg.WithConfig(&cfg),
		daemonpkg.WithLogger(logger),
	)
	if err != nil {
		t.Fatalf("daemon.New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	harness.daemonCancel = cancel
	harness.daemonErrCh = make(chan error, 1)
	go func() {
		harness.daemonErrCh <- daemon.Run(ctx)
	}()

	client, err := cli.NewClient(homePaths.DaemonSocket)
	if err != nil {
		t.Fatalf("cli.NewClient() error = %v", err)
	}
	harness.client = client
	harness.waitForDaemonReady(t)

	t.Cleanup(func() {
		if t.Failed() && harness.logBuffer.Len() > 0 {
			t.Logf("reference daemon logs:\n%s", harness.logBuffer.String())
		}
		harness.shutdown(t)
	})

	return harness
}

func (h *referenceHarness) installExtension(t *testing.T, relativePath string) cli.ExtensionRecord {
	t.Helper()

	root := filepath.Join(h.repoRoot, relativePath)
	checksum, err := extensionpkg.ComputeDirectoryChecksum(root)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum(%q) error = %v", root, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	record, err := h.client.InstallExtension(ctx, cli.InstallExtensionRequest{
		Path:     root,
		Checksum: checksum,
	})
	if err != nil {
		t.Fatalf("InstallExtension(%q) error = %v", relativePath, err)
	}

	waitForCondition(t, 10*time.Second, "extension active "+record.Name, func() bool {
		status, statusErr := h.extensionStatus(record.Name)
		return statusErr == nil && status.State == "active" && status.Health == "healthy"
	})

	status, err := h.extensionStatus(record.Name)
	if err != nil {
		t.Fatalf("ExtensionStatus(%q) error = %v", record.Name, err)
	}
	return status
}

func (h *referenceHarness) extensionStatus(name string) (cli.ExtensionRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return h.client.ExtensionStatus(ctx, name)
}

func (h *referenceHarness) waitForDaemonReady(t *testing.T) cli.DaemonStatus {
	t.Helper()

	var status cli.DaemonStatus
	waitForCondition(t, 10*time.Second, "daemon ready", func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		current, err := h.client.DaemonStatus(ctx)
		if err != nil {
			return false
		}
		status = current
		return strings.TrimSpace(current.Status) == "running"
	})
	return status
}

func (h *referenceHarness) hookCatalog(t *testing.T) []cli.HookCatalogRecord {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	hooks, err := h.client.HookCatalog(ctx, cli.HookCatalogQuery{})
	if err != nil {
		t.Fatalf("HookCatalog() error = %v", err)
	}
	return hooks
}

func (h *referenceHarness) createSession(t *testing.T) cli.SessionRecord {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := h.client.CreateSession(ctx, cli.CreateSessionRequest{
		AgentName:     "coder",
		WorkspacePath: h.workspace.RootDir,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	return session
}

func (h *referenceHarness) promptSession(t *testing.T, sessionID string, message string) ([]cli.AgentEventRecord, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return h.client.PromptSession(ctx, sessionID, message)
}

func (h *referenceHarness) waitForPromptEntries(t *testing.T, want int) []referencePromptLogEntry {
	t.Helper()

	var entries []referencePromptLogEntry
	waitForCondition(t, 10*time.Second, "prompt log entries", func() bool {
		payload, err := os.ReadFile(h.promptLogPath)
		if err != nil {
			return false
		}
		decoded, decodeErr := decodeJSONLines[referencePromptLogEntry](payload)
		if decodeErr != nil {
			return false
		}
		entries = decoded
		return len(entries) >= want
	})
	return entries
}

func (h *referenceHarness) ensurePromptEntryCount(t *testing.T, want int, duration time.Duration) {
	t.Helper()

	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		payload, err := os.ReadFile(h.promptLogPath)
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		entries, decodeErr := decodeJSONLines[referencePromptLogEntry](payload)
		if decodeErr == nil && len(entries) != want {
			t.Fatalf("prompt entry count = %d, want %d", len(entries), want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (h *referenceHarness) hookRuns(t *testing.T, sessionID string, last int) []cli.HookRunRecord {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	runs, err := h.client.HookRuns(ctx, cli.HookRunsQuery{
		Session: sessionID,
		Last:    last,
	})
	if err != nil {
		t.Fatalf("HookRuns() error = %v", err)
	}
	return runs
}

func (h *referenceHarness) shutdown(t *testing.T) {
	t.Helper()

	if h.daemonCancel == nil {
		return
	}

	h.daemonCancel()
	h.daemonCancel = nil

	if h.daemonErrCh == nil {
		return
	}

	select {
	case err := <-h.daemonErrCh:
		h.daemonErrCh = nil
		if err != nil {
			t.Fatalf("daemon.Run() error = %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for daemon shutdown")
	}
}

func (referenceACPAgent) Authenticate(context.Context, acpsdk.AuthenticateRequest) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (referenceACPAgent) Initialize(context.Context, acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		AgentCapabilities: acpsdk.AgentCapabilities{
			LoadSession: true,
		},
		AuthMethods: []acpsdk.AuthMethod{},
	}, nil
}

func (referenceACPAgent) Cancel(context.Context, acpsdk.CancelNotification) error {
	return nil
}

func (referenceACPAgent) NewSession(context.Context, acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	return acpsdk.NewSessionResponse{
		SessionId: "reference-extension-helper",
	}, nil
}

func (referenceACPAgent) LoadSession(context.Context, acpsdk.LoadSessionRequest) (acpsdk.LoadSessionResponse, error) {
	return acpsdk.LoadSessionResponse{}, nil
}

func (referenceACPAgent) Prompt(_ context.Context, params acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	entry := referencePromptLogEntry{
		SessionID: string(params.SessionId),
		Text:      promptText(params.Prompt),
	}
	if err := appendJSONLine(os.Getenv(referencePromptLogPathEnvKey), entry); err != nil {
		return acpsdk.PromptResponse{}, err
	}
	return acpsdk.PromptResponse{
		StopReason: acpsdk.StopReasonEndTurn,
	}, nil
}

func (referenceACPAgent) SetSessionMode(context.Context, acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

func promptText(blocks []acpsdk.ContentBlock) string {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		switch {
		case block.Text != nil:
			parts = append(parts, block.Text.Text)
		case block.ResourceLink != nil:
			parts = append(parts, block.ResourceLink.Uri)
		}
	}
	return strings.Join(parts, "\n\n")
}

func appendJSONLine(path string, value any) error {
	target := strings.TrimSpace(path)
	if target == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(file, "%s\n", payload)
	return err
}

func buildReferenceArtifacts(t *testing.T, repoRoot string) {
	t.Helper()

	runCommand(t, repoRoot, "go", "build", "-o", "./sdk/examples/secret-guard/bin/secret-guard", "./sdk/examples/secret-guard")
	runCommand(t, repoRoot, "npm", "run", "build", "--workspace", "@agh/extension-sdk")
	runCommand(t, repoRoot, "npm", "run", "build", "--workspace", "@agh/example-prompt-enhancer")
}

func runCommand(t *testing.T, dir string, name string, args ...string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s error = %v\n%s", name, strings.Join(args, " "), err, string(output))
	}
}

func referenceRepoRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func referenceHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	homeDir := t.TempDir()
	t.Setenv("AGH_HOME", homeDir)
	t.Setenv("HOME", homeDir)

	homePaths, err := aghconfig.ResolveHomePathsFrom(homeDir)
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	homePaths.DaemonSocket = referenceShortSocketPath(t)
	return homePaths
}

func referenceConfig(t *testing.T, homePaths aghconfig.HomePaths) aghconfig.Config {
	t.Helper()

	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = referenceFreeTCPPort(t)
	cfg.Daemon.Socket = homePaths.DaemonSocket
	cfg.Defaults.Agent = "coder"
	cfg.Defaults.Provider = "claude"
	cfg.Memory.Enabled = false
	cfg.Skills.Enabled = false
	cfg.Providers["claude"] = aghconfig.ProviderConfig{
		Command: referenceACPHelperCommand(t),
	}
	return cfg
}

func referenceSeedWorkspace(t *testing.T, homePaths aghconfig.HomePaths, cfg aghconfig.Config, root string) workspacepkg.ResolvedWorkspace {
	t.Helper()

	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", root, err)
	}

	db, err := globaldb.OpenGlobalDB(testutil.Context(t), homePaths.DatabaseFile)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	defer func() {
		if err := db.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	resolver, err := workspacepkg.NewResolver(
		db,
		workspacepkg.WithHomePaths(homePaths),
		workspacepkg.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		workspacepkg.WithConfigLoader(func(string) (aghconfig.Config, error) {
			return cfg, nil
		}),
	)
	if err != nil {
		t.Fatalf("workspace.NewResolver() error = %v", err)
	}

	resolved, err := resolver.ResolveOrRegister(testutil.Context(t), root)
	if err != nil {
		t.Fatalf("ResolveOrRegister(%q) error = %v", root, err)
	}

	if resolved.Config.Defaults.Agent != cfg.Defaults.Agent {
		t.Fatalf("resolved default agent = %q, want %q", resolved.Config.Defaults.Agent, cfg.Defaults.Agent)
	}
	return resolved
}

func referenceACPHelperCommand(t *testing.T) string {
	t.Helper()

	bin, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	return shellquote.Join(
		"env",
		referenceACPHelperEnvKey+"=1",
		referencePromptLogPathEnvKey+"="+os.Getenv(referencePromptLogPathEnvKey),
		bin,
		"-test.run=TestReferenceExtensionACPHelperProcess",
	)
}

func referenceWriteAgentDef(t *testing.T, homePaths aghconfig.HomePaths, name string, command string) {
	t.Helper()

	path := filepath.Join(homePaths.AgentsDir, name, "AGENT.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}

	content := strings.Join([]string{
		"---",
		"name: " + name,
		"provider: claude",
		"command: " + command,
		"---",
		"You are a coding assistant.",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func referenceShortSocketPath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(os.TempDir(), fmt.Sprintf("agh-reference-%d.sock", time.Now().UTC().UnixNano()))
	t.Cleanup(func() {
		_ = os.Remove(path)
	})
	return path
}

func referenceFreeTCPPort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen(:0) error = %v", err)
	}
	defer func() {
		_ = ln.Close()
	}()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr type = %T, want *net.TCPAddr", ln.Addr())
	}
	return addr.Port
}

func waitForJSONFile[T any](t *testing.T, path string, timeout time.Duration) T {
	t.Helper()

	var decoded T
	waitForCondition(t, timeout, "json file "+path, func() bool {
		payload, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return false
		}
		return true
	})
	return decoded
}

func waitForHandshakeMarker(t *testing.T, path string, pid int, timeout time.Duration) referenceHandshakeMarker {
	t.Helper()

	var marker referenceHandshakeMarker
	waitForCondition(t, timeout, fmt.Sprintf("handshake marker %s for pid=%d", path, pid), func() bool {
		payload, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		if err := json.Unmarshal(payload, &marker); err != nil {
			return false
		}
		return marker.PID == pid
	})
	return marker
}

func waitForHostCallMarker(t *testing.T, path string, pid int, timeout time.Duration) referenceHostCallMarker {
	t.Helper()

	var marker referenceHostCallMarker
	waitForCondition(t, timeout, fmt.Sprintf("host call marker %s for pid=%d", path, pid), func() bool {
		payload, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		if err := json.Unmarshal(payload, &marker); err != nil {
			return false
		}
		return marker.PID == pid
	})
	return marker
}

func waitForStartedPID(t *testing.T, path string, wantCount int, timeout time.Duration) int {
	t.Helper()

	var latest int
	waitForCondition(t, timeout, fmt.Sprintf("started pid %s count>=%d", path, wantCount), func() bool {
		lines, err := readFileLines(path)
		if err != nil || len(lines) < wantCount {
			return false
		}
		pid, ok := parsePIDLine(lines[len(lines)-1])
		if !ok {
			return false
		}
		latest = pid
		return latest > 0
	})
	return latest
}

func parsePIDLine(line string) (int, bool) {
	var pid int
	if _, err := fmt.Sscanf(strings.TrimSpace(line), "pid=%d", &pid); err != nil {
		return 0, false
	}
	return pid, pid > 0
}

func waitForNonEmptyFileLines(t *testing.T, path string, timeout time.Duration) []string {
	t.Helper()

	var lines []string
	waitForCondition(t, timeout, "non-empty file lines "+path, func() bool {
		current, err := readFileLines(path)
		if err != nil || len(current) == 0 {
			return false
		}
		lines = current
		return true
	})
	return lines
}

func readFileLines(path string) ([]string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(string(payload)), nil
}

func waitForCondition(t *testing.T, timeout time.Duration, label string, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s after %v", label, timeout)
}

func hookCatalogContains(hooks []cli.HookCatalogRecord, name string, event string) bool {
	for _, item := range hooks {
		if item.Name == name && item.Event == event && item.ExecutorKind == "subprocess" {
			return true
		}
	}
	return false
}

func hookRunContains(runs []cli.HookRunRecord, event string, outcome string, fragment string) bool {
	for _, item := range runs {
		if item.Event != event || item.Outcome != outcome {
			continue
		}
		if fragment == "" || strings.Contains(string(item.PatchApplied), fragment) || strings.Contains(item.Error, fragment) {
			return true
		}
	}
	return false
}
