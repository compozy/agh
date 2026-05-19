// Package e2e provides shared runtime and artifact helpers for daemon-level end-to-end tests.
package e2e

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/store"
)

// ArtifactKind identifies one stable E2E diagnostic surface.
type ArtifactKind string

const (
	ArtifactKindTranscript           ArtifactKind = "transcript"
	ArtifactKindEvents               ArtifactKind = "events"
	ArtifactKindTransportOutputs     ArtifactKind = "transport_outputs"
	ArtifactKindNetworkMessages      ArtifactKind = "network_messages"
	ArtifactKindNetworkThreads       ArtifactKind = "network_threads"
	ArtifactKindNetworkDirectRooms   ArtifactKind = "network_direct_rooms"
	ArtifactKindNetworkWork          ArtifactKind = "network_work"
	ArtifactKindNetworkAudit         ArtifactKind = "network_audit"
	ArtifactKindAutomationRuns       ArtifactKind = "automation_runs"
	ArtifactKindTasks                ArtifactKind = "tasks"
	ArtifactKindTaskRuns             ArtifactKind = "task_runs"
	ArtifactKindBridgeHealth         ArtifactKind = "bridge_health"
	ArtifactKindBridgeRoutes         ArtifactKind = "bridge_routes"
	ArtifactKindBridgeDeliveryState  ArtifactKind = "bridge_delivery_state"
	ArtifactKindBridgeSecretBindings ArtifactKind = "bridge_secret_bindings"
	ArtifactKindProviderCalls        ArtifactKind = "provider_calls"
	ArtifactKindToolHostDiagnostics  ArtifactKind = "tool_host_diagnostics"
	ArtifactKindCombinedFlow         ArtifactKind = "combined_flow"
	ArtifactKindSessionSandbox       ArtifactKind = "session_sandbox"
	ArtifactKindBrowserTrace         ArtifactKind = "browser_trace"
	ArtifactKindBrowserScreenshots   ArtifactKind = "browser_screenshots"
	ArtifactKindBrowserConsole       ArtifactKind = "browser_console"
	ArtifactKindBrowserNetwork       ArtifactKind = "browser_network"
)

type artifactSpec struct {
	relativePath string
	isDir        bool
}

type captureFileTarget struct {
	sourcePath string
	targetPath string
}

const defaultArtifactSlug = "run"

var artifactSpecs = map[ArtifactKind]artifactSpec{
	ArtifactKindTranscript:           {relativePath: "transcript.json"},
	ArtifactKindEvents:               {relativePath: "events.json"},
	ArtifactKindTransportOutputs:     {relativePath: string(ArtifactKindTransportOutputs), isDir: true},
	ArtifactKindNetworkMessages:      {relativePath: "network_messages.json"},
	ArtifactKindNetworkThreads:       {relativePath: "network_threads.json"},
	ArtifactKindNetworkDirectRooms:   {relativePath: "network_direct_rooms.json"},
	ArtifactKindNetworkWork:          {relativePath: "network_work.json"},
	ArtifactKindNetworkAudit:         {relativePath: "network_audit.json"},
	ArtifactKindAutomationRuns:       {relativePath: "automation_runs.json"},
	ArtifactKindTasks:                {relativePath: "tasks.json"},
	ArtifactKindTaskRuns:             {relativePath: "task_runs.json"},
	ArtifactKindBridgeHealth:         {relativePath: "bridge_health.json"},
	ArtifactKindBridgeRoutes:         {relativePath: "bridge_routes.json"},
	ArtifactKindBridgeDeliveryState:  {relativePath: "bridge_delivery_state.json"},
	ArtifactKindBridgeSecretBindings: {relativePath: "bridge_secret_bindings.json"},
	ArtifactKindProviderCalls:        {relativePath: "provider_calls.json"},
	ArtifactKindToolHostDiagnostics:  {relativePath: "tool_host_diagnostics.json"},
	ArtifactKindCombinedFlow:         {relativePath: "combined_flow.json"},
	ArtifactKindSessionSandbox:       {relativePath: "session_sandbox.json"},
	ArtifactKindBrowserTrace:         {relativePath: "browser_trace.zip"},
	ArtifactKindBrowserScreenshots:   {relativePath: "browser_screenshots", isDir: true},
	ArtifactKindBrowserConsole:       {relativePath: "browser_console.json"},
	ArtifactKindBrowserNetwork:       {relativePath: "browser_network.json"},
}

// ArtifactEntry records one captured diagnostic artifact.
type ArtifactEntry struct {
	Kind      ArtifactKind `json:"kind"`
	Path      string       `json:"path"`
	MediaType string       `json:"media_type,omitempty"`
}

