package recall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	memcontract "github.com/pedronauck/agh/internal/memory/contract"
)

const (
	defaultTopK          = 5
	maxTopK              = 20
	defaultRawCandidates = 40
	maxRawCandidates     = 200
	recencyHalfLifeDays  = 14.0
	trivialTokenFloor    = 2
	nonASCIITokenFloor   = 3
)

var defaultWeights = Weights{
	Unicode: 0.55,
	Trigram: 0.20,
	Recency: 0.15,
	Signal:  0.10,
}

var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {},
	"for": {}, "from": {}, "how": {}, "in": {}, "is": {}, "it": {}, "of": {}, "on": {},
	"or": {}, "the": {}, "to": {}, "what": {}, "when": {}, "where": {}, "with": {},
}

// Weights controls deterministic score fusion for Slice 1 recall.
type Weights struct {
	Unicode float64
	Trigram float64
	Recency float64
	Signal  float64
}

// Candidate is one catalog chunk candidate returned by the storage source.
type Candidate struct {
	ChunkID      string
	EntryID      string
	WorkspaceID  string
	Scope        memcontract.Scope
	AgentName    string
	AgentTier    memcontract.AgentTier
	Type         memcontract.Type
	Slug         string
	Filename     string
	Title        string
	Body         string
	ContentHash  string
	ModTime      time.Time
	Injection    bool
	UnicodeScore float64
	TrigramScore float64
	RecallScore  float64
}

// Signal records that one chunk was surfaced by recall.
type Signal struct {
	ChunkID      string
	WorkspaceID  string
	SurfaceID    string
	Score        float64
	SurfacedAt   time.Time
	SessionID    string
	SignalReason string
}

// Shadow records one candidate suppressed by a deeper scope owner.
type Shadow struct {
	WinnerChunkID string
	LoserChunkID  string
	WorkspaceID   string
	Scope         memcontract.Scope
	AgentName     string
	AgentTier     memcontract.AgentTier
	Type          memcontract.Type
	Slug          string
}

// Source supplies candidates and stores recall side effects.
type Source interface {
	Candidates(ctx context.Context, query memcontract.Query, opts memcontract.RecallOptions) ([]Candidate, error)
	RecordRecall(ctx context.Context, signals []Signal) error
	RecordRecallExecuted(ctx context.Context, query memcontract.Query, resultCount int) error
	RecordRecallSkipped(ctx context.Context, query memcontract.Query, reason string) error
	RecordRecallSignalFailed(ctx context.Context, query memcontract.Query, cause error) error
	RecordRecallSignalDropped(ctx context.Context, query memcontract.Query, signals []Signal, queueDepth int) error
	RecordShadow(ctx context.Context, shadow Shadow) error
}

// Recaller implements deterministic Slice 1 recall.
type Recaller struct {
	source         Source
	signalRecorder *SignalRecorder
	now            func() time.Time
	weights        Weights
	logger         *slog.Logger
}

var _ memcontract.Recaller = (*Recaller)(nil)

// Option customizes a deterministic Recaller.
type Option func(*Recaller)

// WithClock injects a deterministic clock for tests.
func WithClock(now func() time.Time) Option {
	return func(recaller *Recaller) {
		if now != nil {
			recaller.now = now
		}
	}
}

// WithLogger injects the logger used for failure-safe side effects.
func WithLogger(logger *slog.Logger) Option {
	return func(recaller *Recaller) {
		if logger != nil {
			recaller.logger = logger
		}
	}
}

// WithWeights overrides the deterministic score-fusion weights.
func WithWeights(weights Weights) Option {
	return func(recaller *Recaller) {
		recaller.weights = normalizeWeights(weights)
	}
}

// WithSignalRecorder moves recall-signal writes onto a bounded async worker.
func WithSignalRecorder(recorder *SignalRecorder) Option {
	return func(recaller *Recaller) {
		recaller.signalRecorder = recorder
	}
}

// New constructs a deterministic Recaller over a storage source.
func New(source Source, opts ...Option) *Recaller {
	recaller := &Recaller{
		source:  source,
		now:     func() time.Time { return time.Now().UTC() },
		weights: defaultWeights,
		logger:  slog.Default(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(recaller)
		}
	}
	recaller.weights = normalizeWeights(recaller.weights)
	return recaller
}

