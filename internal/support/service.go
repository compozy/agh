// Package support builds daemon-owned support bundles and tracks bundle operations.
package support

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/version"
)

const (
	bundlesDirName              = "support-bundles"
	manifestSchemaVersion       = "support-bundle.v1"
	redactionVersion            = "diagnostics-redaction.v1"
	defaultBundleMaxBytes       = int64(25 << 20)
	defaultArtifactMaxBytes     = int64(1 << 20)
	defaultLogTailMaxBytes      = int64(2 << 20)
	defaultEventSummaryMaxBytes = int64(5 << 20)
	defaultOperationRetention   = 24 * time.Hour
	truncatedJSONArtifact       = `{"truncated":true,"reason":"artifact exceeded byte cap"}` + "\n"
)

var (
	ErrOperationNotFound = errors.New("support: bundle operation not found")
	ErrOperationNotReady = errors.New("support: bundle operation is not ready for download")
)

type OperationStatus string

const (
	OperationPending   OperationStatus = "pending"
	OperationRunning   OperationStatus = "running"
	OperationCompleted OperationStatus = "completed"
	OperationFailed    OperationStatus = "failed"
)

type CreateRequest struct {
	IncludeStatus bool
}

type Operation struct {
	OperationID   string
	Status        OperationStatus
	FileName      string
	FilePath      string
	SizeBytes     int64
	Manifest      *Manifest
	FailureReason string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CompletedAt   *time.Time
}

type Manifest struct {
	SchemaVersion        string             `json:"schema_version"`
	OperationID          string             `json:"operation_id"`
	CreatedAt            time.Time          `json:"created_at"`
	BundleMaxBytes       int64              `json:"bundle_max_bytes"`
	ArtifactMaxBytes     int64              `json:"artifact_max_bytes"`
	LogTailMaxBytes      int64              `json:"log_tail_max_bytes"`
	EventSummaryMaxBytes int64              `json:"event_summary_max_bytes"`
	RedactionVersion     string             `json:"redaction_version"`
	Artifacts            []ManifestArtifact `json:"artifacts"`
}

type ManifestArtifact struct {
	Path             string `json:"path"`
	Included         bool   `json:"included"`
	OmittedReason    string `json:"omitted_reason,omitempty"`
	Bytes            int64  `json:"bytes"`
	Truncated        bool   `json:"truncated"`
	RedactionVersion string `json:"redaction_version,omitempty"`
}

type SnapshotFunc func(context.Context) (any, error)

type Sources struct {
	Status             SnapshotFunc
	Doctor             SnapshotFunc
	Providers          SnapshotFunc
	ConfigApplyRecords SnapshotFunc
	EventSummaries     SnapshotFunc
	Sessions           SnapshotFunc
}

type Builder struct {
	HomePaths            aghconfig.HomePaths
	Config               aghconfig.Config
	Sources              Sources
	Now                  func() time.Time
	BundleMaxBytes       int64
	ArtifactMaxBytes     int64
	LogTailMaxBytes      int64
	EventSummaryMaxBytes int64
}

type Service struct {
	builder   *Builder
	store     *operationStore
	now       func() time.Time
	retention time.Duration
}

func NewService(builder *Builder) *Service {
	if builder == nil {
		builder = &Builder{}
	}
	now := builder.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	builder.Now = now
	return &Service{
		builder:   builder,
		store:     newOperationStore(now),
		now:       now,
		retention: defaultOperationRetention,
	}
}

func BundlesDir(paths aghconfig.HomePaths) string {
	return filepath.Join(paths.HomeDir, bundlesDirName)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Operation, error) {
	if s == nil {
		return Operation{}, errors.New("support: service is required")
	}
	s.store.cleanup(s.retention)
	operationID := uuid.NewString()
	op := s.store.create(operationID)
	runCtx, cancel := detachedContext(ctx)
	go func() {
		defer cancel()
		s.run(runCtx, operationID, req)
	}()
	return op, nil
}

func (s *Service) Get(_ context.Context, operationID string) (Operation, error) {
	if s == nil {
		return Operation{}, errors.New("support: service is required")
	}
	return s.store.get(operationID)
}

