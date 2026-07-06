package daemon

import (
	"context"
	"fmt"
	"strings"

	automationpkg "github.com/compozy/agh/internal/automation"
	aghconfig "github.com/compozy/agh/internal/config"
	looppkg "github.com/compozy/agh/internal/loop"
	loopdsl "github.com/compozy/agh/internal/loop/dsl"
	toolspkg "github.com/compozy/agh/internal/tools"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

type automationLoopStarter struct {
	service  looppkg.Service
	resolver looppkg.DefinitionResolver
}

func newAutomationLoopStarter(
	storeCandidate any,
	catalog *resourceCatalog[looppkg.ResourceSpec],
	toolRegistry toolspkg.Registry,
	homePaths aghconfig.HomePaths,
	workspaceResolver workspacepkg.RuntimeResolver,
) (automationpkg.LoopStarter, error) {
	loopStore, ok := storeCandidate.(looppkg.Store)
	if !ok || catalog == nil {
		return nil, nil
	}
	schemaSource := newLoopToolSchemaSource(toolRegistry)
	resolver := &daemonLoopDefinitionResolver{
		catalog:  catalog,
		compiler: newLoopCompilerWithSchemaSource(schemaSource),
	}
	service, err := looppkg.NewService(
		loopStore,
		resolver,
		looppkg.WithDefaultsResolver(newLoopDefaultsResolver(homePaths, workspaceResolver)),
	)
	if err != nil {
		return nil, fmt.Errorf("daemon: create automation loop starter: %w", err)
	}
	return &automationLoopStarter{
		service:  service,
		resolver: resolver,
	}, nil
}

func (s *automationLoopStarter) ValidateLoopTarget(
	ctx context.Context,
	req automationpkg.LoopTargetValidationRequest,
) error {
	kind, err := automationStartKindToLoopStartKind(req.Kind)
	if err != nil {
		return err
	}
	return looppkg.ValidateStartTarget(ctx, s.resolver, looppkg.StartTargetValidation{
		WorkspaceID:  looppkg.WorkspaceID(strings.TrimSpace(req.WorkspaceID)),
		LoopName:     strings.TrimSpace(req.LoopName),
		Kind:         kind,
		Inputs:       req.Inputs,
		InputMapping: req.InputMapping,
	})
}

func (s *automationLoopStarter) StartLoop(
	ctx context.Context,
	req automationpkg.LoopStartRequest,
) (automationpkg.LoopStartResult, error) {
	kind, err := automationStartKindToLoopStartKind(req.Kind)
	if err != nil {
		return automationpkg.LoopStartResult{}, err
	}
	workspaceID := looppkg.WorkspaceID(strings.TrimSpace(req.WorkspaceID))
	loopName := strings.TrimSpace(req.LoopName)
	values, err := looppkg.ResolveStartTargetInputs(ctx, s.resolver, looppkg.StartTargetResolution{
		StartTargetValidation: looppkg.StartTargetValidation{
			WorkspaceID:  workspaceID,
			LoopName:     loopName,
			Kind:         kind,
			Inputs:       req.Inputs,
			InputMapping: req.InputMapping,
		},
		TriggerPayload: req.TriggerPayload,
	})
	if err != nil {
		return automationpkg.LoopStartResult{}, err
	}
	run, err := s.service.Start(ctx, workspaceID, loopName, looppkg.Inputs{Values: values}, req.Actor)
	if err != nil {
		return automationpkg.LoopStartResult{}, err
	}
	return automationpkg.LoopStartResult{RunID: string(run.ID)}, nil
}

func automationStartKindToLoopStartKind(kind automationpkg.LoopStartKind) (loopdsl.StartKind, error) {
	switch kind {
	case automationpkg.LoopStartKindTrigger:
		return loopdsl.StartTrigger, nil
	case automationpkg.LoopStartKindSchedule:
		return loopdsl.StartSchedule, nil
	case automationpkg.LoopStartKindWebhook:
		return loopdsl.StartWebhook, nil
	default:
		return "", fmt.Errorf("daemon: unsupported automation loop start kind %q", kind)
	}
}