// Recall returns a prompt-ready package using deterministic lexical ranking.
func (r *Recaller) Recall(
	ctx context.Context,
	query memcontract.Query,
	opts memcontract.RecallOptions,
) (memcontract.Packaged, error) {
	if ctx == nil {
		return memcontract.Packaged{}, errors.New("memory recall: context is required")
	}
	if r == nil || r.source == nil {
		return memcontract.Packaged{}, errors.New("memory recall: source is required")
	}

	query.QueryText = strings.TrimSpace(query.QueryText)
	normalizedOpts := normalizeOptions(opts)
	if !normalizedOpts.AllowTrivialQuery && isTrivialQuery(query.QueryText) {
		r.recordSkipped(ctx, query, "trivial_query")
		return emptyPackage(), nil
	}

	candidates, err := r.source.Candidates(ctx, query, normalizedOpts)
	if err != nil {
		return memcontract.Packaged{}, fmt.Errorf("memory recall: load candidates: %w", err)
	}
	now := r.now().UTC()
	ranked, shadows := rankCandidates(candidates, normalizedOpts, r.weights, now)
	for _, shadow := range shadows {
		r.recordShadow(ctx, shadow)
	}

	packaged := packageCandidates(ranked, normalizedOpts.TopK, now)
	if len(packaged.Blocks) > 0 {
		r.recordSignals(ctx, query, signalsForRanked(ranked, normalizedOpts.TopK, now))
	}
	r.recordExecuted(ctx, query, packagedEntryCount(packaged))
	return packaged, nil
}

type rankedCandidate struct {
	Candidate
	score float64
	why   []string
}

func rankCandidates(
	candidates []Candidate,
	opts memcontract.RecallOptions,
	weights Weights,
	now time.Time,
) ([]rankedCandidate, []Shadow) {
	alreadySurfaced := surfacedSet(opts.AlreadySurfaced)
	merged := mergeCandidates(candidates)
	ranked := make([]rankedCandidate, 0, len(merged))
	for _, candidate := range merged {
		if !opts.IncludeSystem && !candidate.Injection {
			continue
		}
		if _, seen := alreadySurfaced[candidate.ChunkID]; seen && !opts.IncludeAlreadySurfaced {
			continue
		}
		if _, seen := alreadySurfaced[candidate.EntryID]; seen && !opts.IncludeAlreadySurfaced {
			continue
		}
		score, why := scoreCandidate(candidate, weights, now)
		if score <= 0 {
			continue
		}
		ranked = append(ranked, rankedCandidate{Candidate: candidate, score: score, why: why})
	}
	sortRanked(ranked)
	return applyShadowRules(ranked)
}

func mergeCandidates(candidates []Candidate) []Candidate {
	byID := make(map[string]Candidate, len(candidates))
	for _, candidate := range candidates {
		candidate = normalizeCandidate(candidate)
		if candidate.ChunkID == "" {
			continue
		}
		current, exists := byID[candidate.ChunkID]
		if !exists {
			byID[candidate.ChunkID] = candidate
			continue
		}
		current.UnicodeScore = math.Max(current.UnicodeScore, candidate.UnicodeScore)
		current.TrigramScore = math.Max(current.TrigramScore, candidate.TrigramScore)
		current.RecallScore = math.Max(current.RecallScore, candidate.RecallScore)
		if current.Body == "" {
			current.Body = candidate.Body
		}
		byID[candidate.ChunkID] = current
	}

	merged := make([]Candidate, 0, len(byID))
	for _, candidate := range byID {
		merged = append(merged, candidate)
	}
	return merged
}