func (s *Service) DownloadPath(ctx context.Context, operationID string) (Operation, string, error) {
	op, err := s.Get(ctx, operationID)
	if err != nil {
		return Operation{}, "", err
	}
	if op.Status != OperationCompleted {
		return Operation{}, "", ErrOperationNotReady
	}
	if strings.TrimSpace(op.FilePath) == "" {
		return Operation{}, "", ErrOperationNotReady
	}
	if _, err := os.Stat(op.FilePath); err != nil {
		return Operation{}, "", fmt.Errorf("support: stat bundle %q: %w", op.FilePath, err)
	}
	return op, op.FilePath, nil
}

func (s *Service) run(ctx context.Context, operationID string, req CreateRequest) {
	s.store.markRunning(operationID)
	result, err := s.builder.Build(ctx, operationID, req)
	if err != nil {
		s.store.markFailed(operationID, diagnostics.RedactAndBound(err.Error(), 4096))
		return
	}
	s.store.markCompleted(operationID, result)
}

type bundleFile struct {
	fileName string
	path     string
	tmpPath  string
	file     *os.File
}

func (b *Builder) Build(ctx context.Context, operationID string, req CreateRequest) (operation Operation, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := b.nowUTC()
	bundle, err := b.openBundleFile(now)
	if err != nil {
		return Operation{}, err
	}
	committed := false
	defer func() {
		if !committed {
			if removeErr := os.Remove(bundle.tmpPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				removeErr = fmt.Errorf("support: remove temporary bundle %q: %w", bundle.tmpPath, removeErr)
				if err == nil {
					err = removeErr
				} else {
					err = errors.Join(err, removeErr)
				}
			}
		}
	}()

	gzipWriter := gzip.NewWriter(bundle.file)
	tarWriter := tar.NewWriter(gzipWriter)
	writer := bundleArchiveWriter{tar: tarWriter, maxBytes: b.bundleMaxBytes(), now: now}
	manifest := b.newManifest(operationID, now)
	b.addArtifacts(ctx, &writer, &manifest, req)
	if err := writer.addManifestJSON(&manifest, b.artifactMaxBytes()); err != nil {
		return Operation{}, err
	}

	size, err := b.closeAndCommitBundle(bundle, tarWriter, gzipWriter)
	if err != nil {
		return Operation{}, err
	}
	committed = true
	completedAt := b.nowUTC()
	return Operation{
		OperationID: operationID,
		Status:      OperationCompleted,
		FileName:    bundle.fileName,
		FilePath:    bundle.path,
		SizeBytes:   size,
		Manifest:    &manifest,
		CreatedAt:   now,
		UpdatedAt:   completedAt,
		CompletedAt: &completedAt,
	}, nil
}

func (b *Builder) openBundleFile(now time.Time) (bundleFile, error) {
	dir := BundlesDir(b.HomePaths)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return bundleFile{}, fmt.Errorf("support: create bundle directory: %w", err)
	}
	fileName := fmt.Sprintf("agh-support-bundle-%s.tar.gz", now.Format("20060102T150405Z"))
	path := filepath.Join(dir, fileName)
	tmpPath := path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return bundleFile{}, fmt.Errorf("support: create bundle file: %w", err)
	}
	return bundleFile{fileName: fileName, path: path, tmpPath: tmpPath, file: file}, nil
}

func (b *Builder) newManifest(operationID string, now time.Time) Manifest {
	return Manifest{
		SchemaVersion:        manifestSchemaVersion,
		OperationID:          operationID,
		CreatedAt:            now,
		BundleMaxBytes:       b.bundleMaxBytes(),
		ArtifactMaxBytes:     b.artifactMaxBytes(),
		LogTailMaxBytes:      b.logTailMaxBytes(),
		EventSummaryMaxBytes: b.eventSummaryMaxBytes(),
		RedactionVersion:     redactionVersion,
		Artifacts:            []ManifestArtifact{},
	}
}

