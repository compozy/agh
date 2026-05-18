package extractor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/fileutil"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"
)

const (
	inboxDirName        = "_inbox"
	systemDirName       = "_system"
	processingSuffix    = ".processing"
	inboxFilePerm       = 0o644
	inboxDirPerm        = 0o755
	defaultScannerLimit = 1024 * 1024
)

// ProposalSink is the controller-backed handoff used by the inbox consumer.
type ProposalSink interface {
	ProposeCandidate(context.Context, memcontract.Candidate) (memcontract.Decision, error)
}

// EventSink persists extractor telemetry.
type EventSink interface {
	RecordExtractorEvent(context.Context, Event) error
}

// Producer writes extractor candidates into the daemon-owned inbox.
type Producer struct {
	root      string
	inboxRoot string
	now       func() time.Time
}

// ProducerOption customizes inbox production.
type ProducerOption func(*Producer)

// WithProducerInboxPath overrides the default <root>/_inbox directory.
func WithProducerInboxPath(path string) ProducerOption {
	return func(p *Producer) {
		if strings.TrimSpace(path) != "" {
			p.inboxRoot = filepath.Clean(path)
		}
	}
}

// NewProducer constructs an inbox producer rooted at the memory directory.
func NewProducer(root string, now func() time.Time, opts ...ProducerOption) (*Producer, error) {
	clean, err := cleanRoot(root)
	if err != nil {
		return nil, err
	}
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	producer := &Producer{
		root:      clean,
		inboxRoot: filepath.Join(clean, inboxDirName),
		now:       now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(producer)
		}
	}
	return producer, nil
}

// Write persists one JSONL inbox file for a completed extractor turn.
func (p *Producer) Write(
	ctx context.Context,
	turn memcontract.TurnRecord,
	candidates []memcontract.Candidate,
) (string, int, error) {
	if p == nil {
		return "", 0, errors.New("memory extractor: producer is required")
	}
	if ctx == nil {
		return "", 0, errors.New("memory extractor: write context is required")
	}
	if err := ctx.Err(); err != nil {
		return "", 0, fmt.Errorf("memory extractor: write canceled: %w", err)
	}
	if len(candidates) == 0 {
		return "", 0, nil
	}
	turn, err := normalizeTurn(turn, p.now)
	if err != nil {
		return "", 0, err
	}
	sessionDir, err := inboxSessionDir(p.inboxRoot, turn.SessionID)
	if err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(sessionDir, inboxDirPerm); err != nil {
		return "", 0, fmt.Errorf("memory extractor: ensure inbox session dir: %w", err)
	}

	var lines bytes.Buffer
	for idx := range candidates {
		candidate := enrichCandidate(candidates[idx], turn, p.now)
		encoded, err := json.Marshal(candidate)
		if err != nil {
			return "", 0, fmt.Errorf("memory extractor: encode candidate: %w", err)
		}
		if _, err := lines.Write(encoded); err != nil {
			return "", 0, fmt.Errorf("memory extractor: buffer candidate: %w", err)
		}
		if err := lines.WriteByte('\n'); err != nil {
			return "", 0, fmt.Errorf("memory extractor: buffer candidate newline: %w", err)
		}
	}

	path := filepath.Join(sessionDir, inboxFilename(p.now(), turn.UntilMessageSeq))
	if err := fileutil.AtomicWriteFile(path, lines.Bytes(), inboxFilePerm); err != nil {
		return "", 0, fmt.Errorf("memory extractor: write inbox file: %w", err)
	}
	return path, len(candidates), nil
}

// InboxConsumer drains daemon-owned extractor inbox files through the write controller.
type InboxConsumer struct {
	root        string
	inboxRoot   string
	failuresDir string
	sink        ProposalSink
	events      EventSink
	logger      *slog.Logger
	now         func() time.Time
}

// ConsumerOption customizes inbox consumption.
type ConsumerOption func(*InboxConsumer)

// WithConsumerEventSink records consumer telemetry.
func WithConsumerEventSink(sink EventSink) ConsumerOption {
	return func(c *InboxConsumer) {
		c.events = sink
	}
}

// WithConsumerLogger configures warning output.
func WithConsumerLogger(logger *slog.Logger) ConsumerOption {
	return func(c *InboxConsumer) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// WithConsumerClock injects deterministic time.
func WithConsumerClock(now func() time.Time) ConsumerOption {
	return func(c *InboxConsumer) {
		if now != nil {
			c.now = now
		}
	}
}

// WithConsumerInboxPath overrides the default <root>/_inbox directory.
func WithConsumerInboxPath(path string) ConsumerOption {
	return func(c *InboxConsumer) {
		if strings.TrimSpace(path) != "" {
			c.inboxRoot = filepath.Clean(path)
		}
	}
}

// WithConsumerFailurePath overrides the default <root>/_system/extractor/failures directory.
func WithConsumerFailurePath(path string) ConsumerOption {
	return func(c *InboxConsumer) {
		if strings.TrimSpace(path) != "" {
			c.failuresDir = filepath.Clean(path)
		}
	}
}

// NewInboxConsumer constructs a FIFO inbox consumer.
func NewInboxConsumer(root string, sink ProposalSink, opts ...ConsumerOption) (*InboxConsumer, error) {
	clean, err := cleanRoot(root)
	if err != nil {
		return nil, err
	}
	if sink == nil {
		return nil, errors.New("memory extractor: proposal sink is required")
	}
	consumer := &InboxConsumer{
		root:        clean,
		inboxRoot:   filepath.Join(clean, inboxDirName),
		failuresDir: filepath.Join(clean, systemDirName, "extractor", "failures"),
		sink:        sink,
		logger:      slog.Default(),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(consumer)
		}
	}
	return consumer, nil
}

