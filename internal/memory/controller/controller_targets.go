package controller

import (
	"context"
	"errors"

	"fmt"

	"slices"
	"strings"

	memcontract "github.com/compozy/agh/internal/memory/contract"
)

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
	ctx context.Context,
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
		return c.ambiguousDecision(ctx, candidate, matches, trace)
	}
}

func (c *Controller) ambiguousDecision(
	ctx context.Context,
	candidate memcontract.Candidate,
	targets []Target,
	trace []memcontract.RuleHit,
) (memcontract.Decision, error) {
	ambiguousTrace := slices.Clone(trace)
	ambiguousTrace = append(
		ambiguousTrace,
		failedRule("ambiguous_targets", "multiple plausible targets require tiebreaker", targetIDs(targets)),
	)
	if c.tiebreaker == nil {
		return c.rulesOnlyAmbiguousDecision(candidate, targets, ambiguousTrace)
	}
	result, err := c.tiebreaker.BreakTie(ctx, TiebreakerRequest{
		Candidate: candidate,
		Targets:   append([]Target(nil), targets...),
		RuleTrace: append([]memcontract.RuleHit(nil), ambiguousTrace...),
	})
	if errors.Is(err, ErrTiebreakerDisabled) {
		return c.rulesOnlyAmbiguousDecision(candidate, targets, ambiguousTrace)
	}
	if err != nil {
		return c.tiebreakerFailureDecision(candidate, targets, ambiguousTrace, result, err)
	}
	decision, applyErr := c.applyTiebreakerResult(candidate, targets, ambiguousTrace, result)
	if applyErr != nil {
		return c.tiebreakerFailureDecision(candidate, targets, ambiguousTrace, result, applyErr)
	}
	return decision, nil
}

func (c *Controller) rulesOnlyAmbiguousDecision(
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
		trace,
		"ambiguous targets; rules-only fallback selected noop",
		nil,
	)
}

func (c *Controller) applyTiebreakerResult(
	candidate memcontract.Candidate,
	targets []Target,
	trace []memcontract.RuleHit,
	result TiebreakerResult,
) (memcontract.Decision, error) {
	op := result.Op.Normalize()
	if err := op.Validate(); err != nil {
		return memcontract.Decision{}, fmt.Errorf("memory controller: validate tiebreaker op: %w", err)
	}
	target := targetByID(targets, result.TargetID)
	var (
		decision memcontract.Decision
		err      error
	)
	switch op {
	case memcontract.OpAdd:
		decision, err = c.addDecision(candidate, trace)
	case memcontract.OpUpdate:
		if target == nil {
			return memcontract.Decision{}, errors.New(
				"memory controller: update tiebreaker result requires a valid target_id",
			)
		}
		decision, err = c.updateDecision(candidate, *target, trace)
	case memcontract.OpDelete:
		if target == nil {
			return memcontract.Decision{}, errors.New(
				"memory controller: delete tiebreaker result requires a valid target_id",
			)
		}
		decision, err = c.decision(
			candidate,
			memcontract.OpDelete,
			[]Target{*target},
			target.TargetFilename,
			"",
			trace,
			result.Reason,
			target,
		)
	case memcontract.OpNoop:
		selected := targets
		if target != nil {
			selected = []Target{*target}
		}
		decision, err = c.decision(
			candidate,
			memcontract.OpNoop,
			selected,
			targetFilename(candidate),
			"",
			trace,
			result.Reason,
			target,
		)
	case memcontract.OpReject:
		decision, err = c.decision(
			candidate,
			memcontract.OpReject,
			nil,
			"",
			"",
			trace,
			result.Reason,
			nil,
		)
	}
	if err != nil {
		return memcontract.Decision{}, err
	}
	return applyTiebreakerMetadata(decision, result), nil
}

func (c *Controller) tiebreakerFailureDecision(
	candidate memcontract.Candidate,
	targets []Target,
	trace []memcontract.RuleHit,
	result TiebreakerResult,
	cause error,
) (memcontract.Decision, error) {
	call := result.Call
	if call == nil {
		call = &memcontract.LLMCall{PromptVersion: c.promptVersion}
	} else {
		cloned := *call
		call = &cloned
	}
	call.Error = boundString(cause.Error(), maxDecisionReasonBytes)
	result.Call = call
	result.Op = c.defaultOpOnFail
	result.Reason = "tiebreaker failed; configured deterministic fallback selected " + c.defaultOpOnFail.String()
	result.Confidence = confidenceForOp(c.defaultOpOnFail)
	return c.applyTiebreakerResult(candidate, targets, trace, result)
}

func applyTiebreakerMetadata(
	decision memcontract.Decision,
	result TiebreakerResult,
) memcontract.Decision {
	decision.Source = memcontract.SourceLLM
	decision.Confidence = result.Confidence
	decision.Reason = boundString(result.Reason, maxDecisionReasonBytes)
	if result.Call != nil {
		call := *result.Call
		decision.LLMTrace = &call
		if strings.TrimSpace(call.PromptVersion) != "" {
			decision.PromptVersion = strings.TrimSpace(call.PromptVersion)
		}
	}
	decision.IdempotencyKey = IdempotencyKey(decision)
	decision.ID = "dec_" + hashString(decision.IdempotencyKey)[:24]
	return decision
}

func targetByID(targets []Target, targetID string) *Target {
	trimmed := strings.TrimSpace(targetID)
	if trimmed == "" {
		return nil
	}
	for i := range targets {
		if strings.TrimSpace(targets[i].ID) == trimmed {
			return &targets[i]
		}
	}
	return nil
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