func (b *Builder) addArtifacts(
	ctx context.Context,
	writer *bundleArchiveWriter,
	manifest *Manifest,
	req CreateRequest,
) {
	artifactMax := b.artifactMaxBytes()
	b.addSnapshotArtifact(ctx, writer, manifest, "status.json", req.IncludeStatus, b.Sources.Status, artifactMax)
	b.addSnapshotArtifact(ctx, writer, manifest, "doctor.json", true, b.Sources.Doctor, artifactMax)
	b.addSnapshotArtifact(ctx, writer, manifest, "providers.json", true, b.Sources.Providers, artifactMax)
	b.addSnapshotArtifact(
		ctx,
		writer,
		manifest,
		"config-apply-records.json",
		true,
		b.Sources.ConfigApplyRecords,
		artifactMax,
	)
	b.addSnapshotArtifact(
		ctx,
		writer,
		manifest,
		"event-summaries.json",
		true,
		b.Sources.EventSummaries,
		b.eventSummaryMaxBytes(),
	)
	b.addSnapshotArtifact(ctx, writer, manifest, "sessions.json", true, b.Sources.Sessions, artifactMax)
	b.addConfigArtifact(writer, manifest)
	b.addLogTailArtifact(writer, manifest)
	b.addVersionsArtifact(writer, manifest)
	b.addHomeTreeArtifact(writer, manifest)
}

func (b *Builder) closeAndCommitBundle(
	bundle bundleFile,
	tarWriter *tar.Writer,
	gzipWriter *gzip.Writer,
) (int64, error) {
	if err := tarWriter.Close(); err != nil {
		return 0, closeBundleFileAfterWriterError(bundle.file, "tar", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return 0, closeBundleFileAfterWriterError(bundle.file, "gzip", err)
	}
	if err := bundle.file.Close(); err != nil {
		return 0, fmt.Errorf("support: close bundle file: %w", err)
	}
	info, err := os.Stat(bundle.tmpPath)
	if err != nil {
		return 0, fmt.Errorf("support: stat bundle file: %w", err)
	}
	size := info.Size()
	if size > b.bundleMaxBytes() {
		return 0, fmt.Errorf("support: bundle size %d exceeds cap %d", size, b.bundleMaxBytes())
	}
	if err := os.Rename(bundle.tmpPath, bundle.path); err != nil {
		return 0, fmt.Errorf("support: finalize bundle file: %w", err)
	}
	return size, nil
}

func closeBundleFileAfterWriterError(file *os.File, stage string, err error) error {
	if closeErr := file.Close(); closeErr != nil {
		err = errors.Join(err, fmt.Errorf("support: close bundle file after %s error: %w", stage, closeErr))
	}
	return fmt.Errorf("support: close %s writer: %w", stage, err)
}

func detachedContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.Background(), func() {}
	}
	detached := context.WithoutCancel(ctx)
	if deadline, ok := ctx.Deadline(); ok {
		return context.WithDeadline(detached, deadline)
	}
	return detached, func() {}
}

func (b *Builder) nowUTC() time.Time {
	if b.Now == nil {
		return time.Now().UTC()
	}
	return b.Now().UTC()
}

func (b *Builder) bundleMaxBytes() int64 {
	if b.BundleMaxBytes > 0 {
		return b.BundleMaxBytes
	}
	return defaultBundleMaxBytes
}

func (b *Builder) artifactMaxBytes() int64 {
	if b.ArtifactMaxBytes > 0 {
		return b.ArtifactMaxBytes
	}
	return defaultArtifactMaxBytes
}

func (b *Builder) logTailMaxBytes() int64 {
	if b.LogTailMaxBytes > 0 {
		return b.LogTailMaxBytes
	}
	return defaultLogTailMaxBytes
}

func (b *Builder) eventSummaryMaxBytes() int64 {
	if b.EventSummaryMaxBytes > 0 {
		return b.EventSummaryMaxBytes
	}
	return defaultEventSummaryMaxBytes
}

