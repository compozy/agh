// Package e2e provides shared runtime and artifact helpers for daemon-level end-to-end tests.
package e2e

import (
	"encoding/json"
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
	ArtifactKindNetworkMessages      ArtifactKind = "network_messages"
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
	ArtifactKindSessionEnvironment   ArtifactKind = "session_environment"
	ArtifactKindBrowserTrace         ArtifactKind = "browser_trace"
	ArtifactKindBrowserScreenshots   ArtifactKind = "browser_screenshots"
	ArtifactKindBrowserConsole       ArtifactKind = "browser_console"
	ArtifactKindBrowserNetwork       ArtifactKind = "browser_network"
)

type artifactSpec struct {
	relativePath string
	isDir        bool
}

const defaultArtifactSlug = "run"

var artifactSpecs = map[ArtifactKind]artifactSpec{
	ArtifactKindTranscript:           {relativePath: "transcript.json"},
	ArtifactKindEvents:               {relativePath: "events.json"},
	ArtifactKindNetworkMessages:      {relativePath: "network_messages.json"},
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
	ArtifactKindSessionEnvironment:   {relativePath: "session_environment.json"},
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

// SessionEnvironmentArtifact captures both the public session environment
// projection and the fuller persisted metadata stored on disk for one session.
type SessionEnvironmentArtifact struct {
	SessionID    string                                 `json:"session_id"`
	SessionState string                                 `json:"session_state,omitempty"`
	StopReason   store.StopReason                       `json:"stop_reason,omitempty"`
	StopDetail   string                                 `json:"stop_detail,omitempty"`
	API          *aghcontract.SessionEnvironmentPayload `json:"api,omitempty"`
	Persisted    *store.SessionEnvironmentMeta          `json:"persisted,omitempty"`
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

	for _, sourcePath := range sourcePaths {
		targetPath := filepath.Join(targetDir, filepath.Base(sourcePath))
		if err := copyFile(sourcePath, targetPath); err != nil {
			return fmt.Errorf("copy %s artifact %q: %w", kind, sourcePath, err)
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

func copyFile(sourcePath string, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	target, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer func() { _ = target.Close() }()

	if _, err := io.Copy(target, source); err != nil {
		return err
	}

	return target.Close()
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
