package settings

import (
	"context"
	"errors"
	"fmt"
	"strings"

	aghconfig "github.com/compozy/agh/internal/config"
)

func (s *service) putMCPServer(
	ctx context.Context,
	scope ScopeKind,
	workspaceID string,
	name string,
	selector TargetSelector,
	server aghconfig.MCPServer,
	secrets MCPSecretValues,
) (MutationResult, error) {
	root, sources, err := s.resolveMCPTargetContext(ctx, scope, workspaceID)
	if err != nil {
		return MutationResult{}, err
	}
	target, err := s.resolveMCPPutTarget(scope, root, name, selector, sources)
	if err != nil {
		return MutationResult{}, err
	}

	normalized := server
	normalized.Name = strings.TrimSpace(normalized.Name)
	if normalized.Name == "" {
		normalized.Name = name
	}
	if normalized.Name != name {
		return MutationResult{}, validationError(fmt.Errorf(
			"settings: MCP server payload name %q does not match request name %q",
			normalized.Name,
			name,
		))
	}
	secretWrites, err := s.prepareMCPSecretWrites(scope, workspaceID, name, &normalized, secrets)
	if err != nil {
		return MutationResult{}, err
	}
	if err := s.validateMCPServerWrite(
		ctx,
		scope,
		workspaceID,
		name,
		target.Kind(),
		sources,
		normalized,
	); err != nil {
		return MutationResult{}, fmt.Errorf("settings: write MCP server %q: %w", name, err)
	}
	secretCleanup, err := s.prepareMCPSecretCleanupPlan(
		ctx, scope, workspaceID, name, target.Kind(), sources, normalized,
	)
	if err != nil {
		return MutationResult{}, err
	}
	secretMutations, err := s.storeMCPSecrets(ctx, secretWrites)
	if err != nil {
		return MutationResult{}, err
	}

	if err := s.withMCPAuthDefinitionMutation(scope, workspaceID, name, func() error {
		return s.writeMCPServer(root, name, target, normalized)
	}); err != nil {
		rollbackErr := s.rollbackMCPSecretMutations(ctx, secretMutations)
		return MutationResult{}, errors.Join(err, rollbackErr)
	}
	if err := s.executeMCPSecretCleanupPlan(ctx, root, name, target, secretCleanup, secretMutations); err != nil {
		return MutationResult{}, err
	}

	return mutationResultForCollection(CollectionMCPServers, scope, workspaceID, target.Kind()), nil
}

func (s *service) writeMCPServer(
	root string,
	name string,
	target aghconfig.WriteTarget,
	server aghconfig.MCPServer,
) error {
	if target.Kind() == WriteTargetGlobalMCPSidecar || target.Kind() == WriteTargetWorkspaceMCPSidecar {
		if _, err := aghconfig.PutMCPSidecarServer(s.homePaths, root, target, server); err != nil {
			return fmt.Errorf("settings: write MCP server %q: %w", name, err)
		}
		return nil
	}
	if _, err := aghconfig.EditConfigOverlay(
		s.homePaths,
		root,
		target,
		func(editor *aghconfig.OverlayEditor) error {
			return editor.UpsertArrayTableItem([]string{"mcp_servers"}, "name", name, mcpServerMap(server))
		},
	); err != nil {
		return fmt.Errorf("settings: write MCP server %q: %w", name, err)
	}
	return nil
}

func (s *service) validateMCPServerWrite(
	ctx context.Context,
	scope ScopeKind,
	workspaceID string,
	name string,
	target WriteTargetKind,
	sources map[string][]mcpSourceEntry,
	server aghconfig.MCPServer,
) error {
	cfg, _, err := s.loadConfig(ctx, scope, workspaceID)
	if err != nil {
		return err
	}
	if projected, ok := projectedMCPServerForValidation(name, target, sources, server); ok {
		cfg.MCPServers = upsertMCPServer(cfg.MCPServers, projected)
	}
	return cfg.Validate()
}

func projectedMCPServerForValidation(
	name string,
	target WriteTargetKind,
	sources map[string][]mcpSourceEntry,
	server aghconfig.MCPServer,
) (aghconfig.MCPServer, bool) {
	entries := sources[strings.TrimSpace(name)]
	if len(entries) == 0 {
		return server, true
	}
	effective := entries[len(entries)-1]
	if (target == WriteTargetGlobalConfig && effective.Target == WriteTargetGlobalMCPSidecar) ||
		(target == WriteTargetWorkspaceConfig && effective.Target == WriteTargetWorkspaceMCPSidecar) {
		return aghconfig.MCPServer{}, false
	}
	return server, true
}