func normalizeCandidate(candidate Candidate) Candidate {
	candidate.ChunkID = strings.TrimSpace(candidate.ChunkID)
	candidate.EntryID = strings.TrimSpace(candidate.EntryID)
	candidate.WorkspaceID = strings.TrimSpace(candidate.WorkspaceID)
	candidate.Scope = candidate.Scope.Normalize()
	candidate.AgentName = strings.TrimSpace(candidate.AgentName)
	candidate.AgentTier = candidate.AgentTier.Normalize()
	candidate.Type = candidate.Type.Normalize()
	candidate.Slug = strings.TrimSpace(candidate.Slug)
	candidate.Filename = strings.TrimSpace(candidate.Filename)
	candidate.Title = strings.TrimSpace(candidate.Title)
	candidate.Body = strings.TrimSpace(candidate.Body)
	candidate.ContentHash = strings.TrimSpace(candidate.ContentHash)
	if candidate.Title == "" {
		candidate.Title = candidate.Filename
	}
	if candidate.Slug == "" {
		candidate.Slug = strings.TrimSuffix(candidate.Filename, ".md")
	}
	if candidate.ModTime.IsZero() {
		candidate.ModTime = time.Unix(0, 0).UTC()
	}
	return candidate
}

func scoreCandidate(candidate Candidate, weights Weights, now time.Time) (float64, []string) {
	unicodeScore := clamp01(candidate.UnicodeScore)
	trigramScore := clamp01(candidate.TrigramScore)
	recencyScore := recency(candidate.ModTime, now)
	signalScore := clamp01(candidate.RecallScore)
	score := weights.Unicode*unicodeScore +
		weights.Trigram*trigramScore +
		weights.Recency*recencyScore +
		weights.Signal*signalScore
	why := []string{
		fmt.Sprintf("unicode=%.3f", unicodeScore),
		fmt.Sprintf("trigram=%.3f", trigramScore),
		fmt.Sprintf("recency=%.3f", recencyScore),
		fmt.Sprintf("signal=%.3f", signalScore),
		fmt.Sprintf("score=%.3f", score),
	}
	return score, why
}

func recency(modTime time.Time, now time.Time) float64 {
	if modTime.IsZero() {
		return 0
	}
	ageHours := now.Sub(modTime.UTC()).Hours()
	if ageHours <= 0 {
		return 1
	}
	return math.Pow(0.5, (ageHours/24.0)/recencyHalfLifeDays)
}

func sortRanked(ranked []rankedCandidate) {
	sort.SliceStable(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]
		if left.score != right.score {
			return left.score > right.score
		}
		if left.scopeDepth() != right.scopeDepth() {
			return left.scopeDepth() > right.scopeDepth()
		}
		if !left.ModTime.Equal(right.ModTime) {
			return left.ModTime.After(right.ModTime)
		}
		return left.ChunkID < right.ChunkID
	})
}

func applyShadowRules(ranked []rankedCandidate) ([]rankedCandidate, []Shadow) {
	winners := make(map[string]rankedCandidate, len(ranked))
	shadows := make([]Shadow, 0)
	for _, candidate := range ranked {
		key := shadowKey(candidate.Candidate)
		if key == "" {
			winners[candidate.ChunkID] = candidate
			continue
		}
		current, exists := winners[key]
		if !exists {
			winners[key] = candidate
			continue
		}
		winner, loser := pickShadowWinner(current, candidate)
		winners[key] = winner
		shadows = append(shadows, Shadow{
			WinnerChunkID: winner.ChunkID,
			LoserChunkID:  loser.ChunkID,
			WorkspaceID:   winner.WorkspaceID,
			Scope:         winner.Scope,
			AgentName:     winner.AgentName,
			AgentTier:     winner.AgentTier,
			Type:          winner.Type,
			Slug:          winner.Slug,
		})
	}

	out := make([]rankedCandidate, 0, len(winners))
	for _, candidate := range winners {
		out = append(out, candidate)
	}
	sortRanked(out)
	return out, shadows
}

func shadowKey(candidate Candidate) string {
	typeName := strings.TrimSpace(string(candidate.Type.Normalize()))
	slug := strings.TrimSpace(candidate.Slug)
	if typeName == "" || slug == "" {
		return ""
	}
	return typeName + "::" + slug
}

func pickShadowWinner(left rankedCandidate, right rankedCandidate) (rankedCandidate, rankedCandidate) {
	if left.scopeDepth() != right.scopeDepth() {
		if left.scopeDepth() > right.scopeDepth() {
			return left, right
		}
		return right, left
	}
	if left.score >= right.score {
		return left, right
	}
	return right, left
}