// ArtifactManifest is the stable per-run artifact index.
type ArtifactManifest struct {
	Version   int             `json:"version"`
	Artifacts []ArtifactEntry `json:"artifacts"`
}

// RuntimeArtifactManifest captures the stable daemon-runtime surfaces later
// integration suites need for debugging and parity assertions.
type RuntimeArtifactManifest struct {
	Version              int                      `json:"version"`
	WorkspaceRoot        string                   `json:"workspace_root,omitempty"`
	Home                 RuntimeHomeArtifact      `json:"home"`
	Logs                 RuntimeLogArtifact       `json:"logs"`
	Runs                 RuntimeRunArtifact       `json:"runs"`
	Transport            RuntimeTransportArtifact `json:"transport"`
	ArtifactRootDir      string                   `json:"artifact_root_dir,omitempty"`
	ArtifactManifestPath string                   `json:"artifact_manifest_path,omitempty"`
	CapturedArtifacts    ArtifactManifest         `json:"captured_artifacts"`
}

// RuntimeHomeArtifact captures the isolated AGH home layout used by a harness.
type RuntimeHomeArtifact struct {
	HomeDir          string `json:"home_dir,omitempty"`
	ConfigFile       string `json:"config_file,omitempty"`
	DatabaseFile     string `json:"database_file,omitempty"`
	DaemonSocket     string `json:"daemon_socket,omitempty"`
	DaemonInfo       string `json:"daemon_info,omitempty"`
	LogsDir          string `json:"logs_dir,omitempty"`
	NetworkAuditFile string `json:"network_audit_file,omitempty"`
}

// RuntimeLogArtifact captures the daemon log surfaces retained by the harness.
type RuntimeLogArtifact struct {
	DaemonLogFile  string `json:"daemon_log_file,omitempty"`
	ProcessLogFile string `json:"process_log_file,omitempty"`
}

// RuntimeRunArtifact captures the stable run-root surfaces retained by the harness.
type RuntimeRunArtifact struct {
	RootDir     string   `json:"root_dir,omitempty"`
	Directories []string `json:"directories,omitempty"`
}

// RuntimeTransportArtifact captures the public transport metadata shared by the harness.
type RuntimeTransportArtifact struct {
	HTTPBaseURL string `json:"http_base_url,omitempty"`
	HTTPHost    string `json:"http_host,omitempty"`
	HTTPPort    int    `json:"http_port,omitempty"`
	UDSBaseURL  string `json:"uds_base_url,omitempty"`
	SocketPath  string `json:"socket_path,omitempty"`
	CLIBinary   string `json:"cli_binary,omitempty"`
	CLIWorkdir  string `json:"cli_workdir,omitempty"`
}