func (b *Builder) addSnapshotArtifact(
	ctx context.Context,
	writer *bundleArchiveWriter,
	manifest *Manifest,
	path string,
	enabled bool,
	snapshot SnapshotFunc,
	maxBytes int64,
) {
	if !enabled {
		manifest.omit(path, "disabled by request")
		return
	}
	if snapshot == nil {
		manifest.omit(path, "source unavailable")
		return
	}
	value, err := snapshot(ctx)
	if err != nil {
		manifest.omit(path, diagnostics.RedactAndBound(err.Error(), 512))
		return
	}
	if err := writer.addJSON(path, value, maxBytes, true, manifest); err != nil {
		manifest.omit(path, diagnostics.RedactAndBound(err.Error(), 512))
	}
}

func (b *Builder) addConfigArtifact(writer *bundleArchiveWriter, manifest *Manifest) {
	if err := writer.addJSON("config-redacted.json", b.Config, b.artifactMaxBytes(), true, manifest); err != nil {
		manifest.omit("config-redacted.json", diagnostics.RedactAndBound(err.Error(), 512))
	}
}

func (b *Builder) addLogTailArtifact(writer *bundleArchiveWriter, manifest *Manifest) {
	path := strings.TrimSpace(b.HomePaths.LogFile)
	if path == "" {
		manifest.omit("logs-tail.txt", "log file path unavailable")
		return
	}
	data, truncated, err := readTail(path, b.logTailMaxBytes())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			manifest.omit("logs-tail.txt", "log file not found")
			return
		}
		manifest.omit("logs-tail.txt", diagnostics.RedactAndBound(err.Error(), 512))
		return
	}
	redacted := []byte(diagnostics.RedactAndBound(string(data), int(b.logTailMaxBytes())))
	if err := writer.addBytes("logs-tail.txt", redacted, truncated, redactionVersion, manifest); err != nil {
		manifest.omit("logs-tail.txt", diagnostics.RedactAndBound(err.Error(), 512))
	}
}

func (b *Builder) addVersionsArtifact(writer *bundleArchiveWriter, manifest *Manifest) {
	value := map[string]any{
		"agh":       version.Current(),
		"generated": b.nowUTC(),
	}
	if err := writer.addJSON("versions.json", value, b.artifactMaxBytes(), false, manifest); err != nil {
		manifest.omit("versions.json", diagnostics.RedactAndBound(err.Error(), 512))
	}
}

func (b *Builder) addHomeTreeArtifact(writer *bundleArchiveWriter, manifest *Manifest) {
	entries, err := collectHomeTree(b.HomePaths.HomeDir, BundlesDir(b.HomePaths), 2000)
	if err != nil {
		manifest.omit("home-tree.json", diagnostics.RedactAndBound(err.Error(), 512))
		return
	}
	if err := writer.addJSON("home-tree.json", entries, b.artifactMaxBytes(), true, manifest); err != nil {
		manifest.omit("home-tree.json", diagnostics.RedactAndBound(err.Error(), 512))
	}
}

type bundleArchiveWriter struct {
	tar      *tar.Writer
	maxBytes int64
	written  int64
	now      time.Time
}

func (w *bundleArchiveWriter) addJSON(
	path string,
	value any,
	maxBytes int64,
	redact bool,
	manifest *Manifest,
) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("support: marshal %s: %w", path, err)
	}
	data = append(data, '\n')
	if redact {
		data = []byte(diagnostics.Redact(string(data)))
	}
	truncated := false
	if int64(len(data)) > maxBytes {
		data = []byte(truncatedJSONArtifact)
		truncated = true
	}
	redaction := ""
	if redact {
		redaction = redactionVersion
	}
	return w.addBytes(path, data, truncated, redaction, manifest)
}

