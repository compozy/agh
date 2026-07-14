package core

import (
	"context"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/doctor"
)

func (h *BaseHandlers) doctorPayload(ctx context.Context, opts doctor.RunOptions) (contract.DoctorPayload, error) {
	start := h.nowUTC()
	registry := doctor.NewRegistry()
	if err := registry.Register(&doctor.ProbeFunc{
		ProbeID:       "runtime.status",
		ProbeCategory: contract.CategoryDaemon,
		RunFunc: func(ctx context.Context, _ *doctor.ProbeEnv) ([]contract.DiagnosticItem, error) {
			status, err := h.statusPayload(ctx, "")
			if err != nil {
				return nil, err
			}
			return diagnosticItemsFromStatus(&status, false), nil
		},
	}); err != nil {
		return contract.DoctorPayload{}, err
	}
	if err := registry.Register(&doctor.ProbeFunc{
		ProbeID:       "runtime.providers",
		ProbeCategory: contract.CategoryProvider,
		RunFunc: func(ctx context.Context, _ *doctor.ProbeEnv) ([]contract.DiagnosticItem, error) {
			providers, err := h.providerStatusPayloads(ctx)
			if err != nil {
				return nil, err
			}
			return providerDiagnosticItems(providers), nil
		},
	}); err != nil {
		return contract.DoctorPayload{}, err
	}
	if h.Bridges != nil {
		if err := registry.Register(&doctor.BridgeProbe{Source: h.Bridges}); err != nil {
			return contract.DoctorPayload{}, err
		}
	}

	runner, err := doctor.NewRunner(registry)
	if err != nil {
		return contract.DoctorPayload{}, err
	}
	items, err := runner.Run(ctx, opts)
	if err != nil {
		return contract.DoctorPayload{}, err
	}
	generatedAt := h.nowUTC()
	return contract.DoctorPayload{
		SchemaVersion: contract.StatusSchemaVersion,
		GeneratedAt:   generatedAt,
		DurationMS:    max(generatedAt.Sub(start).Milliseconds(), 0),
		Status:        diagnosticStatus(items),
		Summary:       doctorSummary(items),
		Items:         items,
	}, nil
}
