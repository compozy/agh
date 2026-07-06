package daemon

import (
	"encoding/json"
	"sort"

	"github.com/compozy/agh/internal/api/contract"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
)

func loopDefinitionPayload(
	spec looppkg.ResourceSpec,
	def dsl.Definition,
) (contract.LoopDefinitionPayload, error) {
	document, err := loopDefinitionDocument(def)
	if err != nil {
		return contract.LoopDefinitionPayload{}, err
	}
	return contract.LoopDefinitionPayload{
		Name:        spec.Name,
		Version:     spec.Version,
		Description: spec.Description,
		Source:      contract.LoopSource(spec.Source.Normalize()),
		Catalog:     loopCatalogResourcePayload(spec.Catalog),
		Definition:  document,
	}, nil
}

func loopCatalogEntryPayload(
	spec looppkg.ResourceSpec,
	def dsl.Definition,
	runs []looppkg.Run,
) (contract.LoopCatalogEntryPayload, error) {
	aggregate := loopCatalogAggregate(runs)
	document, err := loopDefinitionDocument(def)
	if err != nil {
		return contract.LoopCatalogEntryPayload{}, err
	}
	return contract.LoopCatalogEntryPayload{
		Name:          spec.Name,
		Version:       spec.Version,
		Description:   spec.Description,
		Source:        contract.LoopSource(spec.Source.Normalize()),
		Catalog:       loopCatalogResourcePayload(spec.Catalog),
		Inputs:        document.Inputs,
		Start:         document.Start,
		Contract:      document.Contract,
		LastRun:       firstLoopRunPayload(runs),
		Aggregate30d:  aggregate,
		SuccessRate30: loopSuccessRate(aggregate),
	}, nil
}

func loopCatalogResourcePayload(spec looppkg.CatalogResourceSpec) contract.LoopCatalogResourceSpec {
	return contract.LoopCatalogResourceSpec{
		UseWhen:  spec.UseWhen,
		Keywords: append([]string(nil), spec.Keywords...),
		Category: spec.Category,
	}
}

func loopCatalogAggregate(runs []looppkg.Run) contract.LoopCatalogAggregatePayload {
	var aggregate contract.LoopCatalogAggregatePayload
	for _, run := range runs {
		aggregate.Runs++
		switch run.Status {
		case looppkg.StatusDone:
			aggregate.Succeeded++
		case looppkg.StatusFailed, looppkg.StatusBlocked, looppkg.StatusExhausted, looppkg.StatusStalled:
			aggregate.Failed++
		default:
		}
	}
	return aggregate
}

func loopSuccessRate(aggregate contract.LoopCatalogAggregatePayload) float64 {
	if aggregate.Runs == 0 {
		return 0
	}
	return float64(aggregate.Succeeded) / float64(aggregate.Runs)
}

func firstLoopRunPayload(runs []looppkg.Run) *contract.LoopRunPayload {
	if len(runs) == 0 {
		return nil
	}
	payload := loopRunPayload(runs[0])
	return &payload
}

func loopRunPayload(run looppkg.Run) contract.LoopRunPayload {
	return contract.LoopRunPayload{
		ID:                  string(run.ID),
		WorkspaceID:         string(run.WorkspaceID),
		LoopName:            run.LoopName,
		Status:              contract.LoopRunStatus(run.Status),
		Generation:          run.Generation,
		ReattemptStrategy:   contract.LoopReattemptStrategy(run.ReattemptStrategy),
		CreatedAt:           run.CreatedAt,
		LastProgressAt:      run.LastProgressAt,
		StartedByKind:       string(run.StartedBy.Kind),
		StartedByRef:        run.StartedBy.Ref,
		StartedOriginKind:   string(run.StartedOrigin.Kind),
		StartedOriginRef:    run.StartedOrigin.Ref,
		ConsecutiveFailures: run.ConsecutiveFailures,
		IterationCap:        run.IterationCap,
		BudgetTokens:        run.BudgetTokens,
		BudgetWallSec:       run.BudgetWallSec,
		BudgetOnExceeded:    contract.LoopBudgetExceeded(run.BudgetOnExceeded),
		TokensUsed:          run.TokensUsed,
		ParentLoopRunID:     string(run.ParentLoopRunID),
		PauseRequested:      run.PauseRequested,
		Inputs:              cloneLoopAPIMap(run.Inputs),
	}
}

func loopRunsAggregate(runs []looppkg.Run) contract.LoopRunsAggregatePayload {
	var aggregate contract.LoopRunsAggregatePayload
	for _, run := range runs {
		aggregate.Total++
		if run.Status.Live() {
			aggregate.Live++
		}
		if run.Status.Terminal() {
			aggregate.Terminal++
		}
		switch run.Status {
		case looppkg.StatusDone:
			aggregate.Succeeded++
		case looppkg.StatusFailed, looppkg.StatusBlocked, looppkg.StatusExhausted, looppkg.StatusStalled:
			aggregate.Failed++
		default:
		}
	}
	return aggregate
}

func loopRunEventPayload(event looppkg.RunEvent) contract.LoopRunEventPayload {
	return contract.LoopRunEventPayload{
		ID:          event.ID,
		LoopRunID:   string(event.LoopRunID),
		WorkspaceID: string(event.WorkspaceID),
		Seq:         event.Seq,
		Kind:        contract.LoopRunEventKind(event.Kind),
		Payload:     cloneRawMessage(event.Payload),
		At:          event.At,
	}
}

func loopLintErrorsPayload(errors []looppkg.LintError) []contract.LoopLintErrorPayload {
	payloads := make([]contract.LoopLintErrorPayload, 0, len(errors))
	for _, item := range errors {
		payloads = append(payloads, contract.LoopLintErrorPayload{
			NodeID:   string(item.NodeID),
			Code:     item.Code,
			Message:  item.Message,
			Severity: contract.LoopLintSeverity(item.Severity),
		})
	}
	sort.SliceStable(payloads, func(left, right int) bool {
		if payloads[left].NodeID != payloads[right].NodeID {
			return payloads[left].NodeID < payloads[right].NodeID
		}
		return payloads[left].Code < payloads[right].Code
	})
	return payloads
}

func cloneLoopAPIMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	data, err := json.Marshal(input)
	if err != nil {
		return map[string]any{}
	}
	var cloned map[string]any
	if err := json.Unmarshal(data, &cloned); err != nil {
		return map[string]any{}
	}
	return cloned
}

func cloneRawMessage(input json.RawMessage) json.RawMessage {
	if len(input) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), input...)
}