func (candidate rankedCandidate) scopeDepth() int {
	switch candidate.Scope.Normalize() {
	case memcontract.ScopeAgent:
		if candidate.AgentTier.Normalize() == memcontract.AgentTierWorkspace {
			return 3
		}
		return 2
	case memcontract.ScopeWorkspace:
		return 1
	case memcontract.ScopeGlobal:
		return 0
	default:
		return 0
	}
}

func packageCandidates(ranked []rankedCandidate, topK int, now time.Time) memcontract.Packaged {
	if len(ranked) == 0 {
		return emptyPackage()
	}
	if len(ranked) > topK {
		ranked = ranked[:topK]
	}

	blocks := groupBlocks(ranked, now)
	header := stableHeader(blocks, ranked)
	return memcontract.Packaged{Blocks: blocks, Header: header}
}

func groupBlocks(ranked []rankedCandidate, now time.Time) []memcontract.Block {
	groups := make(map[string][]memcontract.PackagedEntry)
	order := make([]string, 0)
	blockMeta := make(map[string]struct {
		scope memcontract.Scope
		tier  memcontract.AgentTier
		depth int
	})
	for _, candidate := range ranked {
		key := blockKey(candidate.Candidate)
		if _, exists := groups[key]; !exists {
			order = append(order, key)
			blockMeta[key] = struct {
				scope memcontract.Scope
				tier  memcontract.AgentTier
				depth int
			}{scope: candidate.Scope.Normalize(), tier: candidate.AgentTier.Normalize(), depth: candidate.scopeDepth()}
		}
		groups[key] = append(groups[key], packagedEntry(candidate, now))
	}
	sort.SliceStable(order, func(i, j int) bool {
		left := blockMeta[order[i]]
		right := blockMeta[order[j]]
		if left.depth != right.depth {
			return left.depth < right.depth
		}
		return order[i] < order[j]
	})

	blocks := make([]memcontract.Block, 0, len(order))
	for _, key := range order {
		entries := groups[key]
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].ID < entries[j].ID
		})
		meta := blockMeta[key]
		blocks = append(blocks, memcontract.Block{
			Scope:     meta.scope,
			AgentTier: meta.tier,
			Entries:   entries,
		})
	}
	return blocks
}

func blockKey(candidate Candidate) string {
	return string(candidate.Scope.Normalize()) + "::" + string(candidate.AgentTier.Normalize())
}

func packagedEntry(candidate rankedCandidate, now time.Time) memcontract.PackagedEntry {
	age := ageDays(candidate.ModTime, now)
	return memcontract.PackagedEntry{
		ID:              candidate.ChunkID,
		Filename:        candidate.Filename,
		Title:           candidate.Title,
		Type:            candidate.Type.Normalize(),
		WorkspaceID:     candidate.WorkspaceID,
		Body:            candidate.Body,
		ModTime:         candidate.ModTime.UTC(),
		AgeDays:         age,
		StalenessBanner: stalenessBanner(age),
		WhyRecalled:     append([]string(nil), candidate.why...),
	}
}