// TransportOutputArtifact records one CLI/HTTP/UDS result retained for later diagnostics.
type TransportOutputArtifact struct {
	Name       string   `json:"name,omitempty"`
	Transport  string   `json:"transport,omitempty"`
	Command    []string `json:"command,omitempty"`
	URL        string   `json:"url,omitempty"`
	Method     string   `json:"method,omitempty"`
	StatusCode int      `json:"status_code,omitempty"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	Error      string   `json:"error,omitempty"`
	Payload    any      `json:"payload,omitempty"`
}

// SessionSandboxArtifact captures both the public session sandbox
// projection and the fuller persisted metadata stored on disk for one session.
type SessionSandboxArtifact struct {
	SessionID    string                             `json:"session_id"`
	SessionState string                             `json:"session_state,omitempty"`
	StopReason   store.StopReason                   `json:"stop_reason,omitempty"`
	StopDetail   string                             `json:"stop_detail,omitempty"`
	API          *aghcontract.SessionSandboxPayload `json:"api,omitempty"`
	Persisted    *store.SessionSandboxMeta          `json:"persisted,omitempty"`
}

// ToolHostOperationOutcome classifies one tool-host operation result.
type ToolHostOperationOutcome string

const (
	ToolHostOutcomeAllowed ToolHostOperationOutcome = "allowed"
	ToolHostOutcomeBlocked ToolHostOperationOutcome = "blocked"
)

// ToolHostOperationDiagnostic records one observed tool-host action.
type ToolHostOperationDiagnostic struct {
	Operation        string                   `json:"operation"`
	Path             string                   `json:"path,omitempty"`
	Outcome          ToolHostOperationOutcome `json:"outcome,omitempty"`
	Error            string                   `json:"error,omitempty"`
	SideEffectPath   string                   `json:"side_effect_path,omitempty"`
	SideEffectExists bool                     `json:"side_effect_exists,omitempty"`
}

// ToolHostDiagnosticsArtifact groups tool-host observations for one session.
type ToolHostDiagnosticsArtifact struct {
	SessionID  string                        `json:"session_id,omitempty"`
	Operations []ToolHostOperationDiagnostic `json:"operations,omitempty"`
}

// CombinedFlowArtifact records the cross-domain identifiers and side effects
// that make a multi-domain failure diagnosable from one retained run.
type CombinedFlowArtifact struct {
	Scenario          string   `json:"scenario"`
	SessionID         string   `json:"session_id,omitempty"`
	Channel           string   `json:"channel,omitempty"`
	AutomationRunID   string   `json:"automation_run_id,omitempty"`
	TriggerID         string   `json:"trigger_id,omitempty"`
	JobID             string   `json:"job_id,omitempty"`
	TaskID            string   `json:"task_id,omitempty"`
	TaskRunID         string   `json:"task_run_id,omitempty"`
	BridgeID          string   `json:"bridge_id,omitempty"`
	NetworkMessageIDs []string `json:"network_message_ids,omitempty"`
	SideEffectPaths   []string `json:"side_effect_paths,omitempty"`
}

// Allowed returns the matching allowed operation diagnostic when present.
func (a ToolHostDiagnosticsArtifact) Allowed(operation string) (ToolHostOperationDiagnostic, bool) {
	return FindAllowedToolHostOperation(a.Operations, operation)
}

// Blocked returns the matching blocked operation diagnostic when present.
func (a ToolHostDiagnosticsArtifact) Blocked(operation string) (ToolHostOperationDiagnostic, bool) {
	return FindBlockedToolHostOperation(a.Operations, operation)
}

// FindAllowedToolHostOperation locates an allowed operation without relying on transcript text.
func FindAllowedToolHostOperation(
	operations []ToolHostOperationDiagnostic,
	operation string,
) (ToolHostOperationDiagnostic, bool) {
	return findToolHostOperation(operations, operation, ToolHostOutcomeAllowed, false)
}

// FindBlockedToolHostOperation locates a blocked operation without relying on transcript text.
func FindBlockedToolHostOperation(
	operations []ToolHostOperationDiagnostic,
	operation string,
) (ToolHostOperationDiagnostic, bool) {
	return findToolHostOperation(operations, operation, ToolHostOutcomeBlocked, true)
}

// ArtifactCollector captures and indexes stable E2E diagnostics.
type ArtifactCollector struct {
	rootDir      string
	manifestPath string

	mu      sync.Mutex
	entries map[ArtifactKind]ArtifactEntry
}

// NewArtifactCollector creates a per-test artifact directory. Passing tests clean it up,
// while failing tests keep it around for inspection.
func NewArtifactCollector(t testing.TB) *ArtifactCollector {
	t.Helper()

	dir, err := os.MkdirTemp("", "agh-e2e-"+sanitizePathComponent(t.Name())+"-")
	if err != nil {
		t.Fatalf("os.MkdirTemp(artifacts) error = %v", err)
	}

	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("retained E2E artifacts at %s", dir)
			return
		}
		_ = os.RemoveAll(dir)
	})

	return &ArtifactCollector{
		rootDir:      dir,
		manifestPath: filepath.Join(dir, "manifest.json"),
		entries:      make(map[ArtifactKind]ArtifactEntry),
	}
}

// RootDir returns the artifact root for the run.
func (c *ArtifactCollector) RootDir() string {
	return c.rootDir
}

// ManifestPath returns the stable manifest location.
func (c *ArtifactCollector) ManifestPath() string {
	return c.manifestPath
}

// ArtifactPath returns the absolute path for one captured artifact kind.
func (c *ArtifactCollector) ArtifactPath(kind ArtifactKind) (string, bool) {
	spec, ok := artifactSpecs[kind]
	if !ok {
		return "", false
	}
	targetPath, err := c.targetPath(spec)
	if err != nil {
		return "", false
	}
	return targetPath, true
}

// CaptureJSON writes one artifact as indented JSON and updates the manifest.
func (c *ArtifactCollector) CaptureJSON(kind ArtifactKind, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s artifact: %w", kind, err)
	}
	data = append(data, '\n')
	return c.captureBytes(kind, data, "application/json")
}

// CaptureText writes one text artifact and updates the manifest.
func (c *ArtifactCollector) CaptureText(kind ArtifactKind, text string) error {
	return c.captureBytes(kind, []byte(text), "text/plain")
}

// CaptureNamedJSON writes one JSON file inside a directory artifact and keeps
// the directory registered in the shared manifest.
func (c *ArtifactCollector) CaptureNamedJSON(kind ArtifactKind, name string, value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal %s named artifact %q: %w", kind, name, err)
	}
	data = append(data, '\n')
	return c.captureNamedBytes(kind, name, ".json", data, "application/json")
}

// CaptureNamedText writes one text file inside a directory artifact and keeps
// the directory registered in the shared manifest.
func (c *ArtifactCollector) CaptureNamedText(kind ArtifactKind, name string, text string) (string, error) {
	return c.captureNamedBytes(kind, name, ".txt", []byte(text), "text/plain")
}

// CaptureFile copies a single file into the canonical artifact location.
func (c *ArtifactCollector) CaptureFile(kind ArtifactKind, sourcePath string, mediaType string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read %s artifact source %q: %w", kind, sourcePath, err)
	}
	return c.captureBytes(kind, data, strings.TrimSpace(mediaType))
}

// CaptureFiles copies one or more files into a canonical directory artifact.
func (c *ArtifactCollector) CaptureFiles(kind ArtifactKind, sourcePaths []string, mediaType string) error {
	spec, err := c.spec(kind)
	if err != nil {
		return err
	}
	if !spec.isDir {
		return fmt.Errorf("artifact %s does not accept multiple files", kind)
	}

	targetDir, err := c.targetPath(spec)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s artifact dir %q: %w", kind, targetDir, err)
	}

	targets, err := captureFileTargets(targetDir, sourcePaths)
	if err != nil {
		return fmt.Errorf("plan %s artifact files: %w", kind, err)
	}
	for _, target := range targets {
		if err := copyFile(target.sourcePath, target.targetPath); err != nil {
			return fmt.Errorf("copy %s artifact %q: %w", kind, target.sourcePath, err)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[kind] = ArtifactEntry{
		Kind:      kind,
		Path:      spec.relativePath,
		MediaType: strings.TrimSpace(mediaType),
	}
	return c.writeManifestLocked()
}

// Manifest returns a stable snapshot of the captured artifacts.
func (c *ArtifactCollector) Manifest() ArtifactManifest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.manifestLocked()
}

// WriteManifest persists the current manifest snapshot and returns it.
func (c *ArtifactCollector) WriteManifest() (ArtifactManifest, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	manifest := c.manifestLocked()
	if err := c.writeManifestLocked(); err != nil {
		return ArtifactManifest{}, err
	}
	return manifest, nil
}

func (c *ArtifactCollector) captureBytes(kind ArtifactKind, data []byte, mediaType string) error {
	spec, err := c.spec(kind)
	if err != nil {
		return err
	}
	if spec.isDir {
		return fmt.Errorf("artifact %s requires file collection", kind)
	}

	targetPath, err := c.targetPath(spec)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s artifact parent %q: %w", kind, filepath.Dir(targetPath), err)
	}
	// #nosec G703 -- targetPath is validated by targetPath() to stay within the collector root.
	if err := os.WriteFile(targetPath, data, 0o600); err != nil {
		return fmt.Errorf("write %s artifact %q: %w", kind, targetPath, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[kind] = ArtifactEntry{
		Kind:      kind,
		Path:      spec.relativePath,
		MediaType: strings.TrimSpace(mediaType),
	}
	return c.writeManifestLocked()
}

func (c *ArtifactCollector) captureNamedBytes(
	kind ArtifactKind,
	name string,
	extension string,
	data []byte,
	mediaType string,
) (string, error) {
	spec, err := c.spec(kind)
	if err != nil {
		return "", err
	}
	if !spec.isDir {
		return "", fmt.Errorf("artifact %s does not accept named entries", kind)
	}

	targetDir, err := c.targetPath(spec)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s artifact dir %q: %w", kind, targetDir, err)
	}

	fileName := sanitizePathComponent(name) + extension
	targetPath := filepath.Join(targetDir, fileName)
	if err := os.WriteFile(targetPath, data, 0o600); err != nil {
		return "", fmt.Errorf("write %s named artifact %q: %w", kind, targetPath, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[kind] = ArtifactEntry{
		Kind:      kind,
		Path:      spec.relativePath,
		MediaType: strings.TrimSpace(mediaType),
	}
	if err := c.writeManifestLocked(); err != nil {
		return "", err
	}
	return targetPath, nil
}

func (c *ArtifactCollector) spec(kind ArtifactKind) (artifactSpec, error) {
	spec, ok := artifactSpecs[kind]
	if !ok {
		return artifactSpec{}, fmt.Errorf("unknown artifact kind %q", kind)
	}
	return spec, nil
}

func (c *ArtifactCollector) targetPath(spec artifactSpec) (string, error) {
	cleanRelativePath := filepath.Clean(spec.relativePath)
	parentPrefix := ".." + string(filepath.Separator)
	if cleanRelativePath == "." || cleanRelativePath == ".." || strings.HasPrefix(cleanRelativePath, parentPrefix) {
		return "", fmt.Errorf("invalid artifact path %q", spec.relativePath)
	}

	targetPath := filepath.Join(c.rootDir, cleanRelativePath)
	relativePath, err := filepath.Rel(c.rootDir, targetPath)
	if err != nil {
		return "", fmt.Errorf("rel artifact path %q: %w", targetPath, err)
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, parentPrefix) {
		return "", fmt.Errorf("artifact path %q escapes root %q", targetPath, c.rootDir)
	}
	return targetPath, nil
}

func (c *ArtifactCollector) manifestLocked() ArtifactManifest {
	items := make([]ArtifactEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		items = append(items, entry)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Path != items[j].Path {
			return items[i].Path < items[j].Path
		}
		return items[i].Kind < items[j].Kind
	})
	return ArtifactManifest{
		Version:   1,
		Artifacts: items,
	}
}

func (c *ArtifactCollector) writeManifestLocked() error {
	manifest := c.manifestLocked()
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal artifact manifest: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(c.manifestPath, data, 0o600); err != nil {
		return fmt.Errorf("write artifact manifest %q: %w", c.manifestPath, err)
	}
	return nil
}

func captureFileTargets(targetDir string, sourcePaths []string) ([]captureFileTarget, error) {
	usedNames := make(map[string]struct{}, len(sourcePaths))
	targets := make([]captureFileTarget, 0, len(sourcePaths))
	for index, sourcePath := range sourcePaths {
		name, err := uniqueCaptureFileName(targetDir, sourcePath, index, usedNames)
		if err != nil {
			return nil, err
		}
		targets = append(targets, captureFileTarget{
			sourcePath: sourcePath,
			targetPath: filepath.Join(targetDir, name),
		})
	}
	return targets, nil
}

func uniqueCaptureFileName(
	targetDir string,
	sourcePath string,
	sourceIndex int,
	usedNames map[string]struct{},
) (string, error) {
	baseName := filepath.Base(sourcePath)
	if baseName == "." || baseName == string(filepath.Separator) {
		return "", fmt.Errorf("source %q does not have a file name", sourcePath)
	}
	for attempt := 0; ; attempt++ {
		candidate := baseName
		if attempt == 1 {
			candidate = fmt.Sprintf("%03d-%s", sourceIndex+1, baseName)
		}
		if attempt > 1 {
			candidate = fmt.Sprintf("%03d-%d-%s", sourceIndex+1, attempt, baseName)
		}
		key := strings.ToLower(candidate)
		if _, ok := usedNames[key]; ok {
			continue
		}
		exists, err := fileExists(filepath.Join(targetDir, candidate))
		if err != nil {
			return "", err
		}
		if exists {
			continue
		}
		usedNames[key] = struct{}{}
		return candidate, nil
	}
}

func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func copyFile(sourcePath string, targetPath string) (err error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := source.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := target.Close()
		if err != nil {
			if removeErr := os.Remove(targetPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove partial artifact %q: %w", targetPath, removeErr))
			}
		}
		if closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = errors.Join(err, closeErr)
			}
		}
	}()

	if _, err := io.Copy(target, source); err != nil {
		return err
	}
	return nil
}

func sanitizePathComponent(value string) string {
	clean := strings.TrimSpace(strings.ToLower(value))
	if clean == "" {
		return defaultArtifactSlug
	}

	var builder strings.Builder
	builder.Grow(len(clean))
	lastDash := false
	for _, r := range clean {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if lastDash || builder.Len() == 0 {
				continue
			}
			builder.WriteByte('-')
			lastDash = true
		}
	}

	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return defaultArtifactSlug
	}
	return result
}

func findToolHostOperation(
	operations []ToolHostOperationDiagnostic,
	operation string,
	outcome ToolHostOperationOutcome,
	requireError bool,
) (ToolHostOperationDiagnostic, bool) {
	wantOperation := strings.TrimSpace(operation)
	for _, item := range operations {
		if strings.TrimSpace(item.Operation) != wantOperation {
			continue
		}
		if item.Outcome != outcome {
			continue
		}
		if requireError && strings.TrimSpace(item.Error) == "" {
			continue
		}
		if !requireError && strings.TrimSpace(item.Error) != "" {
			continue
		}
		return item, true
	}
	return ToolHostOperationDiagnostic{}, false
}