func (w *bundleArchiveWriter) addBytes(
	path string,
	data []byte,
	truncated bool,
	redaction string,
	manifest *Manifest,
) error {
	if w == nil || w.tar == nil {
		return errors.New("support: archive writer is required")
	}
	if strings.TrimSpace(path) == "" {
		return errors.New("support: artifact path is required")
	}
	if w.maxBytes > 0 && w.written+int64(len(data)) > w.maxBytes {
		return fmt.Errorf("support: artifact %s exceeds bundle cap", path)
	}
	header := &tar.Header{Name: path, Mode: 0o600, Size: int64(len(data)), ModTime: w.now}
	if err := w.tar.WriteHeader(header); err != nil {
		return fmt.Errorf("support: write %s header: %w", path, err)
	}
	if _, err := w.tar.Write(data); err != nil {
		return fmt.Errorf("support: write %s content: %w", path, err)
	}
	w.written += int64(len(data))
	manifest.Artifacts = append(manifest.Artifacts, ManifestArtifact{
		Path:             path,
		Included:         true,
		Bytes:            int64(len(data)),
		Truncated:        truncated,
		RedactionVersion: redaction,
	})
	return nil
}

func (w *bundleArchiveWriter) addManifestJSON(manifest *Manifest, maxBytes int64) error {
	if manifest == nil {
		return errors.New("support: manifest is required")
	}
	const path = "manifest.json"
	manifest.upsertArtifact(ManifestArtifact{Path: path, Included: true})
	var data []byte
	truncated := false
	for range 8 {
		encoded, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return fmt.Errorf("support: marshal %s: %w", path, err)
		}
		encoded = append(encoded, '\n')
		truncated = false
		if int64(len(encoded)) > maxBytes {
			encoded = []byte(truncatedJSONArtifact)
			truncated = true
		}
		previous := manifest.artifact(path)
		previousBytes := int64(0)
		previousTruncated := false
		previousFound := false
		if previous != nil {
			previousBytes = previous.Bytes
			previousTruncated = previous.Truncated
			previousFound = true
		}
		entry := ManifestArtifact{
			Path:      path,
			Included:  true,
			Bytes:     int64(len(encoded)),
			Truncated: truncated,
		}
		manifest.upsertArtifact(entry)
		data = encoded
		if previousFound && previousBytes == entry.Bytes && previousTruncated == entry.Truncated {
			break
		}
	}
	return w.writeBytes(path, data)
}

func (w *bundleArchiveWriter) writeBytes(path string, data []byte) error {
	if w == nil || w.tar == nil {
		return errors.New("support: archive writer is required")
	}
	if strings.TrimSpace(path) == "" {
		return errors.New("support: artifact path is required")
	}
	if w.maxBytes > 0 && w.written+int64(len(data)) > w.maxBytes {
		return fmt.Errorf("support: artifact %s exceeds bundle cap", path)
	}
	header := &tar.Header{Name: path, Mode: 0o600, Size: int64(len(data)), ModTime: w.now}
	if err := w.tar.WriteHeader(header); err != nil {
		return fmt.Errorf("support: write %s header: %w", path, err)
	}
	if _, err := w.tar.Write(data); err != nil {
		return fmt.Errorf("support: write %s content: %w", path, err)
	}
	w.written += int64(len(data))
	return nil
}

func (m *Manifest) artifact(path string) *ManifestArtifact {
	if m == nil {
		return nil
	}
	for i := range m.Artifacts {
		if m.Artifacts[i].Path == path {
			return &m.Artifacts[i]
		}
	}
	return nil
}

func (m *Manifest) upsertArtifact(entry ManifestArtifact) {
	if m == nil {
		return
	}
	for i := range m.Artifacts {
		if m.Artifacts[i].Path == entry.Path {
			m.Artifacts[i] = entry
			return
		}
	}
	m.Artifacts = append(m.Artifacts, entry)
}

func (m *Manifest) omit(path string, reason string) {
	if m == nil {
		return
	}
	m.Artifacts = append(m.Artifacts, ManifestArtifact{
		Path:          strings.TrimSpace(path),
		Included:      false,
		OmittedReason: strings.TrimSpace(reason),
	})
}

