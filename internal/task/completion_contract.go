package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const completionContractMetadataKey = "completion_contract"

type completionContract struct {
	RequiredArtifacts []completionRequiredArtifact `json:"required_artifacts,omitempty"`
	RequiredPaths     []string                     `json:"required_paths,omitempty"`
	MissingPolicy     string                       `json:"missing_policy,omitempty"`
}

type completionRequiredArtifact struct {
	Path string `json:"path"`
}

func (m *Service) validateCompletionContract(ctx context.Context, taskRecord Task, run Run) error {
	if m == nil {
		return nil
	}
	contracts, err := completionContractsFromMetadata(taskRecord.Metadata, run.Metadata)
	if err != nil {
		return err
	}
	if len(contracts) == 0 {
		return nil
	}
	var missing []string
	for _, contract := range contracts {
		if err := validateCompletionMissingPolicy(contract.MissingPolicy); err != nil {
			return err
		}
		required := contract.RequiredArtifactPaths()
		for _, rawPath := range required {
			resolved, err := m.resolveCompletionArtifactPath(ctx, taskRecord, run, rawPath)
			if err != nil {
				return err
			}
			info, err := os.Stat(resolved)
			if err != nil {
				if os.IsNotExist(err) {
					missing = append(missing, rawPath)
					continue
				}
				return fmt.Errorf("%w: completion contract artifact %q: %w", ErrValidation, rawPath, err)
			}
			if info.IsDir() {
				missing = append(missing, rawPath)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"%w: task run %q completion contract missing required artifact(s): %s",
			ErrValidation,
			run.ID,
			strings.Join(missing, ", "),
		)
	}
	return nil
}

func (m *Service) resolveCompletionArtifactPath(
	ctx context.Context,
	taskRecord Task,
	run Run,
	rawPath string,
) (string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", fmt.Errorf("%w: completion contract artifact path is required", ErrValidation)
	}
	if strings.Contains(trimmed, "agh_claim_") {
		return "", fmt.Errorf("%w: completion contract artifact path must not embed a claim token", ErrValidation)
	}
	if filepath.IsAbs(trimmed) {
		return "", fmt.Errorf(
			"%w: completion contract artifact %q must be relative to the workspace root",
			ErrValidation,
			trimmed,
		)
	}
	if m.contractRoot == nil {
		return "", fmt.Errorf(
			"%w: completion contract artifact %q is relative but no workspace root resolver is configured",
			ErrValidation,
			trimmed,
		)
	}
	root, err := m.contractRoot(ctx, taskRecord, run)
	if err != nil {
		return "", fmt.Errorf("%w: resolve completion contract root: %w", ErrValidation, err)
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf(
			"%w: completion contract root is required for relative artifact %q",
			ErrValidation,
			trimmed,
		)
	}
	cleanRel := filepath.Clean(trimmed)
	if cleanRel == "." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) || cleanRel == ".." {
		return "", fmt.Errorf(
			"%w: completion contract artifact %q must stay under workspace root",
			ErrValidation,
			trimmed,
		)
	}
	return filepath.Join(root, cleanRel), nil
}

func completionContractsFromMetadata(rawValues ...json.RawMessage) ([]completionContract, error) {
	var contracts []completionContract
	for _, raw := range rawValues {
		if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" {
			continue
		}
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, fmt.Errorf("%w: decode completion contract metadata: %w", ErrValidation, err)
		}
		payload, ok := envelope[completionContractMetadataKey]
		if !ok || len(payload) == 0 || string(payload) == "null" {
			continue
		}
		contract, err := decodeCompletionContract(payload)
		if err != nil {
			return nil, err
		}
		if len(contract.RequiredArtifactPaths()) > 0 {
			contracts = append(contracts, contract)
		}
	}
	return contracts, nil
}

func decodeCompletionContract(raw json.RawMessage) (completionContract, error) {
	var contract completionContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		return completionContract{}, fmt.Errorf("%w: decode completion contract: %w", ErrValidation, err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return completionContract{}, fmt.Errorf("%w: decode completion contract fields: %w", ErrValidation, err)
	}
	if rawArtifacts, ok := envelope["required_artifacts"]; ok {
		paths, err := decodeCompletionRequiredArtifacts(rawArtifacts)
		if err != nil {
			return completionContract{}, err
		}
		contract.RequiredArtifacts = paths
	}
	return contract, nil
}

func decodeCompletionRequiredArtifacts(raw json.RawMessage) ([]completionRequiredArtifact, error) {
	var asObjects []completionRequiredArtifact
	if err := json.Unmarshal(raw, &asObjects); err == nil {
		return asObjects, nil
	}
	var asStrings []string
	if err := json.Unmarshal(raw, &asStrings); err == nil {
		artifacts := make([]completionRequiredArtifact, 0, len(asStrings))
		for _, path := range asStrings {
			artifacts = append(artifacts, completionRequiredArtifact{Path: path})
		}
		return artifacts, nil
	}
	return nil, fmt.Errorf("%w: completion_contract.required_artifacts must be an array", ErrValidation)
}

func (c completionContract) RequiredArtifactPaths() []string {
	paths := make([]string, 0, len(c.RequiredArtifacts)+len(c.RequiredPaths))
	for _, artifact := range c.RequiredArtifacts {
		if trimmed := strings.TrimSpace(artifact.Path); trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	for _, path := range c.RequiredPaths {
		if trimmed := strings.TrimSpace(path); trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	return paths
}

func validateCompletionMissingPolicy(policy string) error {
	switch strings.TrimSpace(policy) {
	case "", "reject":
		return nil
	default:
		return fmt.Errorf("%w: completion_contract.missing_policy must be reject", ErrValidation)
	}
}

func validateActiveLeasePreconditions(run Run, rawToken string, now time.Time) error {
	if strings.TrimSpace(run.ClaimTokenHash) == "" {
		return fmt.Errorf("%w: task run %q has no current claim token hash", ErrInvalidClaimToken, run.ID)
	}
	if !VerifyClaimToken(rawToken, run.ClaimTokenHash) {
		return fmt.Errorf("%w: task run %q token mismatch", ErrInvalidClaimToken, run.ID)
	}
	switch run.Status.Normalize() {
	case TaskRunStatusClaimed, TaskRunStatusStarting, TaskRunStatusRunning:
	default:
		return fmt.Errorf("%w: task run %q is not actively leased", ErrInvalidStatusTransition, run.ID)
	}
	if run.LeaseUntil.IsZero() || !run.LeaseUntil.After(now.UTC()) {
		return fmt.Errorf("%w: task run %q lease expired", ErrLeaseExpired, run.ID)
	}
	return nil
}
