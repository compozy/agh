package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/goccy/go-yaml"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"
	"github.com/pedronauck/agh/internal/memory/prompts"
	"github.com/pedronauck/agh/internal/memory/scan"
)

const (
	defaultPromptVersion       = prompts.VersionV1
	metadataOperationKey       = "operation"
	metadataOpKey              = "op"
	metadataFilenameKey        = "filename"
	metadataTargetFilenameKey  = "target_filename"
	metadataRawContentKey      = "raw_content"
	metadataReasonKey          = "reason"
	metadataSourceKey          = "source"
	metadataTargetEntityKey    = "target_entity"
	metadataTargetAttributeKey = "target_attribute"
	minSurfaceOverlap          = 2
	maxDecisionReasonBytes     = 240
)

var filenameUnsafePattern = regexp.MustCompile(`[^a-z0-9]+`)

// Target is one existing curated memory entry visible to the controller.
type Target struct {
	ID             string
	WorkspaceID    string
	Scope          memcontract.Scope
	AgentName      string
	AgentTier      memcontract.AgentTier
	TargetFilename string
	Frontmatter    memcontract.Header
	Entity         string
	Attribute      string
	Content        string
	RawContent     string
	ContentHash    string
	LastUpdatedAt  time.Time
}

// TargetIndex supplies current curated memory candidates for rule decisions.
type TargetIndex interface {
	ListTargets(ctx context.Context, candidate memcontract.Candidate) ([]Target, error)
}

// Controller decides Memory v2 write outcomes with deterministic Slice 1 rules.
type Controller struct {
	index         TargetIndex
	now           func() time.Time
	promptVersion string
}

// Option customizes a Controller.
type Option func(*Controller)

// WithClock injects a deterministic clock for tests.
func WithClock(now func() time.Time) Option {
	return func(c *Controller) {
		if now != nil {
			c.now = now
		}
	}
}

// WithPromptVersion pins the decision prompt version recorded in WAL rows.
func WithPromptVersion(version string) Option {
	return func(c *Controller) {
		if strings.TrimSpace(version) != "" {
			c.promptVersion = strings.TrimSpace(version)
		}
	}
}