func readTail(path string, maxBytes int64) (data []byte, truncated bool, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			closeErr = fmt.Errorf("support: close log tail file: %w", closeErr)
			if err == nil {
				err = closeErr
				return
			}
			err = errors.Join(err, closeErr)
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("support: stat log tail: %w", err)
	}
	if maxBytes <= 0 || info.Size() <= maxBytes {
		data, err := io.ReadAll(file)
		if err != nil {
			return nil, false, fmt.Errorf("support: read log tail: %w", err)
		}
		return data, false, nil
	}
	if _, err := file.Seek(-maxBytes, io.SeekEnd); err != nil {
		return nil, false, fmt.Errorf("support: seek log tail: %w", err)
	}
	data, err = io.ReadAll(file)
	if err != nil {
		return nil, false, fmt.Errorf("support: read bounded log tail: %w", err)
	}
	return data, true, nil
}

type HomeTreeEntry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Size int64  `json:"size"`
	Mode string `json:"mode"`
}

func collectHomeTree(root string, supportDir string, limit int) ([]HomeTreeEntry, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("support: home directory is required")
	}
	var entries []HomeTreeEntry
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if len(entries) >= limit {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if samePath(path, supportDir) && path != root {
			return filepath.SkipDir
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			rel = ""
		}
		entries = append(entries, HomeTreeEntry{
			Path: filepath.ToSlash(rel),
			Kind: fileKind(info),
			Size: info.Size(),
			Mode: info.Mode().String(),
		})
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("support: collect home tree: %w", walkErr)
	}
	sort.SliceStable(entries, func(i int, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func samePath(left string, right string) bool {
	if strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return false
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func fileKind(info os.FileInfo) string {
	mode := info.Mode()
	switch {
	case mode.IsDir():
		return "dir"
	case mode.IsRegular():
		return "file"
	case mode&os.ModeSymlink != 0:
		return "symlink"
	default:
		return "other"
	}
}

type operationStore struct {
	mu  sync.RWMutex
	now func() time.Time
	ops map[string]Operation
}

func newOperationStore(now func() time.Time) *operationStore {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &operationStore{now: now, ops: make(map[string]Operation)}
}

func (s *operationStore) create(operationID string) Operation {
	now := s.now().UTC()
	op := Operation{OperationID: operationID, Status: OperationPending, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	s.ops[operationID] = op
	s.mu.Unlock()
	return op
}

func (s *operationStore) get(operationID string) (Operation, error) {
	s.mu.RLock()
	op, ok := s.ops[strings.TrimSpace(operationID)]
	s.mu.RUnlock()
	if !ok {
		return Operation{}, ErrOperationNotFound
	}
	return cloneOperation(op), nil
}

func (s *operationStore) markRunning(operationID string) {
	s.update(operationID, func(op *Operation, now time.Time) { op.Status = OperationRunning; op.UpdatedAt = now })
}

func (s *operationStore) markCompleted(operationID string, result Operation) {
	s.update(operationID, func(op *Operation, _ time.Time) { *op = cloneOperation(result) })
}

func (s *operationStore) markFailed(operationID string, reason string) {
	s.update(operationID, func(op *Operation, now time.Time) {
		op.Status = OperationFailed
		op.FailureReason = strings.TrimSpace(reason)
		op.UpdatedAt = now
		op.CompletedAt = &now
	})
}

func (s *operationStore) update(operationID string, fn func(*Operation, time.Time)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	op, ok := s.ops[strings.TrimSpace(operationID)]
	if !ok {
		return
	}
	now := s.now().UTC()
	fn(&op, now)
	s.ops[operationID] = op
}

func (s *operationStore) cleanup(retention time.Duration) {
	if retention <= 0 {
		return
	}
	cutoff := s.now().UTC().Add(-retention)
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, op := range s.ops {
		if op.CompletedAt == nil || op.CompletedAt.After(cutoff) {
			continue
		}
		if strings.TrimSpace(op.FilePath) != "" {
			if removeErr := os.Remove(op.FilePath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				continue
			}
		}
		delete(s.ops, id)
	}
}

func cloneOperation(op Operation) Operation {
	if op.Manifest != nil {
		manifest := *op.Manifest
		manifest.Artifacts = append([]ManifestArtifact(nil), op.Manifest.Artifacts...)
		op.Manifest = &manifest
	}
	if op.CompletedAt != nil {
		completed := *op.CompletedAt
		op.CompletedAt = &completed
	}
	return op
}