func stableHeader(blocks []memcontract.Block, ranked []rankedCandidate) memcontract.CacheStableHeader {
	parts := make([]string, 0, len(ranked))
	for _, candidate := range ranked {
		parts = append(parts, strings.Join([]string{
			candidate.ChunkID,
			candidate.ContentHash,
			string(candidate.Scope.Normalize()),
			string(candidate.AgentTier.Normalize()),
		}, "|"))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	hash := hex.EncodeToString(sum[:])
	return memcontract.CacheStableHeader{
		Text:        fmt.Sprintf("AGH memory recall v1 entries=%d hash=%s", packagedBlockEntryCount(blocks), hash),
		ContentHash: hash,
	}
}

func emptyPackage() memcontract.Packaged {
	return memcontract.Packaged{
		Blocks: []memcontract.Block{},
		Header: memcontract.CacheStableHeader{Text: "AGH memory recall v1 entries=0 hash=", ContentHash: ""},
	}
}

func signalsForRanked(ranked []rankedCandidate, topK int, now time.Time) []Signal {
	if len(ranked) > topK {
		ranked = ranked[:topK]
	}
	signals := make([]Signal, 0, len(ranked))
	for _, candidate := range ranked {
		signals = append(signals, Signal{
			ChunkID:      candidate.ChunkID,
			WorkspaceID:  candidate.WorkspaceID,
			SurfaceID:    candidate.ChunkID,
			Score:        candidate.score,
			SurfacedAt:   now,
			SignalReason: strings.Join(candidate.why, ";"),
		})
	}
	return signals
}

func (r *Recaller) recordSignals(ctx context.Context, query memcontract.Query, signals []Signal) {
	if len(signals) == 0 {
		return
	}
	if r.signalRecorder != nil {
		r.signalRecorder.Submit(ctx, query, signals)
		return
	}
	if err := r.source.RecordRecall(ctx, signals); err != nil {
		r.warn("memory recall: record recall signal failed", "error", err)
		if eventErr := r.source.RecordRecallSignalFailed(ctx, query, err); eventErr != nil {
			r.warn("memory recall: record signal failure event failed", "error", eventErr)
		}
	}
}

func (r *Recaller) recordExecuted(ctx context.Context, query memcontract.Query, resultCount int) {
	if err := r.source.RecordRecallExecuted(ctx, query, resultCount); err != nil {
		r.warn("memory recall: record executed event failed", "error", err)
	}
}

func (r *Recaller) recordSkipped(ctx context.Context, query memcontract.Query, reason string) {
	if err := r.source.RecordRecallSkipped(ctx, query, reason); err != nil {
		r.warn("memory recall: record skipped event failed", "error", err)
	}
}

func (r *Recaller) recordShadow(ctx context.Context, shadow Shadow) {
	if err := r.source.RecordShadow(ctx, shadow); err != nil {
		r.warn("memory recall: record shadow event failed", "error", err)
	}
}

func (r *Recaller) warn(msg string, args ...any) {
	if r != nil && r.logger != nil {
		r.logger.Warn(msg, args...)
	}
}

func isTrivialQuery(query string) bool {
	tokens := meaningfulTokens(query)
	if len(tokens) >= trivialTokenFloor {
		return false
	}
	for _, token := range tokens {
		if containsNonASCII(token) && len([]rune(token)) >= nonASCIITokenFloor {
			return false
		}
	}
	return true
}

func meaningfulTokens(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(query)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	tokens := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		token := strings.TrimSpace(field)
		if len([]rune(token)) < 2 {
			continue
		}
		if _, stop := stopWords[token]; stop {
			continue
		}
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	return tokens
}

func normalizeOptions(opts memcontract.RecallOptions) memcontract.RecallOptions {
	if opts.TopK <= 0 {
		opts.TopK = defaultTopK
	}
	opts.TopK = min(opts.TopK, maxTopK)
	if opts.RawCandidates <= 0 {
		opts.RawCandidates = defaultRawCandidates
	}
	opts.RawCandidates = min(opts.RawCandidates, maxRawCandidates)
	return opts
}

func normalizeWeights(weights Weights) Weights {
	total := weights.Unicode + weights.Trigram + weights.Recency + weights.Signal
	if total <= 0 {
		return defaultWeights
	}
	return Weights{
		Unicode: weights.Unicode / total,
		Trigram: weights.Trigram / total,
		Recency: weights.Recency / total,
		Signal:  weights.Signal / total,
	}
}

func surfacedSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out[trimmed] = struct{}{}
	}
	return out
}

func ageDays(modTime time.Time, now time.Time) int {
	if modTime.IsZero() {
		return 0
	}
	days := calendarDayNumber(now) - calendarDayNumber(modTime)
	if days < 0 {
		return 0
	}
	return days
}

func stalenessBanner(age int) string {
	if age <= 1 {
		return ""
	}
	return fmt.Sprintf("This memory is %d days old. Verify against current state before asserting as fact.", age)
}

func calendarDayNumber(value time.Time) int {
	year, month, day := value.UTC().Date()
	return int(time.Date(year, month, day, 12, 0, 0, 0, time.UTC).Unix() / int64(24*time.Hour/time.Second))
}

func packagedEntryCount(packaged memcontract.Packaged) int {
	return packagedBlockEntryCount(packaged.Blocks)
}

func packagedBlockEntryCount(blocks []memcontract.Block) int {
	count := 0
	for _, block := range blocks {
		count += len(block.Entries)
	}
	return count
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func containsNonASCII(value string) bool {
	for _, r := range value {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}