// ConsumeResult summarizes one inbox pass.
type ConsumeResult struct {
	Files     int
	Proposed  int
	Failed    int
	Decisions []memcontract.Decision
	Failures  []string
}

// ConsumeOnce processes all currently visible JSONL files in FIFO order.
func (c *InboxConsumer) ConsumeOnce(ctx context.Context) (ConsumeResult, error) {
	if c == nil {
		return ConsumeResult{}, errors.New("memory extractor: consumer is required")
	}
	if ctx == nil {
		return ConsumeResult{}, errors.New("memory extractor: consume context is required")
	}
	files, err := c.pendingFiles()
	if err != nil {
		return ConsumeResult{}, err
	}
	result := ConsumeResult{Decisions: make([]memcontract.Decision, 0), Failures: make([]string, 0)}
	var joined error
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return result, fmt.Errorf("memory extractor: consume canceled: %w", err)
		}
		result.Files++
		fileResult, fileErr := c.consumeFile(ctx, file)
		result.Proposed += fileResult.Proposed
		result.Failed += fileResult.Failed
		result.Decisions = append(result.Decisions, fileResult.Decisions...)
		result.Failures = append(result.Failures, fileResult.Failures...)
		if fileErr != nil {
			joined = errors.Join(joined, fileErr)
		}
	}
	return result, joined
}

type pendingFile struct {
	path    string
	modTime time.Time
	claimed bool
}

func (c *InboxConsumer) pendingFiles() ([]pendingFile, error) {
	sessionDirs, err := os.ReadDir(c.inboxRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []pendingFile{}, nil
		}
		return nil, fmt.Errorf("memory extractor: read inbox root: %w", err)
	}
	files := make([]pendingFile, 0)
	for _, sessionDir := range sessionDirs {
		if !sessionDir.IsDir() {
			continue
		}
		dirPath := filepath.Join(c.inboxRoot, sessionDir.Name())
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, fmt.Errorf("memory extractor: read inbox session %q: %w", sessionDir.Name(), err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			claimed := false
			switch {
			case strings.HasSuffix(entry.Name(), ".jsonl"):
			case strings.HasSuffix(entry.Name(), ".jsonl"+processingSuffix):
				claimed = true
			default:
				continue
			}
			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("memory extractor: stat inbox file %q: %w", entry.Name(), err)
			}
			files = append(files, pendingFile{
				path:    filepath.Join(dirPath, entry.Name()),
				modTime: info.ModTime().UTC(),
				claimed: claimed,
			})
		}
	}
	slices.SortFunc(files, func(a, b pendingFile) int {
		if !a.modTime.Equal(b.modTime) {
			return a.modTime.Compare(b.modTime)
		}
		return strings.Compare(a.path, b.path)
	})
	return files, nil
}

func (c *InboxConsumer) consumeFile(ctx context.Context, file pendingFile) (ConsumeResult, error) {
	result := ConsumeResult{Decisions: make([]memcontract.Decision, 0), Failures: make([]string, 0)}
	path := file.path
	processing := path + processingSuffix
	if file.claimed {
		processing = file.path
		path = strings.TrimSuffix(file.path, processingSuffix)
	} else if err := os.Rename(path, processing); err != nil {
		return result, fmt.Errorf("memory extractor: claim inbox file %q: %w", path, err)
	}

	candidates, err := decodeCandidateFile(processing)
	if err != nil {
		result.Failed++
		failurePath, moveErr := c.moveToDLQ(processing, "decode", err)
		result.Failures = append(result.Failures, failurePath)
		c.recordFailure(ctx, "", failurePath, "decode", err)
		return result, errors.Join(err, moveErr)
	}

	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return result, c.requeueClaimedFile(processing, path, err)
		}
		decision, err := c.sink.ProposeCandidate(ctx, candidate)
		if err != nil {
			if isRetryableInboxError(ctx, err) {
				return result, c.requeueClaimedFile(processing, path, err)
			}
			result.Failed++
			failurePath, moveErr := c.moveToDLQ(processing, "controller", err)
			result.Failures = append(result.Failures, failurePath)
			c.recordFailure(ctx, candidate.Metadata["session_id"], failurePath, "controller", err)
			return result, errors.Join(
				fmt.Errorf("memory extractor: propose candidate from %q: %w", path, err),
				moveErr,
			)
		}
		result.Proposed++
		result.Decisions = append(result.Decisions, decision)
	}

	if err := fileutil.AtomicRemoveFile(processing); err != nil {
		return result, fmt.Errorf("memory extractor: remove consumed inbox file: %w", err)
	}
	return result, nil
}