func upsertMCPServer(servers []aghconfig.MCPServer, server aghconfig.MCPServer) []aghconfig.MCPServer {
	name := strings.TrimSpace(server.Name)
	for idx := range servers {
		if strings.TrimSpace(servers[idx].Name) != name {
			continue
		}
		servers[idx] = server
		return servers
	}
	return append(servers, server)
}

func (s *service) deleteMCPServer(
	ctx context.Context,
	scope ScopeKind,
	workspaceID string,
	name string,
	selector TargetSelector,
) (MutationResult, error) {
	root, sources, err := s.resolveMCPTargetContext(ctx, scope, workspaceID)
	if err != nil {
		return MutationResult{}, err
	}
	target, err := s.resolveMCPDeleteTarget(scope, root, name, selector, sources)
	if err != nil {
		return MutationResult{}, err
	}
	deletedSource, sourceFound := mcpSourceForTarget(name, target.Kind(), sources)
	var ownedSecrets []ownedMCPSecretSnapshot
	if sourceFound {
		ownedSecrets, err = s.prepareOwnedMCPSecretDeletes(ctx, scope, workspaceID, deletedSource.Server)
		if err != nil {
			return MutationResult{}, fmt.Errorf("settings: prepare MCP server %q secret cleanup: %w", name, err)
		}
	}

	if err := s.withMCPAuthDefinitionMutation(scope, workspaceID, name, func() error {
		return s.deleteMCPServerDefinition(root, name, target)
	}); err != nil {
		return MutationResult{}, err
	}

	if len(ownedSecrets) > 0 {
		cleanupCtx, cancel := mcpSecretRollbackContext(ctx)
		cleanupErr := s.deleteOwnedMCPSecrets(cleanupCtx, ownedSecrets)
		cancel()
		if cleanupErr != nil {
			restoreErr := s.writeMCPServer(root, name, target, deletedSource.Server)
			if restoreErr != nil {
				restoreErr = fmt.Errorf(
					"settings: restore MCP server %q after secret cleanup failure: %w",
					name,
					restoreErr,
				)
			}
			return MutationResult{}, errors.Join(
				fmt.Errorf("settings: garbage-collect MCP server %q secrets: %w", name, cleanupErr),
				restoreErr,
			)
		}
	}
	return mutationResultForCollection(CollectionMCPServers, scope, workspaceID, target.Kind()), nil
}

func (s *service) deleteMCPServerDefinition(root string, name string, target aghconfig.WriteTarget) error {
	if target.Kind() == WriteTargetGlobalMCPSidecar || target.Kind() == WriteTargetWorkspaceMCPSidecar {
		_, deleted, err := aghconfig.DeleteMCPSidecarServer(s.homePaths, root, target, name)
		if err != nil {
			return fmt.Errorf("settings: delete MCP server %q: %w", name, err)
		}
		if !deleted {
			return notFoundError(fmt.Errorf("settings: MCP server %q not found in %q", name, target.Kind()))
		}
		return nil
	}
	if _, err := aghconfig.EditConfigOverlay(
		s.homePaths,
		root,
		target,
		func(editor *aghconfig.OverlayEditor) error {
			deleted, deleteErr := editor.DeleteArrayTableItem([]string{"mcp_servers"}, "name", name)
			if deleteErr != nil {
				return deleteErr
			}
			if !deleted {
				return notFoundError(
					fmt.Errorf("settings: MCP server %q not found in %q", name, target.Kind()),
				)
			}
			return nil
		},
	); err != nil {
		return fmt.Errorf("settings: delete MCP server %q: %w", name, err)
	}
	return nil
}

func (s *service) withMCPAuthDefinitionMutation(
	scope ScopeKind,
	workspaceID string,
	name string,
	mutate func() error,
) error {
	if mutate == nil {
		return errors.New("settings: MCP definition mutation is required")
	}
	if s.mcpAuth == nil {
		return mutate()
	}
	target, err := normalizeMCPAuthTarget(MCPAuthTargetRequest{
		Scope: scope, WorkspaceID: workspaceID, Name: name,
	})
	if err != nil {
		return err
	}
	if err := s.mcpAuth.MCPAuthInvalidate(target); err != nil {
		return fmt.Errorf("settings: invalidate pending MCP auth before definition mutation: %w", err)
	}
	mutationErr := mutate()
	invalidateErr := s.mcpAuth.MCPAuthInvalidate(target)
	if invalidateErr != nil {
		invalidateErr = fmt.Errorf("settings: invalidate pending MCP auth after definition mutation: %w", invalidateErr)
	}
	return errors.Join(mutationErr, invalidateErr)
}