// New constructs a rule-first controller over the provided target index.
func New(index TargetIndex, opts ...Option) *Controller {
	c := &Controller{
		index: index,
		now: func() time.Time {
			return time.Now().UTC()
		},
		promptVersion: defaultPromptVersion,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

// Decide returns a deterministic Decision. Mutation and WAL persistence are owned by Store.ApplyDecision.
func (c *Controller) Decide(ctx context.Context, candidate memcontract.Candidate) (memcontract.Decision, error) {
	normalized, targets, err := c.prepareDecision(ctx, candidate)
	if err != nil {
		return memcontract.Decision{}, err
	}

	scanResult := scan.Candidate(normalized)
	trace := scanRuleHits(scanResult, normalized)
	if scanResult.Rejected() {
		return c.rejectDecision(normalized, scanResult, trace)
	}
	if opFromCandidate(normalized) == memcontract.OpDelete {
		return c.deleteDecision(normalized, targets, trace)
	}
	if exact := exactContentTarget(normalized, targets); exact != nil {
		return c.decision(
			normalized,
			memcontract.OpNoop,
			[]Target{*exact},
			exact.TargetFilename,
			"",
			append(trace, passedRule("exact_hash", "candidate content already exists", exact.ID)),
			"exact duplicate memory content",
			nil,
		)
	}
	return c.writeDecision(normalized, targets, trace)
}

func (c *Controller) prepareDecision(
	ctx context.Context,
	candidate memcontract.Candidate,
) (memcontract.Candidate, []Target, error) {
	if ctx == nil {
		return memcontract.Candidate{}, nil, errors.New("memory controller: context is required")
	}
	if c == nil {
		return memcontract.Candidate{}, nil, errors.New("memory controller: controller is required")
	}
	normalized, err := normalizeCandidate(candidate, c.now())
	if err != nil {
		return memcontract.Candidate{}, nil, err
	}
	if err := ctx.Err(); err != nil {
		return memcontract.Candidate{}, nil, fmt.Errorf("memory controller: decide canceled: %w", err)
	}

	targets, err := c.targets(ctx, normalized)
	if err != nil {
		return memcontract.Candidate{}, nil, err
	}
	return normalized, targets, nil
}

func (c *Controller) rejectDecision(
	candidate memcontract.Candidate,
	scanResult scan.Result,
	trace []memcontract.RuleHit,
) (memcontract.Decision, error) {
	return c.decision(
		candidate,
		memcontract.OpReject,
		nil,
		"",
		"",
		trace,
		scanReason(scanResult, candidate),
		nil,
	)
}

func (c *Controller) writeDecision(
	normalized memcontract.Candidate,
	targets []Target,
	trace []memcontract.RuleHit,
) (memcontract.Decision, error) {
	if collision := exactFilenameTarget(normalized, targets); collision != nil {
		return c.updateDecision(normalized, *collision, append(
			trace,
			passedRule("exact_slug_collision", "target filename already exists", collision.ID),
		))
	}
	slotMatches := entitySlotTargets(normalized, targets)
	switch len(slotMatches) {
	case 0:
		if surface := surfaceTargets(normalized, targets); len(surface) > 1 &&
			normalized.Origin.Normalize() != memcontract.OriginDreaming {
			if !isAutonomousWriteOrigin(normalized.Origin) &&
				directTargetIdentifiesNewMemory(normalized, surface) {
				return c.addDecision(normalized, append(
					trace,
					passedRule(
						"explicit_target_filename",
						"direct write supplied a distinct target filename",
						targetFilename(normalized),
					),
				))
			}
			return c.ambiguousDecision(normalized, surface, trace)
		}
		return c.addDecision(normalized, trace)
	case 1:
		target := slotMatches[0]
		if equalMemoryBody(normalized.Content, target.Content) {
			return c.decision(
				normalized,
				memcontract.OpNoop,
				[]Target{target},
				target.TargetFilename,
				"",
				append(
					trace,
					passedRule("entity_slot_no_change", "slot target content is unchanged", target.ID),
				),
				"entity slot already has this content",
				nil,
			)
		}
		return c.updateDecision(normalized, target, append(
			trace,
			passedRule("entity_slot_update", "single entity slot target differs", target.ID),
		))
	default:
		if !isAutonomousWriteOrigin(normalized.Origin) &&
			directTargetIdentifiesNewMemory(normalized, slotMatches) {
			return c.addDecision(normalized, append(
				trace,
				passedRule(
					"explicit_target_filename",
					"direct write supplied a distinct target filename",
					targetFilename(normalized),
				),
			))
		}
		return c.ambiguousDecision(normalized, slotMatches, trace)
	}
}

func (c *Controller) targets(ctx context.Context, candidate memcontract.Candidate) ([]Target, error) {
	if c.index == nil {
		return nil, nil
	}
	targets, err := c.index.ListTargets(ctx, candidate)
	if err != nil {
		return nil, fmt.Errorf("memory controller: list targets: %w", err)
	}
	slices.SortFunc(targets, func(a Target, b Target) int {
		return strings.Compare(targetSortKey(a), targetSortKey(b))
	})
	return targets, nil
}

func (c *Controller) addDecision(
	candidate memcontract.Candidate,
	trace []memcontract.RuleHit,
) (memcontract.Decision, error) {
	postContent, err := postContentForCandidate(candidate)
	if err != nil {
		return memcontract.Decision{}, err
	}
	return c.decision(
		candidate,
		memcontract.OpAdd,
		nil,
		targetFilename(candidate),
		postContent,
		append(trace, passedRule("fresh_slot", "no matching target found", "")),
		"fresh memory slot",
		nil,
	)
}

func (c *Controller) updateDecision(
	candidate memcontract.Candidate,
	target Target,
	trace []memcontract.RuleHit,
) (memcontract.Decision, error) {
	postContent, err := postContentForCandidate(candidate)
	if err != nil {
		return memcontract.Decision{}, err
	}
	return c.decision(
		candidate,
		memcontract.OpUpdate,
		[]Target{target},
		target.TargetFilename,
		postContent,
		trace,
		"single target updated",
		&target,
	)
}

func (c *Controller) deleteDecision(
	candidate memcontract.Candidate,
	targets []Target,
	trace []memcontract.RuleHit,
) (memcontract.Decision, error) {
	filename := targetFilename(candidate)
	matches := make([]Target, 0, 1)
	for _, target := range targets {
		if target.TargetFilename == filename {
			matches = append(matches, target)
		}
	}
	switch len(matches) {
	case 0:
		return c.decision(
			candidate,
			memcontract.OpNoop,
			nil,
			filename,
			"",
			append(trace, passedRule("delete_missing", "delete target is already absent", filename)),
			"delete target is already absent",
			nil,
		)
	case 1:
		target := matches[0]
		return c.decision(
			candidate,
			memcontract.OpDelete,
			[]Target{target},
			target.TargetFilename,
			"",
			append(trace, passedRule("delete_target", "single delete target found", target.ID)),
			"delete target found",
			&target,
		)
	default:
		return c.ambiguousDecision(candidate, matches, trace)
	}
}

func (c *Controller) ambiguousDecision(
	candidate memcontract.Candidate,
	targets []Target,
	trace []memcontract.RuleHit,
) (memcontract.Decision, error) {
	return c.decision(
		candidate,
		memcontract.OpNoop,
		targets,
		targetFilename(candidate),
		"",
		append(
			trace,
			failedRule("ambiguous_targets", "multiple plausible targets require tiebreaker", targetIDs(targets)),
		),
		"ambiguous targets; rules-only fallback selected noop",
		nil,
	)
}

func (c *Controller) decision(
	candidate memcontract.Candidate,
	op memcontract.Op,
	targets []Target,
	filename string,
	postContent string,
	trace []memcontract.RuleHit,
	reason string,
	target *Target,
) (memcontract.Decision, error) {
	if err := op.Validate(); err != nil {
		return memcontract.Decision{}, err
	}
	now := c.now().UTC()
	frontmatter := candidate.Frontmatter
	frontmatter.Scope = candidate.Scope.Normalize()
	frontmatter.AgentName = strings.TrimSpace(candidate.AgentName)
	frontmatter.AgentTier = candidate.AgentTier.Normalize()
	postContentHash := ""
	if postContent != "" {
		postContentHash = hashString(postContent)
	}
	priorContent := ""
	if target != nil {
		priorContent = target.RawContent
	}
	if reasonFromMetadata := metadataValue(candidate.Metadata, metadataReasonKey); reasonFromMetadata != "" {
		reason = reasonFromMetadata
	}
	decision := memcontract.Decision{
		CandidateHash:   CandidateHash(candidate),
		Op:              op,
		Targets:         targetIDs(targets),
		TargetFilename:  filename,
		Frontmatter:     frontmatter,
		PostContent:     postContent,
		PostContentHash: postContentHash,
		PriorContent:    priorContent,
		Confidence:      confidenceForOp(op),
		Source:          memcontract.SourceRule,
		RuleTrace:       boundRuleTrace(trace),
		Reason:          boundString(reason, maxDecisionReasonBytes),
		PromptVersion:   c.promptVersion,
		DecidedAt:       now,
	}
	decision.IdempotencyKey = IdempotencyKey(decision)
	decision.ID = "dec_" + hashString(decision.IdempotencyKey)[:24]
	return decision, nil
}

func normalizeCandidate(candidate memcontract.Candidate, now time.Time) (memcontract.Candidate, error) {
	candidate.WorkspaceID = strings.TrimSpace(candidate.WorkspaceID)
	candidate.Scope = candidate.Scope.Normalize()
	candidate.AgentName = strings.TrimSpace(firstNonEmpty(candidate.AgentName, candidate.Frontmatter.AgentName))
	candidate.AgentTier = firstNonEmptyAgentTier(candidate.AgentTier, candidate.Frontmatter.AgentTier).Normalize()
	candidate.Origin = candidate.Origin.Normalize()
	candidate.Content = strings.TrimSpace(candidate.Content)
	candidate.Entity = normalizeSlot(candidate.Entity)
	candidate.Attribute = normalizeSlot(candidate.Attribute)
	if candidate.Metadata == nil {
		candidate.Metadata = map[string]string{}
	}
	if candidate.SubmittedAt.IsZero() {
		candidate.SubmittedAt = now.UTC()
	}
	if candidate.Origin == "" {
		candidate.Origin = memcontract.OriginFile
	}
	if err := candidate.Origin.Validate(); err != nil {
		return memcontract.Candidate{}, fmt.Errorf("memory controller: candidate origin: %w", err)
	}
	if candidate.Scope == "" {
		candidate.Scope = candidate.Frontmatter.Scope.Normalize()
	}
	if candidate.Scope == "" && candidate.Frontmatter.Type.Normalize() != "" {
		scope, err := memcontract.DefaultScopeForType(candidate.Frontmatter.Type)
		if err != nil {
			return memcontract.Candidate{}, fmt.Errorf("memory controller: infer candidate scope: %w", err)
		}
		candidate.Scope = scope
	}
	if err := candidate.Scope.Validate(); err != nil {
		return memcontract.Candidate{}, fmt.Errorf("memory controller: candidate scope: %w", err)
	}
	if opFromCandidate(candidate) != memcontract.OpDelete {
		candidate.Frontmatter.Scope = candidate.Scope.Normalize()
		if candidate.AgentName != "" {
			candidate.Frontmatter.AgentName = candidate.AgentName
		}
		if candidate.AgentTier != "" {
			candidate.Frontmatter.AgentTier = candidate.AgentTier
		}
		if err := candidate.Frontmatter.Validate(); err != nil {
			return memcontract.Candidate{}, fmt.Errorf("memory controller: candidate frontmatter: %w", err)
		}
		if candidate.Content == "" {
			return memcontract.Candidate{}, errors.New("memory controller: candidate content is required")
		}
	} else if targetFilename(candidate) == "" {
		return memcontract.Candidate{}, errors.New("memory controller: delete candidate target filename is required")
	}
	return candidate, nil
}

// CandidateHash returns the stable hash used for decision audit rows.
func CandidateHash(candidate memcontract.Candidate) string {
	payload := map[string]any{
		"workspace_id": candidate.WorkspaceID,
		"scope":        candidate.Scope.Normalize(),
		"agent_name":   strings.TrimSpace(candidate.AgentName),
		"agent_tier":   candidate.AgentTier.Normalize(),
		"origin":       candidate.Origin.Normalize(),
		"content":      strings.TrimSpace(candidate.Content),
		"frontmatter":  candidate.Frontmatter,
		"entity":       normalizeSlot(candidate.Entity),
		"attribute":    normalizeSlot(candidate.Attribute),
		"metadata":     sortedMetadata(candidate.Metadata),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return hashString(fmt.Sprintf("%#v", payload))
	}
	return hashString(string(encoded))
}

// FrontmatterHash returns a stable hash for a decision's frontmatter material.
func FrontmatterHash(header memcontract.Header) string {
	encoded, err := json.Marshal(header)
	if err != nil {
		return hashString(fmt.Sprintf("%#v", header))
	}
	return hashString(string(encoded))
}

// IdempotencyKey returns the write-ahead-log uniqueness key for a decision.
func IdempotencyKey(decision memcontract.Decision) string {
	parts := []string{
		decision.CandidateHash,
		decision.Op.String(),
		strings.Join(decision.Targets, ","),
		decision.TargetFilename,
		decision.PostContentHash,
		FrontmatterHash(decision.Frontmatter),
		decision.PromptVersion,
	}
	return hashString(strings.Join(parts, "\x00"))
}

func postContentForCandidate(candidate memcontract.Candidate) (string, error) {
	if candidate.Metadata != nil {
		if raw, ok := candidate.Metadata[metadataRawContentKey]; ok && raw != "" {
			return raw, nil
		}
	}
	if raw := metadataValue(candidate.Metadata, metadataRawContentKey); raw != "" {
		return raw, nil
	}
	metadata, err := yaml.Marshal(candidate.Frontmatter)
	if err != nil {
		return "", fmt.Errorf("memory controller: render frontmatter: %w", err)
	}
	return "---\n" + string(metadata) + "---\n\n" + strings.TrimSpace(candidate.Content) + "\n", nil
}

func exactContentTarget(candidate memcontract.Candidate, targets []Target) *Target {
	candidateBody := canonicalBody(candidate.Content)
	for idx := range targets {
		if canonicalBody(targets[idx].Content) == candidateBody {
			return &targets[idx]
		}
		if raw := metadataValue(candidate.Metadata, metadataRawContentKey); raw != "" &&
			targets[idx].ContentHash == hashString(raw) {
			return &targets[idx]
		}
	}
	return nil
}

func exactFilenameTarget(candidate memcontract.Candidate, targets []Target) *Target {
	filename := targetFilename(candidate)
	for idx := range targets {
		if targets[idx].TargetFilename == filename {
			return &targets[idx]
		}
	}
	return nil
}

func entitySlotTargets(candidate memcontract.Candidate, targets []Target) []Target {
	entity := normalizeSlot(firstNonEmpty(candidate.Entity, metadataValue(candidate.Metadata, metadataTargetEntityKey)))
	attribute := normalizeSlot(
		firstNonEmpty(candidate.Attribute, metadataValue(candidate.Metadata, metadataTargetAttributeKey)),
	)
	if entity == "" || attribute == "" {
		return nil
	}
	matches := make([]Target, 0)
	for _, target := range targets {
		if normalizeSlot(target.Entity) == entity && normalizeSlot(target.Attribute) == attribute {
			matches = append(matches, target)
		}
	}
	return matches
}

func surfaceTargets(candidate memcontract.Candidate, targets []Target) []Target {
	candidateTokens := tokenSet(candidate.Content)
	matches := make([]Target, 0)
	for _, target := range targets {
		overlap := 0
		for token := range tokenSet(target.Content) {
			if _, exists := candidateTokens[token]; exists {
				overlap++
			}
		}
		if overlap >= minSurfaceOverlap {
			matches = append(matches, target)
		}
	}
	return matches
}

func targetFilename(candidate memcontract.Candidate) string {
	for _, key := range []string{metadataTargetFilenameKey, metadataFilenameKey} {
		if filename := cleanFilename(metadataValue(candidate.Metadata, key)); filename != "" {
			return filename
		}
	}
	if filename := cleanFilename(candidate.Frontmatter.Filename); filename != "" {
		return filename
	}
	base := firstNonEmpty(candidate.Entity, candidate.Frontmatter.Name, firstWords(candidate.Content, 5), "memory")
	prefix := string(candidate.Frontmatter.Type.Normalize())
	if prefix == "" {
		prefix = "memory"
	}
	return prefix + "_" + slugify(base) + ".md"
}

func isAutonomousWriteOrigin(origin memcontract.Origin) bool {
	switch origin.Normalize() {
	case memcontract.OriginDreaming, memcontract.OriginExtractor, memcontract.OriginProvider:
		return true
	default:
		return false
	}
}

func hasExplicitTargetFilename(candidate memcontract.Candidate) bool {
	for _, key := range []string{metadataTargetFilenameKey, metadataFilenameKey} {
		if cleanFilename(metadataValue(candidate.Metadata, key)) != "" {
			return true
		}
	}
	return cleanFilename(candidate.Frontmatter.Filename) != ""
}

func directTargetIdentifiesNewMemory(candidate memcontract.Candidate, targets []Target) bool {
	hasFilename := hasExplicitTargetFilename(candidate)
	candidateName := normalizeSlot(candidate.Frontmatter.Name)
	if !hasFilename && candidateName == "" {
		return false
	}
	if candidateName == "" {
		return true
	}
	for _, target := range targets {
		if normalizeSlot(target.Frontmatter.Name) == candidateName {
			return false
		}
	}
	return true
}

func cleanFilename(filename string) string {
	trimmed := strings.TrimSpace(filename)
	if trimmed == "" || trimmed == "." || trimmed == ".." {
		return ""
	}
	if strings.ContainsAny(trimmed, `/\`) {
		return ""
	}
	if filepath.Ext(trimmed) == "" {
		trimmed += ".md"
	}
	return trimmed
}

func scanRuleHits(result scan.Result, candidate memcontract.Candidate) []memcontract.RuleHit {
	hits := result.RuleHits()
	if len(hits) == 0 {
		return nil
	}
	sampleBytes := len([]byte(candidate.Content))
	for idx := range hits {
		hits[idx].Details = strings.TrimSpace(hits[idx].Details + fmt.Sprintf(" sample_bytes=%d", sampleBytes))
	}
	return hits
}

func scanReason(result scan.Result, candidate memcontract.Candidate) string {
	return fmt.Sprintf("%s sample_bytes=%d", result.Reason(), len([]byte(candidate.Content)))
}

func opFromCandidate(candidate memcontract.Candidate) memcontract.Op {
	raw := firstNonEmpty(
		metadataValue(candidate.Metadata, metadataOperationKey),
		metadataValue(candidate.Metadata, metadataOpKey),
	)
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case memcontract.OpDelete.String(), "forget", "remove":
		return memcontract.OpDelete
	default:
		return memcontract.OpAdd
	}
}

func passedRule(name string, reason string, target string) memcontract.RuleHit {
	return memcontract.RuleHit{Name: "controller." + name, Passed: true, Reason: reason, Target: target}
}

func failedRule(name string, reason string, targets []string) memcontract.RuleHit {
	return memcontract.RuleHit{
		Name:    "controller." + name,
		Passed:  false,
		Reason:  reason,
		Details: strings.Join(targets, ","),
	}
}

func targetIDs(targets []Target) []string {
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		if id := strings.TrimSpace(target.ID); id != "" {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

func targetSortKey(target Target) string {
	return strings.Join([]string{
		string(target.Scope.Normalize()),
		strings.TrimSpace(target.WorkspaceID),
		strings.TrimSpace(target.AgentName),
		string(target.AgentTier.Normalize()),
		strings.TrimSpace(target.TargetFilename),
	}, "\x00")
}

func equalMemoryBody(left string, right string) bool {
	return canonicalBody(left) == canonicalBody(right)
}

func canonicalBody(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func tokenSet(value string) map[string]struct{} {
	fields := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if len(trimmed) < 3 {
			continue
		}
		out[trimmed] = struct{}{}
	}
	return out
}

func sortedMetadata(metadata map[string]string) [][2]string {
	if len(metadata) == 0 {
		return nil
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := make([][2]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, [2]string{key, metadata[key]})
	}
	return out
}

func metadataValue(metadata map[string]string, key string) string {
	if metadata == nil {
		return ""
	}
	return strings.TrimSpace(metadata[key])
}

func normalizeSlot(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func slugify(value string) string {
	normalized := filenameUnsafePattern.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return "memory"
	}
	return normalized
}

func firstWords(value string, limit int) string {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return ""
	}
	if len(fields) > limit {
		fields = fields[:limit]
	}
	return strings.Join(fields, " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonEmptyAgentTier(values ...memcontract.AgentTier) memcontract.AgentTier {
	for _, value := range values {
		if normalized := value.Normalize(); normalized != "" {
			return normalized
		}
	}
	return ""
}

func confidenceForOp(op memcontract.Op) float32 {
	switch op {
	case memcontract.OpReject:
		return 1.0
	case memcontract.OpNoop:
		return 0.95
	default:
		return 0.9
	}
}

func boundRuleTrace(trace []memcontract.RuleHit) []memcontract.RuleHit {
	if len(trace) == 0 {
		return []memcontract.RuleHit{passedRule("default", "rule path completed", "")}
	}
	return trace
}

func boundString(value string, maxBytes int) string {
	trimmed := strings.TrimSpace(value)
	if maxBytes <= 0 || len(trimmed) <= maxBytes {
		return trimmed
	}
	return strings.TrimSpace(trimmed[:maxBytes])
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
