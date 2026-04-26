package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/agentidentity"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

func agentCredentialsFromEnv(deps commandDeps) agentidentity.Credentials {
	return agentidentity.Credentials{
		SessionID: strings.TrimSpace(deps.getenv(agentidentity.EnvSessionID)),
		AgentName: strings.TrimSpace(deps.getenv(agentidentity.EnvAgent)),
	}
}

func resolveAgentCallerFromEnv(
	ctx context.Context,
	deps commandDeps,
	client DaemonClient,
	expectedWorkspaceID string,
	originRef string,
) (agentidentity.Caller, error) {
	if client == nil {
		return agentidentity.Resolve(ctx, agentidentity.ResolveOptions{
			Credentials: agentCredentialsFromEnv(deps),
			OriginKind:  taskpkg.OriginKindCLI,
			OriginRef:   originRef,
		})
	}
	return agentidentity.Resolve(ctx, agentidentity.ResolveOptions{
		Credentials:         agentCredentialsFromEnv(deps),
		Lookup:              agentSessionLookup(client),
		ExpectedWorkspaceID: strings.TrimSpace(expectedWorkspaceID),
		OriginKind:          taskpkg.OriginKindCLI,
		OriginRef:           originRef,
	})
}

func agentSessionLookup(client DaemonClient) agentidentity.SessionLookup {
	return func(ctx context.Context, sessionID string) (agentidentity.SessionSnapshot, error) {
		record, err := client.GetSession(ctx, sessionID)
		if err != nil {
			return agentidentity.SessionSnapshot{}, fmt.Errorf("cli: lookup agent session %q: %w", sessionID, err)
		}
		return agentidentity.SessionSnapshot{
			ID:            record.ID,
			Name:          record.Name,
			AgentName:     record.AgentName,
			Provider:      record.Provider,
			WorkspaceID:   record.WorkspaceID,
			WorkspacePath: record.WorkspacePath,
			Channel:       record.Channel,
			Type:          record.Type,
			State:         record.State,
			CreatedAt:     record.CreatedAt,
			UpdatedAt:     record.UpdatedAt,
		}, nil
	}
}

func cliExitCodeForError(err error) int {
	if err == nil {
		return 0
	}
	if code := agentidentity.ExitCodeForError(err); code != agentidentity.ExitUnavailable {
		return code
	}
	return 1
}