func (c *InboxConsumer) requeueClaimedFile(processing string, path string, cause error) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("memory extractor: requeue inbox file %q: target already exists: %w", path, cause)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("memory extractor: stat inbox requeue target %q: %w", path, err)
	}
	if err := os.Rename(processing, path); err != nil {
		return errors.Join(
			fmt.Errorf("memory extractor: requeue inbox file %q: %w", path, err),
			cause,
		)
	}
	return fmt.Errorf("memory extractor: consume interrupted for %q: %w", path, cause)
}

func isRetryableInboxError(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if ctx == nil {
		return false
	}
	ctxErr := ctx.Err()
	return ctxErr != nil && errors.Is(err, ctxErr)
}

func decodeCandidateFile(path string) ([]memcontract.Candidate, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("memory extractor: open candidate file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Default().Warn("memory extractor: close candidate file failed", "path", path, "error", closeErr)
		}
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), defaultScannerLimit)
	candidates := make([]memcontract.Candidate, 0)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var candidate memcontract.Candidate
		if err := json.Unmarshal([]byte(raw), &candidate); err != nil {
			return nil, fmt.Errorf("memory extractor: decode candidate line %d: %w", line, err)
		}
		candidates = append(candidates, candidate)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("memory extractor: scan candidate file: %w", err)
	}
	return candidates, nil
}

func (c *InboxConsumer) moveToDLQ(path string, stage string, cause error) (string, error) {
	if err := os.MkdirAll(c.failuresDir, inboxDirPerm); err != nil {
		return "", fmt.Errorf("memory extractor: ensure dlq dir: %w", err)
	}
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		return "", fmt.Errorf("memory extractor: read failed inbox file: %w", readErr)
	}
	report := map[string]string{
		"stage":       strings.TrimSpace(stage),
		"source":      path,
		"error":       cause.Error(),
		"content":     string(content),
		"recorded_at": c.now().UTC().Format(time.RFC3339Nano),
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("memory extractor: encode dlq report: %w", err)
	}
	target := filepath.Join(c.failuresDir, dlqFilename(c.now(), filepath.Base(path)))
	if err := fileutil.AtomicWriteFile(target, append(encoded, '\n'), inboxFilePerm); err != nil {
		return "", fmt.Errorf("memory extractor: write dlq report: %w", err)
	}
	if err := fileutil.AtomicRemoveFile(path); err != nil {
		return target, fmt.Errorf("memory extractor: remove failed inbox file: %w", err)
	}
	return target, nil
}

func (c *InboxConsumer) recordFailure(
	ctx context.Context,
	sessionID string,
	target string,
	stage string,
	cause error,
) {
	if c.events == nil {
		return
	}
	event := Event{
		Op:        EventFailed,
		SessionID: sessionID,
		TargetID:  target,
		Error:     cause.Error(),
		Metadata: map[string]string{
			"stage": strings.TrimSpace(stage),
		},
		At: c.now().UTC(),
	}
	if err := c.events.RecordExtractorEvent(ctx, event); err != nil {
		c.logger.Warn("memory extractor: record failure event failed", "error", err)
	}
}

func cleanRoot(root string) (string, error) {
	clean := strings.TrimSpace(root)
	if clean == "" {
		return "", errors.New("memory extractor: memory root is required")
	}
	return filepath.Clean(clean), nil
}

func inboxSessionDir(inboxRoot string, sessionID string) (string, error) {
	segment, err := safeSegment(sessionID)
	if err != nil {
		return "", err
	}
	return filepath.Join(inboxRoot, segment), nil
}

func safeSegment(raw string) (string, error) {
	segment := strings.TrimSpace(raw)
	if segment == "" {
		return "", errors.New("memory extractor: path segment is required")
	}
	if strings.Contains(segment, "/") || strings.Contains(segment, `\`) || segment == "." || segment == ".." {
		return "", fmt.Errorf("memory extractor: unsafe path segment %q", raw)
	}
	return segment, nil
}

func inboxFilename(now time.Time, seq int64) string {
	return now.UTC().Format("20060102T150405.000000000Z") + "-" + strconv.FormatInt(seq, 10) + ".jsonl"
}

func dlqFilename(now time.Time, base string) string {
	cleanBase := strings.TrimSpace(base)
	if cleanBase == "" {
		cleanBase = "inbox.jsonl"
	}
	return now.UTC().Format("20060102T150405.000000000Z") + "-" + cleanBase + ".json"
}
